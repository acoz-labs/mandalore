package server

import (
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"strings"
)

type Scope string

const (
	ScopeRead      Scope = "read"
	ScopeReadWrite Scope = "read-write"
)

func (s Scope) valid() bool { return s == ScopeRead || s == ScopeReadWrite }

// Client is one approved identity. DeviceID is its enrolled provenance device
// in the signet; the server fills it in before the client's first write.
type Client struct {
	ID       string
	Kind     string
	DeviceID string
	Scope    Scope
}

func (c Client) CanWrite() bool { return c.Scope == ScopeReadWrite }

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("client not authorized for this signet")
)

// Authenticator identifies the client of a request to one signet. It returns
// ErrForbidden for a known client without access to that signet.
type Authenticator interface {
	Authenticate(r *http.Request) (Client, error)
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

// loopbackTrust accepts any loopback peer addressing the server by a loopback
// name as the single client "local". Any local process can use it.
type loopbackTrust struct{ scope Scope }

func (a loopbackTrust) Authenticate(r *http.Request) (Client, error) {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	host, _, hostErr := net.SplitHostPort(r.Host)
	if hostErr != nil {
		host = r.Host
	}
	if ip := net.ParseIP(peer); err != nil || ip == nil || !ip.IsLoopback() || (host != "127.0.0.1" && host != "localhost") {
		return Client{}, ErrUnauthenticated
	}
	return Client{ID: "local", Kind: "local", Scope: a.scope}, nil
}
