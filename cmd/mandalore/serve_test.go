package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerCommandsRequireExplicitConfiguration(t *testing.T) {
	for _, args := range [][]string{{"serve"}, {"serve", "--config"}, {"serve", "--unknown"}, {"serve", "--config", "missing", "extra"}, {"serve", "--config", "/fixture/private/missing-config"},
		{"server"}, {"server", "start"}, {"server", "status"}, {"server", "status", "--signet", "a", "--config", "x"}, {"server", "reconcile", "--config", "/fixture/private/missing-config"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var out, errout bytes.Buffer
			if code := run(context.Background(), args, strings.NewReader(""), &out, &errout); code == 0 {
				t.Fatal("invalid command succeeded")
			}
			if !json.Valid(out.Bytes()) || strings.Contains(out.String(), "/fixture/private/") {
				t.Fatal("error is not machine-readable or leaked a path", out.String())
			}
		})
	}
}

func TestServerConfigurationIsStrictAndDoesNotLeak(t *testing.T) {
	valid := func(mutate func(map[string]any)) string {
		c := map[string]any{"listen": "127.0.0.1:8730", "public_url": "http://127.0.0.1:8730", "state_dir": "/fixture/private/state", "owner": map[string]any{"name": "Example"}, "auth": "loopback-trust", "local_scope": "read",
			"signets": []any{map[string]any{"slug": "primary", "root": "/fixture/private/signet", "signet_id": "signet-fixture", "device_id": "device-fixture", "remote": "file:///fixture/private/remote.git"}}}
		mutate(c)
		raw, _ := json.Marshal(c)
		return string(raw)
	}
	for _, test := range []struct{ name, body string }{
		{"malformed", `{"auth":`},
		{"unknown field", valid(func(c map[string]any) { c["private_token"] = "never-emit-fixture-token" })},
		{"unknown signet field", valid(func(c map[string]any) {
			c["signets"].([]any)[0].(map[string]any)["canon"] = []string{"never-emit-fixture-token"}
		})},
		{"duplicate field", `{"listen":"127.0.0.1:0","listen":"0.0.0.0:0"}`},
		{"trailing value", `{} {}`},
		{"oversized", strings.Repeat(" ", 65537)},
		{"missing requirements", `{}`},
		{"loopback trust on a public listener", valid(func(c map[string]any) { c["listen"] = "0.0.0.0:8730" })},
		{"insecure test on a public listener", valid(func(c map[string]any) {
			c["listen"], c["auth"] = "0.0.0.0:8730", "insecure-test"
			delete(c, "local_scope")
			c["test_clients"] = []any{map[string]any{"id": "c1", "kind": "test", "scope": "read", "token": "never-emit-fixture-token", "signets": []string{"primary"}}}
		})},
		{"plain HTTP public URL", valid(func(c map[string]any) { c["public_url"] = "http://memory.example.com" })},
		{"public URL with a path", valid(func(c map[string]any) { c["public_url"] = "http://127.0.0.1:8730/mcp" })},
		{"loopback trust behind a public URL", valid(func(c map[string]any) { c["public_url"] = "https://memory.example.com" })},
		{"insecure test in a release build", valid(func(c map[string]any) {
			c["auth"] = "insecure-test"
			delete(c, "local_scope")
			c["test_clients"] = []any{map[string]any{"id": "c1", "kind": "test", "scope": "read", "token": "never-emit-fixture-token", "signets": []string{"primary"}}}
		})},
		{"missing auth", valid(func(c map[string]any) { delete(c, "auth") })},
		{"signet root unavailable", valid(func(map[string]any) {})},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(test.body), 0600); err != nil {
				t.Fatal(err)
			}
			var out, errout bytes.Buffer
			if code := run(context.Background(), []string{"serve", "--config", path}, strings.NewReader(""), &out, &errout); code == 0 {
				t.Fatal("invalid configuration accepted")
			}
			if !json.Valid(out.Bytes()) {
				t.Fatal("error is not JSON", out.String())
			}
			for _, sensitive := range []string{path, "/fixture/private/", "never-emit-fixture-token"} {
				if strings.Contains(out.String(), sensitive) || strings.Contains(errout.String(), sensitive) {
					t.Fatal("private configuration leaked", out.String(), errout.String())
				}
			}
		})
	}
}
