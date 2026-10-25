package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
)

const (
	maxDocumentBytes = 16384
	documentTimeout  = 5 * time.Second
	documentTTL      = 10 * time.Minute
	maxDocuments     = 64
	maxRedirectURIs  = 10
)

// checkRedirectURI accepts an HTTPS URL, or plain HTTP to localhost or
// 127.0.0.1, without user info or a fragment.
func checkRedirectURI(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.ForceQuery || len(raw) > 512 {
		return errors.New("redirect_uris must be absolute URLs without user info or a fragment")
	}
	if u.Scheme == "https" || (u.Scheme == "http" && loopbackRedirect(u)) {
		return nil
	}
	return errors.New("redirect_uris must use HTTPS, or HTTP to localhost or 127.0.0.1")
}

func loopbackRedirect(u *url.URL) bool {
	h := u.Hostname()
	return h == "localhost" || h == "127.0.0.1"
}

// redirectMatches compares a requested redirect URI with a registered one
// exactly, except that a loopback HTTP redirect ignores the port on both
// sides, since native clients pick a free port per sign-in.
func redirectMatches(registered, requested string) bool {
	if registered == requested {
		return true
	}
	a, errA := url.Parse(registered)
	b, errB := url.Parse(requested)
	if errA != nil || errB != nil || a.Scheme != "http" || b.Scheme != "http" || !loopbackRedirect(a) || a.Hostname() != b.Hostname() {
		return false
	}
	if b.User != nil || b.Fragment != "" || b.ForceQuery {
		return false
	}
	return a.EscapedPath() == b.EscapedPath() && a.RawQuery == b.RawQuery
}

func (c *oauthClient) allows(redirect string) bool {
	for _, r := range c.RedirectURIs {
		if redirectMatches(r, redirect) {
			return true
		}
	}
	return false
}

// clientMetadata is the subset of RFC 7591 client metadata the server reads,
// from a registration request or a client ID metadata document.
type clientMetadata struct {
	ClientID                string   `json:"client_id,omitempty"`
	ClientName              string   `json:"client_name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
}

// check validates redirect URIs and the grant and response types a public
// authorization-code client needs.
func (m clientMetadata) check() error {
	if len(m.RedirectURIs) == 0 || len(m.RedirectURIs) > maxRedirectURIs {
		return fmt.Errorf("redirect_uris requires 1–%d URLs", maxRedirectURIs)
	}
	for _, r := range m.RedirectURIs {
		if err := checkRedirectURI(r); err != nil {
			return err
		}
	}
	if len(m.GrantTypes) > 0 && !contains(m.GrantTypes, "authorization_code") {
		return errors.New("grant_types must include authorization_code")
	}
	if len(m.ResponseTypes) > 0 && !contains(m.ResponseTypes, "code") {
		return errors.New("response_types must include code")
	}
	return nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// displayName keeps a client-supplied name short and printable. It is shown
// to the owner on the consent page, so it never overrides the redirect host.
func displayName(name, fallback string) string {
	name = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, strings.TrimSpace(name))
	if len(name) > 64 {
		name = strings.ToValidUTF8(name[:64], "")
	}
	if strings.TrimSpace(name) == "" {
		return fallback
	}
	return name
}

// clientKind is the harness recorded on the client's writes: the host of its
// metadata document, or of its first non-loopback redirect.
func clientKind(id string, document bool, redirects []string) string {
	if document {
		u, _ := url.Parse(id)
		return u.Hostname()
	}
	for _, r := range redirects {
		if u, err := url.Parse(r); err == nil && !loopbackRedirect(u) {
			return u.Hostname()
		}
	}
	return "localhost"
}

// isDocumentID reports whether a client_id is a client ID metadata document
// URL: HTTPS with a path, no user info, query or fragment.
func isDocumentID(id string) bool {
	u, err := url.Parse(id)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.Path != "" && u.Path != "/" && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && len(id) <= 512
}

// publicAddr reports whether an address is globally routable unicast, so a
// metadata fetch can't reach loopback, private, link-local or other special
// networks.
func publicAddr(a netip.Addr) bool {
	a = a.Unmap()
	if !a.IsGlobalUnicast() || a.IsPrivate() || a.IsLoopback() || a.IsLinkLocalUnicast() {
		return false
	}
	for _, p := range specialPrefixes {
		if p.Contains(a) {
			return false
		}
	}
	return true
}

var specialPrefixes = func() []netip.Prefix {
	var out []netip.Prefix
	for _, s := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64", "2001::/23", "2001:db8::/32", "2002::/16", "fec0::/10"} {
		out = append(out, netip.MustParsePrefix(s))
	}
	return out
}()

// documentClient fetches metadata documents over HTTPS to public addresses
// only. The address is checked after DNS resolution, at connect time, so a
// name that resolves to a private address is refused too. No proxy, no
// redirects.
func documentClient() *http.Client {
	dialer := &net.Dialer{Timeout: documentTimeout, Control: func(_, address string, _ syscall.RawConn) error {
		ap, err := netip.ParseAddrPort(address)
		if err != nil || !publicAddr(ap.Addr()) {
			return errors.New("metadata document address is not public")
		}
		return nil
	}}
	return &http.Client{
		Timeout:       documentTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("metadata document redirected") },
		Transport: &http.Transport{Proxy: nil, DialContext: dialer.DialContext, TLSHandshakeTimeout: documentTimeout,
			ResponseHeaderTimeout: documentTimeout, MaxResponseHeaderBytes: 16384, ForceAttemptHTTP2: true},
	}
}

type document struct {
	meta    clientMetadata
	fetched time.Time
}

// documents caches fetched client ID metadata documents.
type documents struct {
	client *http.Client
	now    func() time.Time

	mu    sync.Mutex
	cache map[string]document
}

func (d *documents) fetch(ctx context.Context, id string) (clientMetadata, error) {
	d.mu.Lock()
	cached, ok := d.cache[id]
	d.mu.Unlock()
	if ok && d.now().Before(cached.fetched.Add(documentTTL)) {
		return cached.meta, nil
	}
	meta, err := d.get(ctx, id)
	if err != nil {
		return clientMetadata{}, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.cache) >= maxDocuments {
		clear(d.cache)
	}
	d.cache[id] = document{meta: meta, fetched: d.now()}
	return meta, nil
}

func (d *documents) get(ctx context.Context, id string) (clientMetadata, error) {
	var meta clientMetadata
	ctx, cancel := context.WithTimeout(ctx, documentTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, id, nil)
	if err != nil {
		return meta, errors.New("client metadata document URL is invalid")
	}
	req.Header.Set("Accept", "application/json")
	resp, err := d.client.Do(req)
	if err != nil {
		return meta, errors.New("client metadata document unavailable")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxDocumentBytes+1))
	if err != nil || resp.StatusCode != http.StatusOK {
		return meta, errors.New("client metadata document unavailable")
	}
	if len(raw) > maxDocumentBytes {
		return meta, errors.New("client metadata document exceeds 16 KiB")
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return meta, errors.New("client metadata document is not valid JSON")
	}
	if meta.ClientID != id {
		return meta, errors.New("client metadata document client_id does not match its URL")
	}
	if m := meta.TokenEndpointAuthMethod; m != "" && m != "none" {
		return meta, errors.New("client metadata document must use token_endpoint_auth_method none")
	}
	if err := meta.check(); err != nil {
		return meta, err
	}
	return meta, nil
}
