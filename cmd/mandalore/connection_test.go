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

func TestArmorerAliasSharesDoctorContractWithoutWrites(t *testing.T) {
	dir := t.TempDir()
	native := filepath.Join(dir, "native-tool")
	if err := os.WriteFile(native, []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	flags := []string{"--state-dir", filepath.Join(dir, "state"), "--native-home", filepath.Join(dir, "profile"), "--native-binary", native, "--read-only"}
	doctor, dc := cli(t, append([]string{"connection", "doctor"}, flags...), "")
	armorer, ac := cli(t, append([]string{"connection", "armorer"}, flags...), "")
	if ac != dc || !reflect.DeepEqual(armorer, doctor) {
		t.Fatalf("alias differs from doctor: armorer=%+v (%d), doctor=%+v (%d)", armorer, ac, doctor, dc)
	}
	if armorer.Error == nil || armorer.Error.ConnectionReport == nil {
		t.Fatal("missing structured diagnostic report", armorer)
	}
	for _, path := range []string{filepath.Join(dir, "state"), filepath.Join(dir, "profile")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("inspection created state", path, err)
		}
	}
}

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
	acknowledged, code := cli(t, []string{"connection", "apply", "--sessions-stopped"}, string(envelope))
	if code != 1 || acknowledged.Error == nil || acknowledged.Error.ConnectionResult == nil || acknowledged.Error.ConnectionResult.Phase != "preflight" {
		t.Fatal("acknowledged plan did not reach the same validated boundary", acknowledged, code)
	}
	raw, _ = json.Marshal(install.ApplyInput{Plan: shared.Result.(install.Plan), SessionsStopped: true})
	typed := api.New(nil, false).Call(context.Background(), "connection_apply", raw)
	if typed.OK || typed.Error == nil || typed.Error.ConnectionResult == nil || typed.Error.ConnectionResult.Phase != "preflight" {
		t.Fatal("typed acknowledgement rejected as schema input rather than native preflight", typed)
	}
}

func TestSessionAcknowledgementIsNotAcceptedOnOtherJourneys(t *testing.T) {
	for _, args := range [][]string{
		{"connection", "plan", "--sessions-stopped"},
		{"connection", "apply", "--harness", "pi", "--sessions-stopped"},
		{"connection", "repair", "--sessions-stopped"},
	} {
		v, code := cli(t, args, "{}")
		if code == 0 || v.OK {
			t.Fatal("invalid acknowledgement accepted", args, v)
		}
	}
	v, _ := cli(t, []string{"connection", "apply", "--sessions-stopped", "--read-only"}, "{}")
	if v.Error == nil || v.Error.Code != "operation.read_only" {
		t.Fatal("acknowledgement bypassed read-only", v)
	}
}
