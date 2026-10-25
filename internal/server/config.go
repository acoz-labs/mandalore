// Package server is Mandalore's single-writer MCP server. It owns one or more
// signets, serves each over Streamable HTTP and backs each up by Git push.
package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

const maxConfigBytes = 65536

type Config struct {
	Listen    string `json:"listen"`
	PublicURL string `json:"public_url"`
	StateDir  string `json:"state_dir"`
	Owner     Owner  `json:"owner"`
	// Auth selects the authenticator: oauth (the default), or loopback-trust
	// on a loopback listener. insecure-test is accepted only by test builds.
	Auth        string       `json:"auth,omitempty"`
	LocalScope  Scope        `json:"local_scope,omitempty"`
	TestClients []TestClient `json:"test_clients,omitempty"`
	// CanonRefreshMinutes is the canon refresh interval: default 10, minimum 1.
	CanonRefreshMinutes int            `json:"canon_refresh_minutes,omitempty"`
	Signets             []SignetConfig `json:"signets"`
}

type Owner struct {
	Name string `json:"name"`
}

type SignetConfig struct {
	Slug     string `json:"slug"`
	Root     string `json:"root"`
	SignetID string `json:"signet_id"`
	DeviceID string `json:"device_id"`
	Remote   string `json:"remote"`
	// Canon lists the canon foundlings to refresh and serve; each must be
	// registered in the signet. Legacy maps legacy foundlings to local roots
	// the server can read.
	Canon  []string          `json:"canon,omitempty"`
	Legacy map[string]string `json:"legacy,omitempty"`
}

// TestClient is a static bearer token for the insecure-test authenticator.
type TestClient struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"`
	Scope   Scope    `json:"scope"`
	Token   string   `json:"token"`
	Signets []string `json:"signets"`
}

const (
	AuthOAuth         = "oauth"
	AuthLoopbackTrust = "loopback-trust"
	AuthInsecureTest  = "insecure-test"
)

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

// LoadConfig strictly decodes a configuration file and refuses unknown fields.
func LoadConfig(path string) (Config, error) {
	var c Config
	f, err := os.Open(path)
	if err != nil {
		return c, errors.New("server configuration unavailable")
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxConfigBytes+1))
	if err != nil {
		return c, errors.New("server configuration unavailable")
	}
	if err := strictjson.Decode(raw, &c, maxConfigBytes); err != nil {
		return c, errors.New("invalid server configuration: unknown, duplicate or mistyped field")
	}
	return c, c.Validate()
}

func loopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c Config) Validate() error {
	host, port, err := net.SplitHostPort(c.Listen)
	if err != nil || host == "" || port == "" {
		return errors.New("listen requires an explicit host and port")
	}
	u, err := url.Parse(c.PublicURL)
	if err != nil || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return errors.New("public_url must be a bare origin such as https://memory.example.com")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && loopbackHost(u.Hostname())) {
		return errors.New("public_url must use HTTPS unless it is loopback")
	}
	if !filepath.IsAbs(c.StateDir) {
		return errors.New("state_dir must be an absolute path")
	}
	if strings.TrimSpace(c.Owner.Name) == "" || len(c.Owner.Name) > 256 {
		return errors.New("owner.name requires 1–256 bytes")
	}
	switch c.auth() {
	case AuthOAuth:
		// A passkey binds to a host name, never to an IP address.
		if net.ParseIP(u.Hostname()) != nil {
			return errors.New("auth oauth needs a public_url host name, such as http://localhost:8730 on loopback")
		}
	case AuthLoopbackTrust, AuthInsecureTest:
		if !loopbackHost(host) {
			return fmt.Errorf("auth %q is refused on a non-loopback listener", c.Auth)
		}
	default:
		return errors.New("auth must be oauth or loopback-trust")
	}
	if c.Auth == AuthInsecureTest && newTestAuthenticator == nil {
		return errors.New("auth insecure-test exists only in test builds")
	}
	// A non-loopback public URL means a proxy or tunnel relays requests, and a
	// relayed request must never become the trusted local client.
	if c.Auth == AuthLoopbackTrust && !loopbackHost(u.Hostname()) {
		return errors.New("auth loopback-trust requires a loopback public_url")
	}
	if (c.Auth == AuthLoopbackTrust) != c.LocalScope.valid() {
		return errors.New("local_scope (read or read-write) is required with loopback-trust and only there")
	}
	if len(c.Signets) == 0 || len(c.Signets) > 16 {
		return errors.New("signets requires 1–16 entries")
	}
	slugs, roots := map[string]bool{}, map[string]bool{}
	for _, s := range c.Signets {
		if !slugPattern.MatchString(s.Slug) || slugs[s.Slug] {
			return errors.New("each signet needs a unique lowercase slug")
		}
		root := filepath.Clean(s.Root)
		if !filepath.IsAbs(s.Root) || roots[root] {
			return errors.New("each signet needs a unique absolute root")
		}
		if s.SignetID == "" || s.DeviceID == "" || s.Remote == "" {
			return fmt.Errorf("signet %s requires signet_id, device_id and remote", s.Slug)
		}
		if err := s.validateFoundlings(); err != nil {
			return err
		}
		slugs[s.Slug], roots[root] = true, true
	}
	if c.CanonRefreshMinutes != 0 && (c.CanonRefreshMinutes < 1 || c.CanonRefreshMinutes > 1440) {
		return errors.New("canon_refresh_minutes requires 1–1440")
	}
	if (c.Auth == AuthInsecureTest) != (len(c.TestClients) > 0) {
		return errors.New("test_clients are required with insecure-test and only there")
	}
	ids, tokens := map[string]bool{}, map[string]bool{}
	for _, t := range c.TestClients {
		if t.ID == "" || ids[t.ID] || strings.TrimSpace(t.Kind) == "" || !t.Scope.valid() || len(t.Token) < 16 || tokens[t.Token] || len(t.Signets) == 0 {
			return errors.New("each test client needs a unique id, a kind, a scope, a unique token of 16+ bytes and signets")
		}
		for _, slug := range t.Signets {
			if !slugs[slug] {
				return fmt.Errorf("test client %s names unknown signet %s", t.ID, slug)
			}
		}
		ids[t.ID], tokens[t.Token] = true, true
	}
	return nil
}

func (c Config) auth() string {
	if c.Auth == "" {
		return AuthOAuth
	}
	return c.Auth
}

func (s SignetConfig) validateFoundlings() error {
	if len(s.Canon) > 64 || len(s.Legacy) > 64 {
		return fmt.Errorf("signet %s allows at most 64 canon and 64 legacy foundlings", s.Slug)
	}
	seen := map[string]bool{}
	for _, id := range s.Canon {
		if !foundlings.ValidID(id) || seen[id] {
			return fmt.Errorf("signet %s: canon needs unique foundling IDs", s.Slug)
		}
		seen[id] = true
	}
	for id, root := range s.Legacy {
		if !foundlings.ValidID(id) || seen[id] {
			return fmt.Errorf("signet %s: legacy needs foundling IDs not listed as canon", s.Slug)
		}
		if !filepath.IsAbs(root) || filepath.Clean(root) != root {
			return fmt.Errorf("signet %s: legacy %s needs a clean absolute root", s.Slug, id)
		}
	}
	return nil
}

func (c Config) canonInterval() time.Duration {
	if c.CanonRefreshMinutes == 0 {
		return 10 * time.Minute
	}
	return time.Duration(c.CanonRefreshMinutes) * time.Minute
}

// mcpPath is the signet's route: /mcp alone, /s/<slug>/mcp among several.
func (c Config) mcpPath(slug string) string {
	if len(c.Signets) == 1 {
		return "/mcp"
	}
	return "/s/" + slug + "/mcp"
}

func (c Config) origin() string {
	u, _ := url.Parse(c.PublicURL)
	return u.Scheme + "://" + u.Host
}
