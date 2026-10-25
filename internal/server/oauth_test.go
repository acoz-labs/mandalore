package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

const invalidTokenChallenge = `Bearer error="invalid_token", resource_metadata="` + publicURL + `/.well-known/oauth-protected-resource"`

func toolNames(t *testing.T, e *oauthEnv, path, token string) []string {
	t.Helper()
	tools, err := e.bearer(t, path, token).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	return names
}

// checkTimings asserts acceptance 8: every OAuth endpoint answered in under
// 200 ms.
func checkTimings(t *testing.T, e *oauthEnv) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	seen := 0
	for path, ds := range e.timings {
		if strings.HasSuffix(path, "/mcp") {
			continue
		}
		for _, d := range ds {
			seen++
			if d >= 200*time.Millisecond {
				t.Errorf("%s took %s", path, d)
			}
		}
	}
	if seen == 0 {
		t.Fatal("no OAuth requests were timed")
	}
}

// Acceptance 1 and 8: the go-sdk authorization code handler registers by
// DCR, the owner approves, it gets tokens and makes an MCP write.
func TestOAuthFlowWithDynamicRegistration(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	e := startOAuth(t, a)
	e.enrollOwner(t)
	h, err := auth.NewAuthorizationCodeHandler(&auth.AuthorizationCodeHandlerConfig{
		DynamicClientRegistrationConfig: &auth.DynamicClientRegistrationConfig{Metadata: &oauthex.ClientRegistrationMetadata{
			ClientName: "Synthetic Chat", RedirectURIs: []string{"https://claude.ai/api/mcp/auth_callback"}, GrantTypes: []string{"authorization_code", "refresh_token"}}},
		AuthorizationCodeFetcher: e.fetcher(t, ScopeReadWrite), RequestRefreshToken: true, Client: e.agent,
	})
	if err != nil {
		t.Fatal(err)
	}
	session := e.connect(t, "/mcp", h)
	// Approval enrolled the client's device before its first write.
	devices := map[string]string{}
	if err := readState(filepath.Join(e.config.StateDir, "signets", "a", "devices.json"), &devices); err != nil || len(devices) != 1 {
		t.Fatalf("devices %v %v", devices, err)
	}
	client := onlyClient(t, e)
	device := devices[client]
	var d memory.Device
	readJSON(t, filepath.Join(a.Root, "provenance/devices", device+".json"), &d)
	if !strings.HasPrefix(d.Label, "Synthetic Chat (claude.ai), approved ") {
		t.Fatalf("device label %q", d.Label)
	}
	_, text := call(t, session, "memory_remember", rememberArgs("oauth-dcr-request-0001", "Saved through OAuth"))
	saved := receipt(t, text)
	var r memory.Revision
	readJSON(t, filepath.Join(a.Root, "memory/records", saved.RecordID, saved.ID+".json"), &r)
	if want := (memory.Authorship{DeviceID: device, Actor: "Synthetic Owner", Harness: "claude.ai"}); r.Authorship != want {
		t.Fatalf("authorship %+v", r.Authorship)
	}
	ts, _ := h.TokenSource(context.Background())
	tok, err := ts.Token()
	if err != nil || tok.RefreshToken == "" {
		t.Fatalf("no refresh token issued: %v", err)
	}
	if got := strings.Fields(tok.Extra("scope").(string)); !slices.Equal(got, []string{scopeRead, scopeWrite}) {
		t.Fatalf("scope %v", got)
	}
	// A second authorization by the approved client skips consent.
	_, challenge := pkce()
	resp, err := e.browser.Get(authorizeURL(client, "https://claude.ai/api/mcp/auth_callback", challenge, publicURL+"/mcp", ""))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "code=") {
		t.Fatalf("approved client did not skip consent: %d", resp.StatusCode)
	}
	checkTimings(t, e)
}

func onlyClient(t *testing.T, e *oauthEnv) string {
	t.Helper()
	clients, err := Clients(e.config)
	if err != nil || len(clients) != 1 {
		t.Fatalf("clients %v %v", clients, err)
	}
	return clients[0].ID
}

// Acceptance 1 (CIMD), 2 and 8: a client ID metadata document served over
// TLS, a loopback redirect whose port differs from the registered one, and
// no metadata fetch on the token path.
func TestOAuthFlowWithClientMetadataDocument(t *testing.T) {
	var hits atomic.Int32
	var docURL string
	doc := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"client_id": docURL, "client_name": "Synthetic Code", "redirect_uris": []string{"http://127.0.0.1/callback"},
			"grant_types": []string{"authorization_code", "refresh_token"}, "token_endpoint_auth_method": "none"})
	}))
	t.Cleanup(doc.Close)
	docURL = doc.URL + "/oauth/client-metadata"
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	// The test document is on loopback, which production fetches refuse.
	e.server.oauth.documents.client = doc.Client()
	e.enrollOwner(t)
	h, err := auth.NewAuthorizationCodeHandler(&auth.AuthorizationCodeHandlerConfig{
		ClientIDMetadataDocumentConfig: &auth.ClientIDMetadataDocumentConfig{URL: docURL}, RedirectURL: "http://127.0.0.1:9/callback",
		AuthorizationCodeFetcher: e.fetcher(t, ScopeRead), RequestRefreshToken: true, Client: e.agent,
	})
	if err != nil {
		t.Fatal(err)
	}
	session := e.connect(t, "/mcp", h)
	if _, text := call(t, session, "memory_scopes", map[string]any{}); !strings.Contains(text, `"ok":true`) {
		t.Fatalf("read failed: %s", text)
	}
	if hits.Load() != 1 {
		t.Fatalf("metadata document fetched %d times, want 1", hits.Load())
	}
	ts, _ := h.TokenSource(context.Background())
	tok, _ := ts.Token()
	before := hits.Load()
	resp, out := e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {docURL}, "refresh_token": {tok.RefreshToken}})
	if resp.StatusCode != http.StatusOK || hits.Load() != before {
		t.Fatalf("refresh: %d %v, fetches %d → %d", resp.StatusCode, out, before, hits.Load())
	}
	if c := onlyClient(t, e); c != docURL {
		t.Fatalf("stored client %q", c)
	}
	checkTimings(t, e)
}

// An approved client skips at most the consent page, never the owner: client
// IDs are not secret, so a request without the owner session gets the sign-in
// page and no code, for DCR and metadata-document clients alike.
func TestApprovedClientStillNeedsTheOwner(t *testing.T) {
	var docURL string
	doc := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"client_id": docURL, "client_name": "Synthetic Code", "redirect_uris": []string{"http://127.0.0.1/callback"}})
	}))
	t.Cleanup(doc.Close)
	docURL = doc.URL + "/oauth/client-metadata"
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.server.oauth.documents.client = doc.Client()
	e.enrollOwner(t)
	web := "https://claude.ai/api/mcp/auth_callback"
	dcr := e.registerClient(t, "Synthetic Chat", web)
	e.tokens(t, dcr, web, "/mcp", "", ScopeReadWrite)
	_, challenge := pkce()
	// The consent page names the host that publishes a document client.
	resp, err := e.browser.Get(authorizeURL(docURL, "http://127.0.0.1:7/callback", challenge, publicURL+"/mcp", ""))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "Published by <strong>127.0.0.1</strong>") {
		t.Fatalf("consent page does not name the document's publisher: %s", body)
	}
	e.visit(t, authorizeURL(docURL, "http://127.0.0.1:7/callback", challenge, publicURL+"/mcp", ""), ScopeReadWrite)
	stranger := &http.Client{Transport: e.agent.Transport, CheckRedirect: e.browser.CheckRedirect}
	for client, redirect := range map[string]string{dcr: web, docURL: "http://127.0.0.1:6666/callback"} {
		verifier, challenge := pkce()
		codes := func() int {
			e.server.oauth.mu.Lock()
			defer e.server.oauth.mu.Unlock()
			return len(e.server.oauth.codes)
		}
		before := codes()
		resp, err := stranger.Get(authorizeURL(client, redirect, challenge, publicURL+"/mcp", scopeRead+" "+scopeWrite))
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || resp.Header.Get("Location") != "" || !strings.Contains(string(body), `id="login"`) || strings.Contains(string(body), "code=") {
			t.Fatalf("%s without the owner: %d %q", client, resp.StatusCode, resp.Header.Get("Location"))
		}
		if codes() != before {
			t.Fatalf("%s: a cookieless request created a code", client)
		}
		if resp, out := e.postForm(t, stranger, "/token", url.Values{"grant_type": {"authorization_code"}, "client_id": {client}, "code": {"guessed"}, "redirect_uri": {redirect}, "code_verifier": {verifier}}); resp.StatusCode != http.StatusBadRequest || out["error"] != "invalid_grant" {
			t.Fatalf("%s token: %d %v", client, resp.StatusCode, out)
		}
		// With the owner's live session the same request gets a code: at once
		// for the DCR client, after consent for a new document installation.
		if loc := e.visit(t, authorizeURL(client, redirect, challenge, publicURL+"/mcp", ""), ScopeRead); loc.Query().Get("code") == "" {
			t.Fatalf("%s with the owner: %s", client, loc)
		}
	}
}

// Acceptance 2: exact redirect matching, loopback ports ignored.
func TestRedirectMatching(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	client := e.registerClient(t, "Synthetic", "https://claude.ai/api/mcp/auth_callback", "http://localhost/callback", "http://127.0.0.1/callback")
	_, challenge := pkce()
	for redirect, ok := range map[string]bool{
		"https://claude.ai/api/mcp/auth_callback":      true,
		"http://localhost:53122/callback":              true,
		"http://127.0.0.1:9/callback":                  true,
		"http://localhost/callback":                    true,
		"http://evil.test/":                            false,
		"https://claude.ai.evil/api/mcp/auth_callback": false,
		"https://claude.ai/api/mcp/auth_callback/":     false,
		"https://claude.ai:8443/api/mcp/auth_callback": false,
		"http://claude.ai/api/mcp/auth_callback":       false,
		"http://localhost:53122/other":                 false,
		"http://localhost:53122/callback?x=1":          false,
		"http://127.0.0.1:9/callback#frag":             false,
		"http://localhost.evil.test/callback":          false,
		"http://[::1]:9/callback":                      false,
	} {
		resp, err := e.browser.Get(authorizeURL(client, redirect, challenge, publicURL+"/mcp", ""))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		// Without an owner session an accepted request shows the sign-in page;
		// a refused one shows an error and never redirects.
		if got := resp.StatusCode == http.StatusOK; got != ok || resp.Header.Get("Location") != "" {
			t.Errorf("%s: status %d, want accepted=%v", redirect, resp.StatusCode, ok)
		}
	}
	for _, bad := range []string{"http://evil.test/callback", "ftp://claude.ai/x", "https://user@claude.ai/x", "https://claude.ai/x#f", "/relative"} {
		if resp, _ := e.postJSON(t, e.agent, "/register", map[string]any{"redirect_uris": []string{bad}}); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("registered %s: %d", bad, resp.StatusCode)
		}
	}
}

// Acceptance 3: S256 PKCE is required and checked.
func TestPKCEIsRequired(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	verifier, challenge := pkce()
	for name, mutate := range map[string]func(url.Values){
		"plain":             func(v url.Values) { v.Set("code_challenge_method", "plain"); v.Set("code_challenge", verifier) },
		"missing challenge": func(v url.Values) { v.Del("code_challenge") },
		"missing method":    func(v url.Values) { v.Del("code_challenge_method") },
		"short challenge":   func(v url.Values) { v.Set("code_challenge", "abc") },
	} {
		u, _ := url.Parse(authorizeURL(client, redirect, challenge, publicURL+"/mcp", ""))
		q := u.Query()
		mutate(q)
		u.RawQuery = q.Encode()
		loc := e.visit(t, u.String(), ScopeRead)
		if loc.Query().Get("error") != "invalid_request" || loc.Query().Get("code") != "" || loc.Query().Get("state") != "synthetic-state" {
			t.Errorf("%s: redirected to %s", name, loc)
		}
	}
	loc := e.visit(t, authorizeURL(client, redirect, challenge, publicURL+"/mcp", ""), ScopeRead)
	_, wrong := pkce()
	resp, out := e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"authorization_code"}, "client_id": {client}, "code": {loc.Query().Get("code")}, "redirect_uri": {redirect}, "code_verifier": {wrong}})
	if resp.StatusCode != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Fatalf("wrong verifier: %d %v", resp.StatusCode, out)
	}
	// The code was spent by the failed attempt.
	resp, out = e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"authorization_code"}, "client_id": {client}, "code": {loc.Query().Get("code")}, "redirect_uri": {redirect}, "code_verifier": {verifier}})
	if resp.StatusCode != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Fatalf("reused code: %d %v", resp.StatusCode, out)
	}
}

// Acceptance 4: refresh tokens rotate, and replaying a rotated one revokes
// the whole family, including the access token issued with its successor.
func TestRefreshRotationAndReplay(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	first := e.tokens(t, client, redirect, "/mcp", "", ScopeReadWrite)
	refresh := func(token string) (int, map[string]any) {
		resp, out := e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {token}})
		return resp.StatusCode, out
	}
	code, second := refresh(first["refresh_token"].(string))
	if code != http.StatusOK || second["refresh_token"] == first["refresh_token"] {
		t.Fatalf("rotation: %d %v", code, second)
	}
	if e.mcpStatus(t, "/mcp", second["access_token"].(string)) != http.StatusOK {
		t.Fatal("rotated access token refused")
	}
	if code, out := refresh(first["refresh_token"].(string)); code != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Fatalf("replay: %d %v", code, out)
	}
	if code, out := refresh(second["refresh_token"].(string)); code != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Fatalf("successor after replay: %d %v", code, out)
	}
	for _, tok := range []any{first["access_token"], second["access_token"]} {
		if e.mcpStatus(t, "/mcp", tok.(string)) != http.StatusUnauthorized {
			t.Fatal("family access token survived the replay")
		}
	}
	// A refresh token expires 30 days after its last use.
	fresh := e.tokens(t, client, redirect, "/mcp", "", ScopeReadWrite)
	e.clock.advance(29 * 24 * time.Hour)
	code, next := refresh(fresh["refresh_token"].(string))
	if code != http.StatusOK {
		t.Fatalf("refresh within 30 days: %d %v", code, next)
	}
	e.clock.advance(29 * 24 * time.Hour)
	if code, _ := refresh(next["refresh_token"].(string)); code != http.StatusOK {
		t.Fatal("sliding refresh expired early")
	}
	e.clock.advance(31 * 24 * time.Hour)
	if code, _ := refresh(fresh["refresh_token"].(string)); code != http.StatusBadRequest {
		t.Fatal("refresh token outlived 30 days")
	}
}

func TestAccessTokensExpireAfterAnHour(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	tok := e.tokens(t, e.registerClient(t, "Synthetic", redirect), redirect, "/mcp", "", ScopeRead)
	if tok["expires_in"].(float64) != 3600 {
		t.Fatalf("expires_in %v", tok["expires_in"])
	}
	e.clock.advance(time.Hour)
	if code, challenge := e.mcpChallenge(t, "/mcp", tok["access_token"].(string)); code != http.StatusUnauthorized || challenge != invalidTokenChallenge {
		t.Fatalf("access token outlived an hour: %d %q", code, challenge)
	}
}

// Acceptance 5: a token is bound to one signet's resource.
func TestTokenAudienceIsOneSignet(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"), testfixture.NewServerSignet(t, "beta"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	tok := e.tokens(t, client, redirect, "/s/a/mcp", "", ScopeReadWrite)
	if e.mcpStatus(t, "/s/a/mcp", tok["access_token"].(string)) != http.StatusOK {
		t.Fatal("token refused on its own signet")
	}
	if e.mcpStatus(t, "/s/b/mcp", tok["access_token"].(string)) != http.StatusUnauthorized {
		t.Fatal("signet A's token accepted on signet B")
	}
	// The grant is per signet: B still needs the owner's consent.
	_, challenge := pkce()
	resp, err := e.browser.Get(authorizeURL(client, redirect, challenge, publicURL+"/s/b/mcp", ""))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !csrfField.Match(body) {
		t.Fatal("signet B skipped consent")
	}
	// A refresh can't move a token to another resource.
	resp, out := e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {tok["refresh_token"].(string)}, "resource": {publicURL + "/s/b/mcp"}})
	if resp.StatusCode != http.StatusBadRequest || out["error"] != "invalid_target" {
		t.Fatalf("refresh to B: %d %v", resp.StatusCode, out)
	}
	for _, resource := range []string{"", publicURL + "/s/c/mcp", "https://other.example.com/s/a/mcp", publicURL + "/mcp"} {
		loc := e.visit(t, authorizeURL(client, redirect, challenge, resource, ""), ScopeRead)
		if loc.Query().Get("error") != "invalid_target" {
			t.Errorf("resource %q: %s", resource, loc)
		}
	}
}

// Acceptance 6: a read grant never yields write tools, whatever is requested,
// and set-scope narrows live tokens.
func TestReadGrantCannotWrite(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	reader := e.registerClient(t, "Reader", redirect)
	tok := e.tokens(t, reader, redirect, "/mcp", scopeRead+" "+scopeWrite, ScopeRead)
	if tok["scope"] != scopeRead {
		t.Fatalf("scope %v", tok["scope"])
	}
	if names := toolNames(t, e, "/mcp", tok["access_token"].(string)); slices.Contains(names, "memory_remember") || !slices.Contains(names, "memory_recall") {
		t.Fatalf("read grant tools %v", names)
	}
	if _, err := e.bearer(t, "/mcp", tok["access_token"].(string)).CallTool(context.Background(), &sdk.CallToolParams{Name: "memory_remember", Arguments: rememberArgs("read-grant-request-01", "Never saved")}); err == nil || !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("read grant called memory_remember: %v", err)
	}
	// Refresh can't widen a read grant either.
	resp, out := e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {reader}, "refresh_token": {tok["refresh_token"].(string)}, "scope": {scopeWrite}})
	if resp.StatusCode != http.StatusOK || out["scope"] != scopeRead {
		t.Fatalf("refresh widened: %v", out)
	}
	writer := e.registerClient(t, "Writer", redirect)
	wtok := e.tokens(t, writer, redirect, "/mcp", "", ScopeReadWrite)
	if names := toolNames(t, e, "/mcp", wtok["access_token"].(string)); !slices.Contains(names, "memory_remember") {
		t.Fatalf("read-write tools %v", names)
	}
	if err := SetClientScope(e.config, writer, "", ScopeRead); err != nil {
		t.Fatal(err)
	}
	if names := toolNames(t, e, "/mcp", wtok["access_token"].(string)); slices.Contains(names, "memory_remember") {
		t.Fatalf("set-scope read left write tools %v", names)
	}
}

// Acceptance 7: revoking a client from the admin CLI makes its next MCP call
// fail with 401, and its refresh token dies with it.
func TestRevokeTakesEffectImmediately(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	tok := e.tokens(t, client, redirect, "/mcp", "", ScopeReadWrite)
	if e.mcpStatus(t, "/mcp", tok["access_token"].(string)) != http.StatusOK {
		t.Fatal("token refused before revoke")
	}
	if _, err := RevokeClient(e.config, client); err != nil {
		t.Fatal(err)
	}
	if code, challenge := e.mcpChallenge(t, "/mcp", tok["access_token"].(string)); code != http.StatusUnauthorized || challenge != invalidTokenChallenge {
		t.Fatalf("after revoke: %d %q", code, challenge)
	}
	resp, out := e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {tok["refresh_token"].(string)}})
	if resp.StatusCode != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Fatalf("refresh after revoke: %d %v", resp.StatusCode, out)
	}
	if _, err := RevokeClient(e.config, client); err == nil {
		t.Fatal("revoking an unknown client succeeded")
	}
}

// A revoked metadata document client is told to authorize again, and doing
// so needs the owner's passkey and a fresh consent, like a new client.
func TestRevokedClientReauthorizesThroughTheOwner(t *testing.T) {
	var docURL string
	doc := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"client_id": docURL, "client_name": "Synthetic Code", "redirect_uris": []string{"http://127.0.0.1/callback"}})
	}))
	t.Cleanup(doc.Close)
	docURL = doc.URL + "/oauth/client-metadata"
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.server.oauth.documents.client = doc.Client()
	e.enrollOwner(t)
	redirect := "http://127.0.0.1:7/callback"
	tok := e.tokens(t, docURL, redirect, "/mcp", "", ScopeReadWrite)
	if _, err := RevokeClient(e.config, docURL); err != nil {
		t.Fatal(err)
	}
	for _, access := range []string{tok["access_token"].(string), "not-a-token"} {
		if code, challenge := e.mcpChallenge(t, "/mcp", access); code != http.StatusUnauthorized || challenge != invalidTokenChallenge {
			t.Fatalf("revoked or unknown token: %d %q", code, challenge)
		}
	}
	if code, challenge := e.mcpChallenge(t, "/mcp", ""); code != http.StatusUnauthorized || challenge != `Bearer resource_metadata="`+publicURL+`/.well-known/oauth-protected-resource"` {
		t.Fatalf("no token: %d %q", code, challenge)
	}
	resp, out := e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {docURL}, "refresh_token": {tok["refresh_token"].(string)}})
	if resp.StatusCode != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Fatalf("refresh after revoke: %d %v", resp.StatusCode, out)
	}
	// Without the owner's session the client gets the sign-in page, no code.
	_, challenge := pkce()
	stranger := &http.Client{Transport: e.agent.Transport, CheckRedirect: e.browser.CheckRedirect}
	resp, err := stranger.Get(authorizeURL(docURL, redirect, challenge, publicURL+"/mcp", ""))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Location") != "" || !strings.Contains(string(body), `id="login"`) || strings.Contains(string(body), "code=") {
		t.Fatalf("without the owner: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	// The owner sees the consent page again: the revoke forgot the approval.
	resp, err = e.browser.Get(authorizeURL(docURL, redirect, challenge, publicURL+"/mcp", ""))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || csrfField.Find(body) == nil {
		t.Fatalf("owner after revoke: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	tok = e.tokens(t, docURL, redirect, "/mcp", "", ScopeRead)
	if e.mcpStatus(t, "/mcp", tok["access_token"].(string)) != http.StatusOK {
		t.Fatal("token refused after re-authorization")
	}
}

// Acceptance 9: metadata document URLs that resolve to loopback, private or
// link-local addresses are refused before any connection.
func TestMetadataDocumentSSRF(t *testing.T) {
	var hits atomic.Int32
	local := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	t.Cleanup(local.Close)
	_, port, _ := strings.Cut(strings.TrimPrefix(local.URL, "https://127.0.0.1"), ":")
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	_, challenge := pkce()
	for _, id := range []string{"https://127.0.0.1:" + port + "/client", "https://localhost:" + port + "/client", "https://[::1]:" + port + "/client",
		"https://10.0.0.1/client", "https://169.254.169.254/latest/meta-data", "https://[::ffff:127.0.0.1]:" + port + "/client", "https://0.0.0.0:" + port + "/client"} {
		resp, err := e.browser.Get(authorizeURL(id, "https://claude.ai/api/mcp/auth_callback", challenge, publicURL+"/mcp", ""))
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "unavailable") {
			t.Errorf("%s: %d %s", id, resp.StatusCode, body)
		}
	}
	if hits.Load() != 0 {
		t.Fatalf("a refused fetch reached the local server %d times", hits.Load())
	}
	for addr, public := range map[string]bool{"127.0.0.1": false, "10.1.2.3": false, "172.16.0.1": false, "192.168.1.1": false, "169.254.169.254": false,
		"100.64.0.1": false, "0.0.0.0": false, "::1": false, "fe80::1": false, "fd00::1": false, "::ffff:10.0.0.1": false, "64:ff9b::a00:1": false,
		"224.0.0.1": false, "255.255.255.255": false, "::127.0.0.1": false, "::10.0.0.1": false, "192.88.99.1": false, "8.8.8.8": true, "2606:4700::1111": true} {
		if publicAddr(netip.MustParseAddr(addr)) != public {
			t.Errorf("%s: public=%v", addr, !public)
		}
	}
}

func TestMetadataDocumentLimits(t *testing.T) {
	var body func(id string) string
	doc := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/moved" {
			http.Redirect(w, r, "/client", http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, body("https://"+r.Host+r.URL.Path))
	}))
	t.Cleanup(doc.Close)
	d := &documents{client: doc.Client(), now: time.Now, cache: map[string]document{}}
	d.client.CheckRedirect = documentClient().CheckRedirect
	valid := func(id string) string {
		return `{"client_id":"` + id + `","redirect_uris":["https://claude.ai/api/mcp/auth_callback"]}`
	}
	for name, test := range map[string]struct {
		path string
		body func(string) string
	}{
		"oversize": {"/client", func(id string) string {
			return valid(id)[:len(valid(id))-1] + `,"x":"` + strings.Repeat("a", maxDocumentBytes) + `"}`
		}},
		"wrong id":      {"/client", func(string) string { return valid("https://elsewhere.example.com/client") }},
		"no redirects":  {"/client", func(id string) string { return `{"client_id":"` + id + `"}` }},
		"http redirect": {"/client", func(id string) string { return `{"client_id":"` + id + `","redirect_uris":["http://evil.test/cb"]}` }},
		"secret client": {"/client", func(id string) string {
			return valid(id)[:len(valid(id))-1] + `,"token_endpoint_auth_method":"client_secret_basic"}`
		}},
		"redirected": {"/moved", valid},
		"not json":   {"/client", func(string) string { return "<html>" }},
	} {
		body = test.body
		if _, err := d.get(context.Background(), doc.URL+test.path); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	body = valid
	if _, err := d.get(context.Background(), doc.URL+"/client"); err != nil {
		t.Fatalf("valid document refused: %v", err)
	}
	for _, id := range []string{"http://example.com/client", "https://example.com", "https://example.com/", "https://u@example.com/c", "https://example.com/c?q=1", "https://example.com/c#f"} {
		if isDocumentID(id) {
			t.Errorf("%s treated as a metadata document", id)
		}
	}
}

// Acceptance 10: a software authenticator registers and signs in; without
// the owner session or its CSRF token the consent POST is refused.
func TestOwnerPasskeyAndConsentCSRF(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	if resp, _ := e.postJSON(t, e.browser, "/owner/login/begin", map[string]string{}); resp.StatusCode != http.StatusForbidden {
		t.Fatal("login began with no passkey enrolled")
	}
	e.enrollOwner(t)
	cookies := e.browser.Jar.Cookies(&url.URL{Scheme: "https", Host: "memory.example.com"})
	if len(cookies) != 1 || cookies[0].Name != sessionCookie {
		t.Fatalf("cookies %v", cookies)
	}
	e.login(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	_, challenge := pkce()
	resp, err := e.browser.Get(authorizeURL(client, redirect, challenge, publicURL+"/mcp", ""))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	for _, want := range []string{"Synthetic", `class="host">claude.ai<`, `value="read"`, `value="read-write"`, `value="approve"`, `value="deny"`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("consent page lacks %q", want)
		}
	}
	if strings.Contains(string(body), `class="warn"`) {
		t.Error("loopback warning on an HTTPS redirect")
	}
	if csp := resp.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Errorf("CSP %q", csp)
	}
	request, csrf := string(requestField.FindSubmatch(body)[1]), string(csrfField.FindSubmatch(body)[1])
	form := url.Values{"request": {request}, "decision": {"approve"}, "scope": {"read-write"}}
	anonymous := &http.Client{Transport: e.browser.Transport, CheckRedirect: e.browser.CheckRedirect}
	form.Set("csrf", csrf)
	if resp, _ := e.postForm(t, anonymous, "/consent", form); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("consent without the owner cookie: %d", resp.StatusCode)
	}
	form.Set("csrf", "forged-"+csrf)
	if resp, _ := e.postForm(t, e.browser, "/consent", form); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("consent with a wrong CSRF token: %d", resp.StatusCode)
	}
	form.Del("csrf")
	if resp, _ := e.postForm(t, e.browser, "/consent", form); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("consent without a CSRF token: %d", resp.StatusCode)
	}
	if clients, _ := Clients(e.config); clients[0].Status != "pending" {
		t.Fatal("a refused consent approved the client")
	}
	// A foreign Origin is refused before the form is read.
	req, _ := http.NewRequest(http.MethodPost, publicURL+"/consent", strings.NewReader(url.Values{"request": {request}, "csrf": {csrf}, "decision": {"approve"}, "scope": {"read"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.test")
	if resp, err := e.browser.Do(req); err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin consent: %v %v", resp, err)
	}
	// Deny returns access_denied to the client.
	form.Set("csrf", csrf)
	form.Set("decision", "deny")
	resp, _ = e.postForm(t, e.browser, "/consent", form)
	if loc, _ := url.Parse(resp.Header.Get("Location")); resp.StatusCode != http.StatusSeeOther || loc.Query().Get("error") != "access_denied" || loc.Host != "claude.ai" {
		t.Fatalf("deny: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	// The session lasts 12 hours.
	e.clock.advance(12 * time.Hour)
	resp, _ = e.browser.Get(authorizeURL(client, redirect, challenge, publicURL+"/mcp", ""))
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), `id="login"`) {
		t.Fatal("owner session outlived 12 hours")
	}
}

func TestOwnerEnrollmentLinkIsOneTimeAndExpires(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	link, expires, err := EnrollOwner(e.config, false)
	if err != nil || !strings.HasPrefix(link, publicURL+"/owner/enroll#") || time.Until(expires) > enrollmentTTL {
		t.Fatalf("link %q expires %s: %v", link, expires, err)
	}
	_, token, _ := strings.Cut(link, "#")
	resp, begin := e.postJSON(t, e.browser, "/owner/register/begin", map[string]string{"token": token})
	if resp.StatusCode != http.StatusOK {
		t.Fatal(begin)
	}
	p := newPasskey(t)
	if resp, _ := e.postJSON(t, e.browser, "/owner/register/finish?ceremony="+begin["ceremony"].(string), p.create(t, begin["options"].(map[string]any))); resp.StatusCode != http.StatusOK {
		t.Fatal("registration failed")
	}
	if resp, _ := e.postJSON(t, e.browser, "/owner/register/begin", map[string]string{"token": token}); resp.StatusCode != http.StatusForbidden {
		t.Fatal("enrollment link reused")
	}
	link, _, _ = EnrollOwner(e.config, false)
	_, token, _ = strings.Cut(link, "#")
	e.clock.advance(enrollmentTTL + time.Second)
	if resp, _ := e.postJSON(t, e.browser, "/owner/register/begin", map[string]string{"token": token}); resp.StatusCode != http.StatusForbidden {
		t.Fatal("expired enrollment link accepted")
	}
	if resp, _ := e.postJSON(t, e.browser, "/owner/register/begin", map[string]string{"token": "wrong"}); resp.StatusCode != http.StatusForbidden {
		t.Fatal("wrong enrollment token accepted")
	}
	raw, _ := os.ReadFile(filepath.Join(e.config.StateDir, oauthStateFile))
	if strings.Contains(string(raw), token) {
		t.Fatal("enrollment token stored in clear")
	}
}

func TestOwnerEnrollmentCanReplaceLostPasskeys(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	lost := e.passkey
	link, _, err := EnrollOwner(e.config, true)
	if err != nil {
		t.Fatal(err)
	}
	_, token, _ := strings.Cut(link, "#")
	_, begin := e.postJSON(t, e.browser, "/owner/register/begin", map[string]string{"token": token})
	e.passkey = newPasskey(t)
	if resp, _ := e.postJSON(t, e.browser, "/owner/register/finish?ceremony="+begin["ceremony"].(string), e.passkey.create(t, begin["options"].(map[string]any))); resp.StatusCode != http.StatusOK {
		t.Fatal("replacement registration failed")
	}
	e.login(t)
	_, begin = e.postJSON(t, e.browser, "/owner/login/begin", map[string]string{})
	if resp, _ := e.postJSON(t, e.browser, "/owner/login/finish?ceremony="+begin["ceremony"].(string), lost.get(t, begin["options"].(map[string]any))); resp.StatusCode != http.StatusForbidden {
		t.Fatal("a replaced passkey still signs in")
	}
}

// A code redeemed twice revokes the tokens it issued.
func TestAuthorizationCodeReplayRevokesItsTokens(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	verifier, challenge := pkce()
	loc := e.visit(t, authorizeURL(client, redirect, challenge, publicURL+"/mcp", ""), ScopeRead)
	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {client}, "code": {loc.Query().Get("code")}, "redirect_uri": {redirect}, "code_verifier": {verifier}}
	resp, tok := e.postForm(t, e.agent, "/token", form)
	if resp.StatusCode != http.StatusOK {
		t.Fatal(tok)
	}
	if resp, out := e.postForm(t, e.agent, "/token", form); resp.StatusCode != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Fatalf("replayed code: %d %v", resp.StatusCode, out)
	}
	if e.mcpStatus(t, "/mcp", tok["access_token"].(string)) != http.StatusUnauthorized {
		t.Fatal("tokens from a replayed code survived")
	}
}

// register posts one anonymous DCR registration from the test listener's
// loopback peer, as every request through a tunnel arrives, and returns the
// status and response.
func (e *oauthEnv) register(t *testing.T, header http.Header) (*http.Response, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, publicURL+"/register", strings.NewReader(`{"redirect_uris":["https://claude.ai/api/mcp/auth_callback"]}`))
	for k, v := range header {
		req.Header[k] = v
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.agent.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func pendingClients(t *testing.T, c Config) []string {
	t.Helper()
	clients, err := Clients(c)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, cl := range clients {
		if cl.Status == "pending" {
			ids = append(ids, cl.ID)
		}
	}
	return ids
}

// DCR is limited globally, never by a forwarded address: a flood from one
// loopback peer delays registration by at most registerWindow once it stops.
func TestRegistrationFloodRecovers(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	for i := 0; i < registerLimit; i++ {
		if resp, out := e.register(t, http.Header{"X-Forwarded-For": {fmt.Sprintf("192.0.2.%d", i)}}); resp.StatusCode != http.StatusCreated {
			t.Fatalf("registration %d: %d %v", i, resp.StatusCode, out)
		}
		e.clock.advance(time.Second)
	}
	for _, h := range []http.Header{{}, {"X-Forwarded-For": {"198.51.100.1"}}, {"X-Real-Ip": {"198.51.100.2"}}, {"Cf-Connecting-Ip": {"198.51.100.3"}}, {"Forwarded": {"for=198.51.100.4"}}} {
		resp, out := e.register(t, h)
		if resp.StatusCode != http.StatusTooManyRequests || out["error"] != "temporarily_unavailable" {
			t.Fatalf("over the limit with %v: %d %v", h, resp.StatusCode, out)
		}
		if got, want := resp.Header.Get("Retry-After"), strconv.Itoa(int((registerWindow - registerLimit*time.Second).Seconds())); got != want {
			t.Fatalf("Retry-After %q, want %q", got, want)
		}
	}
	// The window slides: the first registration ages out first.
	e.clock.advance(registerWindow - registerLimit*time.Second)
	if resp, _ := e.register(t, nil); resp.StatusCode != http.StatusCreated {
		t.Fatalf("after the first registration left the window: %d", resp.StatusCode)
	}
	if resp, _ := e.register(t, nil); resp.StatusCode != http.StatusTooManyRequests || resp.Header.Get("Retry-After") != "1" {
		t.Fatalf("while the rest are in the window: %d %q", resp.StatusCode, resp.Header.Get("Retry-After"))
	}
	e.clock.advance(registerWindow)
	client := e.registerClient(t, "Legitimate", "https://claude.ai/api/mcp/auth_callback")
	if ids := pendingClients(t, e.config); !slices.Contains(ids, client) || len(ids) != registerLimit+2 {
		t.Fatalf("pending after the flood: %v", ids)
	}
	if resp, out := e.postJSON(t, e.agent, "/register", "not an object"); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed registration: %d %v", resp.StatusCode, out)
	}
}

// Pending clients past maxPending are evicted oldest first, and expire after
// pendingTTL. Approved clients are never evicted and gain nothing: the owner
// still signs in for every code. An evicted client can't be approved.
func TestPendingRegistrationsAreEvicted(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	approved := e.registerClient(t, "Approved", redirect)
	tok := e.tokens(t, approved, redirect, "/mcp", "", ScopeRead)

	victim := e.registerClient(t, "Victim", redirect)
	e.clock.advance(registerWindow)
	var (
		flood         []string
		consent, csrf [][]byte
	)
	for len(flood) < 3*maxPending {
		if len(flood) == maxPending-1 {
			// The victim is now the oldest pending client of a full cap:
			// the owner opens its consent page just before the next
			// registration evicts it.
			_, challenge := pkce()
			resp, err := e.browser.Get(authorizeURL(victim, redirect, challenge, publicURL+"/mcp", ""))
			if err != nil {
				t.Fatal(err)
			}
			page, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if consent, csrf = requestField.FindSubmatch(page), csrfField.FindSubmatch(page); consent == nil || csrf == nil {
				t.Fatalf("expected the consent page: %s", page)
			}
		}
		e.clock.advance(registerWindow / registerLimit)
		resp, out := e.register(t, nil)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("registration %d: %d %v", len(flood), resp.StatusCode, out)
		}
		flood = append(flood, out["client_id"].(string))
		pending := pendingClients(t, e.config)
		if len(pending) > maxPending {
			t.Fatalf("%d pending clients", len(pending))
		}
		// A registration outlives a full window of later ones.
		if i := len(flood) - 1 - registerLimit; i >= 0 && !slices.Contains(pending, flood[i]) {
			t.Fatalf("registration %d evicted within one window", i)
		}
		if len(flood) == maxPending {
			if slices.Contains(pending, victim) {
				t.Fatal("the oldest pending client was not evicted")
			}
			// Its consent request is still live but can't approve it.
			resp, _ := e.postForm(t, e.browser, "/consent", url.Values{"request": {string(consent[1])}, "csrf": {string(csrf[1])}, "decision": {"approve"}, "scope": {string(ScopeReadWrite)}})
			if resp.StatusCode != http.StatusBadRequest || slices.Contains(pendingClients(t, e.config), victim) {
				t.Fatalf("approving an evicted client: %d", resp.StatusCode)
			}
		}
	}
	if want := flood[len(flood)-maxPending:]; !slices.Equal(pendingClients(t, e.config), want) {
		t.Fatalf("pending %v, want the newest %v", pendingClients(t, e.config), want)
	}
	resp, refreshed := e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {approved}, "refresh_token": {tok["refresh_token"].(string)}})
	if resp.StatusCode != http.StatusOK || e.mcpStatus(t, "/mcp", refreshed["access_token"].(string)) != http.StatusOK {
		t.Fatalf("the flood evicted an approved client: %d %v", resp.StatusCode, refreshed)
	}
	clients, _ := Clients(e.config)
	if clients[0].ID != approved || clients[0].Status != "approved" {
		t.Fatalf("approved client after the flood: %+v", clients[0])
	}

	if _, out := e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"authorization_code"}, "client_id": {victim}, "code": {"x"}}); out["error"] != "invalid_client" {
		t.Fatalf("evicted client at the token endpoint: %v", out)
	}

	// An approved client without an owner session gets the sign-in page, not a code.
	e.browser.Jar, _ = cookiejar.New(nil)
	_, challenge := pkce()
	resp, err := e.browser.Get(authorizeURL(approved, redirect, challenge, publicURL+"/mcp", ""))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Location") != "" {
		t.Fatalf("approved client without an owner session: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}

	e.clock.advance(pendingTTL)
	if resp, _ := e.register(t, nil); resp.StatusCode != http.StatusCreated {
		t.Fatalf("after pending registrations expired: %d", resp.StatusCode)
	}
	if ids := pendingClients(t, e.config); len(ids) != 1 {
		t.Fatalf("expired registrations kept: %d", len(ids))
	}
	if clients, _ := Clients(e.config); len(clients) != 2 || clients[0].ID != approved {
		t.Fatalf("clients after expiry: %+v", clients)
	}
}

// Token endpoint protocol errors use RFC 6749 codes, and nothing but hashes
// reaches state_dir.
func TestTokenEndpointErrors(t *testing.T) {
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	tok := e.tokens(t, client, redirect, "/mcp", "", ScopeRead)
	for name, test := range map[string]struct {
		form   url.Values
		status int
		code   string
	}{
		"unknown grant":     {url.Values{"grant_type": {"password"}, "client_id": {client}}, 400, "unsupported_grant_type"},
		"no client":         {url.Values{"grant_type": {"refresh_token"}}, 400, "invalid_request"},
		"unknown client":    {url.Values{"grant_type": {"authorization_code"}, "client_id": {"client-unknown"}, "code": {"x"}}, 401, "invalid_client"},
		"unknown refresh":   {url.Values{"grant_type": {"refresh_token"}, "client_id": {"client-unknown"}, "refresh_token": {"x"}}, 400, "invalid_grant"},
		"dead refresh":      {url.Values{"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {"not-a-token"}}, 400, "invalid_grant"},
		"access as refresh": {url.Values{"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {tok["access_token"].(string)}}, 400, "invalid_grant"},
		"missing code":      {url.Values{"grant_type": {"authorization_code"}, "client_id": {client}}, 400, "invalid_request"},
		"bad scope":         {url.Values{"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {tok["refresh_token"].(string)}, "scope": {"admin"}}, 400, "invalid_scope"},
		"repeated":          {url.Values{"grant_type": {"refresh_token", "refresh_token"}, "client_id": {client}}, 400, "invalid_request"},
	} {
		resp, out := e.postForm(t, e.agent, "/token", test.form)
		if resp.StatusCode != test.status || out["error"] != test.code {
			t.Errorf("%s: %d %v", name, resp.StatusCode, out)
		}
	}
	req, _ := http.NewRequest(http.MethodPost, publicURL+"/token", strings.NewReader(`{"grant_type":"refresh_token"}`))
	req.Header.Set("Content-Type", "application/json")
	if resp, err := e.agent.Do(req); err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("JSON token request: %v %v", resp, err)
	}
	raw, _ := os.ReadFile(filepath.Join(e.config.StateDir, oauthStateFile))
	for _, secret := range []string{tok["access_token"].(string), tok["refresh_token"].(string)} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("a token is stored in clear")
		}
	}
}

// Discovery: an unauthenticated MCP call names the protected resource
// metadata, which names this server as the only authorization server.
func TestOAuthDiscovery(t *testing.T) {
	for _, n := range []int{1, 2} {
		var signets []testfixture.ServerSignet
		for i := 0; i < n; i++ {
			signets = append(signets, testfixture.NewServerSignet(t, "alpha"))
		}
		e := startOAuth(t, signets...)
		path, prm := "/mcp", "/.well-known/oauth-protected-resource"
		if n > 1 {
			path, prm = "/s/b/mcp", "/.well-known/oauth-protected-resource/s/b/mcp"
		}
		resp, err := e.agent.Post(publicURL+path, "application/json", strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if want := `Bearer resource_metadata="` + publicURL + prm + `"`; resp.StatusCode != http.StatusUnauthorized || resp.Header.Get("WWW-Authenticate") != want {
			t.Fatalf("%d signets: %d %q", n, resp.StatusCode, resp.Header.Get("WWW-Authenticate"))
		}
		meta, err := oauthex.GetProtectedResourceMetadata(context.Background(), publicURL+prm, publicURL+path, e.agent)
		if err != nil || !slices.Equal(meta.AuthorizationServers, []string{publicURL}) || !slices.Equal(meta.ScopesSupported, []string{scopeRead, scopeWrite}) {
			t.Fatalf("resource metadata %+v: %v", meta, err)
		}
		asm, err := auth.GetAuthServerMetadata(context.Background(), publicURL, e.agent)
		if err != nil || asm.Issuer != publicURL || !asm.ClientIDMetadataDocumentSupported || !slices.Equal(asm.CodeChallengeMethodsSupported, []string{"S256"}) ||
			!slices.Equal(asm.TokenEndpointAuthMethodsSupported, []string{"none"}) || !slices.Contains(asm.ScopesSupported, scopeOffline) || asm.RegistrationEndpoint != publicURL+"/register" ||
			asm.RevocationEndpoint != publicURL+"/revoke" || !slices.Equal(asm.RevocationEndpointAuthMethodsSupported, []string{"none"}) {
			t.Fatalf("authorization server metadata %+v: %v", asm, err)
		}
	}
}

func TestOAuthConfiguration(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	c := oauthConfig(t, a)
	c.Listen = "0.0.0.0:8730"
	if err := c.Validate(); err != nil {
		t.Fatalf("oauth on a public listener: %v", err)
	}
	c.PublicURL = "http://127.0.0.1:8730"
	if err := c.Validate(); err == nil {
		t.Fatal("oauth accepted an IP public_url, which passkeys can't use")
	}
	c.Listen, c.PublicURL = listen, "http://localhost:8730"
	if err := c.Validate(); err != nil {
		t.Fatalf("oauth on localhost: %v", err)
	}
	for _, auth := range []string{AuthLoopbackTrust, AuthInsecureTest} {
		c := oauthConfig(t, a)
		c.Auth, c.Listen = auth, "0.0.0.0:8730"
		if err := c.Validate(); err == nil {
			t.Fatalf("%s on a public listener", auth)
		}
	}
	if _, _, err := EnrollOwner(testConfig(t, a), false); err == nil {
		t.Fatal("owner enrollment without oauth")
	}
}

// liveTokens counts a client's live tokens as `server clients list` shows them.
func liveTokens(t *testing.T, c Config, id string) int {
	t.Helper()
	clients, err := Clients(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, cl := range clients {
		if cl.ID == id {
			return cl.Tokens
		}
	}
	t.Fatalf("client %s is not listed", id)
	return 0
}

func (e *oauthEnv) revoke(t *testing.T, form url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := e.agent.PostForm(publicURL+"/revoke", form)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

// RFC 7009: revoking either token of a pair ends its family, whatever the
// hint says, and the family stays dead. Other families of the same client
// keep working until they are revoked too.
func TestRevocationEndsTheFamily(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	refresh := func(token string) (int, map[string]any) {
		resp, out := e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {token}})
		return resp.StatusCode, out
	}
	dead := func(name string, tok map[string]any) {
		t.Helper()
		if e.mcpStatus(t, "/mcp", tok["access_token"].(string)) != http.StatusUnauthorized {
			t.Fatalf("%s: access token survived revocation", name)
		}
		if code, out := refresh(tok["refresh_token"].(string)); code != http.StatusBadRequest || out["error"] != "invalid_grant" {
			t.Fatalf("%s: refresh after revocation: %d %v", name, code, out)
		}
	}
	first, other := e.tokens(t, client, redirect, "/mcp", "", ScopeReadWrite), e.tokens(t, client, redirect, "/mcp", "", ScopeReadWrite)
	_, rotated := refresh(first["refresh_token"].(string))
	if liveTokens(t, e.config, client) != 5 {
		t.Fatalf("live tokens before revocation: %d", liveTokens(t, e.config, client))
	}

	// The refresh token's family dies: the rotated pair and the first access token.
	resp, body := e.revoke(t, url.Values{"client_id": {client}, "token": {rotated["refresh_token"].(string)}, "token_type_hint": {"refresh_token"}})
	if resp.StatusCode != http.StatusOK || body != "" || resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("revoke refresh: %d %q", resp.StatusCode, body)
	}
	dead("rotated", rotated)
	if e.mcpStatus(t, "/mcp", first["access_token"].(string)) != http.StatusUnauthorized {
		t.Fatal("the family's first access token survived")
	}
	// Replaying the spent refresh token can't revive the family.
	if code, out := refresh(first["refresh_token"].(string)); code != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Fatalf("replay after revocation: %d %v", code, out)
	}
	if e.mcpStatus(t, "/mcp", other["access_token"].(string)) != http.StatusOK || liveTokens(t, e.config, client) != 2 {
		t.Fatal("revocation reached another family")
	}

	// An access token with the wrong hint ends its family too.
	if resp, body := e.revoke(t, url.Values{"client_id": {client}, "token": {other["access_token"].(string)}, "token_type_hint": {"refresh_token"}}); resp.StatusCode != http.StatusOK || body != "" {
		t.Fatalf("revoke access: %d %q", resp.StatusCode, body)
	}
	dead("other", other)
	if n := liveTokens(t, e.config, client); n != 0 {
		t.Fatalf("live tokens after revocation: %d", n)
	}
	// Revoking a dead token again, and an unsupported hint, still answer 200.
	if resp, body := e.revoke(t, url.Values{"client_id": {client}, "token": {other["refresh_token"].(string)}, "token_type_hint": {"id_token"}}); resp.StatusCode != http.StatusOK || body != "" {
		t.Fatalf("revoke again: %d %q", resp.StatusCode, body)
	}
	// The client itself stays approved and can connect again.
	again := e.tokens(t, client, redirect, "/mcp", "", ScopeReadWrite)
	if resp, _ := e.revoke(t, url.Values{"client_id": {client}, "token": {again["refresh_token"].(string)}, "token_type_hint": {"unknown-hint"}}); resp.StatusCode != http.StatusOK {
		t.Fatalf("unknown hint: %d", resp.StatusCode)
	}
	dead("again", again)
}

// A client can revoke only its own tokens. Another client's token gets the
// same answer as an unknown one and leaves the state file untouched; a
// request without a known client_id revokes nothing.
func TestRevocationIsPerClient(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	owner, other := e.registerClient(t, "Synthetic", redirect), e.registerClient(t, "Synthetic Other", redirect)
	tok := e.tokens(t, owner, redirect, "/mcp", "", ScopeReadWrite)
	e.tokens(t, other, redirect, "/mcp", "", ScopeRead)
	state := filepath.Join(e.config.StateDir, oauthStateFile)
	before, _ := os.ReadFile(state)

	unknown, unknownBody := e.revoke(t, url.Values{"client_id": {other}, "token": {"not-a-token"}})
	// Like /token, revocation takes requests from a client's own origin.
	req, _ := http.NewRequest(http.MethodPost, publicURL+"/revoke", strings.NewReader(url.Values{"client_id": {other}, "token": {"not-a-token"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://client.example.com")
	if resp, err := e.agent.Do(req); err != nil || resp.StatusCode != http.StatusOK || resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("cross-origin revocation: %v %v", resp, err)
	}
	for _, token := range []string{tok["access_token"].(string), tok["refresh_token"].(string)} {
		resp, body := e.revoke(t, url.Values{"client_id": {other}, "token": {token}})
		if resp.StatusCode != unknown.StatusCode || body != unknownBody || resp.Header.Get("Content-Type") != unknown.Header.Get("Content-Type") || resp.StatusCode != http.StatusOK {
			t.Fatalf("another client's token: %d %q, unknown token: %d %q", resp.StatusCode, body, unknown.StatusCode, unknownBody)
		}
	}
	if after, _ := os.ReadFile(state); string(after) != string(before) {
		t.Fatal("revoking another client's token changed the state")
	}

	access, refresh := tok["access_token"].(string), tok["refresh_token"].(string)
	for name, test := range map[string]struct {
		form   url.Values
		status int
		code   string
	}{
		"no client":      {url.Values{"token": {refresh}}, 400, "invalid_request"},
		"unknown client": {url.Values{"client_id": {"client-unknown"}, "token": {refresh}}, 401, "invalid_client"},
		"no token":       {url.Values{"client_id": {owner}}, 400, "invalid_request"},
		"repeated":       {url.Values{"client_id": {owner}, "token": {refresh, access}}, 400, "invalid_request"},
	} {
		resp, out := e.postForm(t, e.agent, "/revoke", test.form)
		if resp.StatusCode != test.status || out["error"] != test.code {
			t.Errorf("%s: %d %v", name, resp.StatusCode, out)
		}
	}
	// A bearer token in place of client_id authenticates nothing.
	req, _ = http.NewRequest(http.MethodPost, publicURL+"/revoke", strings.NewReader(url.Values{"token": {refresh}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+access)
	if resp, err := e.agent.Do(req); err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bearer without client_id: %v %v", resp, err)
	}
	req, _ = http.NewRequest(http.MethodPost, publicURL+"/revoke", strings.NewReader(`{"client_id":"`+owner+`","token":"`+refresh+`"}`))
	req.Header.Set("Content-Type", "application/json")
	if resp, err := e.agent.Do(req); err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("JSON revocation: %v %v", resp, err)
	}
	if resp, err := e.agent.Get(publicURL + "/revoke?client_id=" + owner + "&token=" + refresh); err != nil || resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET revocation: %v %v", resp, err)
	}
	if after, _ := os.ReadFile(state); string(after) != string(before) {
		t.Fatal("a refused revocation changed the state")
	}
	if e.mcpStatus(t, "/mcp", access) != http.StatusOK || liveTokens(t, e.config, owner) != 2 {
		t.Fatal("the owner's tokens were revoked by someone else")
	}
}

// A client's own expired access token still ends its family until it is
// pruned, so a late logout leaves nothing live.
func TestRevocationByExpiredToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	tok := e.tokens(t, client, redirect, "/mcp", "", ScopeReadWrite)
	e.clock.advance(time.Hour)
	if resp, body := e.revoke(t, url.Values{"client_id": {client}, "token": {tok["access_token"].(string)}}); resp.StatusCode != http.StatusOK || body != "" {
		t.Fatalf("revoke expired access token: %d %q", resp.StatusCode, body)
	}
	resp, out := e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {tok["refresh_token"].(string)}})
	if resp.StatusCode != http.StatusBadRequest || out["error"] != "invalid_grant" || liveTokens(t, e.config, client) != 0 {
		t.Fatalf("refresh after revoking the expired access token: %d %v", resp.StatusCode, out)
	}
}

// When the OAuth state can't be read, because it is corrupt or the server
// may not read it, /revoke and /token say the server failed rather than that
// the client is unknown, which would send it to authorize again. Nothing is
// revoked, and the tokens work once the state reads again.
func TestOAuthStateReadFailure(t *testing.T) {
	for name, unreadable := range map[string]func(t *testing.T, state string, good []byte) error{
		"corrupt": func(_ *testing.T, state string, _ []byte) error {
			return os.WriteFile(state, []byte(`{"clients": corrupt`), 0600)
		},
		"permission denied": func(t *testing.T, state string, good []byte) error {
			if os.Geteuid() == 0 {
				t.Skip("root reads the unreadable state file this test uses to fail the read")
			}
			if err := os.WriteFile(state, append(good, '\n'), 0600); err != nil {
				return err
			}
			return os.Chmod(state, 0)
		},
	} {
		t.Run(name, func(t *testing.T) { testOAuthStateReadFailure(t, unreadable) })
	}
}

func testOAuthStateReadFailure(t *testing.T, unreadable func(t *testing.T, state string, good []byte) error) {
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	tok := e.tokens(t, client, redirect, "/mcp", "", ScopeReadWrite)
	state := filepath.Join(e.config.StateDir, oauthStateFile)
	good, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := unreadable(t, state, good); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(state, 0600) })

	for name, test := range map[string]struct {
		path   string
		form   url.Values
		status int
		code   string
	}{
		"revoke":          {"/revoke", url.Values{"client_id": {client}, "token": {tok["refresh_token"].(string)}}, 503, "temporarily_unavailable"},
		"revoke unknown":  {"/revoke", url.Values{"client_id": {"client-unknown"}, "token": {"not-a-token"}}, 503, "temporarily_unavailable"},
		"refresh":         {"/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {tok["refresh_token"].(string)}}, 500, "server_error"},
		"code":            {"/token", url.Values{"grant_type": {"authorization_code"}, "client_id": {client}, "code": {"x"}}, 500, "server_error"},
		"refresh unknown": {"/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {"client-unknown"}, "refresh_token": {"x"}}, 500, "server_error"},
	} {
		resp, out := e.postForm(t, e.agent, test.path, test.form)
		if resp.StatusCode != test.status || out["error"] != test.code || strings.Contains(fmt.Sprint(out["error_description"]), "unknown client") {
			t.Errorf("%s: %d %v", name, resp.StatusCode, out)
		}
	}

	if err := os.Chmod(state, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state, good, 0600); err != nil {
		t.Fatal(err)
	}
	if e.mcpStatus(t, "/mcp", tok["access_token"].(string)) != http.StatusOK || liveTokens(t, e.config, client) != 2 {
		t.Fatal("a read failure revoked tokens")
	}
}

// A revocation that can't save the state answers 503 and leaves the family live.
func TestRevocationWriteFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores the read-only state directory this test uses to fail the write")
	}
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	tok := e.tokens(t, client, redirect, "/mcp", "", ScopeReadWrite)
	if err := os.Chmod(e.config.StateDir, 0500); err != nil {
		t.Fatal(err)
	}
	resp, out := e.postForm(t, e.agent, "/revoke", url.Values{"client_id": {client}, "token": {tok["refresh_token"].(string)}})
	if err := os.Chmod(e.config.StateDir, 0700); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable || out["error"] != "temporarily_unavailable" {
		t.Fatalf("revoke without a writable state: %d %v", resp.StatusCode, out)
	}
	if e.mcpStatus(t, "/mcp", tok["access_token"].(string)) != http.StatusOK || liveTokens(t, e.config, client) != 2 {
		t.Fatal("a failed revocation revoked tokens")
	}
}

// When /token can't save the state, redemption and refresh answer 500
// server_error rather than invalid_grant and issue nothing. The code and the
// refresh token stay usable once the state saves again.
func TestTokenSaveFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores the read-only state directory this test uses to fail the write")
	}
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	tok := e.tokens(t, client, redirect, "/mcp", "", ScopeReadWrite)
	grants := pendingGrants(t, e, client, redirect, tok)
	state := filepath.Join(e.config.StateDir, oauthStateFile)
	before, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(e.config.StateDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(e.config.StateDir, 0700) })
	for name, form := range grants {
		resp, out := e.postForm(t, e.agent, "/token", form)
		if resp.StatusCode != http.StatusInternalServerError || out["error"] != "server_error" || out["access_token"] != nil {
			t.Errorf("%s without a writable state: %d %v", name, resp.StatusCode, out)
		}
	}
	if err := os.Chmod(e.config.StateDir, 0700); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(state); string(after) != string(before) {
		t.Fatal("a failed save changed the state")
	}
	for name, form := range grants {
		if resp, out := e.postForm(t, e.agent, "/token", form); resp.StatusCode != http.StatusOK {
			t.Errorf("%s once the state saves: %d %v", name, resp.StatusCode, out)
		}
	}
}

// When the state can't be read by the update that redeems or refreshes, as
// when it changes after the client check, both grants fail as the store's
// failure, and the code and refresh token still work once it reads again.
func TestTokenReadFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads the unreadable state file this test uses to fail the read")
	}
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	tok := e.tokens(t, client, redirect, "/mcp", "", ScopeReadWrite)
	grants := pendingGrants(t, e, client, redirect, tok)
	state := filepath.Join(e.config.StateDir, oauthStateFile)
	good, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	// The server reads the client before it updates, so the update is called
	// directly, with the file changed and unreadable since that read.
	if err := os.WriteFile(state, append(good, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(state, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(state, 0600) })
	a := e.server.oauth
	for name, grant := range map[string]func(string, url.Values) (tokenResponse, error){"redeem": a.redeem, "refresh": a.refresh} {
		if out, err := grant(client, grants[name]); !errors.Is(err, errStore) || out.AccessToken != "" {
			t.Errorf("%s without a readable state: %v %+v", name, err, out)
		}
	}
	if err := os.Chmod(state, 0600); err != nil {
		t.Fatal(err)
	}
	for name, form := range grants {
		if resp, out := e.postForm(t, e.agent, "/token", form); resp.StatusCode != http.StatusOK {
			t.Errorf("%s once the state reads: %d %v", name, resp.StatusCode, out)
		}
	}
}

// pendingGrants returns a redemption of a new, unused authorization code and
// a refresh of tok, both for client.
func pendingGrants(t *testing.T, e *oauthEnv, client, redirect string, tok map[string]any) map[string]url.Values {
	t.Helper()
	verifier, challenge := pkce()
	loc := e.visit(t, authorizeURL(client, redirect, challenge, publicURL+"/mcp", ""), ScopeReadWrite)
	return map[string]url.Values{
		"redeem": {"grant_type": {"authorization_code"}, "client_id": {client}, "code": {loc.Query().Get("code")},
			"redirect_uri": {redirect}, "code_verifier": {verifier}},
		"refresh": {"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {tok["refresh_token"].(string)}},
	}
}

// Real grant errors keep their exact responses: an unknown, expired or reused
// code, and a rotated refresh token and the family its reuse revoked.
func TestTokenGrantErrorBodies(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := startOAuth(t, testfixture.NewServerSignet(t, "alpha"))
	e.enrollOwner(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := e.registerClient(t, "Synthetic", redirect)
	post := func(form url.Values) string {
		t.Helper()
		resp, err := e.agent.PostForm(publicURL+"/token", form)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return resp.Status + " " + string(raw)
	}
	grant := func(reason string) string {
		return `400 Bad Request {"error":"invalid_grant","error_description":"` + reason + `"}` + "\n"
	}
	code := func() url.Values {
		verifier, challenge := pkce()
		loc := e.visit(t, authorizeURL(client, redirect, challenge, publicURL+"/mcp", ""), ScopeReadWrite)
		return url.Values{"grant_type": {"authorization_code"}, "client_id": {client}, "code": {loc.Query().Get("code")},
			"redirect_uri": {redirect}, "code_verifier": {verifier}}
	}
	refresh := func(token any) url.Values {
		return url.Values{"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {token.(string)}}
	}

	unknown := code()
	unknown.Set("code", "unknown")
	if got := post(unknown); got != grant("authorization code is invalid or expired") {
		t.Errorf("unknown code: %s", got)
	}
	expired := code()
	e.clock.advance(codeTTL)
	if got := post(expired); got != grant("authorization code is invalid or expired") {
		t.Errorf("expired code: %s", got)
	}
	reused := code()
	if got := post(reused); !strings.HasPrefix(got, "200 ") {
		t.Fatalf("redeem: %s", got)
	}
	if got := post(reused); got != grant("authorization code was already used") {
		t.Errorf("reused code: %s", got)
	}
	first := e.tokens(t, client, redirect, "/mcp", "", ScopeReadWrite)
	_, second := e.postForm(t, e.agent, "/token", refresh(first["refresh_token"]))
	if got := post(refresh(first["refresh_token"])); got != grant("refresh token was already used; its tokens are revoked") {
		t.Errorf("reused refresh token: %s", got)
	}
	if got := post(refresh(second["refresh_token"])); got != grant("refresh token is invalid or expired") {
		t.Errorf("refresh token of a revoked family: %s", got)
	}
}
