package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// #249 acceptance 1 and 4: the harness names the client and the host that
// vouches for it, filtered and bounded.
func TestHarnessNamesTheClient(t *testing.T) {
	long := strings.Repeat("a", 190) + ".example.com"
	for _, c := range []struct {
		name string
		in   oauthClient
		want string
	}{
		{"document client", oauthClient{Name: "Claude Code", Kind: "claude.ai", Document: true}, "Claude Code (document on claude.ai)"},
		{"DCR client", oauthClient{Name: "Claude", Kind: "claude.ai"}, "Claude (redirect to claude.ai)"},
		{"same name, other registration", oauthClient{Name: "Claude Code", Kind: "claude.ai"}, "Claude Code (redirect to claude.ai)"},
		{"nameless DCR client", oauthClient{Name: displayName("", "claude.ai"), Kind: "claude.ai"}, "claude.ai (redirect to claude.ai)"},
		{"controls and escapes", oauthClient{Name: "Synthetic\x1b[31m Agent\x00\u0085", Kind: "example.com", Document: true}, "Synthetic[31m Agent (document on example.com)"},
		{"bidi and zero-width", oauthClient{Name: "Synthetic\u202e tnegA\u200b\u2066", Kind: "example.com"}, "Synthetic tnegA (redirect to example.com)"},
		{"line separators", oauthClient{Name: "Synthetic\u2028Spoofed line\u2029\r\nNext", Kind: "example.com"}, "SyntheticSpoofed lineNext (redirect to example.com)"},
		{"long host", oauthClient{Name: "Synthetic Agent", Kind: long, Document: true}, "Synthetic Agent (document on …" + long[len(long)-78:] + ")"},
	} {
		if got := c.in.harness(); got != c.want {
			t.Errorf("%s: harness %q, want %q", c.name, got, c.want)
		}
	}
	// A 300-byte multibyte name on a long host: at most 128 bytes, valid
	// UTF-8, suffix whole.
	for _, kind := range []string{"example.com", long} {
		got := (&oauthClient{Name: strings.Repeat("é", 150), Kind: kind, Document: true}).harness()
		if len(got) > maxHarnessBytes || !utf8.ValidString(got) || !strings.HasSuffix(got, ")") || !strings.Contains(got, " (document on ") {
			t.Errorf("bounded harness %q (%d bytes)", got, len(got))
		}
		if _, suffix, _ := strings.Cut(got, " (document on "); len(suffix)+len(" (document on ") > maxHarnessSuffixBytes {
			t.Errorf("suffix of %q is over %d bytes", got, maxHarnessSuffixBytes)
		}
	}
}

// #249 acceptance 1 against a running server: a metadata-document client and
// a DCR client on the same host write with different harnesses, both naming
// the client.
func TestOAuthWritesRecordTheClientHarness(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	e := startOAuth(t, a)
	docURL := serveDocument(t, e)
	e.enrollOwner(t)
	redirect := "http://127.0.0.1:7/callback"
	doc := e.tokens(t, docURL, redirect, "/mcp", "", ScopeReadWrite)
	dcr := e.registerClient(t, "Synthetic Code", "https://127.0.0.1.example/callback", redirect)
	chat := e.tokens(t, dcr, redirect, "/mcp", "", ScopeReadWrite)
	write := func(token, id string) memory.Authorship {
		t.Helper()
		_, text := call(t, e.bearer(t, "/mcp", token), "memory_remember", rememberArgs(id, "Written by "+id))
		return authorOf(t, a, receipt(t, text))
	}
	byDocument := write(doc["access_token"].(string), "harness-document-0001")
	byDCR := write(chat["access_token"].(string), "harness-redirect-0001")
	if byDocument.Harness != "Synthetic Code (document on 127.0.0.1)" {
		t.Fatalf("document client harness %q", byDocument.Harness)
	}
	if byDCR.Harness != "Synthetic Code (redirect to 127.0.0.1.example)" || byDCR.Harness == byDocument.Harness {
		t.Fatalf("DCR client harness %q", byDCR.Harness)
	}
	if byDocument.SessionID != nil || byDCR.SessionID != nil || byDocument.Model != nil {
		t.Fatalf("sessionless writes recorded a session or model: %+v %+v", byDocument, byDCR)
	}
}

// mcpTraffic records the JSON-RPC methods a client sent and the session IDs
// the server issued.
type mcpTraffic struct {
	mu      sync.Mutex
	methods []string
	issued  []string
}

func (m *mcpTraffic) seen() ([]string, []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.methods), slices.Clone(m.issued)
}

// sdkSession connects the SDK client. With initialize, it answers
// server/discover with -32601 so the client falls back to initialize, as
// Claude Code does against a server without 2026-07-28; the SDK client can't
// be pinned to an older protocol otherwise. extra headers go on every request.
func sdkSession(t *testing.T, h *httptest.Server, token string, initialize bool, extra http.Header) (*sdk.ClientSession, *mcpTraffic) {
	t.Helper()
	traffic := &mcpTraffic{}
	base := httpClient(t, token).Transport
	client := &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(raw))
			_ = json.Unmarshal(raw, &msg)
		}
		traffic.mu.Lock()
		traffic.methods = append(traffic.methods, msg.Method)
		traffic.mu.Unlock()
		if initialize && msg.Method == "server/discover" {
			body := `{"jsonrpc":"2.0","id":` + string(msg.ID) + `,"error":{"code":-32601,"message":"Method not found"}}`
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
		}
		r = r.Clone(r.Context())
		for k, v := range extra {
			r.Header[k] = v
		}
		resp, err := base.RoundTrip(r)
		if err == nil && resp.Header.Get(sessionIDHeader) != "" {
			traffic.mu.Lock()
			traffic.issued = append(traffic.issued, resp.Header.Get(sessionIDHeader))
			traffic.mu.Unlock()
		}
		return resp, err
	})}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	session, err := sdk.NewClient(&sdk.Implementation{Name: "synthetic-client", Version: "1"}, nil).Connect(ctx,
		&sdk.StreamableClientTransport{Endpoint: h.URL + "/mcp", HTTPClient: client, MaxRetries: -1, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session, traffic
}

func journalArgs(id string) map[string]any {
	return map[string]any{"request_id": id, "entry": map[string]any{"kind": "outcome", "summary": "Synthetic outcome " + id + "."}}
}

// journalAuthor reads a journal entry's authorship from the signet.
func journalAuthor(t *testing.T, s testfixture.ServerSignet, saved Receipt) memory.Authorship {
	t.Helper()
	paths, _ := filepath.Glob(filepath.Join(s.Root, "memory/events/*/*", saved.ID+".json"))
	if len(paths) != 1 {
		t.Fatalf("journal entry %s: %v", saved.ID, paths)
	}
	var e memory.JournalEntry
	readJSON(t, paths[0], &e)
	return e.Authorship
}

func session(a memory.Authorship) string {
	if a.SessionID == nil {
		return "<null>"
	}
	return *a.SessionID
}

// #249 acceptance 2(a): on the initialize path each connection gets its own
// session ID, which its writes carry, and reads return it.
func TestWritesRecordTheTransportSession(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	var device string
	ids := map[string]bool{}
	for i, conn := range []string{"one", "two"} {
		s, traffic := sdkSession(t, h, writeToken, true, nil)
		// The hook's own session value is an input, never authorship.
		call(t, s, "memory_context", map[string]any{"event": "prompt", "prompt": "synthetic", "session": "hook-session-" + conn})
		_, text := call(t, s, "memory_remember", rememberArgs("session-remember-"+conn+"-01", "Saved on connection "+conn))
		record := authorOf(t, a, receipt(t, text))
		_, text = call(t, s, "memory_journal_append", journalArgs("session-journal-"+conn+"-001"))
		entry := journalAuthor(t, a, receipt(t, text))
		methods, issued := traffic.seen()
		if !slices.Contains(methods, "initialize") || len(issued) != 1 {
			t.Fatalf("connection %s did not take the initialize path: methods %v, issued %v", conn, methods, issued)
		}
		if session(record) != issued[0] || session(entry) != issued[0] {
			t.Fatalf("connection %s: issued %s, recorded %s and %s", conn, issued[0], session(record), session(entry))
		}
		if i == 0 {
			device = record.DeviceID
		}
		if record.DeviceID != device || entry.DeviceID != device || record.Harness != "claude-code" {
			t.Fatalf("connection %s authorship %+v %+v", conn, record, entry)
		}
		ids[issued[0]] = true
		if i == 1 {
			// Scope labels and moves carry it too, and reads return it.
			_, text = call(t, s, "memory_name_scope", nameArgs("session-name-scope-01", "Atlas launch"))
			label := receipt(t, text)
			var l memory.ScopeLabel
			paths, _ := filepath.Glob(filepath.Join(a.Root, "memory/scopes/*/*/labels", label.ID+".json"))
			if len(paths) != 1 {
				t.Fatalf("label %s: %v", label.ID, paths)
			}
			readJSON(t, paths[0], &l)
			first := receipt(t, mustCall(t, s, "memory_remember", inScope("session-move-record-1", "Synthetic codename", projectX, "")))
			m := decoded[MoveReceipt](t, mustCall(t, s, "memory_move", moveArgs("session-move-request1", first.RecordID, projectY, []string{first.ID}, []string{})))
			history := mustCall(t, s, "memory_history", map[string]any{"record_id": first.RecordID})
			journal := mustCall(t, s, "memory_journal", map[string]any{"query": "Synthetic outcome"})
			// Prime returns authorship on its journal entries.
			receipt(t, mustCall(t, s, "memory_journal_append", map[string]any{"request_id": "session-journal-scoped1", "entry": map[string]any{"kind": "outcome", "summary": "Synthetic scoped outcome.", "scope": projectY}}))
			prime := mustCall(t, s, "memory_prime", map[string]any{"scope": projectY})
			want := `"session_id":"` + issued[0] + `"`
			if session(l.Authorship) != issued[0] || !strings.Contains(history, want) || strings.Count(history, want) < 2 || !strings.Contains(journal, want) || !strings.Contains(prime, want) {
				t.Fatalf("label %s, move %s: reads lack %s:\nhistory %s\njournal %s\nprime %s", session(l.Authorship), m.ID, want, history, journal, prime)
			}
		}
	}
	if len(ids) != 2 {
		t.Fatalf("two connections share a session ID: %v", ids)
	}
}

// #249 acceptance 2(b): a default SDK client speaks sessionless 2026-07-28;
// its writes succeed with session_id null, even with a stray session header.
func TestSessionlessWritesRecordNoSession(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	for _, stray := range []http.Header{nil, {sessionIDHeader: {"STRAYSESSIONID00000000000A"}}} {
		s, traffic := sdkSession(t, h, writeToken, false, stray)
		tag := "plain"
		if stray != nil {
			tag = "stray"
		}
		_, text := call(t, s, "memory_remember", rememberArgs("sessionless-remember-"+tag, "Saved sessionless"))
		record := authorOf(t, a, receipt(t, text))
		_, text = call(t, s, "memory_journal_append", journalArgs("sessionless-journal-"+tag))
		entry := journalAuthor(t, a, receipt(t, text))
		methods, issued := traffic.seen()
		if !slices.Contains(methods, "server/discover") || slices.Contains(methods, "initialize") || len(issued) != 0 {
			t.Fatalf("%s: not sessionless: methods %v, issued %v", tag, methods, issued)
		}
		if record.SessionID != nil || entry.SessionID != nil {
			t.Fatalf("%s: sessionless writes recorded %s and %s", tag, session(record), session(entry))
		}
	}
}

// rawWrite sends one tools/call memory_remember with the given headers and
// returns the write's authorship.
func rawWrite(t *testing.T, h *httptest.Server, a testfixture.ServerSignet, id string, headers map[string][]string) memory.Authorship {
	t.Helper()
	params := map[string]any{"name": "memory_remember", "arguments": rememberArgs(id, "Raw write "+id)}
	if v := headers[protocolVersionHeader]; len(v) == 1 && v[0] == "2026-07-28" {
		params["_meta"] = map[string]any{"io.modelcontextprotocol/protocolVersion": v[0], "io.modelcontextprotocol/clientInfo": map[string]any{"name": "synthetic-client", "version": "1"}, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}
	}
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": params})
	req, _ := http.NewRequest(http.MethodPost, h.URL+"/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header[k] = v
	}
	if params["_meta"] != nil {
		req.Header.Set("Mcp-Method", "tools/call")
		req.Header.Set("Mcp-Name", "memory_remember")
	}
	resp, err := httpClient(t, writeToken).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Result sdk.CallToolResult `json:"result"`
	}
	raw, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Result.Content) == 0 {
		t.Fatalf("%s: %d %s", id, resp.StatusCode, raw)
	}
	return authorOf(t, a, receipt(t, out.Result.Content[0].(*sdk.TextContent).Text))
}

// #249 failure paths: a malformed, missing or doubled session header, or one
// on a sessionless protocol, records null and the write succeeds. The server
// keeps no session table, so an ID survives a restart.
func TestSessionHeaderIsBoundedAndNeverFailsAWrite(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	c := testConfig(t, a)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	s, h, _ := startServerUntil(t, ctx, c)
	old := "2025-11-25"
	for name, headers := range map[string]map[string][]string{
		"129 bytes":  {protocolVersionHeader: {old}, sessionIDHeader: {strings.Repeat("A", 129)}},
		"a space":    {protocolVersionHeader: {old}, sessionIDHeader: {"SYNTHETIC SESSION"}},
		"a tab":      {protocolVersionHeader: {old}, sessionIDHeader: {"SYNTHETIC\tSESSION"}},
		"non-ASCII":  {protocolVersionHeader: {old}, sessionIDHeader: {"SYNTHÉTIC"}},
		"two values": {protocolVersionHeader: {old}, sessionIDHeader: {"SYNTHETICA", "SYNTHETICB"}},
		"missing":    {protocolVersionHeader: {old}},
		"2026-07-28": {protocolVersionHeader: {"2026-07-28"}, sessionIDHeader: {"SYNTHETICSESSION2026"}},
	} {
		if got := rawWrite(t, h, a, "bounded-"+strings.ReplaceAll(name, " ", "-")+"-0001", headers); got.SessionID != nil {
			t.Errorf("%s: recorded %q", name, *got.SessionID)
		}
	}
	id := strings.Repeat("Z", 128)
	if got := rawWrite(t, h, a, "bounded-128-bytes-0001", map[string][]string{protocolVersionHeader: {old}, sessionIDHeader: {id}}); session(got) != id {
		t.Fatalf("128-byte ID recorded as %s", session(got))
	}
	// No protocol header means 2025-03-26, which has sessions.
	if got := rawWrite(t, h, a, "bounded-no-version-0001", map[string][]string{sessionIDHeader: {"SYNTHETICSESSION0325"}}); session(got) != "SYNTHETICSESSION0325" {
		t.Fatalf("2025-03-26 write recorded %s", session(got))
	}
	// A restarted server accepts an ID the previous one issued.
	sdkS, traffic := sdkSession(t, h, writeToken, true, nil)
	call(t, sdkS, "memory_scopes", map[string]any{})
	_, issued := traffic.seen()
	if len(issued) != 1 {
		t.Fatalf("issued %v", issued)
	}
	g := s.signets["/mcp"]
	eventually(t, 5*time.Second, "first server's backup settles", func() bool {
		g.backupMu.Lock()
		defer g.backupMu.Unlock()
		return !g.backup.Pending
	})
	stop()
	h.Close()
	_, h2, _ := startServer(t, c)
	if got := rawWrite(t, h2, a, "bounded-after-restart-1", map[string][]string{protocolVersionHeader: {old}, sessionIDHeader: {issued[0]}}); session(got) != issued[0] {
		t.Fatalf("after restart recorded %s, want %s", session(got), issued[0])
	}
}

// A read-only client gets a session ID on initialize and still can't write.
func TestReadClientGetsASessionButCannotWrite(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	s, traffic := sdkSession(t, h, readToken, true, nil)
	_, err := s.CallTool(context.Background(), &sdk.CallToolParams{Name: "memory_remember", Arguments: rememberArgs("read-client-write-0001", "Refused")})
	if _, issued := traffic.seen(); len(issued) != 1 || err == nil || !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("read client: issued %v, write error %v", issued, err)
	}
	// GET and DELETE keep their stateless answers.
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		req, _ := http.NewRequest(method, h.URL+"/mcp", nil)
		req.Header.Set(sessionIDHeader, "SYNTHETICSESSIONGETDELETE")
		resp, err := httpClient(t, readToken).Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("%s: %d", method, resp.StatusCode)
		}
	}
}

// A retry from a new connection is the same request: the session ID is not
// part of the idempotency digest.
func TestRetryFromAnotherConnectionReturnsTheReceipt(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	first, _ := sdkSession(t, h, writeToken, true, nil)
	second, _ := sdkSession(t, h, writeToken, true, nil)
	_, text := call(t, first, "memory_remember", rememberArgs("retry-connection-0001", "Saved once"))
	saved := receipt(t, text)
	_, text = call(t, second, "memory_remember", rememberArgs("retry-connection-0001", "Saved once"))
	if again := receipt(t, text); again.ID != saved.ID {
		t.Fatalf("retry from a new connection wrote again: %s vs %s", again.ID, saved.ID)
	}
	if _, text = call(t, second, "memory_remember", rememberArgs("retry-connection-0001", "Changed")); !strings.Contains(text, "request.conflict") {
		t.Fatalf("changed retry accepted: %s", text)
	}
}

// Only a known protocol with sessions records an ID: an unknown or doubled
// version records null.
func TestTransportSessionNeedsASessionProtocol(t *testing.T) {
	for version, want := range map[string]bool{"2024-11-05": true, "2025-03-26": true, "2025-06-18": true, "2025-11-25": true, "2026-07-28": false, "2099-01-01": false, "": false} {
		h := http.Header{sessionIDHeader: {"SYNTHETICSESSION"}, protocolVersionHeader: {version}}
		if got := transportSession(h) != nil; got != want {
			t.Errorf("version %q: recorded %v, want %v", version, got, want)
		}
	}
	if transportSession(http.Header{sessionIDHeader: {"SYNTHETICSESSION"}, protocolVersionHeader: {"2025-11-25", "2025-06-18"}}) != nil {
		t.Error("two protocol versions recorded a session")
	}
	if transportSession(nil) != nil {
		t.Error("no headers recorded a session")
	}
}

// #249 acceptance 3: a record and journal entry written before 2.4.0
// (harness the client's host, no session) read back after it with their
// authorship byte for byte, and the files are untouched by new writes.
func TestEarlierAuthorshipIsUnchanged(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	before := memory.Authorship{DeviceID: a.DeviceID, Actor: "Synthetic Owner", Harness: "claude.ai"}
	m, err := memory.OpenService(a.Root, before)
	if err != nil {
		t.Fatal(err)
	}
	r, err := m.Remember(memory.Write{Kind: "fact", Summary: "Written before 2.4.0", Body: "Synthetic body.", Basis: "user-direction", Reason: "Synthetic test"})
	if err != nil {
		t.Fatal(err)
	}
	e, err := m.AppendJournal("outcome", "Synthetic outcome before 2.4.0.", nil)
	if err != nil {
		t.Fatal(err)
	}
	recordPath := filepath.Join(a.Root, "memory/records", r.RecordID, r.ID+".json")
	journalPaths, _ := filepath.Glob(filepath.Join(a.Root, "memory/events/*/*", e.ID+".json"))
	if len(journalPaths) != 1 {
		t.Fatalf("journal entry: %v", journalPaths)
	}
	files := map[string][]byte{}
	for _, p := range []string{recordPath, journalPaths[0]} {
		files[p] = readFile(t, p)
	}
	_, h, _ := startServer(t, testConfig(t, a))
	s, _ := sdkSession(t, h, writeToken, true, nil)
	receipt(t, mustCall(t, s, "memory_remember", rememberArgs("after-change-write-01", "Written after 2.4.0")))
	stored := func(p string) string {
		var v struct {
			Authorship json.RawMessage `json:"authorship"`
		}
		if err := json.Unmarshal(files[p], &v); err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		if err := json.Compact(&b, v.Authorship); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}
	for p, raw := range files {
		if !bytes.Equal(readFile(t, p), raw) {
			t.Fatalf("%s changed", p)
		}
	}
	if want := `{"device_id":"` + a.DeviceID + `","actor":"Synthetic Owner","harness":"claude.ai","model":null,"session_id":null}`; stored(recordPath) != want {
		t.Fatalf("stored authorship %s", stored(recordPath))
	}
	history := mustCall(t, s, "memory_history", map[string]any{"record_id": r.RecordID})
	journal := mustCall(t, s, "memory_journal", map[string]any{"query": "before 2.4.0"})
	if !strings.Contains(history, `"authorship":`+stored(recordPath)) || !strings.Contains(journal, `"authorship":`+stored(journalPaths[0])) {
		t.Fatalf("reads changed earlier authorship:\nhistory %s\njournal %s", history, journal)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// A metadata document's new name reaches the harness once the server
// refetches it, also for a token issued before: the cached MCP handler is
// keyed by the harness, so it can't keep the old name.
func TestRenamedDocumentClientWritesItsNewName(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	e := startOAuth(t, a)
	var docURL string
	var name atomic.Value
	name.Store("Synthetic Code")
	doc := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"client_id": docURL, "client_name": name.Load(), "redirect_uris": []string{"http://127.0.0.1/callback"}})
	}))
	t.Cleanup(doc.Close)
	docURL = doc.URL + "/oauth/client-metadata"
	e.server.oauth.documents.client = doc.Client()
	e.enrollOwner(t)
	redirect := "http://127.0.0.1:7/callback"
	token := e.tokens(t, docURL, redirect, "/mcp", "", ScopeReadWrite)["access_token"].(string)
	write := func(id string) string {
		t.Helper()
		_, text := call(t, e.bearer(t, "/mcp", token), "memory_remember", rememberArgs(id, "Written by "+id))
		return authorOf(t, a, receipt(t, text)).Harness
	}
	if got := write("renamed-client-before1"); got != "Synthetic Code (document on 127.0.0.1)" {
		t.Fatalf("before: %q", got)
	}
	name.Store("Synthetic Code Renamed")
	e.clock.advance(documentTTL + time.Second)
	e.tokens(t, docURL, redirect, "/mcp", "", ScopeReadWrite)
	if got := write("renamed-client-after01"); got != "Synthetic Code Renamed (document on 127.0.0.1)" {
		t.Fatalf("after: %q", got)
	}
}
