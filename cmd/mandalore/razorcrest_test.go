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

func TestRazorCrestCLIRequiresExplicitConfiguration(t *testing.T) {
	for _, args := range [][]string{{}, {"start"}, {"serve"}, {"serve", "--config"}, {"serve", "--unknown"}, {"serve", "--config", "missing", "extra"}, {"serve", "--config", "/fixture/private/missing-config"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var out bytes.Buffer
			code := runRazorCrest(context.Background(), args, &out)
			if code == 0 {
				t.Fatal("invalid command succeeded")
			}
			if !json.Valid(out.Bytes()) {
				t.Fatal("error is not machine-readable", out.String())
			}
			if strings.Contains(out.String(), "/fixture/private/") {
				t.Fatal("private path leaked")
			}
		})
	}
}
func TestRazorCrestCLIRejectsInvalidConfigWithoutLeakingContents(t *testing.T) {
	for _, test := range []struct{ name, body string }{
		{"malformed", `{"auth":`},
		{"unknown field", `{"private_token":"never-emit-fixture-token"}`},
		{"duplicate field", `{"listen":"127.0.0.1:0","listen":"0.0.0.0:0"}`},
		{"trailing value", `{} {}`},
		{"oversized", strings.Repeat(" ", 65537)},
		{"missing requirements", `{}`},
		{"invalid binding", `{"binding":"/fixture/private/binding","binding_sha256":"invalid","signet_id":"signet-fixture","listen":"127.0.0.1:0","hosts":["memory.example"],"origin_secret_file":"/fixture/private/secret"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(test.body), 0600); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			code := runRazorCrest(context.Background(), []string{"serve", "--config", path}, &out)
			if code == 0 {
				t.Fatal("invalid configuration accepted")
			}
			if !json.Valid(out.Bytes()) {
				t.Fatal("error is not JSON", out.String())
			}
			for _, sensitive := range []string{path, "/fixture/private/", "never-emit-fixture-token"} {
				if strings.Contains(out.String(), sensitive) {
					t.Fatal("private configuration leaked", out.String())
				}
			}
		})
	}
}
func TestRazorCrestCommandDispatch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"razor-crest", "serve"}, strings.NewReader(""), &stdout, &stderr); code == 0 {
		t.Fatal("missing configuration succeeded")
	}
	if !strings.Contains(stdout.String(), "Explicit Razor Crest config required") {
		t.Fatal("command did not reach Razor Crest", stdout.String(), stderr.String())
	}
}
