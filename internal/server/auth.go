package server

import (
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

// newTestAuthenticator builds the insecure-test authenticator. Only test
// builds of this package set it, so release binaries refuse insecure-test.
var newTestAuthenticator func(clients []TestClient, slug string) Authenticator

// forwardingHeaders mark a request relayed by a proxy or tunnel. A relayed
// request is not the local user, even when the proxy runs on this host. Any
// header whose name contains "forwarded" counts too, so variants fail closed.
var forwardingHeaders = []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Forwarded-Port", "X-Original-Forwarded-For", "X-Real-Ip", "Cf-Connecting-Ip", "Cf-Ray", "True-Client-Ip", "X-Client-Ip", "Client-Ip", "Fastly-Client-Ip", "X-Cluster-Client-Ip", "X-Envoy-External-Address", "Tailscale-User-Login", "Via"}

func forwarded(h http.Header) bool {
	for name := range h {
		if strings.Contains(strings.ToLower(name), "forwarded") {
			return true
		}
	}
	for _, name := range forwardingHeaders {
		if len(h.Values(name)) > 0 {
			return true
		}
	}
	return false
}

// loopbackTrust accepts a direct loopback peer addressing the server by a
// loopback name as the single client "local". Any local process can use it;
// a request carrying proxy forwarding headers is refused.
type loopbackTrust struct{ scope Scope }

func (a loopbackTrust) Authenticate(r *http.Request) (Client, error) {
	if forwarded(r.Header) {
		return Client{}, ErrUnauthenticated
	}
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
