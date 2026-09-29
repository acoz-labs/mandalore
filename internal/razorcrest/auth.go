package razorcrest

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Grant is the explicit permission assigned to one authenticated subject.
type Grant struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

// AuthConfig binds authentication to one trusted issuer and application audience.
// Subjects are immutable issuer subject identifiers, never caller-supplied emails.
type AuthConfig struct {
	Issuer   string           `json:"issuer"`
	Audience string           `json:"audience"`
	Subjects map[string]Grant `json:"subjects"`
}

// IdentityProvider is the remote boundary contract for validated assertions.
// Deployment adapters must authenticate every request; grants remain server-owned.
type IdentityProvider interface {
	Authenticate(*http.Request) (Principal, error)
}

type Principal struct {
	Subject     string
	Read, Write bool
}

var errUnauthorized = errors.New("unauthorized")

// Authenticator validates the Access assertion independently of the edge. It does
// not trust identity headers or use token-provided URLs to discover signing keys.
type Authenticator struct {
	config      AuthConfig
	client      *http.Client
	now         func() time.Time
	mu          sync.Mutex
	keys        map[string]*rsa.PublicKey
	expires     time.Time
	lastAttempt time.Time
}

func NewAuthenticator(config AuthConfig) (*Authenticator, error) {
	u, err := url.Parse(config.Issuer)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Path != "" || u.RawPath != "" || strings.TrimSpace(config.Issuer) != config.Issuer {
		return nil, errors.New("issuer must be an HTTPS origin without a trailing slash")
	}
	if strings.TrimSpace(config.Audience) == "" || config.Audience != strings.TrimSpace(config.Audience) || len(config.Subjects) == 0 {
		return nil, errors.New("audience and authorized subjects are required")
	}
	subjects := make(map[string]Grant, len(config.Subjects))
	for subject, grant := range config.Subjects {
		if strings.TrimSpace(subject) == "" || subject != strings.TrimSpace(subject) || !grant.Read {
			return nil, errors.New("each subject requires an identifier and permission")
		}
		subjects[subject] = grant
	}
	config.Subjects = subjects
	return &Authenticator{config: config, client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("key endpoint redirects forbidden") }}, now: time.Now}, nil
}

func (a *Authenticator) Authenticate(r *http.Request) (Principal, error) {
	tokens := r.Header.Values("Cf-Access-Jwt-Assertion")
	if len(tokens) != 1 || len(tokens[0]) > 16384 {
		return Principal{}, errUnauthorized
	}
	parts := strings.Split(tokens[0], ".")
	if len(parts) != 3 {
		return Principal{}, errUnauthorized
	}
	var header struct {
		Alg  string   `json:"alg"`
		Kid  string   `json:"kid"`
		Crit []string `json:"crit"`
	}
	if decodeJWT(parts[0], &header) != nil || header.Alg != "RS256" || header.Kid == "" || len(header.Crit) > 0 {
		return Principal{}, errUnauthorized
	}
	key, err := a.key(r.Context(), header.Kid)
	if err != nil {
		return Principal{}, errUnauthorized
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Principal{}, errUnauthorized
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) != nil {
		return Principal{}, errUnauthorized
	}
	var claims struct {
		Issuer    string          `json:"iss"`
		Subject   string          `json:"sub"`
		Audience  json.RawMessage `json:"aud"`
		Expiry    int64           `json:"exp"`
		NotBefore int64           `json:"nbf"`
	}
	if decodeJWT(parts[1], &claims) != nil {
		return Principal{}, errUnauthorized
	}
	now := a.now().Unix()
	if claims.Issuer != a.config.Issuer || claims.Subject == "" || claims.Expiry <= now || claims.NotBefore > now {
		return Principal{}, errUnauthorized
	}
	var audiences []string
	if json.Unmarshal(claims.Audience, &audiences) != nil {
		var audience string
		if json.Unmarshal(claims.Audience, &audience) != nil {
			return Principal{}, errUnauthorized
		}
		audiences = []string{audience}
	}
	found := false
	for _, audience := range audiences {
		if audience == a.config.Audience {
			found = true
		}
	}
	if !found {
		return Principal{}, errUnauthorized
	}
	grant, ok := a.config.Subjects[claims.Subject]
	if !ok {
		return Principal{}, errUnauthorized
	}
	return Principal{Subject: claims.Subject, Read: grant.Read, Write: grant.Write}, nil
}

func decodeJWT(encoded string, out any) error {
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func (a *Authenticator) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	if now.Before(a.expires) {
		if key := a.keys[kid]; key != nil {
			return key, nil
		}
	}
	// Bound unknown-key refreshes so arbitrary requests cannot flood the issuer.
	if !a.lastAttempt.IsZero() && now.Sub(a.lastAttempt) < 30*time.Second {
		return nil, errUnauthorized
	}
	a.lastAttempt = now
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.config.Issuer+"/cdn-cgi/access/certs", nil)
	if err != nil {
		return nil, err
	}
	response, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errUnauthorized
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1048577))
	if err != nil || len(body) > 1048576 {
		return nil, errUnauthorized
	}
	var document struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if json.Unmarshal(body, &document) != nil || len(document.Keys) == 0 || len(document.Keys) > 100 {
		return nil, errUnauthorized
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, jwk := range document.Keys {
		if jwk.Kty != "RSA" || (jwk.Alg != "" && jwk.Alg != "RS256") || (jwk.Use != "" && jwk.Use != "sig") {
			continue
		}
		n, ne := base64.RawURLEncoding.DecodeString(jwk.N)
		e, ee := base64.RawURLEncoding.DecodeString(jwk.E)
		if ne != nil || ee != nil || len(e) == 0 || len(e) > 4 || jwk.Kid == "" {
			return nil, errUnauthorized
		}
		exponent := int64(0)
		for _, b := range e {
			exponent = (exponent << 8) | int64(b)
		}
		modulus := new(big.Int).SetBytes(n)
		if modulus.BitLen() < 2048 || modulus.BitLen() > 8192 || exponent < 3 || exponent > 2147483647 || exponent%2 == 0 {
			return nil, errUnauthorized
		}
		if _, duplicate := keys[jwk.Kid]; duplicate {
			return nil, errUnauthorized
		}
		keys[jwk.Kid] = &rsa.PublicKey{N: modulus, E: int(exponent)}
	}
	a.keys = keys
	a.expires = now.Add(5 * time.Minute)
	if key := keys[kid]; key != nil {
		return key, nil
	}
	return nil, errUnauthorized
}
