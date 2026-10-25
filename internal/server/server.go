package server

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Server struct {
	config  Config
	log     *log.Logger
	signets map[string]*signet // by MCP path
	hosts   map[string]bool
	origins map[string]bool
}

var defaultTiming = timing{debounce: 2 * time.Second, retry: time.Minute, push: 30 * time.Second}

// New opens every configured signet. It does not touch the network; Start
// checks the remotes.
func New(c Config, logs io.Writer) (*Server, error) {
	return newServer(c, logs, defaultTiming)
}

func newServer(c Config, logs io.Writer, t timing) (*Server, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	s := &Server{config: c, log: log.New(logs, "mandalore: ", log.LstdFlags), signets: map[string]*signet{}, hosts: map[string]bool{}, origins: map[string]bool{c.origin(): true}}
	public, _ := url.Parse(c.PublicURL)
	_, port, _ := net.SplitHostPort(c.Listen)
	s.hosts[strings.ToLower(public.Host)] = true
	for _, host := range []string{"127.0.0.1", "localhost", "[::1]"} {
		s.hosts[host+":"+port] = true
		s.origins["http://"+host+":"+port] = true
	}
	for _, sc := range c.Signets {
		g, err := openSignet(c, sc, s.log, t)
		if err != nil {
			return nil, err
		}
		s.signets[c.mcpPath(sc.Slug)] = g
	}
	return s, nil
}

// Start checks each remote, blocks any signet whose remote is ahead, and
// starts backup delivery for the rest. It returns when the checks finish.
func (s *Server) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for _, g := range s.signets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.startup(ctx)
		}()
	}
	wg.Wait()
	for _, g := range s.signets {
		if !g.blocked {
			go g.deliver(ctx)
		}
	}
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path == "/healthz" {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() && r.Method == http.MethodGet {
				_, _ = io.WriteString(w, "alive\n")
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		g, ok := s.signets[r.URL.Path]
		if !ok || !s.hosts[strings.ToLower(r.Host)] {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if origins := r.Header.Values("Origin"); len(origins) > 1 || (len(origins) == 1 && !s.origins[origins[0]]) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		c, err := g.auth.Authenticate(r)
		if errors.Is(err, ErrForbidden) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err != nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if g.blocked {
			http.Error(w, "signet unavailable: remote ahead; the operator must run mandalore server reconcile", http.StatusServiceUnavailable)
			return
		}
		g.mu.Lock()
		h := g.handler(c)
		g.mu.Unlock()
		h.ServeHTTP(w, r)
	})
}

// Run serves until ctx ends, then makes one last bounded push of pending
// backups so a clean stop leaves the remotes current.
func (s *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.config.Listen)
	if err != nil {
		return err
	}
	delivery, stop := context.WithCancel(context.Background())
	defer stop()
	s.Start(delivery)
	server := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	s.log.Printf("serving %d signet(s) on %s", len(s.signets), s.config.Listen)
	select {
	case err = <-served:
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = server.Shutdown(shutdown)
		cancel()
	}
	stop()
	final, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, g := range s.signets {
		g.backupMu.Lock()
		pending := g.backup.Pending && !g.blocked
		g.backupMu.Unlock()
		if pending {
			g.push(final)
		}
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
