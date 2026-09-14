package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/install"
)

func TestConnectionPlanCLIAndAgentCatalogShareTheContract(t *testing.T) {
	dir := t.TempDir()
	root, bindingPath := filepath.Join(dir, "signet"), filepath.Join(dir, "binding.json")
	for _, args := range [][]string{
		{"signet", "create", "--repository", root, "--name", "Synthetic", "--device-label", "Test"},
		{"signet", "bind", "--repository", root, "--binding", bindingPath, "--device-label", "Test", "--actor", "Synthetic"},
	} {
		if v, code := cli(t, args, ""); code != 0 || !v.OK {
			t.Fatal(v, code)
		}
	}
	binary := filepath.Join(dir, "trusted synthetic binary")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 99\n"), 0700); err != nil {
		t.Fatal(err)
	}
	o := install.Options{StateDir: filepath.Join(dir, "state"), NativeHome: filepath.Join(dir, "native"), NativeBinary: binary, Binary: binary, Binding: bindingPath}
	v, code := cli(t, []string{"connection", "plan", "--state-dir", o.StateDir, "--native-home", o.NativeHome, "--native-binary", binary, "--binary", binary, "--binding", bindingPath}, "")
	if code != 0 || !v.OK {
		t.Fatal("connection plan unavailable", v, code)
	}
	raw, _ := json.Marshal(o)
	shared := api.New(nil, true).Call(context.Background(), "connection_plan", raw)
	b1, _ := json.Marshal(v.Result)
	b2, _ := json.Marshal(shared.Result)
	var humanResult, apiResult any
	if err := json.Unmarshal(b1, &humanResult); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b2, &apiResult); err != nil {
		t.Fatal(err)
	}
	if !shared.OK || !reflect.DeepEqual(humanResult, apiResult) {
		t.Fatal("different menu/agent contract", shared)
	}
	if _, err := os.Lstat(o.StateDir); !os.IsNotExist(err) {
		t.Fatal("plan changed state")
	}
	envelope, _ := json.Marshal(v)
	denied, code := cli(t, []string{"connection", "apply", "--read-only"}, string(envelope))
	if code != 2 || denied.OK || denied.Error.Code != "operation.read_only" {
		t.Fatal("no-write bypass", denied, code)
	}
	// Valid plan reaches execution and refuses the fake native process, retaining
	// an explicit phase rather than losing the result in a generic input error.
	failed, code := cli(t, []string{"connection", "apply"}, string(envelope))
	if code != 1 || failed.OK || failed.Error.ConnectionResult == nil || failed.Error.ConnectionResult.Phase != "preflight" {
		t.Fatal("missing failure receipt", failed, code)
	}
}
