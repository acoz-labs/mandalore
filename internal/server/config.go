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

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

const maxConfigBytes = 65536

type Config struct {
	Listen    string `json:"listen"`
	PublicURL string `json:"public_url"`
	StateDir  string `json:"state_dir"`
	Owner     Owner  `json:"owner"`
	// Auth selects the authenticator. OAuth is the production default once it
	// exists; until then a loopback-only mode must be named explicitly.
	Auth        string         `json:"auth"`
	LocalScope  Scope          `json:"local_scope,omitempty"`
	TestClients []TestClient   `json:"test_clients,omitempty"`
	Signets     []SignetConfig `json:"signets"`
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
	switch c.Auth {
	case AuthLoopbackTrust, AuthInsecureTest:
		if !loopbackHost(host) {
			return fmt.Errorf("auth %q is refused on a non-loopback listener", c.Auth)
		}
	default:
		return errors.New("auth must be loopback-trust or insecure-test")
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
		slugs[s.Slug], roots[root] = true, true
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
