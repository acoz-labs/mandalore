package server

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// The insecure-test authenticator exists only in test builds.
func init() {
	newTestAuthenticator = func(all []TestClient, slug string) Authenticator { return newStaticTokens(all, slug) }
}

// staticTokens is the insecure-test authenticator: bearer tokens from config,
// accepted only on a loopback listener.
type staticTokens struct {
	clients map[string]Client
	known   []string
}

func newStaticTokens(all []TestClient, slug string) staticTokens {
	a := staticTokens{clients: map[string]Client{}}
	for _, t := range all {
		a.known = append(a.known, t.Token)
		for _, s := range t.Signets {
			if s == slug {
				a.clients[t.Token] = Client{ID: t.ID, Kind: t.Kind, Scope: t.Scope}
			}
		}
	}
	return a
}

func (a staticTokens) Authenticate(r *http.Request) (Client, error) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || len(r.Header.Values("Authorization")) != 1 {
		return Client{}, ErrUnauthenticated
	}
	for _, known := range a.known {
		if subtle.ConstantTimeCompare([]byte(token), []byte(known)) == 1 {
			if c, ok := a.clients[known]; ok {
				return c, nil
			}
			return Client{}, ErrForbidden
		}
	}
	return Client{}, ErrUnauthenticated
}
