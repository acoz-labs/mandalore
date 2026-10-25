package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	listen     = "127.0.0.1:8730"
	writeToken = "token-c1-read-write"
	readToken  = "token-r1-read-only"
)

var fastTiming = timing{debounce: 50 * time.Millisecond, retry: 200 * time.Millisecond, push: 5 * time.Second}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}
func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

// testConfig serves the given synthetic signets with c1 (read-write) and r1
// (read) authorized for the first one only.
func testConfig(t *testing.T, signets ...testfixture.ServerSignet) Config {
	t.Helper()
	c := Config{Listen: listen, PublicURL: "https://memory.example.com", StateDir: t.TempDir(), Owner: Owner{Name: "Synthetic Owner"}, Auth: AuthInsecureTest}
	for i, s := range signets {
		c.Signets = append(c.Signets, SignetConfig{Slug: string(rune('a' + i)), Root: s.Root, SignetID: s.SignetID, DeviceID: s.DeviceID, Remote: s.Remote})
	}
	c.TestClients = []TestClient{
		{ID: "c1", Kind: "claude-code", Scope: ScopeReadWrite, Token: writeToken, Signets: []string{"a"}},
		{ID: "r1", Kind: "claude.ai", Scope: ScopeRead, Token: readToken, Signets: []string{"a"}},
	}
	return c
}

func startServer(t *testing.T, c Config) (*Server, *httptest.Server, *syncBuffer) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return startServerUntil(t, ctx, c)
}

func startServerUntil(t *testing.T, ctx context.Context, c Config) (*Server, *httptest.Server, *syncBuffer) {
	t.Helper()
	logs := &syncBuffer{}
	s, err := newServer(c, logs, fastTiming)
	if err != nil {
		t.Fatal(err)
	}
	s.Start(ctx)
	h := httptest.NewServer(s.Handler())
	t.Cleanup(h.Close)
	return s, h, logs
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// httpClient addresses the test listener by the configured loopback host.
func httpClient(t *testing.T, token string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	t.Cleanup(transport.CloseIdleConnections)
	return &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		r = r.Clone(r.Context())
		r.Host = listen
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		return transport.RoundTrip(r)
	})}
}

func connect(t *testing.T, h *httptest.Server, path, token string) *sdk.ClientSession {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	session, err := sdk.NewClient(&sdk.Implementation{Name: "synthetic-client", Version: "1"}, nil).Connect(ctx, &sdk.StreamableClientTransport{Endpoint: h.URL + path, HTTPClient: httpClient(t, token), MaxRetries: -1, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func call(t *testing.T, session *sdk.ClientSession, name string, args any) (*sdk.CallToolResult, string) {
	t.Helper()
	r, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for _, c := range r.Content {
		if tc, ok := c.(*sdk.TextContent); ok {
			text.WriteString(tc.Text)
		}
	}
	return r, text.String()
}

func receipt(t *testing.T, text string) Receipt {
	t.Helper()
	var v struct {
		OK     bool    `json:"ok"`
		Result Receipt `json:"result"`
	}
	if err := json.Unmarshal([]byte(text), &v); err != nil || !v.OK {
		t.Fatalf("not a successful receipt: %s", text)
	}
	return v.Result
}

func rememberArgs(id, body string) map[string]any {
	return map[string]any{"request_id": id, "record": map[string]any{"kind": "fact", "summary": "Synthetic convention", "body": body, "basis": "user-direction", "reason": "Confirmed in a synthetic test"}}
}

func remoteHead(t *testing.T, s testfixture.ServerSignet) string {
	t.Helper()
	out := testfixture.Git(t, s.Root, "ls-remote", s.Remote, "refs/heads/main")
	head, _, _ := strings.Cut(out, "\t")
	return head
}

func eventually(t *testing.T, within time.Duration, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(within)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("%s within %s", what, within)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// Acceptance 1: a client authorized for A is refused on B, and A's writes
// never reach B's checkout or remote.
func TestSignetsAreIsolated(t *testing.T) {
	a, b := testfixture.NewServerSignet(t, "alpha"), testfixture.NewServerSignet(t, "beta")
	_, h, _ := startServer(t, testConfig(t, a, b))
	for _, path := range []string{"/mcp", "/s/c/mcp"} {
		if code := post(t, h, path, writeToken); code != http.StatusNotFound {
			t.Fatalf("%s: status %d", path, code)
		}
	}
	if code := post(t, h, "/s/b/mcp", writeToken); code != http.StatusForbidden {
		t.Fatalf("A's client on B: status %d", code)
	}
	if code := post(t, h, "/s/b/mcp", "token-unknown-client"); code != http.StatusUnauthorized {
		t.Fatalf("unknown token on B: status %d", code)
	}
	bHead := testfixture.Git(t, b.Root, "rev-parse", "HEAD")
	_, text := call(t, connect(t, h, "/s/a/mcp", writeToken), "memory_remember", rememberArgs("isolation-request-0001", "Alpha only marker"))
	receipt(t, text)
	eventually(t, 5*time.Second, "A's remote receives the write", func() bool { return grepRepo(t, a.RemoteDir, "Alpha only marker") })
	if testfixture.Git(t, b.Root, "rev-parse", "HEAD") != bHead || testfixture.Git(t, b.Root, "status", "--porcelain") != "" {
		t.Fatal("B's checkout changed")
	}
	if grepRepo(t, b.RemoteDir, "Alpha only marker") || grepRepo(t, b.Root, "Alpha only marker") {
		t.Fatal("A's write reached B's remote")
	}
}

func post(t *testing.T, h *httptest.Server, path, token string) int {
	t.Helper()
	r, err := http.NewRequest(http.MethodPost, h.URL+path, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := httpClient(t, token).Do(r)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// grepRepo reports whether any commit in a bare repository contains text.
func grepRepo(t *testing.T, repo, text string) bool {
	t.Helper()
	refs := testfixture.Git(t, repo, "for-each-ref", "--format=%(objectname)")
	if refs == "" {
		return false
	}
	return testfixture.Git(t, repo, "log", "--all", "-p", "--format=", "-S", text) != ""
}

// Acceptance 2: c1's revision and journal entry carry c1's enrolled device
// and kind.
func TestWritesRecordTheClient(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	c := testConfig(t, a)
	_, h, _ := startServer(t, c)
	session := connect(t, h, "/mcp", writeToken)
	_, text := call(t, session, "memory_remember", rememberArgs("authorship-request-01", "Recorded by c1"))
	saved := receipt(t, text)
	_, text = call(t, session, "memory_journal_append", map[string]any{"request_id": "authorship-journal-01", "entry": map[string]any{"kind": "outcome", "summary": "Synthetic outcome by c1."}})
	entry := receipt(t, text)
	var devices map[string]string
	if err := readState(filepath.Join(c.StateDir, "signets", "a", "devices.json"), &devices); err != nil || devices["c1"] == "" {
		t.Fatalf("c1 not enrolled: %v %v", devices, err)
	}
	var device memory.Device
	readJSON(t, filepath.Join(a.Root, "provenance/devices", devices["c1"]+".json"), &device)
	if !strings.HasPrefix(device.Label, "c1 (claude-code), approved ") {
		t.Fatalf("device label %q", device.Label)
	}
	var r memory.Revision
	readJSON(t, filepath.Join(a.Root, "memory/records", saved.RecordID, saved.ID+".json"), &r)
	want := memory.Authorship{DeviceID: devices["c1"], Actor: "Synthetic Owner", Harness: "claude-code"}
	if r.Authorship != want {
		t.Fatalf("revision authorship %+v", r.Authorship)
	}
	journals, _ := filepath.Glob(filepath.Join(a.Root, "memory/events/*/*", entry.ID+".json"))
	if len(journals) != 1 {
		t.Fatal("journal entry missing")
	}
	var j memory.JournalEntry
	readJSON(t, journals[0], &j)
	if j.Authorship != want {
		t.Fatalf("journal authorship %+v", j.Authorship)
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatal(err)
	}
}

// Acceptance 3 (ported from razor-crest): an identical retry returns the
// original receipt, changed input is refused, and restarts keep the receipt.
func TestRequestIDReplay(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	c := testConfig(t, a)
	c.TestClients = append(c.TestClients, TestClient{ID: "c2", Kind: "claude-code", Scope: ScopeReadWrite, Token: "token-c2-read-write", Signets: []string{"a"}})
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	s, h, _ := startServerUntil(t, ctx, c)
	session := connect(t, h, "/mcp", writeToken)
	_, text := call(t, session, "memory_remember", rememberArgs("fixture-request-0001", "Use synthetic fixtures."))
	first := receipt(t, text)
	_, text = call(t, session, "memory_remember", rememberArgs("fixture-request-0001", "Use synthetic fixtures."))
	if again := receipt(t, text); again.ID != first.ID || again.RecordID != first.RecordID || !again.Durable {
		t.Fatalf("identical retry changed receipt: %+v vs %+v", again, first)
	}
	r, text := call(t, session, "memory_remember", rememberArgs("fixture-request-0001", "Different input"))
	if !r.IsError || !strings.Contains(text, "request.conflict") {
		t.Fatalf("changed input accepted: %s", text)
	}
	if r, _ := call(t, session, "memory_remember", rememberArgs("short", "Too short a key")); !r.IsError {
		t.Fatal("short request_id accepted")
	}
	journal := map[string]any{"request_id": "fixture-journal-0001", "entry": map[string]any{"kind": "outcome", "summary": "Synthetic outcome."}}
	_, text = call(t, session, "memory_journal_append", journal)
	firstEntry := receipt(t, text)
	_, text = call(t, session, "memory_journal_append", journal)
	if receipt(t, text).ID != firstEntry.ID {
		t.Fatal("journal retry changed receipt")
	}
	// Keys are namespaced by client: c2's identical key is a new operation.
	_, text = call(t, connect(t, h, "/mcp", "token-c2-read-write"), "memory_remember", rememberArgs("fixture-request-0001", "Use synthetic fixtures."))
	if other := receipt(t, text); other.ID == first.ID {
		t.Fatal("clients share idempotency keys")
	}
	_, recalled := call(t, session, "memory_recall", map[string]any{"query": "synthetic fixtures"})
	if strings.Count(recalled, "Use synthetic fixtures.") != 2 {
		t.Fatalf("retries duplicated memory: %s", recalled)
	}
	// A restarted server returns the original receipt. Only one server may
	// own the signet, so the first stops once its backup settles.
	g := s.signets["/mcp"]
	eventually(t, 5*time.Second, "first server's backup settles", func() bool {
		g.backupMu.Lock()
		defer g.backupMu.Unlock()
		return !g.backup.Pending
	})
	stop()
	h.Close()
	_, h2, _ := startServer(t, c)
	_, text = call(t, connect(t, h2, "/mcp", writeToken), "memory_remember", rememberArgs("fixture-request-0001", "Use synthetic fixtures."))
	if receipt(t, text).ID != first.ID {
		t.Fatal("restart lost the receipt")
	}
}

// Acceptance 4: a read client sees no write tools and cannot write.
func TestReadScopeHasNoWriteTools(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	s, h, _ := startServer(t, testConfig(t, a))
	tools := func(token string) map[string]bool {
		list, err := connect(t, h, "/mcp", token).ListTools(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		names := map[string]bool{}
		for _, tool := range list.Tools {
			names[tool.Name] = true
		}
		return names
	}
	read, write := tools(readToken), tools(writeToken)
	for _, name := range append([]string{"memory_context"}, readTools...) {
		if !read[name] || !write[name] {
			t.Fatalf("read tool %s missing", name)
		}
	}
	for _, name := range append([]string{"memory_remember", "memory_journal_append"}, writeTools...) {
		if read[name] || !write[name] {
			t.Fatalf("write tool %s: read=%v write=%v", name, read[name], write[name])
		}
	}
	if len(read) != len(readTools)+1 || len(write) != len(readTools)+len(writeTools)+3 {
		t.Fatalf("unexpected tools: %v", write)
	}
	r, err := connect(t, h, "/mcp", readToken).CallTool(context.Background(), &sdk.CallToolParams{Name: "memory_remember", Arguments: rememberArgs("read-only-request-01", "Must not save")})
	if err == nil && !r.IsError {
		t.Fatal("read client wrote through MCP")
	}
	// The call path refuses writes by scope too, not only by tool listing.
	g := s.signets["/mcp"]
	if res := g.call(context.Background(), Client{ID: "r1", Kind: "claude.ai", Scope: ScopeRead}, "memory_remember", []byte(`{}`)); !res.IsError {
		t.Fatal("read client write accepted by call path")
	}
	if p, err := g.memory.Recall("Must not save", nil, 5, 4096); err != nil || p.MatchingCount != 0 {
		t.Fatal("read client's write was saved")
	}
}

// Acceptance 5: a write reaches the bare remote within 5 s; with the remote
// missing the receipt says pending and a later push succeeds.
func TestBackupPushesAndRetries(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	c := testConfig(t, a)
	s, h, _ := startServer(t, c)
	session := connect(t, h, "/mcp", writeToken)
	_, text := call(t, session, "memory_remember", rememberArgs("backup-request-00001", "First backup"))
	if got := receipt(t, text); got.Backup != "pending" || !got.Durable {
		t.Fatalf("receipt %+v", got)
	}
	start := time.Now()
	eventually(t, 5*time.Second, "push reaches the remote", func() bool {
		return remoteHead(t, a) == testfixture.Git(t, a.Root, "rev-parse", "HEAD") && grepRepo(t, a.RemoteDir, "First backup")
	})
	t.Logf("pushed after %s", time.Since(start))
	g := s.signets["/mcp"]
	eventually(t, 5*time.Second, "backup state pushed", func() bool {
		g.backupMu.Lock()
		defer g.backupMu.Unlock()
		return g.backup.State == "pushed" && !g.backup.Pending
	})

	hidden := a.RemoteDir + ".away"
	if err := os.Rename(a.RemoteDir, hidden); err != nil {
		t.Fatal(err)
	}
	_, text = call(t, session, "memory_remember", rememberArgs("backup-request-00002", "Second backup"))
	if got := receipt(t, text); got.Backup != "pending" {
		t.Fatalf("receipt with missing remote %+v", got)
	}
	time.Sleep(4 * fastTiming.retry)
	statuses, err := Status(context.Background(), c)
	if err != nil || statuses[0].Remote != "unreachable" || !statuses[0].Backup.Pending || statuses[0].Backup.State != "pending" {
		t.Fatalf("status while remote missing %+v: %v", statuses, err)
	}
	if err := os.Rename(hidden, a.RemoteDir); err != nil {
		t.Fatal(err)
	}
	eventually(t, 5*time.Second, "retry pushes once the remote is back", func() bool { return grepRepo(t, a.RemoteDir, "Second backup") })
	eventually(t, 5*time.Second, "status reports pushed", func() bool {
		st, err := Status(context.Background(), c)
		return err == nil && st[0].Remote == "current" && st[0].Backup.State == "pushed" && !st[0].Backup.Pending && st[0].Backup.Head == st[0].Head && st[0].Backup.LastPush != ""
	})
}

// Acceptance 6: a remote ahead at startup blocks the signet until reconcile.
func TestRemoteAheadBlocksUntilReconcile(t *testing.T) {
	a, b := testfixture.NewServerSignet(t, "alpha"), testfixture.NewServerSignet(t, "beta")
	c := testConfig(t, a, b)
	testfixture.Git(t, a.Root, "push", a.Remote, "main")
	// Another 1.x writer pushes a valid memory to A's remote.
	other := filepath.Join(t.TempDir(), "other")
	testfixture.Git(t, a.Root, "clone", a.Remote, other)
	m, err := memory.OpenService(other, memory.Authorship{DeviceID: a.DeviceID, Actor: "Synthetic Owner", Harness: "cli"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Remember(memory.Write{Kind: "fact", Summary: "Late 1.x write", Body: "Written by another clone", Basis: "user-direction", Reason: "Synthetic"}); err != nil {
		t.Fatal(err)
	}
	testfixture.Git(t, other, "add", "memory")
	testfixture.Git(t, other, "commit", "-m", "Late write")
	testfixture.Git(t, other, "push", "origin", "main")

	_, h, logs := startServer(t, c)
	if !strings.Contains(logs.String(), "signet a: remote ahead: run mandalore server reconcile") {
		t.Fatalf("log %q", logs.String())
	}
	if code := post(t, h, "/s/a/mcp", writeToken); code != http.StatusServiceUnavailable {
		t.Fatalf("blocked signet served: %d", code)
	}
	st, err := Status(context.Background(), c)
	if err != nil || st[0].Remote != "ahead" || st[1].Remote != "current" {
		t.Fatalf("status %+v: %v", st, err)
	}
	if _, err := Reconcile(context.Background(), c, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := Reconcile(context.Background(), c, "missing"); err == nil {
		t.Fatal("unknown signet reconciled")
	}
	_, h2, logs2 := startServer(t, c)
	if strings.Contains(logs2.String(), "remote ahead") {
		t.Fatalf("still blocked after reconcile: %s", logs2.String())
	}
	_, text := call(t, connect(t, h2, "/s/a/mcp", readToken), "memory_recall", map[string]any{"query": "late"})
	if !strings.Contains(text, "Late 1.x write") {
		t.Fatalf("reconciled memory missing: %s", text)
	}
}

func TestLoopbackTrustAndHTTPBoundary(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	c := testConfig(t, a)
	c.Auth, c.TestClients, c.LocalScope = AuthLoopbackTrust, nil, ScopeRead
	s, h, _ := startServer(t, c)
	session := connect(t, h, "/mcp", "")
	init := session.InitializeResult()
	if init.Instructions != instructions(false) {
		t.Fatal("read instructions not served")
	}
	if _, text := call(t, session, "memory_scopes", map[string]any{}); !strings.Contains(text, `"ok":true`) {
		t.Fatal(text)
	}
	for _, tc := range []struct {
		name, host, origin, path, method string
		status                           int
	}{
		{"public host is not loopback-trusted", "memory.example.com", "", "/mcp", "POST", 401},
		{"unknown host", "other.example", "", "/mcp", "POST", 404},
		{"foreign origin", listen, "https://other.example", "/mcp", "POST", 403},
		{"unknown route", listen, "", "/admin", "POST", 404},
		{"healthz", listen, "", "/healthz", "GET", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "http://"+tc.host+tc.path, strings.NewReader(`{}`))
			r.RemoteAddr = "127.0.0.1:50000"
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, r)
			if w.Code != tc.status || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("status %d, cache %q", w.Code, w.Header().Get("Cache-Control"))
			}
			if strings.Contains(w.Body.String(), a.Root) {
				t.Fatal("response leaked the signet path")
			}
		})
	}
	r := httptest.NewRequest("GET", "http://"+listen+"/healthz", nil)
	r.RemoteAddr = "192.0.2.1:50000"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatal("healthz served to a non-loopback peer")
	}
	r = httptest.NewRequest("POST", "http://"+listen+"/mcp", strings.NewReader(`{}`))
	r.RemoteAddr = "192.0.2.1:50000"
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("non-loopback peer trusted")
	}
}

func TestInstructionsAreShort(t *testing.T) {
	read, write := instructions(false), instructions(true)
	if read == write || len(read) >= 600 || len(write) >= 600 {
		t.Fatalf("instructions: %d and %d bytes", len(read), len(write))
	}
	for _, text := range []string{read, write} {
		for _, gone := range []string{"memory_sync", "session", "binding"} {
			if strings.Contains(text, gone) {
				t.Fatalf("instructions mention %s", gone)
			}
		}
	}
}
