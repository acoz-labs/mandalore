package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/testfixture"
	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const publicURL = "https://memory.example.com"

// passkey is a software authenticator: one P-256 credential with "none"
// attestation, user presence and user verification.
type passkey struct {
	key    *ecdsa.PrivateKey
	id     []byte
	user   []byte
	origin string
	count  uint32
}

func newPasskey(t *testing.T) *passkey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	return &passkey{key: key, id: id, origin: publicURL}
}

var b64 = base64.RawURLEncoding

func (p *passkey) clientData(kind, challenge string) []byte {
	raw, _ := json.Marshal(map[string]any{"type": kind, "challenge": challenge, "origin": p.origin, "crossOrigin": false})
	return raw
}

func (p *passkey) authData(rpID string, flags byte, attested []byte) []byte {
	rp := sha256.Sum256([]byte(rpID))
	p.count++
	out := append(rp[:], flags)
	out = binary.BigEndian.AppendUint32(out, p.count)
	return append(out, attested...)
}

// create answers navigator.credentials.create options.
func (p *passkey) create(t *testing.T, options map[string]any) map[string]any {
	t.Helper()
	pk := options["publicKey"].(map[string]any)
	rpID := pk["rp"].(map[string]any)["id"].(string)
	user, err := b64.DecodeString(pk["user"].(map[string]any)["id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	p.user = user
	x, y := p.key.PublicKey.X.FillBytes(make([]byte, 32)), p.key.PublicKey.Y.FillBytes(make([]byte, 32))
	cose, err := webauthncbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: x, -3: y})
	if err != nil {
		t.Fatal(err)
	}
	attested := append(make([]byte, 16), byte(len(p.id)>>8), byte(len(p.id)))
	attested = append(append(attested, p.id...), cose...)
	object, err := webauthncbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": p.authData(rpID, 0x45, attested)})
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{"id": b64.EncodeToString(p.id), "rawId": b64.EncodeToString(p.id), "type": "public-key",
		"response": map[string]any{"clientDataJSON": b64.EncodeToString(p.clientData("webauthn.create", pk["challenge"].(string))), "attestationObject": b64.EncodeToString(object)}}
}

// get answers navigator.credentials.get options.
func (p *passkey) get(t *testing.T, options map[string]any) map[string]any {
	t.Helper()
	pk := options["publicKey"].(map[string]any)
	data := p.authData(pk["rpId"].(string), 0x05, nil)
	client := p.clientData("webauthn.get", pk["challenge"].(string))
	digest := sha256.Sum256(append(append([]byte(nil), data...), func() []byte { h := sha256.Sum256(client); return h[:] }()...))
	sig, err := ecdsa.SignASN1(rand.Reader, p.key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{"id": b64.EncodeToString(p.id), "rawId": b64.EncodeToString(p.id), "type": "public-key",
		"response": map[string]any{"clientDataJSON": b64.EncodeToString(client), "authenticatorData": b64.EncodeToString(data), "signature": b64.EncodeToString(sig), "userHandle": b64.EncodeToString(p.user)}}
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// oauthEnv is a synthetic OAuth server at memory.example.com, reached over a
// test TLS listener, with a browser for the owner and a timed agent client.
type oauthEnv struct {
	config  Config
	server  *Server
	tls     *httptest.Server
	clock   *clock
	browser *http.Client // the owner's browser: cookie jar, no redirects followed
	agent   *http.Client // an MCP client: no cookies
	passkey *passkey

	mu      sync.Mutex
	timings map[string][]time.Duration
}

func oauthConfig(t *testing.T, signets ...testfixture.ServerSignet) Config {
	t.Helper()
	c := Config{Listen: listen, PublicURL: publicURL, StateDir: t.TempDir(), Owner: Owner{Name: "Synthetic Owner"}}
	for i, s := range signets {
		c.Signets = append(c.Signets, SignetConfig{Slug: string(rune('a' + i)), Root: s.Root, SignetID: s.SignetID, DeviceID: s.DeviceID, Remote: s.Remote})
	}
	return c
}

func startOAuth(t *testing.T, signets ...testfixture.ServerSignet) *oauthEnv {
	t.Helper()
	e := &oauthEnv{config: oauthConfig(t, signets...), clock: &clock{t: time.Now()}, timings: map[string][]time.Duration{}}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s, err := newServer(e.config, io.Discard, fastTiming)
	if err != nil {
		t.Fatal(err)
	}
	s.oauth.now, s.oauth.store.now, s.oauth.documents.now = e.clock.now, e.clock.now, e.clock.now
	s.Start(ctx)
	e.server = s
	e.tls = httptest.NewTLSServer(s.Handler())
	t.Cleanup(e.tls.Close)
	jar, _ := cookiejar.New(nil)
	e.browser = &http.Client{Jar: jar, Transport: e.transport(t), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	e.agent = &http.Client{Transport: e.transport(t)}
	return e
}

// transport sends memory.example.com to the test listener, trusting its
// certificate, and records how long each server path takes.
func (e *oauthEnv) transport(t *testing.T) http.RoundTripper {
	base := e.tls.Client().Transport.(*http.Transport).Clone()
	addr := e.tls.Listener.Addr().String()
	dialer := &net.Dialer{}
	base.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address == "memory.example.com:443" {
			address = addr
		}
		return dialer.DialContext(ctx, network, address)
	}
	t.Cleanup(base.CloseIdleConnections)
	return roundTripper(func(r *http.Request) (*http.Response, error) {
		start := time.Now()
		resp, err := base.RoundTrip(r)
		if r.URL.Host == "memory.example.com" {
			e.mu.Lock()
			e.timings[r.URL.Path] = append(e.timings[r.URL.Path], time.Since(start))
			e.mu.Unlock()
		}
		return resp, err
	})
}

func (e *oauthEnv) postJSON(t *testing.T, c *http.Client, path string, body any) (*http.Response, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, publicURL+path, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func (e *oauthEnv) postForm(t *testing.T, c *http.Client, path string, form url.Values) (*http.Response, map[string]any) {
	t.Helper()
	resp, err := c.PostForm(publicURL+path, form)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

// enrollOwner registers the owner passkey through a one-time link, which
// also signs the browser in.
func (e *oauthEnv) enrollOwner(t *testing.T) {
	t.Helper()
	link, _, err := EnrollOwner(e.config, false)
	if err != nil {
		t.Fatal(err)
	}
	_, token, _ := strings.Cut(link, "#")
	e.passkey = newPasskey(t)
	resp, begin := e.postJSON(t, e.browser, "/owner/register/begin", map[string]string{"token": token})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register begin: %d %v", resp.StatusCode, begin)
	}
	resp, out := e.postJSON(t, e.browser, "/owner/register/finish?ceremony="+begin["ceremony"].(string), e.passkey.create(t, begin["options"].(map[string]any)))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register finish: %d %v", resp.StatusCode, out)
	}
}

// login signs a fresh browser in with the enrolled passkey.
func (e *oauthEnv) login(t *testing.T) {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	e.browser.Jar = jar
	resp, begin := e.postJSON(t, e.browser, "/owner/login/begin", map[string]string{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login begin: %d %v", resp.StatusCode, begin)
	}
	resp, out := e.postJSON(t, e.browser, "/owner/login/finish?ceremony="+begin["ceremony"].(string), e.passkey.get(t, begin["options"].(map[string]any)))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login finish: %d %v", resp.StatusCode, out)
	}
}

var (
	requestField = regexp.MustCompile(`name="request" value="([^"]+)"`)
	csrfField    = regexp.MustCompile(`name="csrf" value="([^"]+)"`)
)

// visit opens an authorization URL in the owner's browser. If the consent
// page appears it approves with the given scope. It returns the redirect.
func (e *oauthEnv) visit(t *testing.T, authURL string, scope Scope) *url.URL {
	t.Helper()
	resp, err := e.browser.Get(authURL)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		req, csrf := requestField.FindSubmatch(body), csrfField.FindSubmatch(body)
		if req == nil || csrf == nil {
			t.Fatalf("expected the consent page, got: %s", body)
		}
		resp, _ = e.postForm(t, e.browser, "/consent", url.Values{"request": {string(req[1])}, "csrf": {string(csrf[1])}, "decision": {"approve"}, "scope": {string(scope)}})
	}
	if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusFound {
		t.Fatalf("authorization did not redirect: %d %s", resp.StatusCode, body)
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

// fetcher approves authorizations in the owner's browser for the go-sdk
// authorization code handler.
func (e *oauthEnv) fetcher(t *testing.T, scope Scope) auth.AuthorizationCodeFetcher {
	return func(_ context.Context, args *auth.AuthorizationArgs) (*auth.AuthorizationResult, error) {
		loc := e.visit(t, args.URL, scope)
		if loc.Query().Get("error") != "" {
			return nil, fmt.Errorf("authorization refused: %s", loc.Query().Get("error"))
		}
		return &auth.AuthorizationResult{Code: loc.Query().Get("code"), State: loc.Query().Get("state")}, nil
	}
}

func (e *oauthEnv) connect(t *testing.T, path string, h auth.OAuthHandler) *sdk.ClientSession {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	session, err := sdk.NewClient(&sdk.Implementation{Name: "synthetic-client", Version: "1"}, nil).Connect(ctx,
		&sdk.StreamableClientTransport{Endpoint: publicURL + path, HTTPClient: e.agent, OAuthHandler: h, MaxRetries: -1, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// pkce returns a verifier and its S256 challenge.
func pkce() (string, string) {
	v := secret()
	sum := sha256.Sum256([]byte(v))
	return v, b64.EncodeToString(sum[:])
}

// registerClient registers a DCR client and returns its ID.
func (e *oauthEnv) registerClient(t *testing.T, name string, redirects ...string) string {
	t.Helper()
	resp, out := e.postJSON(t, e.agent, "/register", map[string]any{"client_name": name, "redirect_uris": redirects})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: %d %v", resp.StatusCode, out)
	}
	return out["client_id"].(string)
}

func authorizeURL(client, redirect, challenge, resource, scope string) string {
	v := url.Values{"response_type": {"code"}, "client_id": {client}, "redirect_uri": {redirect}, "code_challenge": {challenge},
		"code_challenge_method": {"S256"}, "resource": {resource}, "state": {"synthetic-state"}}
	if scope != "" {
		v.Set("scope", scope)
	}
	return publicURL + "/authorize?" + v.Encode()
}

// tokens runs a whole authorization for a registered client and returns the
// token response.
func (e *oauthEnv) tokens(t *testing.T, client, redirect, path, scope string, grant Scope) map[string]any {
	t.Helper()
	verifier, challenge := pkce()
	loc := e.visit(t, authorizeURL(client, redirect, challenge, publicURL+path, scope), grant)
	resp, out := e.postForm(t, e.agent, "/token", url.Values{"grant_type": {"authorization_code"}, "client_id": {client}, "code": {loc.Query().Get("code")},
		"redirect_uri": {redirect}, "code_verifier": {verifier}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token: %d %v", resp.StatusCode, out)
	}
	return out
}

// mcpStatus sends one MCP ping with a bearer token and returns the status.
func (e *oauthEnv) mcpStatus(t *testing.T, path, token string) int {
	t.Helper()
	status, _ := e.mcpChallenge(t, path, token)
	return status
}

// mcpChallenge sends one MCP ping and returns the status and WWW-Authenticate.
func (e *oauthEnv) mcpChallenge(t *testing.T, path, token string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, publicURL+path, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := e.agent.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode, resp.Header.Get("WWW-Authenticate")
}

// bearer returns an MCP session that sends a fixed access token.
func (e *oauthEnv) bearer(t *testing.T, path, token string) *sdk.ClientSession {
	t.Helper()
	base := e.agent.Transport
	client := &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		r = r.Clone(r.Context())
		r.Header.Set("Authorization", "Bearer "+token)
		return base.RoundTrip(r)
	})}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	session, err := sdk.NewClient(&sdk.Implementation{Name: "synthetic-client", Version: "1"}, nil).Connect(ctx,
		&sdk.StreamableClientTransport{Endpoint: publicURL + path, HTTPClient: client, MaxRetries: -1, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}
