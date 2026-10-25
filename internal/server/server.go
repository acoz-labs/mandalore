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

	"github.com/modelcontextprotocol/go-sdk/auth"
)

type Server struct {
	config  Config
	log     *log.Logger
	signets map[string]*signet // by MCP path
	hosts   map[string]bool
	origins map[string]bool
	oauth   *authServer
	// refreshing tracks the canon refresh loops Start runs.
	refreshing sync.WaitGroup
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
	if t.canon == 0 {
		t.canon = c.canonInterval()
	}
	s := &Server{config: c, log: log.New(logs, "mandalore: ", log.LstdFlags), signets: map[string]*signet{}, hosts: map[string]bool{}, origins: map[string]bool{c.origin(): true}}
	public, _ := url.Parse(c.PublicURL)
	_, port, _ := net.SplitHostPort(c.Listen)
	s.hosts[strings.ToLower(public.Host)] = true
	for _, host := range []string{"127.0.0.1", "localhost", "[::1]"} {
		s.hosts[host+":"+port] = true
		s.origins["http://"+host+":"+port] = true
	}
	bySlug := map[string]*signet{}
	for _, sc := range c.Signets {
		g, err := openSignet(c, sc, s.log, t)
		if err != nil {
			return nil, err
		}
		s.signets[c.mcpPath(sc.Slug)], bySlug[sc.Slug] = g, g
	}
	if c.auth() == AuthOAuth {
		a, err := newAuthServer(c, bySlug, s.log, time.Now)
		if err != nil {
			return nil, err
		}
		s.oauth = a
	}
	return s, nil
}

// Start checks each remote, blocks any signet whose remote is ahead, and
// starts backup delivery and canon refresh for the rest. It returns when the
// remote checks finish; the first canon refresh runs in the background.
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
			s.refreshing.Add(1)
			go func() {
				defer s.refreshing.Done()
				g.refreshCanon(ctx)
			}()
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
		var endpoint http.Handler
		if s.oauth != nil {
			endpoint = s.oauth.endpoints[r.URL.Path]
		}
		g, ok := s.signets[r.URL.Path]
		if (!ok && endpoint == nil) || !s.hosts[strings.ToLower(r.Host)] {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		// Clients call the token, revocation, registration and metadata
		// endpoints from their own origins; every other route takes requests
		// from this one.
		if origins := r.Header.Values("Origin"); len(origins) > 1 || (len(origins) == 1 && !s.origins[origins[0]] && !crossOrigin(r.URL.Path)) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if endpoint != nil {
			endpoint.ServeHTTP(w, r)
			return
		}
		if g.protected != nil {
			g.protected.ServeHTTP(w, r)
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
		g.serve(c, w, r)
	})
}

func crossOrigin(path string) bool {
	return path == "/token" || path == "/revoke" || path == "/register" || strings.HasPrefix(path, "/.well-known/")
}

// serve runs an authenticated client's MCP request.
func (g *signet) serve(c Client, w http.ResponseWriter, r *http.Request) {
	if g.blocked {
		http.Error(w, "signet unavailable: remote ahead; the operator must run mandalore server reconcile", http.StatusServiceUnavailable)
		return
	}
	g.mu.Lock()
	h := g.handler(c)
	g.mu.Unlock()
	h.ServeHTTP(w, r)
}

// authorized serves MCP for the client a bearer token resolved to.
func (g *signet) authorized() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := auth.TokenInfoFromContext(r.Context())
		c, ok := info.Extra["client"].(Client)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		g.serve(c, w, r)
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
	// A refresh in flight finishes recording its receipts, within its budget.
	s.refreshing.Wait()
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
