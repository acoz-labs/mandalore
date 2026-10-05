package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type fakeHermes struct {
	fail, interference bool
	calls              []string
}

func (f *fakeHermes) run(_ context.Context, o Options, args ...string) ([]byte, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	if args[0] == "--version" {
		return []byte("Hermes Agent v0.21.5"), nil
	}
	if args[len(args)-1] == "--help" {
		return []byte("plugins enable disable"), nil
	}
	if f.fail {
		return nil, errors.New("synthetic interruption")
	}
	s, err := inspectHermesSettings(o.NativeHome)
	if err != nil {
		return nil, err
	}
	plugins, _ := s.Other["plugins"].(map[string]any)
	if plugins == nil {
		plugins = map[string]any{}
		s.Other["plugins"] = plugins
	}
	names, _ := plugins["enabled"].([]string)
	plugins["enabled"] = append(names, "mandalore")
	if f.interference {
		s.Other["memory"] = map[string]any{"provider": "unexpected"}
	}
	raw, err := yaml.Marshal(s.Other)
	if err != nil {
		return nil, err
	}
	return nil, os.WriteFile(filepath.Join(o.NativeHome, "config.yaml"), raw, 0600)
}

func noHermesProbe(context.Context, HermesPlan) error { return nil }

func TestHermesNativeCommandsPinRootAgainstStickyProfile(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "hermes")
	script := `#!/bin/sh
selected="$HERMES_HOME"
case "$HERMES_HOME" in
  */profiles/*) ;;
  *)
    if [ "$1" = "--profile" ] && [ "$2" = "default" ]; then
      shift 2
    elif [ -f "$HERMES_HOME/active_profile" ]; then
      selected="$HERMES_HOME/profiles/$(cat "$HERMES_HOME/active_profile")"
    fi ;;
esac
printf '%s\n' "$selected"
`
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	for _, home := range []string{filepath.Join(root, "custom"), filepath.Join(root, "custom", "profiles", "selected")} {
		if err := os.MkdirAll(home, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, "active_profile"), []byte("other"), 0600); err != nil {
			t.Fatal(err)
		}
		o := Options{NativeHome: home, NativeBinary: binary, Binding: filepath.Join(root, "binding.json")}
		raw, err := nativeHermes(context.Background(), o, "plugins", "enable", "mandalore")
		if err != nil || strings.TrimSpace(string(raw)) != home {
			t.Fatalf("native command redirected selected profile: %s %v", raw, err)
		}
	}
}

func TestHermesPreviewPinsProfileAndNeverExecutes(t *testing.T) {
	o := HermesOptions{Options: fixture(t), ReadOnly: true}
	p, err := PrepareHermes(o)
	if err != nil {
		t.Fatal(err)
	}
	again, err := PrepareHermes(o)
	if err != nil || p != again {
		t.Fatal("unstable plan", err)
	}
	for _, path := range []string{o.StateDir, o.NativeHome} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("preview wrote state")
		}
	}
	if err := os.Mkdir(o.NativeHome, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(o.NativeHome, "config.yaml"), []byte("memory:\n  provider: local\n"), 0600); err != nil {
		t.Fatal(err)
	}
	next, err := PrepareHermes(o)
	if err != nil || next.Root == p.Root {
		t.Fatal("profile bytes not pinned", err)
	}
	o.SessionTransportVersion = 1
	if _, err := PrepareHermes(o); err == nil {
		t.Fatal("read-only granted transport")
	}
}

func TestHermesInstallUpdateAndRetainedRecovery(t *testing.T) {
	o := HermesOptions{Options: fixture(t)}
	o.SessionTransportVersion = 1
	f := &fakeHermes{}
	p, err := PrepareHermes(o)
	if err != nil {
		t.Fatal(err)
	}
	r, err := applyHermes(context.Background(), p, f.run, noHermesProbe)
	if err != nil || !r.Installed || r.Uncertain || r.Phase != "verified" {
		t.Fatal(r, err)
	}
	f.calls = nil
	r, err = applyHermes(context.Background(), p, f.run, noHermesProbe)
	if err != nil || !r.AlreadyCurrent || len(f.calls) != 0 {
		t.Fatal("idempotent application changed native state", r, err, f.calls)
	}
	o.Generation = "next"
	next, err := PrepareHermes(o)
	if err != nil {
		t.Fatal(err)
	}
	f.fail = true
	r, err = applyHermes(context.Background(), next, f.run, noHermesProbe)
	if err == nil || !r.Uncertain || r.Phase != "registered" {
		t.Fatal("uncertain activation hidden", r, err)
	}
	for _, root := range []string{p.Root, next.Root} {
		if _, err := ownedHermes(root, p.StateDir, p.NativeHome, false); err != nil {
			t.Fatal("retained generation lost", err)
		}
	}
	if _, err := os.Stat(filepath.Join(r.Attempt, "failed.json")); err != nil {
		t.Fatal("failure receipt missing", err)
	}
	// Restore the old generation through a reviewed repair, not a receipt rewrite.
	if err := os.Remove(filepath.Join(p.NativeHome, "plugins", "mandalore")); err != nil {
		t.Fatal(err)
	}
	repair, err := PrepareHermesRepair(RepairInput{Root: p.Root, NativeBinary: p.NativeBinary})
	if err != nil {
		t.Fatal(err)
	}
	if repair.BindingSHA256 != p.BindingSHA256 || repair.SessionTransportVersion != 1 {
		t.Fatal("recovery changed authority")
	}
	f.fail = false
	r, err = applyHermes(context.Background(), repair, f.run, noHermesProbe)
	if err != nil || !r.Installed {
		t.Fatal("repair failed", r, err)
	}
}

func TestHermesPreservesNativeMemoryAndUnrelatedPlugins(t *testing.T) {
	o := HermesOptions{Options: fixture(t)}
	if err := os.Mkdir(o.NativeHome, 0700); err != nil {
		t.Fatal(err)
	}
	config := []byte("model:\n  default: example-model\nmemory:\n  provider: local\nplugins:\n  enabled: [unrelated]\n  disabled: [mandalore, another]\n")
	if err := os.WriteFile(filepath.Join(o.NativeHome, "config.yaml"), config, 0600); err != nil {
		t.Fatal(err)
	}
	p, err := PrepareHermes(o)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeHermes{}
	if _, err := applyHermes(context.Background(), p, f.run, noHermesProbe); err != nil {
		t.Fatal(err)
	}
	s, err := inspectHermesSettings(o.NativeHome)
	if err != nil || !s.Enabled || s.Other["memory"].(map[string]any)["provider"] != "local" {
		t.Fatal("native memory changed", s, err)
	}
	if err := os.WriteFile(filepath.Join(o.NativeHome, "config.yaml"), config, 0600); err != nil {
		t.Fatal(err)
	}
	report := doctorHermes(context.Background(), Profile{StateDir: o.StateDir, NativeHome: o.NativeHome, NativeBinary: o.NativeBinary}, f.run)
	if report.Healthy {
		t.Fatal("disabled plugin reported healthy")
	}
}

func TestHermesRejectsForeignEditedAndStaleState(t *testing.T) {
	for _, kind := range []string{"foreign", "alias", "stale", "interference", "edited"} {
		t.Run(kind, func(t *testing.T) {
			o := HermesOptions{Options: fixture(t)}
			p, err := PrepareHermes(o)
			if err != nil {
				t.Fatal(err)
			}
			f := &fakeHermes{}
			switch kind {
			case "alias":
				path := filepath.Join(o.NativeHome, "plugins", "category", "other")
				if err := os.MkdirAll(path, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(path, "plugin.yaml"), []byte("name: mandalore\nversion: 1.0.0\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "foreign":
				if err := os.MkdirAll(filepath.Join(o.NativeHome, "plugins", "mandalore"), 0700); err != nil {
					t.Fatal(err)
				}
			case "stale":
				if err := os.Mkdir(o.NativeHome, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(o.NativeHome, "config.yaml"), []byte("theme: changed\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "interference":
				f.interference = true
			case "edited":
				if _, err := applyHermes(context.Background(), p, f.run, noHermesProbe); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(p.Root, "package", "__init__.py"), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := applyHermes(context.Background(), p, f.run, noHermesProbe); err == nil {
				t.Fatal("unsafe state accepted")
			}
		})
	}
}
