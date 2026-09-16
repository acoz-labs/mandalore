package readiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
)

type bindingFixture struct {
	path, root, manifest, device string
	binding                      binding.Binding
}

func writeFixtureJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func newBindingFixture(t *testing.T) bindingFixture {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := bindingFixture{path: filepath.Join(dir, "binding.json"), root: filepath.Join(dir, "bank")}
	f.manifest = filepath.Join(f.root, "signet.json")
	f.device = filepath.Join(f.root, "provenance", "devices", "device-synthetic.json")
	if err := os.MkdirAll(filepath.Dir(f.device), 0700); err != nil {
		t.Fatal(err)
	}
	f.binding = binding.Binding{Version: 1, Root: f.root, SignetID: "signet-synthetic", DeviceID: "device-synthetic", Actor: "PRIVATE_ACTOR_CANARY"}
	writeFixtureJSON(t, f.path, f.binding)
	writeFixtureJSON(t, f.manifest, memory.Signet{Version: 1, ID: f.binding.SignetID, Name: "PRIVATE_BANK_NAME_CANARY"})
	writeFixtureJSON(t, f.device, memory.Device{Version: 1, ID: f.binding.DeviceID, Label: "PRIVATE_DEVICE_LABEL_CANARY"})
	// Deliberately not a healthy record graph; static connection inspection must
	// not open or validate any of these remembered-content locations.
	for _, name := range []string{"memory", "foundlings", ".git"} {
		if err := os.WriteFile(filepath.Join(f.root, name), []byte("PRIVATE_CONTENT_CANARY: invalid content"), 0000); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func TestInspectBindingReadsOnlySelectedMetadata(t *testing.T) {
	f := newBindingFixture(t)
	allowed := map[string]int64{f.path: 16 << 10, f.manifest: 4 << 20, f.device: 4 << 20}
	seen := map[string]int{}
	read := func(ctx context.Context, path string, limit int64) ([]byte, error) {
		if expected, ok := allowed[path]; !ok || limit != expected {
			t.Fatalf("unexpected content read or budget: %q %d", path, limit)
		}
		seen[path]++
		return readMetadata(ctx, path, limit)
	}
	got, err := inspectBinding(context.Background(), f.path, "codex", read)
	if err != nil || got.Setup != "verified-static" || got.Code != "binding-metadata-consistent" || !got.Complete {
		t.Fatal("valid identity metadata not recognized", got, err)
	}
	if !reflect.DeepEqual(seen, map[string]int{f.path: 1, f.manifest: 1, f.device: 1}) {
		t.Fatal("wrong metadata read set", seen)
	}
	raw, err := os.ReadFile(f.path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if got.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("binding digest does not identify decoded bytes")
	}
	encoded, err := json.Marshal(got)
	if err != nil || strings.Contains(string(encoded), "PRIVATE_") || strings.Contains(string(encoded), f.root) {
		t.Fatal("projection exposed private metadata", string(encoded), err)
	}
}

func TestInspectBindingMalformedOrMismatchedMetadata(t *testing.T) {
	for _, which := range []string{"binding", "duplicate-binding", "binding-version", "root-relative", "root-control", "device-traversal", "actor", "manifest", "manifest-version", "identity", "device", "device-id", "legacy", "binding-inside-bank", "redirected-manifest"} {
		t.Run(which, func(t *testing.T) {
			f := newBindingFixture(t)
			switch which {
			case "binding":
				if err := os.WriteFile(f.path, []byte("PRIVATE_INVALID_BINDING"), 0600); err != nil {
					t.Fatal(err)
				}
			case "duplicate-binding":
				if err := os.WriteFile(f.path, []byte(`{"schema_version":1,"schema_version":1}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "binding-version":
				f.binding.Version = 2
				writeFixtureJSON(t, f.path, f.binding)
			case "root-relative":
				f.binding.Root = "relative"
				writeFixtureJSON(t, f.path, f.binding)
			case "root-control":
				f.binding.Root += "\nPRIVATE_CONTROL"
				writeFixtureJSON(t, f.path, f.binding)
			case "device-traversal":
				f.binding.DeviceID = "../../PRIVATE_TRAVERSAL"
				writeFixtureJSON(t, f.path, f.binding)
			case "actor":
				f.binding.Actor = " "
				writeFixtureJSON(t, f.path, f.binding)
			case "manifest":
				if err := os.WriteFile(f.manifest, []byte("PRIVATE_INVALID_MANIFEST"), 0600); err != nil {
					t.Fatal(err)
				}
			case "manifest-version":
				writeFixtureJSON(t, f.manifest, memory.Signet{Version: 2, ID: f.binding.SignetID, Name: "Synthetic"})
			case "identity":
				writeFixtureJSON(t, f.manifest, memory.Signet{Version: 1, ID: "signet-other", Name: "Synthetic"})
			case "device":
				if err := os.WriteFile(f.device, []byte("PRIVATE_INVALID_DEVICE"), 0600); err != nil {
					t.Fatal(err)
				}
			case "device-id":
				writeFixtureJSON(t, f.device, memory.Device{Version: 1, ID: "device-other", Label: "Synthetic"})
			case "legacy":
				if err := os.WriteFile(filepath.Join(f.root, "bank.json"), []byte("PRIVATE_LEGACY"), 0600); err != nil {
					t.Fatal(err)
				}
			case "binding-inside-bank":
				f.path = filepath.Join(f.root, "binding.json")
				writeFixtureJSON(t, f.path, f.binding)
			case "redirected-manifest":
				other := filepath.Join(filepath.Dir(f.root), "other-manifest.json")
				if err := os.Rename(f.manifest, other); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, f.manifest); err != nil {
					t.Fatal(err)
				}
			}
			got, err := inspectBinding(context.Background(), f.path, "pi", readMetadata)
			if err != nil || got.Setup != "inconsistent" || got.Complete {
				t.Fatal("invalid metadata not reported as incomplete/inconsistent", got, err)
			}
			encoded, _ := json.Marshal(got)
			if strings.Contains(string(encoded), "PRIVATE_") || strings.Contains(string(encoded), f.root) {
				t.Fatal("raw metadata leaked", string(encoded))
			}
		})
	}
}

func TestInspectBindingMissingAndCancellation(t *testing.T) {
	f := newBindingFixture(t)
	got, err := inspectBinding(context.Background(), filepath.Join(filepath.Dir(f.root), "absent.json"), "pi", readMetadata)
	if err != nil || got.Setup != "missing" || !got.Complete {
		t.Fatal("missing setup should be a complete finding", got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	read := func(ctx context.Context, path string, limit int64) ([]byte, error) {
		called = true
		return readMetadata(ctx, path, limit)
	}
	got, err = inspectBinding(ctx, f.path, "pi", read)
	if !errors.Is(err, context.Canceled) || called || got != (metadataObservation{}) {
		t.Fatal("cancelled observation did work or returned success", got, err)
	}
}

func TestInspectBindingDanglingRedirectIsNotMissingSetup(t *testing.T) {
	f := newBindingFixture(t)
	link := filepath.Join(filepath.Dir(f.root), "redirected-binding")
	if err := os.Symlink(filepath.Join(filepath.Dir(f.root), "absent.json"), link); err != nil {
		t.Fatal(err)
	}
	got, err := inspectBinding(context.Background(), link, "pi", readMetadata)
	if err != nil || got.Setup != "inconsistent" || got.Code != "metadata-unsafe" || got.Complete {
		t.Fatal("unsafe binding redirect was mislabeled absent", got, err)
	}
}

func TestInspectBindingStopsOnFailureWithoutLeakingRawErrors(t *testing.T) {
	for _, failure := range []string{"permission", "cancel", "missing", "oversized"} {
		t.Run(failure, func(t *testing.T) {
			f := newBindingFixture(t)
			reads := 0
			read := func(ctx context.Context, path string, limit int64) ([]byte, error) {
				reads++
				if reads == 1 {
					return readMetadata(ctx, path, limit)
				}
				if path != f.manifest {
					t.Fatal("continued reading after failed metadata", path)
				}
				switch failure {
				case "permission":
					return nil, &os.PathError{Op: "read", Path: "PRIVATE_ERROR_CANARY", Err: os.ErrPermission}
				case "cancel":
					return nil, context.Canceled
				case "missing":
					return nil, os.ErrNotExist
				default:
					return nil, errMetadataLimit
				}
			}
			got, err := inspectBinding(context.Background(), f.path, "pi", read)
			if reads != 2 {
				t.Fatal("unexpected follow-on reads", reads)
			}
			if failure == "cancel" {
				if !errors.Is(err, context.Canceled) || got != (metadataObservation{}) {
					t.Fatal("cancel lost", got, err)
				}
				return
			}
			if err != nil || got.Complete || got.Setup == "verified-static" {
				t.Fatal("failed read became successful observation", got, err)
			}
			encoded, _ := json.Marshal(got)
			if strings.Contains(string(encoded), "PRIVATE_") {
				t.Fatal("raw error exposed", string(encoded))
			}
		})
	}
}

func TestInspectBindingRejectsDeviceTraversalBeforeDependentReads(t *testing.T) {
	f := newBindingFixture(t)
	f.binding.DeviceID = "../../other"
	writeFixtureJSON(t, f.path, f.binding)
	reads := 0
	read := func(ctx context.Context, path string, limit int64) ([]byte, error) {
		reads++
		if path != f.path {
			t.Fatal("unsafe device ID reached a dependent read", path)
		}
		return readMetadata(ctx, path, limit)
	}
	got, err := inspectBinding(context.Background(), f.path, "pi", read)
	if err != nil || reads != 1 || got.Setup != "inconsistent" {
		t.Fatal("unsafe identity not stopped", got, err)
	}
}
