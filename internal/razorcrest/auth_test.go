package razorcrest

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func signAssertion(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	h, _ := json.Marshal(map[string]any{"alg": "RS256", "kid": kid})
	c, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(c)
	digest := sha256.Sum256([]byte(payload))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return payload + "." + base64.RawURLEncoding.EncodeToString(signature)
}
func authRequest(token string) *http.Request {
	r := httptest.NewRequest("POST", "https://memory.example/mcp", nil)
	r.Header.Set("Cf-Access-Jwt-Assertion", token)
	return r
}
func newAuthFixture(t *testing.T) (*Authenticator, *rsa.PrivateKey, *time.Time, *string, *bool, *int) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "first"
	unavailable := false
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/cdn-cgi/access/certs" {
			t.Errorf("unexpected key path %s", r.URL.Path)
		}
		if unavailable {
			http.Error(w, "offline", 503)
			return
		}
		fmt.Fprintf(w, `{"keys":[{"kty":"RSA","kid":%q,"alg":"RS256","use":"sig","n":%q,"e":"AQAB"}]}`, kid, base64.RawURLEncoding.EncodeToString(key.N.Bytes()))
	}))
	t.Cleanup(server.Close)
	a, err := NewAuthenticator(AuthConfig{Issuer: server.URL, Audience: "test-audience", Subjects: map[string]Grant{"test-subject": {Read: true, Write: true}}})
	if err != nil {
		t.Fatal(err)
	}
	a.client = server.Client()
	now := time.Unix(1800000000, 0)
	a.now = func() time.Time { return now }
	return a, key, &now, &kid, &unavailable, &calls
}
func authClaims(a *Authenticator) map[string]any {
	return map[string]any{"iss": a.config.Issuer, "sub": "test-subject", "aud": []string{"test-audience"}, "exp": a.now().Add(time.Hour).Unix()}
}

func TestAuthenticatorClaimsAndSignature(t *testing.T) {
	a, key, _, _, _, _ := newAuthFixture(t)
	token := signAssertion(t, key, "first", authClaims(a))
	p, err := a.Authenticate(authRequest(token))
	if err != nil || p.Subject != "test-subject" || !p.Read || !p.Write {
		t.Fatalf("valid assertion: %+v %v", p, err)
	}
	for _, test := range []struct {
		name, field string
		value       any
	}{
		{"issuer", "iss", "https://untrusted.example"}, {"audience", "aud", []string{"other"}}, {"subject", "sub", "stranger"}, {"expired", "exp", a.now().Unix()}, {"missing expiry", "exp", nil}, {"future", "nbf", a.now().Add(time.Hour).Unix()},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := authClaims(a)
			c[test.field] = test.value
			if _, err := a.Authenticate(authRequest(signAssertion(t, key, "first", c))); err == nil {
				t.Fatal("accepted invalid claims")
			}
		})
	}
	attacker, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Authenticate(authRequest(signAssertion(t, attacker, "first", authClaims(a)))); err == nil {
		t.Fatal("accepted forged signature")
	}
	c := authClaims(a)
	c["aud"] = "test-audience"
	if _, err := a.Authenticate(authRequest(signAssertion(t, key, "first", c))); err != nil {
		t.Fatal("string audience rejected", err)
	}
}
func TestAuthenticatorRotationAndOutage(t *testing.T) {
	a, key, now, kid, unavailable, calls := newAuthFixture(t)
	if _, err := a.Authenticate(authRequest(signAssertion(t, key, *kid, authClaims(a)))); err != nil {
		t.Fatal(err)
	}
	*kid = "rotated"
	token := signAssertion(t, key, *kid, authClaims(a))
	if _, err := a.Authenticate(authRequest(token)); err == nil {
		t.Fatal("unknown key accepted inside refresh bound")
	}
	if *calls != 1 {
		t.Fatal("unbounded refresh")
	}
	*now = now.Add(31 * time.Second)
	if _, err := a.Authenticate(authRequest(token)); err != nil {
		t.Fatal("rotation failed", err)
	}
	*unavailable = true
	if _, err := a.Authenticate(authRequest(token)); err != nil {
		t.Fatal("valid cached key failed", err)
	}
	*now = now.Add(6 * time.Minute)
	if _, err := a.Authenticate(authRequest(token)); err == nil {
		t.Fatal("expired key cache accepted during outage")
	}
}
func TestAuthenticatorRejectsMalformedAndIdentityHeaders(t *testing.T) {
	a, key, _, _, _, _ := newAuthFixture(t)
	for _, token := range []string{"", "a.b.c", strings.Repeat("x", 16385), "eyJhbGciOiJub25lIiwia2lkIjoiZmlyc3QifQ.e30."} {
		if _, err := a.Authenticate(authRequest(token)); err == nil {
			t.Fatal("accepted malformed token")
		}
	}
	r := authRequest("")
	r.Header.Set("Cf-Access-Authenticated-User-Email", "test-subject")
	if _, err := a.Authenticate(r); err == nil {
		t.Fatal("trusted unsigned identity")
	}
	r = authRequest(signAssertion(t, key, "first", authClaims(a)))
	r.Header.Add("Cf-Access-Jwt-Assertion", r.Header.Get("Cf-Access-Jwt-Assertion"))
	if _, err := a.Authenticate(r); err == nil {
		t.Fatal("accepted multiple assertions")
	}
}
func TestAuthenticatorConfigAndGrantIsolation(t *testing.T) {
	for _, issuer := range []string{"http://issuer.example", "https://issuer.example/", "https://issuer.example/path", "https://issuer.example?x=1", "https://user@issuer.example", "https://issuer.example#fragment"} {
		if _, err := NewAuthenticator(AuthConfig{Issuer: issuer, Audience: "a", Subjects: map[string]Grant{"s": {Read: true}}}); err == nil {
			t.Fatalf("accepted issuer %s", issuer)
		}
	}
	config := AuthConfig{Issuer: "https://issuer.example", Audience: "a", Subjects: map[string]Grant{"s": {Read: true}}}
	a, err := NewAuthenticator(config)
	if err != nil {
		t.Fatal(err)
	}
	config.Subjects["s"] = Grant{Write: true}
	if a.config.Subjects["s"].Write {
		t.Fatal("caller mutated grants")
	}
	config.Subjects = map[string]Grant{"s": {}}
	if _, err := NewAuthenticator(config); err == nil {
		t.Fatal("accepted empty grant")
	}
	config.Subjects = map[string]Grant{"s": {Read: true}}
	config.Audience = ""
	if _, err := NewAuthenticator(config); err == nil {
		t.Fatal("accepted empty audience")
	}
}

func TestAuthenticatorKeyEndpointFailsClosed(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{"malformed", 200, `{"keys":`},
		{"empty", 200, `{"keys":[]}`},
		{"oversized", 200, strings.Repeat("x", 1048577)},
		{"weak key", 200, `{"keys":[{"kty":"RSA","kid":"first","n":"AQAB","e":"AQAB"}]}`},
		{"redirect", 302, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			a, key, _, _, _, _ := newAuthFixture(t)
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "https://untrusted.example/keys")
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			a.config.Issuer = server.URL
			client := server.Client()
			client.CheckRedirect = a.client.CheckRedirect
			client.CheckRedirect = func(*http.Request, []*http.Request) error { return fmt.Errorf("redirect forbidden") }
			a.client = client
			if _, err := a.Authenticate(authRequest(signAssertion(t, key, "first", authClaims(a)))); err == nil {
				t.Fatal("invalid JWKS response authorized request")
			}
		})
	}
}
