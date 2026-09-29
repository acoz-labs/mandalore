package razorcrest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type serviceRoundTripper func(*http.Request) (*http.Response, error)

func (f serviceRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func serviceFixture(t *testing.T, write bool) (*Service, string, string) {
	t.Helper()
	root := t.TempDir()
	store, err := memory.Create(filepath.Join(root, "signet"), "Example", "device-fixture", "Fixture")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "binding.json")
	b, err := binding.Bind(store.Root, path, "Remote test device", "Example actor")
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(bytes)
	secret := strings.Repeat("fixture-secret-", 3)
	secretFile := filepath.Join(root, "origin-secret")
	if err := os.WriteFile(secretFile, []byte(secret), 0600); err != nil {
		t.Fatal(err)
	}
	auth, key, _, _, _, _ := newAuthFixture(t)
	auth.config.Subjects["test-subject"] = Grant{Read: true, Write: write}
	c := Config{Binding: path, BindingSHA256: hex.EncodeToString(digest[:]), SignetID: b.SignetID, Listen: "127.0.0.1:0", Hosts: []string{"memory.example"}, Origins: []string{"https://client.example"}, OriginSecretFile: secretFile, Auth: auth.config}
	service, err := New(c)
	if err != nil {
		t.Fatal(err)
	}
	service.auth = auth
	return service, signAssertion(t, key, "first", authClaims(auth)), secret
}
func serviceClient(t *testing.T, s *Service, token, secret string) (context.Context, *sdk.ClientSession) {
	t.Helper()
	server := httptest.NewServer(s.Handler())
	t.Cleanup(server.Close)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: serviceRoundTripper(func(r *http.Request) (*http.Response, error) {
		copy := r.Clone(r.Context())
		copy.Host = "memory.example"
		copy.Header.Set("Cf-Access-Jwt-Assertion", token)
		copy.Header.Set("X-Razor-Crest-Origin", secret)
		return transport.RoundTrip(copy)
	})}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	session, err := sdk.NewClient(&sdk.Implementation{Name: "integration-fixture", Version: "1"}, nil).Connect(ctx, &sdk.StreamableClientTransport{Endpoint: server.URL + "/mcp", HTTPClient: client, MaxRetries: -1, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return ctx, session
}
func toolText(t *testing.T, r *sdk.CallToolResult) string {
	t.Helper()
	var out strings.Builder
	for _, c := range r.Content {
		if text, ok := c.(*sdk.TextContent); ok {
			out.WriteString(text.Text)
		}
	}
	return out.String()
}
func rememberArguments() map[string]any {
	return map[string]any{"request_id": "fixture-request-0001", "record": map[string]any{"kind": "fact", "summary": "Synthetic remote convention", "body": "Use synthetic fixtures for remote acceptance.", "basis": "observation", "reason": "Integration fixture"}}
}

func TestServiceStreamableHTTPReadsWritesAndRetries(t *testing.T) {
	s, token, secret := serviceFixture(t, true)
	ctx, client := serviceClient(t, s, token, secret)
	list, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range list.Tools {
		names[tool.Name] = true
		if !readTools[tool.Name] && tool.Name != "memory_remember" && tool.Name != "memory_journal_append" {
			t.Fatalf("unexpected remote operation %s", tool.Name)
		}
	}
	if !names["memory_recall"] || !names["memory_remember"] || names["foundling_canon_recall"] {
		t.Fatal("wrong exposed tool set", names)
	}
	call := func(name string, args any) *sdk.CallToolResult {
		t.Helper()
		r, err := client.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	first := call("memory_remember", rememberArguments())
	if first.IsError {
		t.Fatal(toolText(t, first))
	}
	duplicate := call("memory_remember", rememberArguments())
	if duplicate.IsError || toolText(t, first) != toolText(t, duplicate) {
		t.Fatal("identical retry changed receipt", toolText(t, duplicate))
	}
	changed := rememberArguments()
	changed["record"].(map[string]any)["body"] = "Different input must be rejected"
	if !call("memory_remember", changed).IsError {
		t.Fatal("same key with changed input accepted")
	}
	recalled := call("memory_recall", map[string]any{})
	text := toolText(t, recalled)
	if recalled.IsError || !strings.Contains(text, "Synthetic remote convention") {
		t.Fatal("saved memory missing", text)
	}
	if strings.Contains(text, s.memory.Root()) || strings.Contains(text, s.config.Binding) || strings.Contains(text, secret) {
		t.Fatal("remote output leaked local deployment details")
	}
	if strings.Count(text, "Synthetic remote convention") != 1 {
		t.Fatal("retry duplicated memory", text)
	}
	status := call("memory_sync_status", map[string]any{})
	statusText := toolText(t, status)
	if strings.Contains(statusText, s.memory.Root()) || strings.Contains(statusText, s.config.Binding) || strings.Contains(statusText, secret) {
		t.Fatal("sync status leaked deployment details", statusText)
	}
	journalArgs := map[string]any{"request_id": "fixture-journal-0001", "entry": map[string]any{"kind": "outcome", "summary": "Synthetic remote outcome"}}
	journal := call("memory_journal_append", journalArgs)
	if journal.IsError {
		t.Fatal("journal write failed", toolText(t, journal))
	}
	journalRetry := call("memory_journal_append", journalArgs)
	if journalRetry.IsError || toolText(t, journalRetry) != toolText(t, journal) {
		t.Fatal("journal retry changed receipt")
	}
	admin, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "signet_create", Arguments: map[string]any{}})
	if err == nil && !admin.IsError {
		t.Fatal("admin operation exposed")
	}
	// Persistent retries still return the original receipt after service recreation.
	restarted, err := New(s.config)
	if err != nil {
		t.Fatal(err)
	}
	restarted.auth = s.auth
	ctx2, client2 := serviceClient(t, restarted, token, secret)
	retried, err := client2.CallTool(ctx2, &sdk.CallToolParams{Name: "memory_remember", Arguments: rememberArguments()})
	if err != nil || retried.IsError || toolText(t, retried) != toolText(t, first) {
		t.Fatal("restart lost idempotent receipt", err)
	}
}
func TestServiceReadOnlyAndForeignSignetDenied(t *testing.T) {
	s, token, secret := serviceFixture(t, false)
	ctx, client := serviceClient(t, s, token, secret)
	list, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range list.Tools {
		if tool.Name == "memory_remember" || tool.Name == "memory_journal_append" {
			t.Fatal("read-only principal offered write")
		}
	}
	result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "memory_remember", Arguments: rememberArguments()})
	if err == nil && !result.IsError {
		t.Fatal("read-only write succeeded")
	}
	// Exercise enforcement underneath discovery as well as the SDK's allowlist.
	raw, _ := json.Marshal(rememberArguments())
	if s.call(ctx, Principal{Subject: "test-subject", Read: true}, "memory_remember", raw).OK {
		t.Fatal("server grant bypass")
	}
	writer, writerToken, writerSecret := serviceFixture(t, true)
	ctx2, client2 := serviceClient(t, writer, writerToken, writerSecret)
	args := rememberArguments()
	args["record"].(map[string]any)["scope"] = map[string]string{"kind": "signet", "id": "signet-unrelated"}
	result, err = client2.CallTool(ctx2, &sdk.CallToolParams{Name: "memory_remember", Arguments: args})
	if err == nil && !result.IsError {
		t.Fatal("foreign signet accepted")
	}
}
func TestServiceHTTPBoundary(t *testing.T) {
	s, token, secret := serviceFixture(t, true)
	for _, tc := range []struct {
		name, host, origin, secret, token, path string
		status                                  int
	}{
		{"valid boundary", "memory.example", "", secret, token, "/mcp", 0},
		{"absent assertion", "memory.example", "", secret, "", "/mcp", 401},
		{"forged assertion", "memory.example", "", secret, token + "forged", "/mcp", 401},
		{"missing origin secret", "memory.example", "", "", token, "/mcp", 403},
		{"wrong origin secret", "memory.example", "", "incorrect", token, "/mcp", 403},
		{"browser origin", "memory.example", "https://untrusted.example", secret, token, "/mcp", 403},
		{"host", "untrusted.example", "", secret, token, "/mcp", 404},
		{"route", "memory.example", "", secret, token, "/admin", 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "https://"+tc.host+tc.path, strings.NewReader(`{}`))
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("X-Razor-Crest-Origin", tc.secret)
			r.Header.Set("Cf-Access-Jwt-Assertion", tc.token)
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, r)
			if tc.status != 0 && w.Code != tc.status {
				t.Fatalf("status %d want %d", w.Code, tc.status)
			}
			if tc.status == 0 && (w.Code == 401 || w.Code == 403 || w.Code == 404) {
				t.Fatalf("valid boundary denied %d", w.Code)
			}
			if strings.Contains(w.Body.String(), s.memory.Root()) || strings.Contains(w.Body.String(), secret) {
				t.Fatal("boundary leaked deployment data")
			}
		})
	}
}

func TestServiceForeignSignetReadDenied(t *testing.T) {
	s, token, secret := serviceFixture(t, true)
	ctx, client := serviceClient(t, s, token, secret)
	// Seed a real second signet to distinguish denial from an empty result.
	other, err := memory.Create(filepath.Join(t.TempDir(), "other"), "Other", "device-other", "Other")
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := memory.OpenService(other.Root, memory.Authorship{DeviceID: "device-other", Actor: "Fixture", Harness: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = foreign.Remember(memory.Write{Kind: "fact", Summary: "Foreign marker must stay private", Body: "Synthetic foreign memory", Basis: "observation", Reason: "Fixture"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "memory_recall", Arguments: map[string]any{"scope": map[string]string{"kind": "signet", "id": foreign.ID()}}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("foreign read should fail explicitly", toolText(t, result))
	}
	if strings.Contains(toolText(t, result), "Foreign marker") {
		t.Fatal("foreign signet memory leaked")
	}
}
func TestServiceDuplicateOriginHeadersDenied(t *testing.T) {
	s, token, secret := serviceFixture(t, true)
	for _, header := range []string{"Origin", "X-Razor-Crest-Origin"} {
		t.Run(header, func(t *testing.T) {
			r := httptest.NewRequest("POST", "https://memory.example/mcp", strings.NewReader(`{}`))
			r.Header.Set("Cf-Access-Jwt-Assertion", token)
			r.Header.Set("X-Razor-Crest-Origin", secret)
			r.Header.Set("Origin", "https://client.example")
			r.Header.Add(header, r.Header.Get(header))
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, r)
			if w.Code != 403 {
				t.Fatalf("duplicate %s accepted: %d", header, w.Code)
			}
		})
	}
}
func TestServiceRejectsInvalidConfiguredOrigins(t *testing.T) {
	s, _, _ := serviceFixture(t, true)
	for _, origin := range []string{"http://client.example", "https://client.example/path", "https://client.example?query=1", "https://client.example#fragment", "https://user@client.example", "https://client.example?", "https://"} {
		t.Run(origin, func(t *testing.T) {
			c := s.config
			c.Origins = []string{origin}
			if _, err := New(c); err == nil {
				t.Fatalf("invalid configured origin accepted: %s", origin)
			}
		})
	}
}
