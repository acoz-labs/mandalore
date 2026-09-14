package install

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func treeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err == nil {
			files[path] = hash(b)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return files
}

func TestDoctorIsReadOnlyAndDistinguishesUntestedBoundaries(t *testing.T) {
	t.Setenv("MANDALORE_BIN", "")
	t.Setenv("MANDALORE_BINDING", "")
	p, err := Prepare(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeNative{plan: p}
	if _, err := apply(context.Background(), p, f.run, noProbe); err != nil {
		t.Fatal(err)
	}
	before := treeSnapshot(t, filepath.Dir(p.Binding))
	r := doctor(context.Background(), Profile{StateDir: p.StateDir, NativeHome: p.NativeHome, NativeBinary: p.NativeBinary}, f.run)
	if !r.Healthy {
		t.Fatal(r)
	}
	untested := map[string]bool{}
	for _, c := range r.Checks {
		if c.Status == "not-tested" {
			untested[c.Name] = true
		}
	}
	for _, key := range []string{"native-login", "hook-trust", "live-mcp", "remote-freshness", "active-context"} {
		if !untested[key] {
			t.Fatal("boundary not disclosed", key)
		}
	}
	if !reflect.DeepEqual(before, treeSnapshot(t, filepath.Dir(p.Binding))) {
		t.Fatal("doctor wrote state")
	}
	t.Setenv("MANDALORE_BINDING", "/synthetic/another-binding.json")
	if r := doctor(context.Background(), Profile{StateDir: p.StateDir, NativeHome: p.NativeHome, NativeBinary: p.NativeBinary}, f.run); r.Healthy {
		t.Fatal("conflicting override ignored")
	}
}

func TestRepairPreviewRefusesEditsAndMissingSourceButRecoversMissingCache(t *testing.T) {
	for _, kind := range []string{"missing-cache", "edited-cache", "edited-binding", "missing-runtime", "missing-both-runtimes"} {
		t.Run(kind, func(t *testing.T) {
			p, err := Prepare(fixture(t))
			if err != nil {
				t.Fatal(err)
			}
			f := &fakeNative{plan: p}
			if _, err := apply(context.Background(), p, f.run, noProbe); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "missing-cache":
				if err := os.Remove(filepath.Join(cacheRoot(p), "hooks/hooks.json")); err != nil {
					t.Fatal(err)
				}
			case "edited-cache":
				if err := os.WriteFile(filepath.Join(cacheRoot(p), "hooks/hooks.json"), []byte("preserve edit"), 0600); err != nil {
					t.Fatal(err)
				}
			case "edited-binding":
				b, err := os.ReadFile(p.Binding)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p.Binding, append(b, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing-runtime", "missing-both-runtimes":
				if err := os.Remove(p.Runtime); err != nil {
					t.Fatal(err)
				}
				if kind == "missing-both-runtimes" {
					if err := os.Remove(p.Binary); err != nil {
						t.Fatal(err)
					}
				}
			}
			before := treeSnapshot(t, filepath.Dir(p.Binding))
			next, err := PrepareRepair(RepairInput{Root: p.Root})
			want := kind == "missing-cache" || kind == "missing-runtime"
			if (err == nil) != want {
				t.Fatal(kind, next, err)
			}
			if !reflect.DeepEqual(before, treeSnapshot(t, filepath.Dir(p.Binding))) {
				t.Fatal("repair preview wrote files")
			}
			if !want {
				return
			}
			if next.Root == p.Root || next.Binding != p.Binding || next.SignetID != p.SignetID {
				t.Fatal("repair altered identity or reused damaged generation")
			}
			f.plan = next
			if result, err := apply(context.Background(), next, f.run, noProbe); err != nil || !result.Installed {
				t.Fatal("repair application", result, err)
			}
			if _, err := loadReceipt(p.Root); err != nil {
				t.Fatal("previous generation was removed", err)
			}
		})
	}
}
