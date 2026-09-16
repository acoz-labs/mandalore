package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadinessCatalogIsUnboundAndNonNetwork(t *testing.T) {
	count := 0
	for _, op := range Catalog() {
		if op.Name != "connection_assess" {
			continue
		}
		count++
		if !op.CLIOnly || !op.ReadOnly || !op.Idempotent || op.RequiresBinding || op.Network {
			t.Fatalf("readiness authority flags are wrong: %+v", op)
		}
		if op.InputSchema == nil || op.OutputSchema == nil {
			t.Fatal("readiness must have typed input and output discovery")
		}
	}
	if count != 1 {
		t.Fatalf("expected one CLI-only connection_assess operation, got %d", count)
	}
}

func readinessInput(t *testing.T, harness string) (string, map[string]any) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// No installed provider, profile or bank is needed for these observations.
	t.Setenv("PATH", filepath.Join(root, "absent-path"))
	return root, map[string]any{
		"harness": harness, "state_dir": filepath.Join(root, "state"),
		"native_home":    filepath.Join(root, "profile"),
		"native_binary":  filepath.Join(root, "native"),
		"binding":        filepath.Join(root, "binding.json"),
		"include_prompt": true,
	}
}

func TestReadinessMissingSetupIsAReportWithoutCreation(t *testing.T) {
	for _, harness := range []string{"codex", "pi"} {
		t.Run(harness, func(t *testing.T) {
			root, input := readinessInput(t, harness)
			before := privacyInventory(t, root)
			raw, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			out := New(nil, true).Call(context.Background(), "connection_assess", raw)
			if !out.OK {
				t.Fatalf("missing setup should be a successful observation, got %+v", out.Error)
			}
			result, err := json.Marshal(out.Result)
			if err != nil {
				t.Fatal(err)
			}
			var report struct {
				SchemaVersion int    `json:"schema_version"`
				Complete      bool   `json:"complete"`
				Harness       string `json:"harness"`
				Prompt        string `json:"prompt"`
				Components    []struct {
					ID    string `json:"id"`
					Setup string `json:"setup"`
				} `json:"components"`
			}
			if err := json.Unmarshal(result, &report); err != nil {
				t.Fatal(err)
			}
			if report.SchemaVersion != 1 || !report.Complete || report.Harness != harness || report.Prompt == "" {
				t.Fatalf("incomplete readiness report: %s", result)
			}
			foundMissingHarness := false
			for _, c := range report.Components {
				if c.ID == harness && c.Setup == "missing" {
					foundMissingHarness = true
				}
			}
			if !foundMissingHarness {
				t.Fatalf("missing native executable was not reported: %s", result)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(result, &fields); err != nil {
				t.Fatal(err)
			}
			if _, exists := fields["healthy"]; exists {
				t.Fatal("static assessment must not report universal health")
			}
			if !reflect.DeepEqual(before, privacyInventory(t, root)) {
				t.Fatal("assessment created or changed fixture state")
			}
		})
	}
}

func TestReadinessDoesNotRunSelectedNativeExecutable(t *testing.T) {
	root, input := readinessInput(t, "codex")
	// Write beside the explicitly selected script, independent of the child's
	// cwd or filtered environment. No private path is embedded in its contents.
	if err := os.WriteFile(input["native_binary"].(string), []byte("#!/bin/sh\nprintf executed > \"${0%/*}/readiness-execution-canary\"\nexit 97\n"), 0700); err != nil {
		t.Fatal(err)
	}
	before := privacyInventory(t, root)
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	out := New(nil, true).Call(context.Background(), "connection_assess", raw)
	if !out.OK {
		t.Fatalf("selected native bytes should not be executed: %+v", out.Error)
	}
	if !reflect.DeepEqual(before, privacyInventory(t, root)) {
		t.Fatal("assessment changed the selected fixture")
	}
}
