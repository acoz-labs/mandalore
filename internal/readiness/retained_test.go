package readiness

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/acoz-labs/mandalore/internal/install"
)

func retainedFixture(t *testing.T) (install.ReceiptSelection, fileDigest, metadataObservation, install.Plan) {
	t.Helper()
	f := newBindingFixture(t)
	base := filepath.Dir(f.root)
	binary := filepath.Join(base, "runtime")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nprintf executed > \"${0%/*}/execution-canary\"\nexit 97\n"), 0700); err != nil {
		t.Fatal(err)
	}
	p, err := install.Prepare(install.Options{StateDir: filepath.Join(base, "state"), NativeHome: filepath.Join(base, "profile"), NativeBinary: binary, Binary: binary, Binding: f.path})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(p.Root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p.Runtime), 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Runtime, raw, 0700); err != nil {
		t.Fatal(err)
	}
	r := install.Receipt{Plan: p, Files: map[string]string{".agents/plugins/marketplace.json": p.PackageSHA256}}
	writeFixtureJSON(t, filepath.Join(p.Root, "connection.json"), r)
	native, err := hashFile(context.Background(), binary, 512<<20, true)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := inspectBinding(context.Background(), f.path, "codex", readMetadata)
	if err != nil || bound.Setup != "verified-static" {
		t.Fatal(bound, err)
	}
	s := install.ReceiptSelection{Harness: "codex", Root: p.Root, StateDir: p.StateDir, NativeHome: p.NativeHome, NativeBinary: p.NativeBinary, Binding: p.Binding}
	return s, native, bound, p
}

func TestRetainedInspectionMeasuresOnlyOwnedSelectedRuntime(t *testing.T) {
	s, native, bound, p := retainedFixture(t)
	reads, hashes := 0, 0
	read := func(ctx context.Context, path string, limit int64) ([]byte, error) {
		reads++
		if path != filepath.Join(s.Root, "connection.json") || limit != 32<<10 {
			t.Fatal("unexpected receipt read", path, limit)
		}
		return readMetadata(ctx, path, limit)
	}
	fingerprint := func(ctx context.Context, path string, limit int64, redirect bool) (fileDigest, error) {
		hashes++
		if path != p.Runtime || limit != 128<<20 || redirect {
			t.Fatal("unexpected or redirected runtime read", path, limit, redirect)
		}
		return hashFile(ctx, path, limit, redirect)
	}
	got, err := inspectRetained(context.Background(), s, native, bound, read, fingerprint)
	if err != nil || got.Setup != "verified-static" || !got.Complete || got.Runtime.SHA256 != p.BinarySHA256 || reads != 1 || hashes != 1 {
		t.Fatal("selected retained runtime not recognized", got, err, reads, hashes)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(s.Binding), "execution-canary")); !os.IsNotExist(err) {
		t.Fatal("retained program was executed", err)
	}
	// No package files exist. Success must be scoped to the measured runtime,
	// not misrepresented as complete installed-package validation.
	if got.Code != "retained-runtime-and-selection-consistent" {
		t.Fatal("incorrect evidence scope", got)
	}
}

func TestRetainedInspectionDoesNotFollowForeignSelectionOrBinding(t *testing.T) {
	for _, field := range []string{"profile", "state", "native-path", "binding-path", "binding-bytes", "native-bytes", "binding-unverified", "native-not-executable"} {
		t.Run(field, func(t *testing.T) {
			s, native, bound, _ := retainedFixture(t)
			switch field {
			case "profile":
				s.NativeHome += "-other"
			case "state":
				s.StateDir += "-other"
			case "native-path":
				s.NativeBinary += "-other"
			case "binding-path":
				s.Binding += "-other"
			case "binding-bytes":
				bound.SHA256 = "changed"
			case "native-bytes":
				native.SHA256 = "changed"
			case "binding-unverified":
				bound.Setup = "unknown"
			case "native-not-executable":
				native.Executable = false
			}
			called := false
			fingerprint := func(context.Context, string, int64, bool) (fileDigest, error) {
				called = true
				t.Fatal("followed runtime after foreign/unverified selection")
				return fileDigest{}, nil
			}
			got, err := inspectRetained(context.Background(), s, native, bound, readMetadata, fingerprint)
			if err != nil || called || got.Setup == "verified-static" || got.Complete {
				t.Fatal("foreign/stale selection was trusted", got, err)
			}
		})
	}
}

func TestRetainedInspectionRejectsChangedRuntimeAndReceipt(t *testing.T) {
	for _, field := range []string{"runtime", "receipt"} {
		s, native, bound, p := retainedFixture(t)
		if field == "runtime" {
			if err := os.WriteFile(p.Runtime, []byte("changed runtime"), 0700); err != nil {
				t.Fatal(err)
			}
		} else {
			raw, err := os.ReadFile(filepath.Join(s.Root, "connection.json"))
			if err != nil {
				t.Fatal(err)
			}
			var r install.Receipt
			if err := json.Unmarshal(raw, &r); err != nil {
				t.Fatal(err)
			}
			r.Plan.Runtime = filepath.Join(filepath.Dir(s.Binding), "foreign-runtime")
			writeFixtureJSON(t, filepath.Join(s.Root, "connection.json"), r)
		}
		got, err := inspectRetained(context.Background(), s, native, bound, readMetadata, hashFile)
		if err != nil || got.Setup != "inconsistent" || got.Complete {
			t.Fatal("edited retained state certified", got, err)
		}
	}
}

func TestRetainedUnselectedDoesNotInferMissingRegistration(t *testing.T) {
	read := func(context.Context, string, int64) ([]byte, error) {
		t.Fatal("unselected root caused reads")
		return nil, nil
	}
	hash := func(context.Context, string, int64, bool) (fileDigest, error) {
		t.Fatal("unselected root caused hashing")
		return fileDigest{}, nil
	}
	got, err := inspectRetained(context.Background(), install.ReceiptSelection{}, fileDigest{}, metadataObservation{}, read, hash)
	if err != nil || got.Setup != "unknown" || got.Code != "retained-unselected" || !got.Complete {
		t.Fatal("unselected interpreted as missing registration", got, err)
	}
}
