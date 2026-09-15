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

func TestPiConnectionCLIAndTypedPlanShareContract(t *testing.T) {
	dir := t.TempDir()
	root, path := filepath.Join(dir, "signet"), filepath.Join(dir, "binding.json")
	for _, args := range [][]string{
		{"signet", "create", "--repository", root, "--name", "Synthetic", "--device-label", "Test"},
		{"signet", "bind", "--repository", root, "--binding", path, "--device-label", "Test", "--actor", "Synthetic"},
	} {
		if v, code := cli(t, args, ""); code != 0 || !v.OK {
			t.Fatal(v, code)
		}
	}
	binary := filepath.Join(dir, "inert-runtime")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 99\n"), 0700); err != nil {
		t.Fatal(err)
	}
	o := install.PiOptions{Options: install.Options{StateDir: filepath.Join(dir, "state"), NativeHome: filepath.Join(dir, "profile"), NativeBinary: binary, Binary: binary, Binding: path}, ReadOnly: true}
	v, code := cli(t, []string{"connection", "plan", "--harness", "pi", "--state-dir", o.StateDir, "--native-home", o.NativeHome, "--native-binary", binary, "--binary", binary, "--binding", path, "--memory-read-only", "--read-only"}, "")
	if code != 0 || !v.OK {
		t.Fatal("Pi CLI preview unavailable", v, code)
	}
	raw, _ := json.Marshal(o)
	shared := api.New(nil, true).Call(context.Background(), "pi_connection_plan", raw)
	human, _ := json.Marshal(v.Result)
	agent, _ := json.Marshal(shared.Result)
	var humanValue, agentValue any
	if err := json.Unmarshal(human, &humanValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(agent, &agentValue); err != nil {
		t.Fatal(err)
	}
	if !shared.OK || !reflect.DeepEqual(humanValue, agentValue) {
		t.Fatal("human and typed previews differ", shared)
	}
	envelope, _ := json.Marshal(v)
	denied, code := cli(t, []string{"connection", "apply", "--harness", "pi", "--read-only"}, string(envelope))
	if code != 2 || denied.OK || denied.Error.Code != "operation.read_only" {
		t.Fatal("Pi read-only bypass", denied)
	}
	failed, code := cli(t, []string{"connection", "apply", "--harness", "pi"}, string(envelope))
	if code != 1 || failed.OK || failed.Error.PiConnectionResult == nil || failed.Error.PiConnectionResult.Phase != "preflight" {
		t.Fatal("Pi failure receipt missing", failed, code)
	}
}

func TestPiArmorerAliasIsReadOnlyAndHarnessExplicit(t *testing.T) {
	dir := t.TempDir()
	flags := []string{"--harness", "pi", "--state-dir", filepath.Join(dir, "state"), "--native-home", filepath.Join(dir, "profile"), "--native-binary", filepath.Join(dir, "pi"), "--read-only"}
	doctor, dc := cli(t, append([]string{"connection", "doctor"}, flags...), "")
	armorer, ac := cli(t, append([]string{"connection", "armorer"}, flags...), "")
	if ac != dc || !reflect.DeepEqual(doctor, armorer) || armorer.Error == nil || armorer.Error.PiConnectionReport == nil {
		t.Fatal("Pi diagnostic alias mismatch", doctor, armorer)
	}
	if _, err := os.Stat(filepath.Join(dir, "state")); !os.IsNotExist(err) {
		t.Fatal("Pi doctor wrote state")
	}
	if _, err := os.Stat(filepath.Join(dir, "profile")); !os.IsNotExist(err) {
		t.Fatal("Pi doctor wrote profile")
	}
	bad, code := cli(t, []string{"connection", "plan", "--harness", "unknown"}, "")
	if code != 2 || bad.OK {
		t.Fatal("unknown harness accepted")
	}
}
