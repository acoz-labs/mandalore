package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPiPlanPinsInputsWithoutExecutionOrWrites(t *testing.T) {
	o := PiOptions{Options: fixture(t), ReadOnly: true}
	p, err := PreparePi(o)
	if err != nil {
		t.Fatal(err)
	}
	again, err := PreparePi(o)
	if err != nil || !reflect.DeepEqual(p, again) {
		t.Fatal("unstable preview", err)
	}
	if p.Harness != "pi" || p.SchemaVersion != 1 || p.SignetID == "" || !p.ReadOnly || len(p.PackageSHA256) != 64 || p.PackageVersion == "" {
		t.Fatal(p)
	}
	if _, err := os.Stat(o.StateDir); !os.IsNotExist(err) {
		t.Fatal("preview wrote installation state")
	}
	if _, err := os.Stat(o.NativeHome); !os.IsNotExist(err) {
		t.Fatal("preview wrote native profile")
	}
	if err := os.Mkdir(o.NativeHome, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(o.NativeHome, "settings.json"), []byte(`{"theme":"light","packages":["npm:unrelated"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	changed, err := PreparePi(o)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Root == p.Root || !changed.SettingsExists || changed.SettingsSHA256 == "" {
		t.Fatal("profile identity not pinned")
	}
	o.ReadOnly = false
	writable, err := PreparePi(o)
	if err != nil || writable.Root == changed.Root {
		t.Fatal("read-only choice not pinned", err)
	}
}

func TestPiPlanRefusesForeignOrFilteredMemoryPackages(t *testing.T) {
	for _, kind := range []string{"local", "remote", "filtered", "legacy"} {
		t.Run(kind, func(t *testing.T) {
			o := PiOptions{Options: fixture(t)}
			if err := os.Mkdir(o.NativeHome, 0700); err != nil {
				t.Fatal(err)
			}
			pkg := filepath.Join(filepath.Dir(o.Binding), "foreign")
			if err := os.Mkdir(pkg, 0700); err != nil {
				t.Fatal(err)
			}
			name := "mandalore"
			if kind == "legacy" {
				name = "my-friday-memory"
			}
			manifest, _ := json.Marshal(map[string]string{"name": name, "version": "1.0.0"})
			if err := os.WriteFile(filepath.Join(pkg, "package.json"), manifest, 0600); err != nil {
				t.Fatal(err)
			}
			var entry any = pkg
			if kind == "remote" {
				entry = "npm:mandalore@1.0.0"
			}
			if kind == "filtered" {
				entry = map[string]any{"source": pkg, "extensions": []string{}}
			}
			raw, _ := json.Marshal(map[string]any{"packages": []any{entry}})
			if err := os.WriteFile(filepath.Join(o.NativeHome, "settings.json"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := PreparePi(o); err == nil {
				t.Fatal("foreign memory registration accepted")
			}
		})
	}
}

func TestPiPlanRejectsOverlapAndManagedRedirection(t *testing.T) {
	for _, kind := range []string{"profile", "bank", "binding", "redirected"} {
		t.Run(kind, func(t *testing.T) {
			o := PiOptions{Options: fixture(t)}
			switch kind {
			case "profile":
				o.StateDir = o.NativeHome
			case "bank":
				o.NativeHome = filepath.Join(filepath.Dir(o.Binding), "signet")
			case "binding":
				o.StateDir = filepath.Dir(o.Binding)
			case "redirected":
				if err := os.Mkdir(o.StateDir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(o.NativeHome, filepath.Join(o.StateDir, "pi")); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := PreparePi(o); err == nil {
				t.Fatal("unsafe Pi paths accepted")
			}
		})
	}
}
