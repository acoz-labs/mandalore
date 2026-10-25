package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/testfixture"
)

func TestServerCommandsRequireExplicitConfiguration(t *testing.T) {
	for _, args := range [][]string{{"serve"}, {"serve", "--config"}, {"serve", "--unknown"}, {"serve", "--config", "missing", "extra"}, {"serve", "--config", "/fixture/private/missing-config"},
		{"server"}, {"server", "start"}, {"server", "status"}, {"server", "status", "--signet", "a", "--config", "x"}, {"server", "reconcile", "--config", "/fixture/private/missing-config"},
		{"server", "canon"}, {"server", "canon", "status", "--config", "x"}, {"server", "canon", "refresh"}, {"server", "canon", "refresh", "--config", "/fixture/private/missing-config"}} {
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
			c["signets"].([]any)[0].(map[string]any)["notes"] = []string{"never-emit-fixture-token"}
		})},
		{"invalid canon foundling ID", valid(func(c map[string]any) {
			c["signets"].([]any)[0].(map[string]any)["canon"] = []string{"never-emit-fixture-token!"}
		})},
		{"relative legacy root", valid(func(c map[string]any) {
			c["signets"].([]any)[0].(map[string]any)["legacy"] = map[string]string{"foundling-history": "fixture/private/history"}
		})},
		{"canon refresh below a minute", valid(func(c map[string]any) { c["canon_refresh_minutes"] = 0.5 })},
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
		{"local scope under default oauth", valid(func(c map[string]any) { delete(c, "auth") })},
		{"oauth on an IP public URL", valid(func(c map[string]any) { delete(c, "auth"); delete(c, "local_scope") })},
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

func TestServerOAuthAdminCommands(t *testing.T) {
	s := testfixture.NewServerSignet(t, "admin")
	raw, _ := json.Marshal(map[string]any{"listen": "127.0.0.1:8730", "public_url": "https://memory.example.com", "state_dir": filepath.Join(t.TempDir(), "state"), "owner": map[string]any{"name": "Example"},
		"signets": []any{map[string]any{"slug": "primary", "root": s.Root, "signet_id": s.SignetID, "device_id": s.DeviceID, "remote": s.Remote}}})
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	cli := func(args ...string) (int, map[string]any) {
		var out, errout bytes.Buffer
		code := run(context.Background(), append(args, "--config", path), strings.NewReader(""), &out, &errout)
		var v map[string]any
		if err := json.Unmarshal(out.Bytes(), &v); err != nil {
			t.Fatalf("%v: not JSON: %s", args, out.String())
		}
		return code, v
	}
	code, v := cli("server", "owner", "enroll")
	if link, _ := v["result"].(map[string]any)["link"].(string); code != 0 || !strings.HasPrefix(link, "https://memory.example.com/owner/enroll#") {
		t.Fatalf("owner enroll: %d %v", code, v)
	}
	if code, v := cli("server", "clients", "list"); code != 0 || len(v["result"].([]any)) != 0 {
		t.Fatalf("clients list: %d %v", code, v)
	}
	for _, args := range [][]string{
		{"server", "clients", "revoke", "--client", "client-unknown"},
		{"server", "clients", "revoke"},
		{"server", "clients", "set-scope", "--client", "client-unknown", "--scope", "read"},
		{"server", "clients", "set-scope", "--client", "client-unknown", "--scope", "admin"},
		{"server", "clients", "list", "--client", "x"},
		{"server", "clients", "purge"},
		{"server", "owner", "reset"},
	} {
		if code, v := cli(args...); code == 0 || v["ok"] != false {
			t.Errorf("%v succeeded: %v", args, v)
		}
	}
}
