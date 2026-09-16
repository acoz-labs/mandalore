package install

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeNative struct {
	plan               Plan
	root, version      string
	pluginRoot         string
	clearCacheOnRemove bool
	enabled            bool
	fail               string
	calls              []string
}

func encoded(v any) []byte { b, _ := json.Marshal(v); return b }

func (f *fakeNative) run(_ context.Context, o Options, args ...string) ([]byte, error) {
	command := strings.Join(args, " ")
	f.calls = append(f.calls, command)
	// Real Codex requires CODEX_HOME before even read-only plugin inventory.
	if st, err := os.Stat(o.NativeHome); err != nil || !st.IsDir() {
		return nil, errors.New("selected native home must exist before inventory")
	}
	if f.fail != "" && strings.HasPrefix(command, f.fail) {
		return nil, errors.New("synthetic native failure")
	}
	switch {
	case command == "plugin marketplace list --json":
		items := []any{}
		if f.root != "" {
			items = append(items, map[string]any{"name": "mandalore", "root": f.root, "marketplaceSource": map[string]string{"sourceType": "local", "source": f.root}})
		}
		return encoded(map[string]any{"marketplaces": items}), nil
	case command == "plugin list --json":
		items := []any{}
		if f.enabled {
			items = append(items, map[string]any{"pluginId": "mandalore@mandalore", "name": "mandalore", "version": f.version, "enabled": true, "marketplaceSource": map[string]string{"sourceType": "local", "source": f.pluginRoot}})
		}
		return encoded(map[string]any{"installed": items}), nil
	case command == "plugin marketplace remove mandalore --json":
		if f.clearCacheOnRemove && f.version != "" {
			// This is only this fake's generated cache under its disposable profile.
			if err := os.RemoveAll(filepath.Join(f.plan.NativeHome, "plugins/cache/mandalore/mandalore", f.version)); err != nil {
				return nil, err
			}
		}
		f.root = ""
		return []byte(`{}`), nil
	case len(args) == 4 && args[0] == "plugin" && args[1] == "marketplace" && args[2] == "add":
		f.root = args[3]
		return []byte(`{}`), nil
	case command == "plugin add mandalore@mandalore --json":
		files, _, err := bundle(f.plan)
		if err != nil {
			return nil, err
		}
		for name, data := range files {
			if !strings.HasPrefix(name, "plugins/mandalore/") {
				continue
			}
			path := filepath.Join(cacheRoot(f.plan), strings.TrimPrefix(name, "plugins/mandalore/"))
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				return nil, err
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				return nil, err
			}
		}
		f.version, f.enabled = f.plan.Version, true
		f.pluginRoot = f.plan.Root
		return []byte(`{}`), nil
	}
	return nil, errors.New("unexpected native command")
}

func noProbe(context.Context, Plan) error { return nil }

func TestApplyPreparesNativeHomeBeforeInventory(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "existing"}[existing], func(t *testing.T) {
			o := fixture(t)
			sentinel := filepath.Join(o.NativeHome, "unrelated.txt")
			wantMode := os.FileMode(0700)
			if existing {
				if err := os.Mkdir(o.NativeHome, 0700); err != nil {
					t.Fatal(err)
				}
				wantMode = 0750
				if err := os.Chmod(o.NativeHome, wantMode); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(sentinel, []byte("preserve me"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			p, err := Prepare(o)
			if err != nil {
				t.Fatal(err)
			}
			if !existing {
				if _, err := os.Lstat(o.NativeHome); !os.IsNotExist(err) {
					t.Fatal("preview created native home", err)
				}
			}
			f := &fakeNative{plan: p}
			result, err := apply(context.Background(), p, f.run, noProbe)
			if err != nil || !result.Installed {
				t.Fatal(result, err)
			}
			st, err := os.Stat(o.NativeHome)
			if err != nil || st.Mode().Perm() != wantMode {
				t.Fatal("native home permissions changed", st, err)
			}
			if existing {
				data, err := os.ReadFile(sentinel)
				if err != nil || string(data) != "preserve me" {
					t.Fatal("unrelated profile file changed", err)
				}
			}
		})
	}
}

func TestApplyRefusesChangedNativeHomeBeforeNativeCommands(t *testing.T) {
	for _, kind := range []string{"file", "redirected", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			p, err := Prepare(fixture(t))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "file":
				if err := os.WriteFile(p.NativeHome, []byte("preserve me"), 0600); err != nil {
					t.Fatal(err)
				}
			case "redirected":
				if err := os.Symlink(filepath.Dir(p.Binding), p.NativeHome); err != nil {
					t.Fatal(err)
				}
			case "cancelled":
				cancel()
			}
			f := &fakeNative{plan: p}
			result, err := apply(ctx, p, f.run, noProbe)
			if err == nil || result.Installed || len(f.calls) != 0 {
				t.Fatal("unsafe native execution", result, err, f.calls)
			}
			if kind == "cancelled" {
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if _, err := os.Lstat(p.NativeHome); !os.IsNotExist(err) {
					t.Fatal("cancelled apply created home", err)
				}
			}
		})
	}
}

func TestInventoryErrorsNameFailedOperation(t *testing.T) {
	for _, command := range []string{"plugin marketplace list --json", "plugin list --json"} {
		t.Run(command, func(t *testing.T) {
			run := func(_ context.Context, _ Options, args ...string) ([]byte, error) {
				if strings.Join(args, " ") == command {
					return []byte("private subprocess output"), context.Canceled
				}
				return []byte(`{"marketplaces":[]}`), nil
			}
			_, _, err := inventory(context.Background(), Options{}, run)
			if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), command) || strings.Contains(err.Error(), "private subprocess output") {
				t.Fatal("missing bounded operation context or cancellation identity", err)
			}
		})
	}
}

func TestApplyNativeSuccessIdempotenceAndPartialRetry(t *testing.T) {
	p, err := Prepare(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeNative{plan: p, clearCacheOnRemove: true}
	result, err := apply(context.Background(), p, f.run, noProbe)
	if err != nil || !result.Installed || !result.RequiresFreshSession {
		t.Fatal(result, err)
	}
	if err := verifyCache(p, false); err != nil {
		t.Fatal(err)
	}
	if _, err := apply(context.Background(), p, f.run, noProbe); err != nil {
		t.Fatal("idempotent apply", err)
	}
	oldRoot := p.Root
	o := p.Options
	o.Generation = "refresh-1"
	next, err := Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	f.plan, f.fail = next, "plugin marketplace add"
	partial, err := apply(context.Background(), next, f.run, noProbe)
	if err == nil || partial.Installed || partial.PreviousRoot != oldRoot || partial.Phase != "registration-removed" {
		t.Fatal("partial state not reported", partial, err)
	}
	if _, err := loadReceipt(oldRoot); err != nil {
		t.Fatal("old generation lost", err)
	}
	if _, err := loadReceipt(next.Root); err != nil {
		t.Fatal("new generation lost", err)
	}
	f.fail = ""
	if result, err := apply(context.Background(), next, f.run, noProbe); err != nil || !result.Installed {
		t.Fatal("partial retry", result, err)
	}
}

func TestApplyRefusesForeignEditedAndChangedState(t *testing.T) {
	for _, kind := range []string{"foreign-marketplace", "edited-bundle", "edited-cache", "changed-input", "locked", "bad-runtime"} {
		t.Run(kind, func(t *testing.T) {
			p, err := Prepare(fixture(t))
			if err != nil {
				t.Fatal(err)
			}
			f := &fakeNative{plan: p}
			probe := noProbe
			switch kind {
			case "foreign-marketplace":
				f.root = filepath.Join(filepath.Dir(p.Binding), "unowned")
			case "edited-bundle", "edited-cache":
				if _, err := apply(context.Background(), p, f.run, noProbe); err != nil {
					t.Fatal(err)
				}
				file := filepath.Join(p.Root, "plugins/mandalore/hooks/hooks.json")
				if kind == "edited-cache" {
					file = filepath.Join(cacheRoot(p), "hooks/hooks.json")
				}
				if err := os.WriteFile(file, []byte("preserve user changes"), 0600); err != nil {
					t.Fatal(err)
				}
			case "changed-input":
				if err := os.WriteFile(p.Binary, []byte("changed"), 0700); err != nil {
					t.Fatal(err)
				}
			case "locked":
				if err := os.MkdirAll(p.StateDir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(p.StateDir, ".install-lock"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "bad-runtime":
				probe = func(context.Context, Plan) error { return errors.New("incompatible runtime") }
			}
			f.calls = nil
			if result, err := apply(context.Background(), p, f.run, probe); err == nil || result.Installed {
				t.Fatal("unsafe activation accepted", result)
			}
			for _, call := range f.calls {
				if call != "plugin marketplace list --json" && call != "plugin list --json" {
					t.Fatal("native mutation before refusal", call)
				}
			}
		})
	}
}
