// Package razorcrest is the optional authenticated remote boundary for Mandalore.
package razorcrest

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Config struct {
	Binding          string     `json:"binding"`
	BindingSHA256    string     `json:"binding_sha256"`
	SignetID         string     `json:"signet_id"`
	Listen           string     `json:"listen"`
	Hosts            []string   `json:"hosts"`
	Origins          []string   `json:"origins"`
	OriginSecretFile string     `json:"origin_secret_file"`
	Auth             AuthConfig `json:"auth"`
	Synchronization  bool       `json:"synchronization"`
	CanonFoundlings  []string   `json:"canon_foundlings,omitempty"`
}

type canonSession struct {
	subject string
	expires time.Time
}
type Service struct {
	config   Config
	memory   *memory.Service
	api      *api.API
	auth     IdentityProvider
	secret   []byte
	mu       sync.Mutex // serialize foreground memory, canon and background Git operations
	sessions map[string]canonSession
	handlers map[string]http.Handler
}

func New(c Config) (*Service, error) {
	if c.Binding == "" || c.BindingSHA256 == "" || c.SignetID == "" || len(c.Hosts) == 0 {
		return nil, errors.New("explicit guarded binding and allowed hosts required")
	}
	if len(c.CanonFoundlings) > 10 || len(c.Auth.Subjects) > 64 {
		return nil, errors.New("service configuration exceeds bounded grants")
	}
	if _, _, e := net.SplitHostPort(c.Listen); e != nil {
		return nil, errors.New("explicit listen host and port required")
	}
	for _, origin := range c.Origins {
		u, e := url.Parse(origin)
		if e != nil || u.Scheme != "https" || u.User != nil || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.RawPath != "" || u.Fragment != "" {
			return nil, errors.New("origins must be HTTPS origins")
		}
	}
	for _, host := range c.Hosts {
		if host == "" || strings.ContainsAny(host, "/\\ \t\n") {
			return nil, errors.New("invalid host allowlist")
		}
	}
	secret, e := os.ReadFile(c.OriginSecretFile)
	if e != nil {
		return nil, errors.New("origin secret unavailable")
	}
	secret = []byte(strings.TrimSpace(string(secret)))
	if len(secret) < 32 || len(secret) > 256 {
		return nil, errors.New("origin secret requires 32–256 bytes")
	}
	a, e := NewAuthenticator(c.Auth)
	if e != nil {
		return nil, e
	}
	m, e := binding.OpenGuarded(c.Binding, "razor-crest", binding.Guard{SHA256: c.BindingSHA256, SignetID: c.SignetID})
	if e != nil {
		return nil, errors.New("service binding invalid")
	}
	s := &Service{config: c, memory: m, api: api.New(m, false), auth: a, secret: secret, sessions: map[string]canonSession{}, handlers: map[string]http.Handler{}}
	for sub, g := range c.Auth.Subjects {
		p := Principal{Subject: sub, Read: g.Read, Write: g.Write}
		server := s.server(p)
		s.handlers[sub] = sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, &sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: 65536, DisableLocalhostProtection: true, PropagateRequestCancellation: true})
	}
	return s, nil
}

func contains(values []string, v string) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path == "/healthz" {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			if net.ParseIP(host).IsLoopback() && r.Method == "GET" {
				w.WriteHeader(200)
				_, _ = w.Write([]byte("alive\n"))
				return
			}
			http.Error(w, "not found", 404)
			return
		}
		if r.URL.Path != "/mcp" || !contains(s.config.Hosts, r.Host) {
			http.Error(w, "not found", 404)
			return
		}
		if len(r.Header.Values("Origin")) > 1 {
			http.Error(w, "forbidden", 403)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && !contains(s.config.Origins, origin) {
			http.Error(w, "forbidden", 403)
			return
		}
		if len(r.Header.Values("X-Razor-Crest-Origin")) != 1 || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Razor-Crest-Origin")), s.secret) != 1 {
			http.Error(w, "forbidden", 403)
			return
		}
		p, e := s.auth.Authenticate(r)
		if e != nil {
			http.Error(w, "unauthorized", 401)
			return
		}
		h, ok := s.handlers[p.Subject]
		if !ok || !p.Read {
			http.Error(w, "forbidden", 403)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// Run performs bounded startup, periodic and foreground synchronization. The
// persistent replica is never replaced or cloned automatically during recovery.
func (s *Service) Run(ctx context.Context) error {
	server := &http.Server{Addr: s.config.Listen, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			s.mu.Lock()
			s.delivery(ctx)
			s.mu.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-ticker.C:
			}
		}
	}()
	go func() {
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdown)
		case <-done:
		}
	}()
	e := server.ListenAndServe()
	if errors.Is(e, http.ErrServerClosed) {
		return nil
	}
	return e
}
func (s *Service) delivery(ctx context.Context) api.Envelope {
	if !s.config.Synchronization {
		return api.Success(map[string]string{"state": "disabled"})
	}
	return s.api.Call(ctx, "memory_sync", []byte(`{"timeout_seconds":3}`))
}

var readTools = map[string]bool{"memory_scopes": true, "memory_recall": true, "memory_history": true, "memory_journal": true, "memory_visibility_history": true, "memory_sync_status": true, "foundling_canon_scopes": true, "foundling_canon_recall": true, "foundling_canon_heads": true}

func (s *Service) server(p Principal) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: "mandalore-razor-crest", Version: "1"}, &sdk.ServerOptions{Instructions: remoteInstructions(p.Write, len(s.config.CanonFoundlings) > 0)})
	for _, op := range api.Catalog() {
		if !readTools[op.Name] {
			continue
		}
		if strings.HasPrefix(op.Name, "foundling_") && len(s.config.CanonFoundlings) == 0 {
			continue
		}
		server.AddTool(&sdk.Tool{Name: op.Name, Description: remoteDescription(op), InputSchema: remoteSchema(op.InputSchema), Annotations: &sdk.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			return result(s.call(ctx, p, op.Name, req.Params.Arguments)), nil
		})
	}
	add := func(name, description string, input any) {
		schema, e := strictjson.Schema(input)
		if e != nil {
			panic(e)
		}
		server.AddTool(&sdk.Tool{Name: name, Description: description, InputSchema: remoteSchema(schema)}, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			return result(s.call(ctx, p, name, req.Params.Arguments)), nil
		})
	}
	if p.Write {
		add("memory_remember", "Remember confirmed preferences, decisions and project facts in Mandalore for future conversations across agents. Use for requests to remember and confirmed useful learning; honor do-not-remember requests. Recall first to avoid duplicates; corrections preserve record_id, kind and scope and supersede current revision IDs. Never save secrets or transcripts. Generate one unique request_id per new save; retry ambiguous responses with identical input and the same key. Inspect saved and delivery separately.", new(RememberInput))
		add("memory_journal_append", "Record a concise useful work outcome or decision trail in Mandalore, not a transcript or a substitute for current facts. Honor no-journal requests; never save secrets. Generate one unique request_id per new entry; retry ambiguous responses with identical input and the same key. Inspect saved and delivery separately.", new(JournalInput))
	}
	if len(s.config.CanonFoundlings) > 0 {
		add("razor_session_open", "Create an isolated canon snapshot for this conversation.", new(struct{}))
		add("foundling_refresh", "Explicitly advance only this conversation's authorized canon snapshots.", new(api.CanonSessionInput))
		add("foundling_status", "Inspect only this conversation's authorized canon snapshots.", new(api.CanonSessionInput))
	}
	return server
}
func result(v api.Envelope) *sdk.CallToolResult {
	b, e := json.Marshal(v)
	if e != nil || len(b) > api.MaxOutputBytes {
		return &sdk.CallToolResult{IsError: true, Content: []sdk.Content{&sdk.TextContent{Text: "Response encoding failed"}}}
	}
	return &sdk.CallToolResult{IsError: !v.OK, StructuredContent: v, Content: []sdk.Content{&sdk.TextContent{Text: string(b)}}}
}

type RememberInput struct {
	RequestID string       `json:"request_id" jsonschema:"Unique 16–128 character key generated once per new save; reuse unchanged with identical input after an ambiguous response. A new key creates a new operation."`
	Record    memory.Write `json:"record"`
}
type JournalInput struct {
	RequestID string           `json:"request_id" jsonschema:"Unique 16–128 character key generated once per new entry; reuse unchanged with identical input after an ambiguous response. A new key creates a new operation."`
	Entry     api.JournalWrite `json:"entry"`
}

func requestKey(subject, id string) (string, error) {
	if len(id) < 16 || len(id) > 128 || strings.TrimSpace(id) != id {
		return "", errors.New("request_id requires 16–128 characters")
	}
	h := sha256.Sum256([]byte(subject + "\x00" + id))
	return hex.EncodeToString(h[:]), nil
}
func (s *Service) call(ctx context.Context, p Principal, name string, raw []byte) api.Envelope {
	if !p.Read {
		return api.Failure("access.denied", "Access denied.", false)
	}
	if len(raw) == 0 {
		raw = []byte(`{}`)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil {
		return api.Failure("operation.cancelled", "Cancelled before execution.", false)
	}
	if s.memory.ID() != s.config.SignetID {
		return api.Failure("binding.invalid", "Service identity changed.", false)
	}
	if name == "memory_remember" || name == "memory_journal_append" {
		if !p.Write {
			return api.Failure("access.denied", "Writes denied.", false)
		}
		var receipt api.Receipt
		var e error
		if name == "memory_remember" {
			var in RememberInput
			if strictjson.Decode(raw, &in, api.MaxInputBytes) != nil {
				return api.Failure("input.invalid", "Invalid write input.", false)
			}
			key, err := requestKey(s.config.Auth.Issuer+"\x00"+p.Subject, in.RequestID)
			if err != nil {
				return api.Failure("input.invalid", err.Error(), false)
			}
			var r memory.Revision
			r, e = s.memory.RememberIdempotent(key, in.Record)
			receipt = api.Receipt{SignetID: s.memory.ID(), ID: r.ID, RecordID: r.RecordID, DurableLocally: e == nil, Synchronization: "pending"}
		} else {
			var in JournalInput
			if strictjson.Decode(raw, &in, api.MaxInputBytes) != nil {
				return api.Failure("input.invalid", "Invalid journal input.", false)
			}
			key, err := requestKey(s.config.Auth.Issuer+"\x00"+p.Subject, in.RequestID)
			if err != nil {
				return api.Failure("input.invalid", err.Error(), false)
			}
			var r memory.JournalEntry
			r, e = s.memory.AppendJournalIdempotent(key, in.Entry.Kind, in.Entry.Summary)
			receipt = api.Receipt{SignetID: s.memory.ID(), ID: r.ID, DurableLocally: e == nil, Synchronization: "pending"}
		}
		if e != nil {
			return api.Failure("save.incomplete", "Save failed or incomplete; retry only identical input with the same request_id.", true)
		}
		delivery := s.delivery(ctx)
		if !s.config.Synchronization {
			receipt.Synchronization = "disabled"
		}
		if status, ok := delivery.Result.(signetsync.Status); ok {
			receipt.Synchronization = status.State
		}
		return api.Success(struct {
			Saved    api.Receipt  `json:"saved"`
			Delivery api.Envelope `json:"delivery"`
		}{receipt, delivery})
	}
	if name == "razor_session_open" {
		var in struct{}
		if strictjson.Decode(raw, &in, api.MaxInputBytes) != nil {
			return api.Failure("input.invalid", "Invalid session input.", false)
		}
		for id, x := range s.sessions {
			if time.Now().After(x.expires) {
				delete(s.sessions, id)
			}
		}
		if len(s.sessions) >= 1024 {
			return api.Failure("session.capacity", "Session capacity exceeded.", false)
		}
		id := memory.NewID("razor")
		s.sessions[id] = canonSession{subject: p.Subject, expires: time.Now().Add(24 * time.Hour)}
		receipts, e := foundlings.New(s.memory).SessionRefreshAllowed(ctx, id, s.config.CanonFoundlings)
		if e != nil {
			return api.Failure("session.unavailable", "Cannot create canon session.", false)
		}
		return api.Success(struct {
			SessionID  string                    `json:"session_id"`
			References []foundlings.CanonReceipt `json:"references"`
		}{id, receipts})
	}
	if strings.HasPrefix(name, "foundling_") {
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil {
			return api.Failure("input.invalid", "Invalid canon input.", false)
		}
		var id, ref string
		_ = json.Unmarshal(fields["session_id"], &id)
		_ = json.Unmarshal(fields["foundling_id"], &ref)
		session, ok := s.sessions[id]
		if !ok || session.subject != p.Subject || time.Now().After(session.expires) {
			return api.Failure("access.denied", "Unknown or expired canon session.", false)
		}
		if name == "foundling_refresh" || name == "foundling_status" {
			var in api.CanonSessionInput
			if strictjson.Decode(raw, &in, api.MaxInputBytes) != nil {
				return api.Failure("input.invalid", "Invalid canon session input.", false)
			}
			var receipts []foundlings.CanonReceipt
			var e error
			if name == "foundling_refresh" {
				receipts, e = foundlings.New(s.memory).SessionRefreshAllowed(ctx, id, s.config.CanonFoundlings)
			} else {
				receipts, e = foundlings.New(s.memory).SessionStatus(ctx, id)
			}
			if e != nil {
				return api.Failure("session.unavailable", "Canon state unavailable.", false)
			}
			filtered := []foundlings.CanonReceipt{}
			for _, r := range receipts {
				if contains(s.config.CanonFoundlings, r.FoundlingID) {
					filtered = append(filtered, r)
				}
			}
			return api.Success(filtered)
		}
		if !contains(s.config.CanonFoundlings, ref) {
			return api.Failure("access.denied", "Reference denied.", false)
		}
	}
	if !readTools[name] {
		return api.Failure("access.denied", "Operation denied.", false)
	}
	delivery := s.delivery(ctx)
	v := s.api.Call(ctx, name, raw)
	// Return both freshness and semantic results without claiming a failed fetch
	// changed the snapshot or erased evidence already supplied to a model.
	response := api.Success(struct {
		Memory   api.Envelope `json:"memory"`
		Delivery api.Envelope `json:"delivery"`
	}{v, delivery})
	if !v.OK {
		return v
	}
	return response
}
