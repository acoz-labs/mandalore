package install

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type fakeClaude struct {
	calls        []string
	fail         string
	interference bool
}

func (f *fakeClaude) run(_ context.Context, o Options, args ...string) ([]byte, error) {
	command := strings.Join(args, " ")
	f.calls = append(f.calls, command)
	if args[0] == "--version" {
		return []byte(ClaudeNativeVersion + " (Claude Code)\n"), nil
	}
	if strings.Contains(command, f.fail) && f.fail != "" {
		return nil, errors.New("synthetic interruption")
	}
	if !strings.Contains(command, "--scope user") {
		return nil, errors.New("unscoped native mutation")
	}
	s, err := inspectClaudeSettings(o.NativeHome)
	if err != nil {
		return nil, err
	}
	get := func(name string) map[string]any {
		v, _ := s.Other[name].(map[string]any)
		if v == nil {
			v = map[string]any{}
			s.Other[name] = v
		}
		return v
	}
	settings := get("settings.json")
	known := get("plugins/known_marketplaces.json")
	installed := get("plugins/installed_plugins.json")
	enabled, _ := settings["enabledPlugins"].(map[string]any)
	if enabled == nil {
		enabled = map[string]any{}
	}
	extra, _ := settings["extraKnownMarketplaces"].(map[string]any)
	if extra == nil {
		extra = map[string]any{}
	}
	plugins, _ := installed["plugins"].(map[string]any)
	if plugins == nil {
		plugins = map[string]any{}
	}
	if args[1] == "marketplace" {
		if args[2] == "add" {
			entry := map[string]any{"source": map[string]any{"source": "directory", "path": args[3]}, "installLocation": args[3]}
			known["mandalore"] = entry
			extra["mandalore"] = map[string]any{"source": entry["source"]}
		} else {
			delete(known, "mandalore")
			delete(extra, "mandalore")
		}
	} else if args[1] == "uninstall" {
		delete(plugins, "mandalore@mandalore")
		delete(enabled, "mandalore@mandalore")
	} else if args[1] == "install" {
		r, e := loadClaudeReceipt(s.Root)
		if e != nil {
			return nil, e
		}
		p := r.Plan
		cache := filepath.Join(o.NativeHome, "plugins", "cache", "mandalore", "mandalore", p.cacheVersion())
		for name := range r.Files {
			if !strings.HasPrefix(name, "package/") {
				continue
			}
			raw, e := os.ReadFile(filepath.Join(p.Root, name))
			if e != nil {
				return nil, e
			}
			target := filepath.Join(cache, strings.TrimPrefix(name, "package/"))
			if e := os.MkdirAll(filepath.Dir(target), 0700); e != nil {
				return nil, e
			}
			if e := os.WriteFile(target, raw, 0600); e != nil {
				return nil, e
			}
		}
		plugins["mandalore@mandalore"] = []any{map[string]any{"scope": "user", "version": p.nativeVersion(), "installPath": cache}}
		enabled["mandalore@mandalore"] = true
	}
	settings["enabledPlugins"] = enabled
	settings["extraKnownMarketplaces"] = extra
	installed["version"] = 2
	installed["plugins"] = plugins
	if f.interference {
		settings["theme"] = "unexpected"
	}
	for name, value := range s.Other {
		raw, _ := json.Marshal(value)
		path := filepath.Join(o.NativeHome, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, raw, 0600); err != nil {
			return nil, err
		}
	}
	return []byte(`{"outcome":"ok"}`), nil
}
func noClaudeProbe(context.Context, ClaudePlan) error { return nil }
func TestClaudeInstallUpdateDeferralAndRepair(t *testing.T) {
	o := ClaudeOptions{Options: fixture(t), ReadOnly: true}
	p, err := PrepareClaude(o)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(o.StateDir); !os.IsNotExist(err) {
		t.Fatal("preview wrote state")
	}
	f := &fakeClaude{}
	r, err := applyClaude(context.Background(), ClaudeApplyInput{Plan: p}, f.run, noClaudeProbe)
	if err != nil || !r.Installed {
		t.Fatal(r, err)
	}
	f.calls = nil
	r, err = applyClaude(context.Background(), ClaudeApplyInput{Plan: p}, f.run, noClaudeProbe)
	if err != nil || !r.AlreadyCurrent || len(f.calls) != 1 {
		t.Fatal("idempotence", r, err, f.calls)
	}
	o.Generation = "update"
	next, err := PrepareClaude(o)
	if err != nil {
		t.Fatal(err)
	}
	f.calls = nil
	r, err = applyClaude(context.Background(), ClaudeApplyInput{Plan: next}, f.run, noClaudeProbe)
	if err != nil || r.Phase != "deferred" || len(f.calls) != 0 {
		t.Fatal("missing stopped-session gate", r, err)
	}
	r, err = applyClaude(context.Background(), ClaudeApplyInput{Plan: next, SessionsStopped: true}, f.run, noClaudeProbe)
	if err != nil || !r.Installed {
		t.Fatal("update", r, err)
	}
	if _, err := ownedClaude(p.Root, p.StateDir, p.NativeHome, false); err != nil {
		t.Fatal("old generation not retained", err)
	}
	repair, err := PrepareClaudeRepair(RepairInput{Root: next.Root})
	if err != nil || repair.Root == next.Root || !repair.ReadOnly {
		t.Fatal("repair", repair, err)
	}
}
func TestClaudeRejectsChangedBindingForeignRegistrationAndInventory(t *testing.T) {
	for _, kind := range []string{"binding", "foreign", "malformed", "interference", "interrupted"} {
		t.Run(kind, func(t *testing.T) {
			o := ClaudeOptions{Options: fixture(t)}
			p, err := PrepareClaude(o)
			if err != nil {
				t.Fatal(err)
			}
			f := &fakeClaude{}
			switch kind {
			case "binding":
				raw, _ := os.ReadFile(o.Binding)
				if err := os.WriteFile(o.Binding, append(raw, ' '), 0600); err != nil {
					t.Fatal(err)
				}
			case "foreign", "malformed":
				if err := os.MkdirAll(o.NativeHome, 0700); err != nil {
					t.Fatal(err)
				}
				raw := `{"enabledPlugins":{"mandalore@other":true}}`
				if kind == "malformed" {
					raw = `{"enabledPlugins":[]}`
				}
				if err := os.WriteFile(filepath.Join(o.NativeHome, "settings.json"), []byte(raw), 0600); err != nil {
					t.Fatal(err)
				}
			case "interference":
				f.interference = true
			case "interrupted":
				f.fail = "plugin install"
			}
			r, err := applyClaude(context.Background(), ClaudeApplyInput{Plan: p}, f.run, noClaudeProbe)
			if err == nil || r.Installed {
				t.Fatal("unsafe input accepted", r, err)
			}
			if kind == "interrupted" && (!r.Uncertain || r.Attempt == "") {
				t.Fatal("lost partial receipt", r)
			}
		})
	}
}

func TestClaudeRefusesEditedCacheAndRecoversMissingCache(t *testing.T) {
	for _, mode := range []string{"edited", "missing"} {
		t.Run(mode, func(t *testing.T) {
			o := ClaudeOptions{Options: fixture(t)}
			p, err := PrepareClaude(o)
			if err != nil {
				t.Fatal(err)
			}
			f := &fakeClaude{}
			if _, err := applyClaude(context.Background(), ClaudeApplyInput{Plan: p}, f.run, noClaudeProbe); err != nil {
				t.Fatal(err)
			}
			cache := filepath.Join(o.NativeHome, "plugins", "cache", "mandalore", "mandalore", p.cacheVersion())
			if mode == "edited" {
				if err := os.WriteFile(filepath.Join(cache, ".mcp.json"), []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.RemoveAll(cache); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := applyClaude(context.Background(), ClaudeApplyInput{Plan: p}, f.run, noClaudeProbe); err == nil {
				t.Fatal("invalid cache called current")
			}
			repair, err := PrepareClaudeRepair(RepairInput{Root: p.Root})
			if mode == "edited" {
				if err == nil {
					t.Fatal("repair accepted edited cache")
				}
				return
			}
			if err != nil {
				t.Fatal("missing cache repair blocked", err)
			}
			r, err := applyClaude(context.Background(), ClaudeApplyInput{Plan: repair, SessionsStopped: true}, f.run, noClaudeProbe)
			if err != nil || !r.Installed {
				t.Fatal("recovery failed", r, err)
			}
		})
	}
}

func TestClaudeReceiptMetadataUsesActualOwnedBundle(t *testing.T) {
	o := ClaudeOptions{Options: fixture(t)}
	p, err := PrepareClaude(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := publishClaudeBundle(p); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(p.Root, "receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	selected := ReceiptSelection{Harness: "claude-code", Root: p.Root, StateDir: p.StateDir, NativeHome: p.NativeHome, NativeBinary: p.NativeBinary, Binding: p.Binding}
	metadata, err := ProjectReceiptMetadata(selected, raw)
	if err != nil || metadata.PackageSHA256 != p.PackageSHA256 || metadata.BindingSHA256 != p.BindingSHA256 {
		t.Fatal(metadata, err)
	}
	selected.Binding = filepath.Join(filepath.Dir(p.Binding), "other.json")
	if _, err := ProjectReceiptMetadata(selected, raw); err == nil {
		t.Fatal("receipt retarget accepted")
	}
}

func TestClaudeBridgeFailsClosedAfterRuntimeEdit(t *testing.T) {
	o := ClaudeOptions{Options: fixture(t)}
	if err := os.WriteFile(o.Binary, []byte("#!/bin/sh\nprintf '%s\\n' '{\"hookSpecificOutput\":{\"hookEventName\":\"SessionStart\",\"additionalContext\":\"Synthetic bridge\"}}'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	p, err := PrepareClaude(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := stageRuntime(p.runtimePlan()); err != nil {
		t.Fatal(err)
	}
	if err := publishClaudeBundle(p); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(p.Root, "package", "scripts", "run-memory.sh")
	out, err := exec.Command("/bin/sh", script, "hook").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "Synthetic bridge") {
		t.Fatal(string(out), err)
	}
	if err := os.WriteFile(p.Runtime, []byte("#!/bin/sh\nprintf COMPROMISED\n"), 0700); err != nil {
		t.Fatal(err)
	}
	out, err = exec.Command("/bin/sh", script, "hook").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "integrity check failed") || strings.Contains(string(out), "COMPROMISED") {
		t.Fatal(string(out), err)
	}
	if out, err := exec.Command("/bin/sh", script, "mcp").CombinedOutput(); err == nil || strings.Contains(string(out), "COMPROMISED") {
		t.Fatal("MCP executed edited runtime", string(out), err)
	}
}
