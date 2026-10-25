package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/testfixture"
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
