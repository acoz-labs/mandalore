package server

import (
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// serveDocument serves a synthetic client ID metadata document over TLS,
// as every Claude Code installation shares one, and lets e fetch it.
func serveDocument(t *testing.T, e *oauthEnv) string {
	t.Helper()
	var docURL string
	doc := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"client_id": docURL, "client_name": "Synthetic Code", "redirect_uris": []string{"http://127.0.0.1/callback"}})
	}))
	t.Cleanup(doc.Close)
	docURL = doc.URL + "/oauth/client-metadata"
	// The test document is on loopback, which production fetches refuse.
	e.server.oauth.documents.client = doc.Client()
	return docURL
}

// installations lists a client's installations as `server clients list`
// shows them.
func installations(t *testing.T, c Config, client string) []ClientInfo {
	t.Helper()
	all, err := Clients(c)
	if err != nil {
		t.Fatal(err)
	}
	var out []ClientInfo
	for _, info := range all {
		if info.ID == client && info.Installation != "" {
			out = append(out, info)
		}
	}
	return out
}

// authorizePage opens an authorization URL and returns the status and body.
func authorizePage(t *testing.T, c *http.Client, authURL string) (int, string) {
	t.Helper()
	resp, err := c.Get(authURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func (e *oauthEnv) refresh(t *testing.T, client, token string) (*http.Response, map[string]any) {
	t.Helper()
	return e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {token}})
}

// codes counts the authorization codes the server holds.
func (e *oauthEnv) codes() int {
	e.server.oauth.mu.Lock()
	defer e.server.oauth.mu.Unlock()
	return len(e.server.oauth.codes)
}

func canWrite(t *testing.T, e *oauthEnv, token string) bool {
	t.Helper()
	return slices.Contains(toolNames(t, e, "/mcp", token), "memory_remember")
}

// Issue #202: installations sharing one metadata-document client_id are
// consented, scoped, refreshed and revoked separately.
func TestInstallationsOfOneDocumentClient(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	docURL := serveDocument(t, e)
	e.enrollOwner(t)
	redirect := "http://127.0.0.1:7/callback"
	first := e.tokens(t, docURL, redirect, "/mcp", "", ScopeReadWrite)

	// A second installation under the approved client_id gets the consent
	// page, which says other installations exist, and no code.
	e.clock.advance(time.Second)
	_, challenge := pkce()
	codes := e.codes()
	status, body := authorizePage(t, e.browser, authorizeURL(docURL, redirect, challenge, publicURL+"/mcp", ""))
	if status != http.StatusOK || csrfField.FindString(body) == "" || !strings.Contains(body, "Other installations of this application are connected already") {
		t.Fatalf("second installation: %d %s", status, body)
	}
	if e.codes() != codes {
		t.Fatal("the second installation got a code without consent")
	}
	second := e.tokens(t, docURL, redirect, "/mcp", "", ScopeRead)
	if !canWrite(t, e, first["access_token"].(string)) || canWrite(t, e, second["access_token"].(string)) {
		t.Fatal("the second installation inherited the first one's scope, or changed it")
	}
	listed := installations(t, e.config, docURL)
	if len(listed) != 2 || listed[0].Installation == listed[1].Installation {
		t.Fatalf("installations %+v", listed)
	}
	for i, want := range []Scope{ScopeReadWrite, ScopeRead} {
		in := listed[i]
		if in.Status != "approved" || in.Grants["a"] != want || in.Tokens != 2 || in.FirstSeen.IsZero() || in.LastUsed.Before(in.FirstSeen) {
			t.Fatalf("installation %d: %+v", i, in)
		}
	}
	firstID, secondID := listed[0].Installation, listed[1].Installation

	// set-scope on one installation leaves the others alone.
	third := e.tokens(t, docURL, redirect, "/mcp", "", ScopeReadWrite)
	if err := SetInstallationScope(e.config, firstID, "a", ScopeRead); err != nil {
		t.Fatal(err)
	}
	if canWrite(t, e, first["access_token"].(string)) || !canWrite(t, e, third["access_token"].(string)) {
		t.Fatal("set-scope missed its installation or reached another")
	}
	if err := SetInstallationScope(e.config, firstID, "b", ScopeRead); err == nil {
		t.Fatal("set-scope on a signet without a grant succeeded")
	}
	// 2.0.1 would read the client's widest installation grant.
	if got := legacyGrants(t, e.config, docURL); got["a"] != ScopeReadWrite || len(got) != 1 {
		t.Fatalf("2.0.1 grants %v", got)
	}

	// A replayed refresh token ends its own family only.
	resp, rotated := e.refresh(t, docURL, first["refresh_token"].(string))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("refresh: %d %v", resp.StatusCode, rotated)
	}
	if resp, out := e.refresh(t, docURL, first["refresh_token"].(string)); resp.StatusCode != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Fatalf("replay: %d %v", resp.StatusCode, out)
	}
	if e.mcpStatus(t, "/mcp", rotated["access_token"].(string)) != http.StatusUnauthorized {
		t.Fatal("the replayed family survived")
	}
	resp, second = e.refresh(t, docURL, second["refresh_token"].(string))
	if resp.StatusCode != http.StatusOK || e.mcpStatus(t, "/mcp", second["access_token"].(string)) != http.StatusOK {
		t.Fatalf("the other installation's family was revoked: %d %v", resp.StatusCode, second)
	}

	// Revoking one installation leaves the others working.
	if client, err := RevokeInstallation(e.config, secondID); err != nil || client != docURL {
		t.Fatalf("revoke installation: %q %v", client, err)
	}
	if e.mcpStatus(t, "/mcp", second["access_token"].(string)) != http.StatusUnauthorized {
		t.Fatal("the revoked installation's access token still works")
	}
	if resp, out := e.refresh(t, docURL, second["refresh_token"].(string)); resp.StatusCode != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Fatalf("refresh after revoking the installation: %d %v", resp.StatusCode, out)
	}
	if e.mcpStatus(t, "/mcp", third["access_token"].(string)) != http.StatusOK {
		t.Fatal("revoking one installation revoked another")
	}
	if resp, out := e.refresh(t, docURL, third["refresh_token"].(string)); resp.StatusCode != http.StatusOK {
		t.Fatalf("the remaining installation can't refresh: %d %v", resp.StatusCode, out)
	}
	if _, err := RevokeInstallation(e.config, secondID); err == nil {
		t.Fatal("revoking a revoked installation succeeded")
	}

	// Revoking by client_id revokes every installation and counts them: the
	// first (its family already dead) and the third.
	if n, err := RevokeClient(e.config, docURL); err != nil || n != 2 {
		t.Fatalf("revoke client: %d %v", n, err)
	}
	if got := installations(t, e.config, docURL); len(got) != 0 {
		t.Fatalf("installations after revoking the client: %+v", got)
	}
	if got := legacyGrants(t, e.config, docURL); got != nil {
		t.Fatalf("2.0.1 grants after revoking the client: %v", got)
	}
}

// legacyGrants reads a client's grants from the saved state as 2.0.1 does.
func legacyGrants(t *testing.T, c Config, client string) map[string]Scope {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(c.StateDir, oauthStateFile))
	if err != nil {
		t.Fatal(err)
	}
	var old v1State
	if err := json.Unmarshal(raw, &old); err != nil {
		t.Fatal(err)
	}
	if cl := old.Clients[client]; cl != nil {
		return cl.Grants
	}
	return nil
}

// Claude Code's logout revokes the refresh token, then the access token.
// Both answer 200 although the first ends the installation's only family.
func TestDocumentClientLogout(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	docURL := serveDocument(t, e)
	e.enrollOwner(t)
	tok := e.tokens(t, docURL, "http://127.0.0.1:7/callback", "/mcp", "", ScopeRead)
	for _, token := range []string{tok["refresh_token"].(string), tok["access_token"].(string)} {
		if resp, body := e.revoke(t, url.Values{"client_id": {docURL}, "token": {token}}); resp.StatusCode != http.StatusOK || body != "" {
			t.Fatalf("logout: %d %q", resp.StatusCode, body)
		}
	}
	if e.mcpStatus(t, "/mcp", tok["access_token"].(string)) != http.StatusUnauthorized {
		t.Fatal("logout left the access token live")
	}
	// The dead installation is pruned a day after its last use, and the
	// client with it.
	e.clock.advance(pendingTTL)
	e.login(t)
	e.tokens(t, e.registerClient(t, "Synthetic Chat", "https://claude.ai/api/mcp/auth_callback"), "https://claude.ai/api/mcp/auth_callback", "/mcp", "", ScopeRead)
	if err := e.server.oauth.store.view(func(st *oauthState) error {
		if st.Clients[docURL] != nil || len(st.installationsOf(docURL)) != 0 {
			t.Error("the logged-out document client was not pruned")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// Approved client, no owner session: a new installation of an approved
// document client, from a browser without the owner cookie, gets the sign-in
// page and no code; with the owner's session it gets the consent page, never
// a code at once. An approved DCR client still skips only the consent page.
func TestApprovedClientNoOwnerSession(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	docURL := serveDocument(t, e)
	e.enrollOwner(t)
	web := "https://claude.ai/api/mcp/auth_callback"
	dcr := e.registerClient(t, "Synthetic Chat", web)
	e.tokens(t, dcr, web, "/mcp", "", ScopeReadWrite)
	e.tokens(t, docURL, "http://127.0.0.1:7/callback", "/mcp", "", ScopeReadWrite)
	stranger := &http.Client{Transport: e.agent.Transport, CheckRedirect: e.browser.CheckRedirect}
	codes := e.codes
	for client, redirect := range map[string]string{dcr: web, docURL: "http://127.0.0.1:6666/callback"} {
		_, challenge := pkce()
		authURL := authorizeURL(client, redirect, challenge, publicURL+"/mcp", scopeRead+" "+scopeWrite)
		before := codes()
		status, body := authorizePage(t, stranger, authURL)
		if status != http.StatusOK || !strings.Contains(body, `id="login"`) || codes() != before {
			t.Fatalf("%s without the owner: %d", client, status)
		}
		status, body = authorizePage(t, e.browser, authURL)
		consent := status == http.StatusOK && csrfField.FindString(body) != ""
		if client == docURL && (!consent || codes() != before) {
			t.Fatalf("new installation with the owner: %d, codes %d → %d", status, before, codes())
		}
		if client == dcr && (status != http.StatusSeeOther || codes() != before+1) {
			t.Fatalf("approved DCR client with the owner: %d", status)
		}
	}
}

// v1State is the OAuth state as 2.0.1 writes and reads it: grants per
// client, tokens with no installation.
type v1State struct {
	Clients map[string]*struct {
		ID           string           `json:"id"`
		Name         string           `json:"name"`
		Kind         string           `json:"kind"`
		RedirectURIs []string         `json:"redirect_uris"`
		Document     bool             `json:"metadata_document,omitempty"`
		Registered   time.Time        `json:"registered"`
		Grants       map[string]Scope `json:"grants,omitempty"`
	} `json:"clients"`
	Tokens map[string]*struct {
		Kind    string    `json:"kind"`
		Client  string    `json:"client"`
		Signet  string    `json:"signet"`
		Family  string    `json:"family"`
		Scope   Scope     `json:"scope"`
		Expires time.Time `json:"expires"`
		Used    bool      `json:"used,omitempty"`
	} `json:"tokens"`
	Owner json.RawMessage `json:"owner"`
}

// Issue #202 migration: a 2.0.1 state with one metadata-document client used
// by three installations (three live families, one with a rotated refresh
// token kept for replay detection), a separately registered hosted-app DCR
// client, a pending registration and a document client whose tokens all
// expired. After the upgrade every live token keeps working with no consent,
// each family is its own installation, the hosted app still skips consent,
// and the saved state still reads as 2.0.1's grants for a rollback.
func TestMigrationKeepsVersion1Grants(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	c := oauthConfig(t, testfixture.NewServerSignet(t, "alpha"))
	now := time.Now().UTC().Truncate(time.Second)
	code := "https://claude.ai/oauth/claude-code-client-metadata"
	gone := "https://client.example.com/oauth/client-metadata"
	chat, pending := "client-synthetichostedapp0001", "client-syntheticpending00001"
	web := "https://claude.ai/api/mcp/auth_callback"
	type pair struct{ access, refresh, used string }
	families := map[string]pair{}
	tokens := map[string]any{}
	token := func(kind, client, family string, scope Scope, issued time.Time, used bool) string {
		v := secret()
		ttl := accessTTL
		if kind == tokenRefresh {
			ttl = refreshTTL
		}
		tokens[hashSecret(v)] = map[string]any{"kind": kind, "client": client, "signet": "a", "family": family, "scope": scope, "expires": issued.Add(ttl), "used": used}
		return v
	}
	for i, issued := range []time.Duration{-10 * time.Minute, -20 * time.Minute, -30 * time.Minute} {
		family := hashSecret(secret())
		p := pair{access: token(tokenAccess, code, family, ScopeReadWrite, now.Add(issued), false), refresh: token(tokenRefresh, code, family, ScopeReadWrite, now.Add(issued), false)}
		if i == 0 {
			p.used = token(tokenRefresh, code, family, ScopeReadWrite, now.Add(issued-time.Hour), true)
		}
		families["code-"+string(rune('0'+i))] = p
	}
	chatFamily := hashSecret(secret())
	families["chat"] = pair{access: token(tokenAccess, chat, chatFamily, ScopeReadWrite, now.Add(-5*time.Minute), false), refresh: token(tokenRefresh, chat, chatFamily, ScopeReadWrite, now.Add(-5*time.Minute), false)}
	registered := now.Add(-48 * time.Hour)
	state := map[string]any{
		"clients": map[string]any{
			code:    map[string]any{"id": code, "name": "Claude Code", "kind": "claude.ai", "redirect_uris": []string{"http://localhost/callback"}, "metadata_document": true, "registered": registered, "grants": map[string]Scope{"a": ScopeReadWrite}},
			gone:    map[string]any{"id": gone, "name": "Synthetic Expired", "kind": "client.example.com", "redirect_uris": []string{"http://localhost/callback"}, "metadata_document": true, "registered": registered, "grants": map[string]Scope{"a": ScopeRead}},
			chat:    map[string]any{"id": chat, "name": "Synthetic Chat", "kind": "claude.ai", "redirect_uris": []string{web}, "registered": registered, "grants": map[string]Scope{"a": ScopeReadWrite}},
			pending: map[string]any{"id": pending, "name": "Synthetic Pending", "kind": "claude.ai", "redirect_uris": []string{web}, "registered": now.Add(-time.Hour)},
		},
		"tokens": tokens,
		"owner":  map[string]any{},
	}
	path := filepath.Join(c.StateDir, oauthStateFile)
	if err := writeState(path, state); err != nil {
		t.Fatal(err)
	}

	// The admin CLI reads the 2.0.1 file as installations before anything
	// saves it, with the IDs the server will keep.
	before, err := Clients(c)
	if err != nil {
		t.Fatal(err)
	}
	ids := func(list []ClientInfo) map[string][]string {
		out := map[string][]string{}
		for _, info := range list {
			out[info.ID] = append(out[info.ID], info.Installation)
		}
		return out
	}
	if got := ids(before); len(got[code]) != 3 || len(got[chat]) != 1 || !slices.Equal(got[pending], []string{""}) || got[gone] != nil {
		t.Fatalf("migrated list %v", got)
	}
	for _, info := range installations(t, c, code) {
		if info.Grants["a"] != ScopeReadWrite || info.FirstSeen != registered || !info.LastUsed.After(now.Add(-time.Hour)) {
			t.Fatalf("migrated installation %+v", info)
		}
	}

	e := startOAuthConfig(t, c)
	e.enrollOwner(t)
	for name, p := range families {
		if e.mcpStatus(t, "/mcp", p.access) != http.StatusOK || !canWrite(t, e, p.access) {
			t.Fatalf("%s: a migrated access token stopped working", name)
		}
	}
	// The DCR hosted app re-authorizes with no consent page.
	_, challenge := pkce()
	if status, _ := authorizePage(t, e.browser, authorizeURL(chat, web, challenge, publicURL+"/mcp", "")); status != http.StatusSeeOther {
		t.Fatalf("hosted app re-authorization: %d", status)
	}
	// Every migrated refresh token rotates, under its own installation.
	for name, p := range families {
		if resp, out := e.refresh(t, clientOf(name, code, chat), p.refresh); resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: refresh %d %v", name, resp.StatusCode, out)
		}
	}
	if got := ids(mustClients(t, c)); !slices.Equal(got[code], ids(before)[code]) || !slices.Equal(got[chat], ids(before)[chat]) {
		t.Fatalf("installation IDs changed on save: %v → %v", ids(before), got)
	}
	// Replaying the migrated rotated token ends only its family.
	if resp, _ := e.refresh(t, code, families["code-0"].used); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("replay of a migrated rotated token: %d", resp.StatusCode)
	}
	// The other two each hold their old access token and the new pair.
	var live []int
	for _, info := range installations(t, c, code) {
		live = append(live, info.Tokens)
	}
	if slices.Sort(live); !slices.Equal(live, []int{0, 3, 3}) {
		t.Fatalf("live tokens per installation after the replay: %v", live)
	}

	// Rollback: 2.0.1 reads the saved state as per-client grants, so every
	// token it holds still has its client's grant.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var old v1State
	if err := json.Unmarshal(raw, &old); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"version": 2`) {
		t.Fatal("the saved state has no version")
	}
	for h, tok := range old.Tokens {
		if cl := old.Clients[tok.Client]; cl == nil || cl.Grants[tok.Signet] != ScopeReadWrite {
			t.Fatalf("token %s has no 2.0.1 grant", h)
		}
	}
	// And upgrading again after 2.0.1 rewrote the file migrates again.
	stripped, _ := json.Marshal(old)
	if err := os.WriteFile(path, stripped, 0600); err != nil {
		t.Fatal(err)
	}
	if got := ids(mustClients(t, c)); len(got[code]) != 2 || len(got[chat]) != 1 {
		t.Fatalf("re-migrated list %v", got)
	}
}

func clientOf(name, code, chat string) string {
	if name == "chat" {
		return chat
	}
	return code
}

func mustClients(t *testing.T, c Config) []ClientInfo {
	t.Helper()
	out, err := Clients(c)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// userAgent sets the User-Agent of every request the client sends.
func userAgent(c *http.Client, ua string) {
	base := c.Transport
	c.Transport = roundTripper(func(r *http.Request) (*http.Response, error) {
		r = r.Clone(r.Context())
		r.Header.Set("User-Agent", ua)
		return base.RoundTrip(r)
	})
}

// installationOf returns the one listed installation of a client.
func installationOf(t *testing.T, c Config, client string) ClientInfo {
	t.Helper()
	listed := installations(t, c, client)
	if len(listed) != 1 {
		t.Fatalf("installations of %s: %+v", client, listed)
	}
	return listed[0]
}

// Issue #217: the owner tells installations apart by the approving
// browser's hint, a label of their own and a last_used that authenticated
// MCP use moves, at most once a minute.
func TestInstallationIdentity(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	docURL := serveDocument(t, e)
	e.enrollOwner(t)
	browser := "Mozilla/5.0 (Synthetic OS 1.0) SyntheticBrowser/1.0"
	// HTTP carries tabs and non-ASCII bytes in a header, but no escapes.
	userAgent(e.browser, browser+"\u202e\t"+strings.Repeat(" padding", 30))
	tok := e.tokens(t, docURL, "http://127.0.0.1:7/callback", "/mcp", "", ScopeRead)
	access := tok["access_token"].(string)
	in := installationOf(t, e.config, docURL)
	if !strings.HasPrefix(in.Hint, browser+" padding") || len([]rune(in.Hint)) > maxHint || strings.ContainsAny(in.Hint, "\t\u202e") {
		t.Fatalf("hint %q", in.Hint)
	}
	for ua, want := range map[string]string{"\x1b[2J" + browser + "\r\n": "[2J" + browser, "Synthetic\xff/1": "Synthetic/1", "": ""} {
		if got := userAgentHint(ua); got != want {
			t.Errorf("hint of %q: %q, want %q", ua, got, want)
		}
	}
	// The installation records the hint and nothing else about the peer.
	raw, err := os.ReadFile(filepath.Join(e.config.StateDir, oauthStateFile))
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		Installations map[string]map[string]any `json:"installations"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if keys := slices.Sorted(maps.Keys(saved.Installations[in.Installation])); !slices.Equal(keys, []string{"client", "first_seen", "grants", "hint", "id", "last_used"}) {
		t.Fatalf("saved installation fields %v", keys)
	}

	// MCP use within a minute of the last write saves nothing.
	issued := in.LastUsed
	e.clock.advance(lastUsedEvery - time.Second)
	if e.mcpStatus(t, "/mcp", access) != http.StatusOK {
		t.Fatal("MCP call refused")
	}
	if got := installationOf(t, e.config, docURL).LastUsed; !got.Equal(issued) {
		t.Fatalf("last_used moved within a minute: %v → %v", issued, got)
	}
	// Past it, the next call moves last_used, and calls just after it don't.
	e.clock.advance(time.Second)
	used := e.clock.now()
	if e.mcpStatus(t, "/mcp", access) != http.StatusOK {
		t.Fatal("MCP call refused")
	}
	e.clock.advance(30 * time.Second)
	_ = e.mcpStatus(t, "/mcp", access)
	if got := installationOf(t, e.config, docURL).LastUsed; !got.Equal(used) {
		t.Fatalf("last_used %v, want %v", got, used)
	}
	// An unknown token moves nothing.
	e.clock.advance(time.Hour - 2*time.Minute)
	if e.mcpStatus(t, "/mcp", secret()) != http.StatusUnauthorized {
		t.Fatal("unknown token accepted")
	}
	if got := installationOf(t, e.config, docURL).LastUsed; !got.Equal(used) {
		t.Fatalf("an unknown token moved last_used: %v", got)
	}

	// The owner's label, trimmed; refused labels change nothing.
	id := in.Installation
	if err := LabelInstallation(e.config, id, "  Synthetic desk "); err != nil {
		t.Fatal(err)
	}
	for name, label := range map[string]string{
		"escape":      "desk\x1b[2J",
		"newline":     "desk\nforged line",
		"bidi":        "desk\u202e",
		"too long":    strings.Repeat("é", maxLabel+1),
		"not UTF-8":   "desk\xff",
		"zero width":  "de\u200bsk",
		"tab":         "desk\tnote",
		"delete char": "desk\x7f",
	} {
		if err := LabelInstallation(e.config, id, label); err == nil {
			t.Errorf("%s: label accepted", name)
		}
	}
	if err := LabelInstallation(e.config, "inst-unknown", "desk"); err == nil {
		t.Error("labelled an unknown installation")
	}
	if got := installationOf(t, e.config, docURL); got.Label != "Synthetic desk" || got.Hint != in.Hint {
		t.Fatalf("after labelling: %+v", got)
	}
	if err := LabelInstallation(e.config, id, strings.Repeat("é", maxLabel)); err != nil {
		t.Fatalf("a label of %d characters: %v", maxLabel, err)
	}
	if err := LabelInstallation(e.config, id, ""); err != nil || installationOf(t, e.config, docURL).Label != "" {
		t.Fatalf("clearing the label: %v", err)
	}
	if err := LabelInstallation(e.config, id, "Synthetic desk"); err != nil {
		t.Fatal(err)
	}
	// A rotated refresh keeps the label and hint.
	if resp, out := e.refresh(t, docURL, tok["refresh_token"].(string)); resp.StatusCode != http.StatusOK {
		t.Fatalf("refresh: %d %v", resp.StatusCode, out)
	} else {
		access = out["access_token"].(string)
	}
	if got := installationOf(t, e.config, docURL); got.Label != "Synthetic desk" || got.Hint != in.Hint || !got.LastUsed.Equal(e.clock.now()) {
		t.Fatalf("after refresh: %+v", got)
	}
}

// Issue #217: a late failure to save last_used, the label or a prune, on a
// read-only state directory, never refuses MCP use and changes nothing.
func TestInstallationWritesReadOnlyState(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores the read-only state directory this test uses to fail the write")
	}
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	docURL := serveDocument(t, e)
	e.enrollOwner(t)
	access := e.tokens(t, docURL, "http://127.0.0.1:7/callback", "/mcp", "", ScopeRead)["access_token"].(string)
	before := installationOf(t, e.config, docURL)
	if err := os.Chmod(e.config.StateDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(e.config.StateDir, 0700) })
	e.clock.advance(2 * lastUsedEvery)
	for range 3 {
		if e.mcpStatus(t, "/mcp", access) != http.StatusOK {
			t.Fatal("a failed last_used save refused the MCP call")
		}
	}
	if err := LabelInstallation(e.config, before.Installation, "Synthetic desk"); err == nil {
		t.Fatal("label saved on a read-only state directory")
	}
	if _, err := PruneInstallations(e.config); err == nil {
		t.Fatal("prune saved on a read-only state directory")
	}
	if got := installationOf(t, e.config, docURL); !got.LastUsed.Equal(before.LastUsed) || got.Label != "" {
		t.Fatalf("a failed save changed the installation: %+v", got)
	}
	// Writable again, the next call past the minute saves last_used.
	if err := os.Chmod(e.config.StateDir, 0700); err != nil {
		t.Fatal(err)
	}
	e.clock.advance(lastUsedEvery)
	if e.mcpStatus(t, "/mcp", access) != http.StatusOK || !installationOf(t, e.config, docURL).LastUsed.Equal(e.clock.now()) {
		t.Fatal("last_used did not recover once the state was writable")
	}
}

// Issue #217: corrupt state makes labelling, pruning and listing fail
// without rewriting the file.
func TestInstallationWritesCorruptState(t *testing.T) {
	c := oauthConfig(t, testfixture.NewServerSignet(t, "alpha"))
	path := filepath.Join(c.StateDir, oauthStateFile)
	corrupt := []byte(`{"installations": corrupt`)
	if err := os.WriteFile(path, corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	if err := LabelInstallation(c, "inst-0000000000000000", "Synthetic desk"); err == nil {
		t.Error("labelled in corrupt state")
	}
	if _, err := PruneInstallations(c); err == nil {
		t.Error("pruned corrupt state")
	}
	if _, err := Clients(c); err == nil {
		t.Error("listed corrupt state")
	}
	if raw, _ := os.ReadFile(path); string(raw) != string(corrupt) {
		t.Fatal("corrupt state was rewritten")
	}
}

// Issue #217 (#218 item 1): an installation of a document client with no
// live token is listed as expired and can be pruned. A DCR installation
// with no live token stays approved, since its client authorizes again with
// its grant, and an approval whose code is unredeemed is not expired.
func TestExpiredInstallations(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	docURL := serveDocument(t, e)
	e.enrollOwner(t)
	redirect, web := "http://127.0.0.1:7/callback", "https://claude.ai/api/mcp/auth_callback"
	// The admin commands read the wall clock, so the server's clock starts
	// in the past, and what it does now is that long ago for them.
	e.clock.advance(-10 * time.Minute)
	gone := e.tokens(t, docURL, redirect, "/mcp", "", ScopeRead)
	kept := e.tokens(t, docURL, redirect, "/mcp", "", ScopeRead)
	dcr := e.registerClient(t, "Synthetic Chat", web)
	chat := e.tokens(t, dcr, web, "/mcp", "", ScopeRead)
	for client, tok := range map[string]map[string]any{docURL: gone, dcr: chat} {
		if resp, _ := e.revoke(t, url.Values{"client_id": {client}, "token": {tok["refresh_token"].(string)}}); resp.StatusCode != http.StatusOK {
			t.Fatalf("logout: %d", resp.StatusCode)
		}
	}
	// An approval made just now, whose code is unredeemed.
	e.clock.advance(10 * time.Minute)
	_, challenge := pkce()
	if loc := e.visit(t, authorizeURL(docURL, redirect, challenge, publicURL+"/mcp", ""), ScopeRead); loc.Query().Get("code") == "" {
		t.Fatalf("no code: %v", loc)
	}
	status := func() map[string]int {
		out := map[string]int{}
		for _, info := range mustClients(t, e.config) {
			out[info.Status]++
			if info.Status == "expired" && info.Tokens != 0 {
				t.Fatalf("expired with live tokens: %+v", info)
			}
		}
		return out
	}
	// The logged-out installation has expired; the live one, the DCR
	// installation and the unredeemed approval have not.
	if got := status(); got["approved"] != 3 || got["expired"] != 1 {
		t.Fatalf("statuses %v", got)
	}
	// The unredeemed approval expires with its code.
	if err := e.server.oauth.store.view(func(st *oauthState) error {
		for _, at := range []time.Duration{codeTTL - time.Second, codeTTL} {
			now := e.clock.now().Add(at)
			live, n := st.liveTokens(now), 0
			for _, in := range st.Installations {
				if st.expired(in, live, now) {
					n++
				}
			}
			if want := map[bool]int{false: 1, true: 2}[at == codeTTL]; n != want {
				t.Errorf("%v after the approval: %d expired, want %d", at, n, want)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	pruned, err := PruneInstallations(e.config)
	if err != nil || len(pruned) != 1 {
		t.Fatalf("prune: %v %v", pruned, err)
	}
	if got := status(); got["approved"] != 3 || got["expired"] != 0 {
		t.Fatalf("statuses after prune %v", got)
	}
	if e.mcpStatus(t, "/mcp", kept["access_token"].(string)) != http.StatusOK {
		t.Fatal("prune ended a live installation")
	}
	if pruned, err := PruneInstallations(e.config); err != nil || len(pruned) != 0 {
		t.Fatalf("second prune: %v %v", pruned, err)
	}
	// The DCR client still re-authorizes without consent.
	_, challenge = pkce()
	if status, _ := authorizePage(t, e.browser, authorizeURL(dcr, web, challenge, publicURL+"/mcp", "")); status != http.StatusSeeOther {
		t.Fatalf("DCR re-authorization: %d", status)
	}
}

// Issue #218 item 1: a 2.0.1 family whose tokens have all expired, but are
// not yet pruned from the file, migrates to an expired installation that
// prune removes with its client.
func TestMigratedExpiredFamily(t *testing.T) {
	c := oauthConfig(t, testfixture.NewServerSignet(t, "alpha"))
	doc := "https://client.example.com/oauth/client-metadata"
	issued := time.Now().Add(-refreshTTL - time.Hour)
	state := map[string]any{
		"clients": map[string]any{doc: map[string]any{"id": doc, "name": "Synthetic Code", "kind": "client.example.com", "redirect_uris": []string{"http://localhost/callback"},
			"metadata_document": true, "registered": issued.Add(-time.Hour), "grants": map[string]Scope{"a": ScopeRead}}},
		"tokens": map[string]any{
			hashSecret(secret()): map[string]any{"kind": tokenAccess, "client": doc, "signet": "a", "family": "family-gone", "scope": ScopeRead, "expires": issued.Add(accessTTL)},
			hashSecret(secret()): map[string]any{"kind": tokenRefresh, "client": doc, "signet": "a", "family": "family-gone", "scope": ScopeRead, "expires": issued.Add(refreshTTL)},
		},
		"owner": map[string]any{},
	}
	if err := writeState(filepath.Join(c.StateDir, oauthStateFile), state); err != nil {
		t.Fatal(err)
	}
	if in := installationOf(t, c, doc); in.Status != "expired" || in.Tokens != 0 {
		t.Fatalf("migrated expired family: %+v", in)
	}
	pruned, err := PruneInstallations(c)
	if err != nil || len(pruned) != 1 {
		t.Fatalf("prune: %v %v", pruned, err)
	}
	if got := mustClients(t, c); len(got) != 0 {
		t.Fatalf("after prune: %+v", got)
	}
}

// authorOf reads the authorship of a saved revision.
func authorOf(t *testing.T, s testfixture.ServerSignet, saved Receipt) memory.Authorship {
	t.Helper()
	var r memory.Revision
	readJSON(t, filepath.Join(s.Root, "memory/records", saved.RecordID, saved.ID+".json"), &r)
	return r.Authorship
}

// Issue #211: two installations of one metadata-document client write with
// distinct authorship, and the client's device from before 2.0.3 keeps the
// records it authored.
func TestInstallationsHaveTheirOwnAuthorship(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := testfixture.NewServerSignet(t, "alpha")
	e := startOAuth(t, a)
	docURL := serveDocument(t, e)
	e.enrollOwner(t)
	// 2.0.2 keyed a client's device by its client_id alone.
	kind := clientKind(docURL, true, nil)
	res := e.server.signets["/mcp"].call(context.Background(), Client{ID: docURL, Name: "Synthetic Code", Kind: kind, Scope: ScopeReadWrite}, "memory_remember", mustJSON(t, rememberArgs("authorship-before-0001", "Written before 2.0.3")))
	before := receipt(t, res.Content[0].(*sdk.TextContent).Text)
	shared := authorOf(t, a, before)

	redirect := "http://127.0.0.1:7/callback"
	first := e.tokens(t, docURL, redirect, "/mcp", "", ScopeReadWrite)
	e.clock.advance(time.Second)
	second := e.tokens(t, docURL, redirect, "/mcp", "", ScopeReadWrite)
	listed := installations(t, e.config, docURL)
	if len(listed) != 2 {
		t.Fatalf("installations %+v", listed)
	}
	devices := map[string]string{}
	write := func(token, id string) memory.Authorship {
		t.Helper()
		_, text := call(t, e.bearer(t, "/mcp", token), "memory_remember", rememberArgs(id, "Written by "+id))
		return authorOf(t, a, receipt(t, text))
	}
	// Each installation writes twice, interleaved, so a handler or device
	// cached for the first can't serve the second.
	for _, round := range []string{"0001", "0002"} {
		for i, tok := range []map[string]any{first, second} {
			got := write(tok["access_token"].(string), "authorship-installation-"+string(rune('a'+i))+round)
			in := listed[i].Installation
			if prev, ok := devices[in]; ok && prev != got.DeviceID {
				t.Fatalf("installation %s changed device: %s → %s", in, prev, got.DeviceID)
			}
			devices[in] = got.DeviceID
			if got.Actor != "Synthetic Owner" || got.Harness != kind || got.DeviceID == shared.DeviceID {
				t.Fatalf("installation %s authorship %+v", in, got)
			}
		}
	}
	if devices[listed[0].Installation] == devices[listed[1].Installation] {
		t.Fatalf("two installations share a device: %v", devices)
	}
	saved := map[string]string{}
	readJSON(t, filepath.Join(e.config.StateDir, "signets", "a", "devices.json"), &saved)
	if want := map[string]string{docURL: shared.DeviceID, listed[0].Installation: devices[listed[0].Installation], listed[1].Installation: devices[listed[1].Installation]}; !maps.Equal(saved, want) {
		t.Fatalf("devices.json %v, want %v", saved, want)
	}
	// Provenance names each installation, so the owner can match a device
	// to `server clients list`.
	for in, device := range devices {
		var d memory.Device
		readJSON(t, filepath.Join(a.Root, "provenance/devices", device+".json"), &d)
		if !strings.HasPrefix(d.Label, "Synthetic Code installation "+in+" ("+kind+"), approved ") {
			t.Fatalf("device label %q", d.Label)
		}
	}
	// The record written before 2.0.3 keeps its authorship.
	if got := authorOf(t, a, before); got != shared {
		t.Fatalf("historical authorship %+v, want %+v", got, shared)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// Issue #211: state as 2.0.2 saves it, with two Claude Code installations it
// migrated (no hint or label) and a hosted-app DCR client, each with a device
// 2.0.2 enrolled per client_id. After the upgrade every live token keeps
// working with no consent, the hosted app keeps its device, and each Claude
// Code installation writes as its own device.
func TestUpgradeFrom202KeepsGrants(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := testfixture.NewServerSignet(t, "alpha")
	c := oauthConfig(t, a)
	now := time.Now().UTC().Truncate(time.Second)
	code := "https://claude.ai/oauth/claude-code-client-metadata"
	chat, web := "client-synthetichostedapp0001", "https://claude.ai/api/mcp/auth_callback"
	registered := now.Add(-48 * time.Hour)
	st := oauthState{Version: stateVersion, Clients: map[string]*oauthClient{
		code: {ID: code, Name: "Claude Code", Kind: "claude.ai", RedirectURIs: []string{"http://localhost/callback"}, Document: true, Registered: registered, Legacy: map[string]Scope{"a": ScopeReadWrite}},
		chat: {ID: chat, Name: "Synthetic Chat", Kind: "claude.ai", RedirectURIs: []string{web}, Registered: registered, Legacy: map[string]Scope{"a": ScopeReadWrite}},
	}, Installations: map[string]*installation{}, Tokens: map[string]*oauthToken{}}
	type family struct{ client, installation, access, refresh string }
	var families []family
	for i, client := range []string{code, code, chat} {
		in := "inst-" + hashSecret("migrated\x00" + client + string(rune('0'+i)))[:16]
		issued := now.Add(-time.Duration(i+1) * 10 * time.Minute)
		st.Installations[in] = &installation{ID: in, Client: client, Grants: map[string]Scope{"a": ScopeReadWrite}, FirstSeen: registered, LastUsed: issued}
		f := family{client: client, installation: in, access: secret(), refresh: secret()}
		id := hashSecret(secret())
		st.Tokens[hashSecret(f.access)] = &oauthToken{Kind: tokenAccess, Client: client, Installation: in, Signet: "a", Family: id, Scope: ScopeReadWrite, Expires: issued.Add(accessTTL)}
		st.Tokens[hashSecret(f.refresh)] = &oauthToken{Kind: tokenRefresh, Client: client, Installation: in, Signet: "a", Family: id, Scope: ScopeReadWrite, Expires: issued.Add(refreshTTL)}
		families = append(families, f)
	}
	path := filepath.Join(c.StateDir, oauthStateFile)
	if err := writeState(path, st); err != nil {
		t.Fatal(err)
	}
	before := mustClients(t, c)

	e := startOAuthConfig(t, c)
	e.enrollOwner(t)
	// 2.0.2 enrolled one device per client_id.
	old := map[string]memory.Authorship{}
	for _, client := range []string{code, chat} {
		res := e.server.signets["/mcp"].call(context.Background(), Client{ID: client, Name: st.Clients[client].Name, Kind: "claude.ai", Scope: ScopeReadWrite}, "memory_remember", mustJSON(t, rememberArgs("upgrade-before-"+client[len(client)-8:], "Written by 2.0.2")))
		old[client] = authorOf(t, a, receipt(t, res.Content[0].(*sdk.TextContent).Text))
	}

	devices := map[string]string{}
	for _, f := range families {
		if e.mcpStatus(t, "/mcp", f.access) != http.StatusOK || !canWrite(t, e, f.access) {
			t.Fatalf("%s: a 2.0.2 access token stopped working", f.installation)
		}
		_, text := call(t, e.bearer(t, "/mcp", f.access), "memory_remember", rememberArgs("upgrade-after-"+f.installation, "Written by 2.0.3"))
		got := authorOf(t, a, receipt(t, text))
		devices[f.installation] = got.DeviceID
		if shared := got.DeviceID == old[f.client].DeviceID; shared != (f.client == chat) {
			t.Fatalf("%s wrote as %s; 2.0.2's device for %s is %s", f.installation, got.DeviceID, f.client, old[f.client].DeviceID)
		}
		if resp, out := e.refresh(t, f.client, f.refresh); resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: refresh %d %v", f.installation, resp.StatusCode, out)
		}
	}
	if devices[families[0].installation] == devices[families[1].installation] {
		t.Fatalf("the two Claude Code installations share a device: %v", devices)
	}
	// The hosted app re-authorizes with no consent page.
	_, challenge := pkce()
	if status, _ := authorizePage(t, e.browser, authorizeURL(chat, web, challenge, publicURL+"/mcp", "")); status != http.StatusSeeOther {
		t.Fatalf("hosted app re-authorization: %d", status)
	}
	after := mustClients(t, c)
	if len(after) != len(before) {
		t.Fatalf("clients %+v → %+v", before, after)
	}
	for i := range after {
		if after[i].Installation != before[i].Installation || after[i].Hint != "" || after[i].Status != "approved" {
			t.Fatalf("installation %+v, was %+v", after[i], before[i])
		}
	}
}
