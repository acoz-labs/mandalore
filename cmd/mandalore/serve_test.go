package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

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
		{"server", "clients", "revoke", "--installation", "inst-unknown"},
		{"server", "clients", "revoke", "--client", "client-unknown", "--installation", "inst-unknown"},
		{"server", "clients", "set-scope", "--installation", "inst-unknown", "--scope", "read"},
		{"server", "clients", "set-scope", "--client", "client-unknown", "--installation", "inst-unknown", "--scope", "read"},
		{"server", "clients", "set-scope", "--client", "client-unknown", "--scope", "read"},
		{"server", "clients", "set-scope", "--client", "client-unknown", "--scope", "admin"},
		{"server", "clients", "list", "--client", "x"},
		{"server", "clients", "purge"},
		{"server", "clients", "list", "--format", "yaml"},
		{"server", "clients", "list", "extra"},
		{"server", "clients", "label", "Synthetic desk"},
		{"server", "clients", "label", "--installation", "inst-unknown"},
		{"server", "clients", "label", "--installation", "inst-unknown", "one", "two"},
		{"server", "clients", "label", "--installation", "inst-unknown", "Synthetic desk"},
		{"server", "clients", "prune", "extra"},
		{"server", "owner", "reset"},
	} {
		if code, v := cli(args...); code == 0 || v["ok"] != false {
			t.Errorf("%v succeeded: %v", args, v)
		}
	}

	// A 2.0.1 state: one metadata-document client used by two installations,
	// and a third whose token has expired but is not yet pruned.
	doc := "https://client.example.com/oauth/client-metadata"
	expires := time.Now().Add(time.Hour)
	raw, _ = json.Marshal(map[string]any{
		"clients": map[string]any{doc: map[string]any{"id": doc, "name": "Synthetic Code", "kind": "client.example.com", "redirect_uris": []string{"http://localhost/callback"},
			"metadata_document": true, "registered": time.Now().Add(-time.Hour), "grants": map[string]string{"primary": "read-write"}}},
		"tokens": map[string]any{
			"hash-one":  map[string]any{"kind": "access", "client": doc, "signet": "primary", "family": "family-one", "scope": "read-write", "expires": expires},
			"hash-two":  map[string]any{"kind": "access", "client": doc, "signet": "primary", "family": "family-two", "scope": "read-write", "expires": expires},
			"hash-gone": map[string]any{"kind": "access", "client": doc, "signet": "primary", "family": "family-gone", "scope": "read-write", "expires": time.Now().Add(-time.Hour)},
		},
	})
	var config map[string]any
	configRaw, _ := os.ReadFile(path)
	_ = json.Unmarshal(configRaw, &config)
	if err := os.WriteFile(filepath.Join(config["state_dir"].(string), "oauth.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	listText := func() string {
		var text, errout bytes.Buffer
		if code := run(context.Background(), []string{"server", "clients", "list", "--format", "text", "--config", path}, strings.NewReader(""), &text, &errout); code != 0 {
			t.Fatalf("text list: %d %s", code, text.String())
		}
		return text.String()
	}
	_, v = cli("server", "clients", "list")
	statuses := map[string][]string{}
	for _, entry := range v["result"].([]any) {
		in := entry.(map[string]any)
		statuses[in["status"].(string)] = append(statuses[in["status"].(string)], in["installation"].(string))
		if _, ok := in["label"]; !ok {
			t.Fatalf("entry without a label field: %v", in)
		}
	}
	if len(statuses["approved"]) != 2 || len(statuses["expired"]) != 1 {
		t.Fatalf("clients list after migration: %v", v)
	}
	gone := statuses["expired"][0]
	if text := listText(); !strings.Contains(text, gone+"  expired  Synthetic Code (client.example.com)\n") || !strings.Contains(text, "  live tokens  0\n") {
		t.Fatalf("text list lacks the expired installation:\n%s", text)
	}
	if code, v := cli("server", "clients", "prune"); code != 0 || !slices.Equal(v["result"].(map[string]any)["pruned"].([]any), []any{gone}) {
		t.Fatalf("prune: %d %v", code, v)
	}
	_, v = cli("server", "clients", "list")
	listed := v["result"].([]any)
	if len(listed) != 2 {
		t.Fatalf("clients list after prune: %v", v)
	}
	one := listed[0].(map[string]any)["installation"].(string)

	// The label is the one argument, wherever it sits among the flags.
	if code, v := cli("server", "clients", "label", "--installation", one, " Synthetic desk "); code != 0 || v["result"].(map[string]any)["label"] != "Synthetic desk" {
		t.Fatalf("label: %d %v", code, v)
	}
	if code, v := cli("server", "clients", "label", "Synthetic\x1b[2J", "--installation", one); code == 0 || v["error"].(map[string]any)["code"] != "client.refused" {
		t.Fatalf("label with an escape: %d %v", code, v)
	}
	if _, v := cli("server", "clients", "list"); v["result"].([]any)[0].(map[string]any)["label"] != "Synthetic desk" {
		t.Fatalf("listed label: %v", v)
	}
	text := listText()
	for _, want := range []string{one + "  approved  Synthetic Code (client.example.com)\n", "  label        Synthetic desk\n", "  client       " + doc + "\n", "  live tokens  1\n", "  grants       primary=read-write\n"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text list lacks %q:\n%s", want, text)
		}
	}
	if code, v := cli("server", "clients", "prune"); code != 0 || len(v["result"].(map[string]any)["pruned"].([]any)) != 0 {
		t.Fatalf("prune with nothing expired: %d %v", code, v)
	}
	if code, v := cli("server", "clients", "revoke", "--installation", one); code != 0 || v["result"].(map[string]any)["client"] != doc {
		t.Fatalf("revoke installation: %d %v", code, v)
	}
	if _, v := cli("server", "clients", "list"); len(v["result"].([]any)) != 1 {
		t.Fatalf("clients list after revoking one installation: %v", v)
	}
	code, v = cli("server", "clients", "revoke", "--client", doc)
	if result := v["result"].(map[string]any); code != 0 || result["installations"] != 1.0 || !strings.Contains(result["notice"].(string), "every installation") {
		t.Fatalf("revoke client: %d %v", code, v)
	}
}
