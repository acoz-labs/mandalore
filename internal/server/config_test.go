package server

import (
	"strings"
	"testing"
)

func validConfig() Config {
	return Config{Listen: "127.0.0.1:8730", PublicURL: "http://127.0.0.1:8730", StateDir: "/state", Owner: Owner{Name: "Owner"}, Auth: AuthInsecureTest,
		Signets:     []SignetConfig{{Slug: "a", Root: "/signets/a", SignetID: "signet-a", DeviceID: "device-a", Remote: "file:///remotes/a.git"}},
		TestClients: []TestClient{{ID: "c1", Kind: "test", Scope: ScopeRead, Token: "token-0123456789ab", Signets: []string{"a"}}}}
}

func TestConfigValidation(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Config){
		"unknown auth":              func(c *Config) { c.Auth = "oauth" },
		"local scope without trust": func(c *Config) { c.LocalScope = ScopeRead },
		"trust without local scope": func(c *Config) { c.Auth, c.TestClients = AuthLoopbackTrust, nil },
		"test clients without test": func(c *Config) { c.Auth, c.LocalScope = AuthLoopbackTrust, ScopeRead },
		"short token":               func(c *Config) { c.TestClients[0].Token = "short" },
		"client for unknown signet": func(c *Config) { c.TestClients[0].Signets = []string{"b"} },
		"duplicate slug":            func(c *Config) { c.Signets = append(c.Signets, c.Signets[0]) },
		"shared root": func(c *Config) {
			c.Signets = append(c.Signets, SignetConfig{Slug: "b", Root: "/signets/a/", SignetID: "s", DeviceID: "d", Remote: "r"})
		},
		"missing remote":              func(c *Config) { c.Signets[0].Remote = "" },
		"bad slug":                    func(c *Config) { c.Signets[0].Slug = "A/B" },
		"relative state":              func(c *Config) { c.StateDir = "state" },
		"no owner":                    func(c *Config) { c.Owner.Name = " " },
		"no signets":                  func(c *Config) { c.Signets = nil },
		"listen without host":         func(c *Config) { c.Listen = ":8730" },
		"non-loopback test listener":  func(c *Config) { c.Listen = "192.0.2.1:8730" },
		"public URL with credentials": func(c *Config) { c.PublicURL = "https://user@memory.example.com" },
		"loopback trust behind a public URL": func(c *Config) {
			c.Auth, c.TestClients, c.LocalScope, c.PublicURL = AuthLoopbackTrust, nil, ScopeReadWrite, "https://memory.example.com"
		},
	} {
		c := validConfig()
		c.TestClients = append([]TestClient{}, c.TestClients...)
		mutate(&c)
		if err := c.Validate(); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestInsecureTestNeedsATestBuild(t *testing.T) {
	saved := newTestAuthenticator
	t.Cleanup(func() { newTestAuthenticator = saved })
	newTestAuthenticator = nil
	if err := validConfig().Validate(); err == nil || !strings.Contains(err.Error(), "test builds") {
		t.Fatalf("insecure-test accepted outside a test build: %v", err)
	}
}

func TestSignetRoutes(t *testing.T) {
	c := validConfig()
	if c.mcpPath("a") != "/mcp" {
		t.Fatal("single signet is not at /mcp")
	}
	c.Signets = append(c.Signets, SignetConfig{Slug: "b", Root: "/signets/b", SignetID: "signet-b", DeviceID: "device-b", Remote: "file:///remotes/b.git"})
	if c.mcpPath("a") != "/s/a/mcp" || c.mcpPath("b") != "/s/b/mcp" {
		t.Fatal("several signets are not at /s/<slug>/mcp")
	}
	if !strings.HasPrefix(c.origin(), "http://127.0.0.1:8730") {
		t.Fatal(c.origin())
	}
}
