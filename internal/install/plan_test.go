package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func fixture(t *testing.T) Options {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "signet")
	if _, err := memory.Create(root, "Synthetic", "device-bootstrap", "Fixture"); err != nil {
		t.Fatal(err)
	}
	b := filepath.Join(dir, "binding.json")
	if _, err := binding.Bind(root, b, "Test machine", "Synthetic writer"); err != nil {
		t.Fatal(err)
	}
	runtime := filepath.Join(dir, "runtime with spaces")
	// Planning must inspect bytes, never execute this intentionally failing file.
	if err := os.WriteFile(runtime, []byte("#!/bin/sh\nexit 99\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return Options{StateDir: filepath.Join(dir, "state"), NativeHome: filepath.Join(dir, "native"), NativeBinary: runtime, Binary: runtime, Binding: b}
}

func TestPlanIsDeterministicReadOnlyAndPinsAllInputs(t *testing.T) {
	o := fixture(t)
	before, err := os.ReadFile(o.Binding)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Prepare(o)
	if err != nil || !reflect.DeepEqual(p, again) {
		t.Fatal("unstable preview", err)
	}
	if p.PluginID != "mandalore@mandalore" || p.SchemaVersion != 1 || len(p.BinarySHA256) != 64 || len(p.PackageSHA256) != 64 || len(p.BindingSHA256) != 64 {
		t.Fatal(p)
	}
	if !strings.HasPrefix(p.Runtime, o.StateDir+string(filepath.Separator)) || !strings.HasPrefix(p.Root, o.StateDir+string(filepath.Separator)) {
		t.Fatal("escaped managed root", p)
	}
	if _, err := os.Lstat(o.StateDir); !os.IsNotExist(err) {
		t.Fatal("preview wrote state", err)
	}
	after, err := os.ReadFile(o.Binding)
	if err != nil || string(before) != string(after) {
		t.Fatal("preview changed binding", err)
	}
	if err := os.WriteFile(o.Binary, []byte("new trusted bytes"), 0700); err != nil {
		t.Fatal(err)
	}
	changed, err := Prepare(o)
	if err != nil || changed.Root == p.Root || changed.Runtime == p.Runtime {
		t.Fatal("runtime change not reflected", err)
	}
	var b binding.Binding
	if err := json.Unmarshal(before, &b); err != nil {
		t.Fatal(err)
	}
	b.Actor = "Different writer"
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(o.Binding, data, 0600); err != nil {
		t.Fatal(err)
	}
	rebound, err := Prepare(o)
	if err != nil || rebound.Root == changed.Root {
		t.Fatal("binding change not reflected", err)
	}
}

func TestPlanRefusesOverlapAndUnsafeDestinations(t *testing.T) {
	for _, kind := range []string{"relative", "control", "inside-bank", "bank-inside-state", "native-overlap", "redirected-managed", "dangling", "directory-artifact", "state-file", "fifo-artifact", "oversized-artifact"} {
		t.Run(kind, func(t *testing.T) {
			o := fixture(t)
			root := filepath.Join(filepath.Dir(o.Binding), "signet")
			switch kind {
			case "relative":
				o.StateDir = "state"
			case "control":
				o.StateDir += "\x1b"
			case "inside-bank":
				o.StateDir = filepath.Join(root, "state")
			case "bank-inside-state":
				o.StateDir = filepath.Dir(root)
			case "native-overlap":
				o.StateDir = o.NativeHome
			case "redirected-managed":
				if err := os.Mkdir(o.StateDir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(root, filepath.Join(o.StateDir, "connections")); err != nil {
					t.Fatal(err)
				}
			case "dangling":
				if err := os.Symlink(filepath.Join(root, "absent"), o.StateDir); err != nil {
					t.Fatal(err)
				}
			case "directory-artifact":
				o.Binary = root
			case "state-file":
				if err := os.WriteFile(o.StateDir, []byte("preserve"), 0600); err != nil {
					t.Fatal(err)
				}
			case "fifo-artifact":
				o.Binary = filepath.Join(filepath.Dir(root), "fifo")
				if err := syscall.Mkfifo(o.Binary, 0600); err != nil {
					t.Fatal(err)
				}
			case "oversized-artifact":
				if err := os.Truncate(o.Binary, maxBinary+1); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Prepare(o); err == nil {
				t.Fatal("unsafe plan accepted")
			}
		})
	}
}

func TestEmbeddedPackageIncludesHiddenNativeResources(t *testing.T) {
	files, err := packageFiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		".agents/plugins/marketplace.json",
		"plugins/mandalore/.codex-plugin/plugin.json",
		"plugins/mandalore/.mcp.json",
		"plugins/mandalore/hooks/hooks.json",
		"plugins/mandalore/scripts/run-memory.sh",
		"plugins/mandalore/skills/this-is-the-way/SKILL.md",
	} {
		if len(files[name]) == 0 {
			t.Fatal("missing packaged resource", name)
		}
	}
}

func TestDigestUsesTheSelectedSizeLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime")
	if err := os.WriteFile(path, []byte("small synthetic executable"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := digestLimit(path, 4); err == nil {
		t.Fatal("size bound ignored")
	}
	if _, err := digestLimit(path, 128); err != nil {
		t.Fatal(err)
	}
	if maxNativeBinary <= maxBinary {
		t.Fatal("native executable needs its independently bounded size allowance")
	}
}
