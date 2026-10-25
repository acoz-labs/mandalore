package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

const (
	scopeRead      = "memory:read"
	scopeWrite     = "memory:write"
	scopeOffline   = "offline_access"
	codeTTL        = 2 * time.Minute
	requestTTL     = 10 * time.Minute
	maxFormBytes   = 16384
	registerLimit  = 10 // registrations per peer address per registerWindow
	registerWindow = 10 * time.Minute
	maxInFlight    = 64 // codes, consent requests, ceremonies or sessions held in memory
)

// authServer is the built-in OAuth 2.1 authorization server. Clients,
// grants, token hashes and passkeys live in state_dir; authorization codes,
// consent requests, passkey ceremonies and owner sessions live in memory.
type authServer struct {
	issuer    string
	owner     string
	store     *oauthStore
	documents *documents
	webauthn  *webauthn.WebAuthn
	signets   map[string]*signet // by slug
	resources map[string]string  // resource URL → slug
	log       *log.Logger
	now       func() time.Time
	endpoints map[string]http.Handler

	mu         sync.Mutex
	codes      map[string]*authCode
	requests   map[string]*authRequest
	sessions   map[string]*ownerSession
	ceremonies map[string]*ceremony
	registered map[string][]time.Time // peer address → recent registrations
}

type authCode struct {
	client, redirect, challenge, signet string
	scope                               Scope
	expires                             time.Time
	used                                bool
}

// authRequest is a validated /authorize request waiting for the owner.
type authRequest struct {
	client            *oauthClient
	redirect, state   string
	challenge, signet string
	requested         Scope
	expires           time.Time
}

func newAuthServer(c Config, signets map[string]*signet, logger *log.Logger, now func() time.Time) (*authServer, error) {
	store, err := openOAuthStore(c.StateDir, now)
	if err != nil {
		return nil, err
	}
	public, _ := url.Parse(c.PublicURL)
	w, err := webauthn.New(&webauthn.Config{RPID: public.Hostname(), RPDisplayName: "Mandalore", RPOrigins: []string{c.origin()}})
	if err != nil {
		return nil, errors.New("public_url cannot serve passkeys")
	}
	a := &authServer{issuer: c.origin(), owner: c.Owner.Name, store: store, documents: &documents{client: documentClient(), now: now, cache: map[string]document{}},
		webauthn: w, signets: signets, resources: map[string]string{}, log: logger, now: now,
		codes: map[string]*authCode{}, requests: map[string]*authRequest{}, sessions: map[string]*ownerSession{}, ceremonies: map[string]*ceremony{}, registered: map[string][]time.Time{}}
	a.endpoints = map[string]http.Handler{
		"/.well-known/oauth-authorization-server": http.HandlerFunc(a.metadata),
		"/register":              http.HandlerFunc(a.register),
		"/authorize":             http.HandlerFunc(a.authorize),
		"/consent":               http.HandlerFunc(a.consent),
		"/token":                 http.HandlerFunc(a.token),
		"/owner/enroll":          http.HandlerFunc(a.enrollPage),
		"/owner/register/begin":  http.HandlerFunc(a.registerBegin),
		"/owner/register/finish": http.HandlerFunc(a.registerFinish),
		"/owner/login/begin":     http.HandlerFunc(a.loginBegin),
		"/owner/login/finish":    http.HandlerFunc(a.loginFinish),
	}
	for _, sc := range c.Signets {
		resource := a.issuer + c.mcpPath(sc.Slug)
		a.resources[resource] = sc.Slug
		prm := auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{Resource: resource, AuthorizationServers: []string{a.issuer},
			ScopesSupported: []string{scopeRead, scopeWrite}, BearerMethodsSupported: []string{"header"}, ResourceName: "Mandalore " + sc.Slug})
		// RFC 9728 inserts the well-known path before the resource path. One
		// signet also answers at the bare well-known path it advertises.
		a.endpoints["/.well-known/oauth-protected-resource"+c.mcpPath(sc.Slug)] = prm
		if len(c.Signets) == 1 {
			a.endpoints["/.well-known/oauth-protected-resource"] = prm
		}
		g := signets[sc.Slug]
		g.protected = auth.RequireBearerToken(a.verifier(sc.Slug), &auth.RequireBearerTokenOptions{ResourceMetadataURL: a.issuer + resourceMetadataPath(c, sc.Slug)})(g.authorized())
	}
	return a, nil
}

// resourceMetadataPath is the protected resource metadata a signet's 401
// names: the bare well-known path for one signet, path-inserted among several.
func resourceMetadataPath(c Config, slug string) string {
	if len(c.Signets) == 1 {
		return "/.well-known/oauth-protected-resource"
	}
	return "/.well-known/oauth-protected-resource" + c.mcpPath(slug)
}

func (a *authServer) metadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                a.issuer,
		"authorization_endpoint":                a.issuer + "/authorize",
		"token_endpoint":                        a.issuer + "/token",
		"registration_endpoint":                 a.issuer + "/register",
		"scopes_supported":                      []string{scopeRead, scopeWrite, scopeOffline},
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"client_id_metadata_document_supported": true,
	})
}

// oauthError writes an RFC 6749 error response.
func oauthError(w http.ResponseWriter, status int, code, description string) {
	writeJSON(w, status, map[string]string{"error": code, "error_description": description})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// register is RFC 7591 dynamic client registration. The client stays pending,
// with no access, until the owner approves it.
func (a *authServer) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/json" {
		oauthError(w, http.StatusBadRequest, "invalid_client_metadata", "registration requires application/json")
		return
	}
	if !a.allowRegistration(r) {
		oauthError(w, http.StatusTooManyRequests, "temporarily_unavailable", "too many registrations from this address; retry later")
		return
	}
	var meta clientMetadata
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxFormBytes)).Decode(&meta); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_client_metadata", "registration is not valid JSON")
		return
	}
	if err := meta.check(); err != nil {
		code := "invalid_client_metadata"
		if strings.HasPrefix(err.Error(), "redirect_uris") {
			code = "invalid_redirect_uri"
		}
		oauthError(w, http.StatusBadRequest, code, err.Error())
		return
	}
	now := a.now()
	client := &oauthClient{ID: "client-" + secret()[:26], RedirectURIs: meta.RedirectURIs, Registered: now}
	client.Kind = clientKind(client.ID, false, meta.RedirectURIs)
	client.Name = displayName(meta.ClientName, client.Kind)
	err := a.store.update(func(st *oauthState) error {
		if st.pending(now) >= maxPending {
			return errTooManyPending
		}
		st.Clients[client.ID] = client
		return nil
	})
	if errors.Is(err, errTooManyPending) {
		oauthError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "too many registrations are waiting for approval")
		return
	}
	if err != nil {
		oauthError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id": client.ID, "client_id_issued_at": now.Unix(), "client_name": client.Name, "redirect_uris": client.RedirectURIs,
		"token_endpoint_auth_method": "none", "grant_types": []string{"authorization_code", "refresh_token"}, "response_types": []string{"code"},
	})
}

var errTooManyPending = errors.New("too many pending registrations")

// allowRegistration limits registrations per peer address.
func (a *authServer) allowRegistration(r *http.Request) bool {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peer = r.RemoteAddr
	}
	now := a.now()
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.registered) > 1024 {
		clear(a.registered)
	}
	recent := a.registered[peer][:0]
	for _, t := range a.registered[peer] {
		if now.Sub(t) < registerWindow {
			recent = append(recent, t)
		}
	}
	if len(recent) >= registerLimit {
		a.registered[peer] = recent
		return false
	}
	a.registered[peer] = append(recent, now)
	return true
}

// requestedScope parses an OAuth scope parameter. It returns "" when the
// client asked for no memory scope, meaning whatever the grant allows.
func requestedScope(raw string) (Scope, error) {
	var read, write bool
	for _, s := range strings.Fields(raw) {
		switch s {
		case scopeRead:
			read = true
		case scopeWrite:
			write = true
		case scopeOffline:
		default:
			return "", fmt.Errorf("unknown scope %q", s)
		}
	}
	switch {
	case write:
		return ScopeReadWrite, nil
	case read:
		return ScopeRead, nil
	}
	return "", nil
}

// narrow gives the requested scope, never more than the grant.
func narrow(requested, grant Scope) Scope {
	if requested == ScopeRead || grant == ScopeRead {
		return ScopeRead
	}
	return ScopeReadWrite
}

func scopeNames(s Scope) []string {
	if s == ScopeReadWrite {
		return []string{scopeRead, scopeWrite}
	}
	return []string{scopeRead}
}

// resolveClient finds a registered client or fetches a metadata document.
// A document client is stored only once the owner approves it.
func (a *authServer) resolveClient(r *http.Request, id string) (*oauthClient, error) {
	var stored *oauthClient
	_ = a.store.view(func(st *oauthState) error {
		if c := st.Clients[id]; c != nil {
			cp := *c
			stored = &cp
		}
		return nil
	})
	if !isDocumentID(id) {
		if stored == nil {
			return nil, errors.New("unknown client_id")
		}
		return stored, nil
	}
	meta, err := a.documents.fetch(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		stored = &oauthClient{ID: id, Document: true, Registered: a.now()}
	}
	stored.RedirectURIs = meta.RedirectURIs
	stored.Kind = clientKind(id, true, meta.RedirectURIs)
	stored.Name = displayName(meta.ClientName, stored.Kind)
	return stored, nil
}

// authorize validates an authorization request. An approved client gets a
// code at once; otherwise the owner signs in and sees the consent page.
func (a *authServer) authorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	for k, v := range q {
		if len(v) > 1 {
			a.page(w, http.StatusBadRequest, errorPage{Message: "The authorization request repeats " + k + "."})
			return
		}
	}
	client, err := a.resolveClient(r, q.Get("client_id"))
	if err != nil {
		a.page(w, http.StatusBadRequest, errorPage{Message: "This application isn't recognised: " + err.Error() + "."})
		return
	}
	redirect := q.Get("redirect_uri")
	if redirect == "" || !client.allows(redirect) {
		a.page(w, http.StatusBadRequest, errorPage{Message: "The redirect address isn't registered for this application."})
		return
	}
	// From here on, errors go back to the client's redirect.
	fail := func(code, description string) {
		v := url.Values{"error": {code}, "error_description": {description}}
		if s := q.Get("state"); s != "" {
			v.Set("state", s)
		}
		http.Redirect(w, r, withQuery(redirect, v), http.StatusFound)
	}
	if q.Get("response_type") != "code" {
		fail("unsupported_response_type", "response_type must be code")
		return
	}
	challenge := q.Get("code_challenge")
	if q.Get("code_challenge_method") != "S256" || !pkceValue(challenge) {
		fail("invalid_request", "PKCE with code_challenge_method S256 is required")
		return
	}
	slug, ok := a.resources[q.Get("resource")]
	if !ok {
		fail("invalid_target", "resource must be one of this server's signet URLs")
		return
	}
	requested, err := requestedScope(q.Get("scope"))
	if err != nil {
		fail("invalid_scope", err.Error())
		return
	}
	req := &authRequest{client: client, redirect: redirect, state: q.Get("state"), challenge: challenge, signet: slug, requested: requested, expires: a.now().Add(requestTTL)}
	if grant, ok := client.Grants[slug]; ok {
		a.grant(w, r, req, narrow(requested, grant))
		return
	}
	session := a.session(r)
	if session == nil {
		a.loginPage(w, client)
		return
	}
	id := secret()
	a.mu.Lock()
	a.prune()
	makeRoom(a.requests, func(r *authRequest) time.Time { return r.expires })
	a.requests[id] = req
	a.mu.Unlock()
	a.consentPage(w, req, id, session.csrf)
}

// consent records the owner's decision on a pending authorization request.
// It needs the owner session cookie and the session's CSRF token.
func (a *authServer) consent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if r.ParseForm() != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	session := a.session(r)
	if session == nil || subtle.ConstantTimeCompare([]byte(r.PostForm.Get("csrf")), []byte(session.csrf)) != 1 {
		a.page(w, http.StatusForbidden, errorPage{Message: "Your owner session expired or this form is stale. Start the connection again."})
		return
	}
	a.mu.Lock()
	req := a.requests[r.PostForm.Get("request")]
	delete(a.requests, r.PostForm.Get("request"))
	a.mu.Unlock()
	if req == nil || !a.now().Before(req.expires) {
		a.page(w, http.StatusBadRequest, errorPage{Message: "This approval request expired. Start the connection again."})
		return
	}
	if r.PostForm.Get("decision") != "approve" {
		v := url.Values{"error": {"access_denied"}, "error_description": {"the owner denied access"}}
		if req.state != "" {
			v.Set("state", req.state)
		}
		http.Redirect(w, r, withQuery(req.redirect, v), http.StatusSeeOther)
		return
	}
	scope := Scope(r.PostForm.Get("scope"))
	if !scope.valid() {
		a.page(w, http.StatusBadRequest, errorPage{Message: "Choose read or read-write access."})
		return
	}
	err := a.store.update(func(st *oauthState) error {
		c := st.Clients[req.client.ID]
		if c == nil {
			if !req.client.Document {
				return errors.New("the application's registration expired")
			}
			cp := *req.client
			c = &cp
			st.Clients[c.ID] = c
		}
		if req.client.Document {
			c.Name, c.Kind, c.RedirectURIs = req.client.Name, req.client.Kind, req.client.RedirectURIs
		}
		if c.Grants == nil {
			c.Grants = map[string]Scope{}
		}
		c.Grants[req.signet] = scope
		return nil
	})
	if err != nil {
		a.page(w, http.StatusBadRequest, errorPage{Message: "Approval failed: " + err.Error() + "."})
		return
	}
	client := Client{ID: req.client.ID, Name: req.client.Name, Kind: req.client.Kind, Scope: scope}
	if err := a.signets[req.signet].enroll(client); err != nil {
		a.log.Printf("signet %s: cannot enroll an approved client's device yet; it enrolls on first write", req.signet)
	}
	a.grant(w, r, req, narrow(req.requested, scope))
}

// grant issues a single-use authorization code and redirects to the client.
func (a *authServer) grant(w http.ResponseWriter, r *http.Request, req *authRequest, scope Scope) {
	code := secret()
	a.mu.Lock()
	a.prune()
	makeRoom(a.codes, func(c *authCode) time.Time { return c.expires })
	a.codes[hashSecret(code)] = &authCode{client: req.client.ID, redirect: req.redirect, challenge: req.challenge, signet: req.signet, scope: scope, expires: a.now().Add(codeTTL)}
	a.mu.Unlock()
	v := url.Values{"code": {code}}
	if req.state != "" {
		v.Set("state", req.state)
	}
	http.Redirect(w, r, withQuery(req.redirect, v), http.StatusSeeOther)
}

func withQuery(raw string, v url.Values) string {
	u, _ := url.Parse(raw)
	q := u.Query()
	for k, vs := range v {
		q[k] = vs
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// pkceValue checks an RFC 7636 code verifier or S256 challenge: 43–128
// unreserved characters.
func pkceValue(v string) bool {
	if len(v) < 43 || len(v) > 128 {
		return false
	}
	for _, c := range v {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("-._~", c)) {
			return false
		}
	}
	return true
}

// prune drops expired in-memory state. The caller holds mu.
func (a *authServer) prune() {
	now := a.now()
	maps.DeleteFunc(a.codes, func(_ string, c *authCode) bool { return !now.Before(c.expires) })
	maps.DeleteFunc(a.requests, func(_ string, r *authRequest) bool { return !now.Before(r.expires) })
	maps.DeleteFunc(a.sessions, func(_ string, s *ownerSession) bool { return !now.Before(s.expires) })
	maps.DeleteFunc(a.ceremonies, func(_ string, c *ceremony) bool { return !now.Before(c.expires) })
}

// makeRoom bounds an in-memory map before an insert by dropping the entry
// that expires first, so a flood evicts its own entries before a fresh one.
func makeRoom[V any](m map[string]V, expires func(V) time.Time) {
	for len(m) >= maxInFlight {
		var first string
		for k, v := range m {
			if first == "" || expires(v).Before(expires(m[first])) {
				first = k
			}
		}
		delete(m, first)
	}
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// token is the token endpoint for public clients. It reads and writes local
// state only; it never fetches a metadata document.
func (a *authServer) token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/x-www-form-urlencoded" {
		oauthError(w, http.StatusBadRequest, "invalid_request", "the token request must be application/x-www-form-urlencoded")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if r.ParseForm() != nil {
		oauthError(w, http.StatusBadRequest, "invalid_request", "malformed token request")
		return
	}
	f := r.PostForm
	for k, v := range f {
		if len(v) > 1 {
			oauthError(w, http.StatusBadRequest, "invalid_request", "the token request repeats "+k)
			return
		}
	}
	clientID := f.Get("client_id")
	if clientID == "" {
		oauthError(w, http.StatusBadRequest, "invalid_request", "client_id is required")
		return
	}
	var known bool
	_ = a.store.view(func(st *oauthState) error { known = st.Clients[clientID] != nil; return nil })
	if !known {
		oauthError(w, http.StatusUnauthorized, "invalid_client", "unknown client")
		return
	}
	var (
		out  tokenResponse
		code = "invalid_grant"
		err  error
	)
	switch f.Get("grant_type") {
	case "authorization_code":
		out, err = a.redeem(clientID, f)
	case "refresh_token":
		out, err = a.refresh(clientID, f)
	default:
		code, err = "unsupported_grant_type", errors.New("grant_type must be authorization_code or refresh_token")
	}
	var oe *tokenError
	if errors.As(err, &oe) {
		code = oe.code
	}
	if err != nil {
		oauthError(w, http.StatusBadRequest, code, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type tokenError struct{ code, description string }

func (e *tokenError) Error() string { return e.description }

func (a *authServer) redeem(clientID string, f url.Values) (tokenResponse, error) {
	verifier := f.Get("code_verifier")
	if f.Get("code") == "" || f.Get("redirect_uri") == "" || verifier == "" {
		return tokenResponse{}, &tokenError{"invalid_request", "code, redirect_uri and code_verifier are required"}
	}
	key := hashSecret(f.Get("code"))
	a.mu.Lock()
	c := a.codes[key]
	replay := c != nil && c.used
	if c != nil {
		c.used = true
	}
	a.mu.Unlock()
	if c == nil || !a.now().Before(c.expires) {
		return tokenResponse{}, errors.New("authorization code is invalid or expired")
	}
	if replay {
		// A code used twice may be stolen: revoke what it issued.
		_ = a.store.update(func(st *oauthState) error { st.revokeFamily(key); return nil })
		return tokenResponse{}, errors.New("authorization code was already used")
	}
	sum := sha256.Sum256([]byte(verifier))
	if c.client != clientID || c.redirect != f.Get("redirect_uri") || !pkceValue(verifier) ||
		subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(sum[:])), []byte(c.challenge)) != 1 {
		return tokenResponse{}, errors.New("authorization code, client, redirect_uri or code_verifier do not match")
	}
	if res := f.Get("resource"); res != "" && a.resources[res] != c.signet {
		return tokenResponse{}, &tokenError{"invalid_target", "resource does not match the authorization"}
	}
	var out tokenResponse
	err := a.store.update(func(st *oauthState) error {
		client := st.Clients[clientID]
		if client == nil || client.Grants[c.signet] == "" {
			return errors.New("the grant was revoked")
		}
		grant := client.Grants[c.signet]
		out = a.issue(st, clientID, c.signet, key, narrow(c.scope, grant))
		return nil
	})
	return out, err
}

func (a *authServer) refresh(clientID string, f url.Values) (tokenResponse, error) {
	presented := f.Get("refresh_token")
	if presented == "" {
		return tokenResponse{}, &tokenError{"invalid_request", "refresh_token is required"}
	}
	requested, err := requestedScope(f.Get("scope"))
	if err != nil {
		return tokenResponse{}, &tokenError{"invalid_scope", err.Error()}
	}
	var (
		out    tokenResponse
		failed error
	)
	err = a.store.update(func(st *oauthState) error {
		key := hashSecret(presented)
		t := st.Tokens[key]
		switch {
		case t == nil || t.Kind != tokenRefresh || !a.now().Before(t.Expires) || t.Client != clientID:
			failed = errors.New("refresh token is invalid or expired")
			return nil
		case t.Used:
			// Reuse of a rotated token: assume theft and end the family.
			st.revokeFamily(t.Family)
			failed = errors.New("refresh token was already used; its tokens are revoked")
			return nil
		}
		if res := f.Get("resource"); res != "" && a.resources[res] != t.Signet {
			failed = &tokenError{"invalid_target", "resource does not match the refresh token"}
			return nil
		}
		client := st.Clients[clientID]
		if client == nil || client.Grants[t.Signet] == "" {
			failed = errors.New("the grant was revoked")
			return nil
		}
		grant := client.Grants[t.Signet]
		t.Used = true
		scope := narrow(t.Scope, grant)
		if requested != "" {
			scope = narrow(requested, scope)
		}
		out = a.issue(st, clientID, t.Signet, t.Family, scope)
		return nil
	})
	if err != nil {
		return out, err
	}
	return out, failed
}

// issue stores a new access and refresh token pair and returns it.
func (a *authServer) issue(st *oauthState, client, signet, family string, scope Scope) tokenResponse {
	access, refresh := secret(), secret()
	now := a.now()
	st.Tokens[hashSecret(access)] = &oauthToken{Kind: tokenAccess, Client: client, Signet: signet, Family: family, Scope: scope, Expires: now.Add(accessTTL)}
	st.Tokens[hashSecret(refresh)] = &oauthToken{Kind: tokenRefresh, Client: client, Signet: signet, Family: family, Scope: scope, Expires: now.Add(refreshTTL)}
	return tokenResponse{AccessToken: access, TokenType: "Bearer", ExpiresIn: int(accessTTL.Seconds()), RefreshToken: refresh, Scope: strings.Join(scopeNames(scope), " ")}
}

// verifier checks an access token for one signet: its audience, expiry and
// the client's current grant, which can only narrow the token's scope.
func (a *authServer) verifier(slug string) auth.TokenVerifier {
	return func(_ context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		var info *auth.TokenInfo
		err := a.store.view(func(st *oauthState) error {
			t := st.Tokens[hashSecret(token)]
			if t == nil || t.Kind != tokenAccess || t.Signet != slug || !a.now().Before(t.Expires) {
				return auth.ErrInvalidToken
			}
			c := st.Clients[t.Client]
			if c == nil {
				return auth.ErrInvalidToken
			}
			grant, ok := c.Grants[slug]
			if !ok {
				return auth.ErrInvalidToken
			}
			scope := narrow(t.Scope, grant)
			info = &auth.TokenInfo{Scopes: scopeNames(scope), Expiration: t.Expires, UserID: c.ID,
				Extra: map[string]any{"client": Client{ID: c.ID, Name: c.Name, Kind: c.Kind, Scope: scope}}}
			return nil
		})
		return info, err
	}
}
