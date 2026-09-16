package readiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/acoz-labs/mandalore/internal/install"
)

func retainedPiFixture(t *testing.T) (install.ReceiptSelection, fileDigest, metadataObservation, install.PiPlan) {
	t.Helper()
	f := newBindingFixture(t)
	base := filepath.Dir(f.root)
	binary := filepath.Join(base, "runtime")
	raw := []byte("#!/bin/sh\nprintf executed > \"${0%/*}/execution-canary\"\nexit 97\n")
	if err := os.WriteFile(binary, raw, 0700); err != nil {
		t.Fatal(err)
	}
	p, err := install.PreparePi(install.PiOptions{Options: install.Options{StateDir: filepath.Join(base, "state"), NativeHome: filepath.Join(base, "profile"), NativeBinary: binary, Binary: binary, Binding: f.path}})
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{p.Root, filepath.Dir(p.Runtime)} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(p.Runtime, raw, 0700); err != nil {
		t.Fatal(err)
	}
	// Construct only the receipt's administrative projection, without staging
	// a package or invoking native installation. Its digest uses the owned
	// format's existing indentation contract.
	connection, err := json.MarshalIndent(map[string]any{
		"schema_version": 1, "harness": "pi", "runtime": p.Runtime, "runtime_sha256": p.BinarySHA256,
		"binding": p.Binding, "binding_sha256": p.BindingSHA256, "signet_id": p.SignetID,
		"package_sha256": p.PackageSHA256, "package_version": p.PackageVersion, "read_only": p.ReadOnly,
		"native_home": p.NativeHome, "native_binary": p.NativeBinary, "state_dir": p.StateDir, "connection_root": p.Root,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(connection)
	writeFixtureJSON(t, filepath.Join(p.Root, "receipt.json"), install.PiReceipt{Plan: p, Files: map[string]string{"package/package.json": p.PackageSHA256, "package/connection.json": hex.EncodeToString(digest[:])}})
	native, err := hashFile(context.Background(), binary, 512<<20, true)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := inspectBinding(context.Background(), f.path, "pi", readMetadata)
	if err != nil || bound.Setup != "verified-static" {
		t.Fatal(bound, err)
	}
	return install.ReceiptSelection{Harness: "pi", Root: p.Root, StateDir: p.StateDir, NativeHome: p.NativeHome, NativeBinary: p.NativeBinary, Binding: p.Binding}, native, bound, p
}

func TestRetainedPiUsesOnlySelectedReceiptAndRuntime(t *testing.T) {
	s, native, bound, p := retainedPiFixture(t)
	reads, hashes := 0, 0
	read := func(ctx context.Context, path string, limit int64) ([]byte, error) {
		reads++
		if path != filepath.Join(s.Root, "receipt.json") || limit != 64<<10 {
			t.Fatal("unexpected Pi metadata read", path, limit)
		}
		return readMetadata(ctx, path, limit)
	}
	fingerprint := func(ctx context.Context, path string, limit int64, redirect bool) (fileDigest, error) {
		hashes++
		if path != p.Runtime || limit != 128<<20 || redirect {
			t.Fatal("unexpected Pi runtime read", path, limit, redirect)
		}
		return hashFile(ctx, path, limit, redirect)
	}
	r, err := inspectRetained(context.Background(), s, native, bound, read, fingerprint)
	if err != nil || !r.Complete || r.Setup != "verified-static" || r.Code != "retained-runtime-and-selection-consistent" || r.Runtime.SHA256 != p.BinarySHA256 || reads != 1 || hashes != 1 {
		t.Fatal(r, err, reads, hashes)
	}
	if _, err := os.Lstat(filepath.Join(p.Root, "package")); !os.IsNotExist(err) {
		t.Fatal("test must not require a package tree", err)
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(s.Binding), "execution-canary")); !os.IsNotExist(err) {
		t.Fatal("fixture program was executed", err)
	}
}

func TestRetainedPiRejectsStaleSelectionBeforeFollowingRuntime(t *testing.T) {
	for _, changed := range []string{"profile", "binding", "native", "receipt"} {
		t.Run(changed, func(t *testing.T) {
			s, native, bound, p := retainedPiFixture(t)
			switch changed {
			case "profile":
				s.NativeHome += "-other"
			case "binding":
				bound.SHA256 = "changed"
			case "native":
				native.SHA256 = "changed"
			case "receipt":
				if err := os.WriteFile(filepath.Join(p.Root, "receipt.json"), []byte("PRIVATE_INVALID_RECEIPT"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			r, err := inspectRetained(context.Background(), s, native, bound, readMetadata, func(context.Context, string, int64, bool) (fileDigest, error) {
				t.Fatal("stale Pi selection followed runtime")
				return fileDigest{}, nil
			})
			if err != nil || r.Complete || r.Setup != "inconsistent" {
				t.Fatal(r, err)
			}
		})
	}
}
