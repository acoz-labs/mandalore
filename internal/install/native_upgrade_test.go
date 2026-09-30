package install

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func installUpgradeFixture(t *testing.T, harness string, o Options) string {
	t.Helper()
	switch harness {
	case "codex":
		p, err := Prepare(o)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = apply(context.Background(), p, (&fakeNative{plan: p}).run, noProbe); err != nil {
			t.Fatal(err)
		}
		return p.Root
	case "pi":
		p, err := PreparePi(PiOptions{Options: o, ReadOnly: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = applyPi(context.Background(), p, (&fakePi{}).run, noPiProbe); err != nil {
			t.Fatal(err)
		}
		return p.Root
	default:
		p, err := PrepareClaude(ClaudeOptions{Options: o, ReadOnly: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = applyClaude(context.Background(), ClaudeApplyInput{Plan: p}, (&fakeClaude{}).run, noClaudeProbe); err != nil {
			t.Fatal(err)
		}
		return p.Root
	}
}
func writeNativeFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0700); err != nil {
		t.Fatal(err)
	}
}
func compatibleNativeFixture(t *testing.T, h, root string) string {
	t.Helper()
	switch h {
	case "claude-code":
		return "#!/bin/sh\ncase \"$*\" in\n--version) echo '2.9.0 (Claude Code)' ;;\n'plugin --help') echo 'install uninstall list marketplace' ;;\n'plugin marketplace --help') echo 'add remove list' ;;\n*) exit 91 ;;\nesac\n"
	case "pi":
		return "#!/bin/sh\ncase \"$*\" in\n--version) echo 0.99.2 ;;\n--help) echo 'install remove --mode --extension --no-session' ;;\n*) exit 91 ;;\nesac\n"
	default:
		p, err := loadReceipt(root)
		if err != nil {
			t.Fatal(err)
		}
		ms := map[string]any{"marketplaces": []any{map[string]any{"name": "mandalore", "root": root, "marketplaceSource": map[string]any{"sourceType": "local", "source": root}}}}
		ps := map[string]any{"installed": []any{map[string]any{"pluginId": p.Plan.PluginID, "name": "mandalore", "version": p.Plan.Version, "enabled": true, "marketplaceSource": map[string]any{"sourceType": "local", "source": root}}}}
		a, _ := json.Marshal(ms)
		b, _ := json.Marshal(ps)
		return "#!/bin/sh\ncase \"$*\" in\n'plugin marketplace list --json') cat <<'JSON'\n" + string(a) + "\nJSON\n;;\n'plugin list --json') cat <<'JSON'\n" + string(b) + "\nJSON\n;;\n*) exit 91 ;;\nesac\n"
	}
}

func TestNativeUpgradeRevalidatesAllHarnessesWithoutReconnection(t *testing.T) {
	for _, h := range []string{"codex", "pi", "claude-code"} {
		t.Run(h, func(t *testing.T) {
			o := fixture(t)
			o.NativeBinary = filepath.Join(filepath.Dir(o.Binary), "agent")
			writeNativeFixture(t, o.NativeBinary, "#!/bin/sh\nexit 99\n")
			root := installUpgradeFixture(t, h, o)
			selection := ReceiptSelection{Harness: h, Root: root, StateDir: o.StateDir, NativeHome: o.NativeHome, NativeBinary: o.NativeBinary, Binding: o.Binding}
			before, err := InspectLaunch(selection)
			if err != nil {
				t.Fatal(err)
			}
			receiptName := "receipt.json"
			if h == "codex" {
				receiptName = "connection.json"
			}
			receiptBefore, _ := os.ReadFile(filepath.Join(root, receiptName))
			writeNativeFixture(t, o.NativeBinary, compatibleNativeFixture(t, h, root))
			after, err := InspectLaunch(selection)
			if err != nil {
				t.Fatal(err)
			}
			if before.NativeSHA256 == after.NativeSHA256 || before.BindingSHA256 != after.BindingSHA256 || before.ReadOnly != after.ReadOnly || before.SessionTransportVersion != after.SessionTransportVersion {
				t.Fatal("upgrade altered binding/access or was not observed")
			}
			if err := ValidateLaunchNative(context.Background(), h, after); err != nil {
				t.Fatal(err)
			}
			receiptAfter, _ := os.ReadFile(filepath.Join(root, receiptName))
			if string(receiptBefore) != string(receiptAfter) {
				t.Fatal("immutable receipt rewritten")
			}
			writeNativeFixture(t, o.NativeBinary, "#!/bin/sh\necho unsupported\n")
			broken, err := InspectLaunch(selection)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateLaunchNative(context.Background(), h, broken); err == nil {
				t.Fatal("missing native capabilities accepted")
			}
		})
	}
}

func TestLegacyNativeUpdaterLocatorsRecoverRemovedTargets(t *testing.T) {
	for _, h := range []string{"codex", "pi", "claude-code"} {
		t.Run(h, func(t *testing.T) {
			o := fixture(t)
			base := filepath.Dir(o.Binary)
			var original, next, launcher string
			switch h {
			case "claude-code":
				original = filepath.Join(base, "local/share/claude/versions/2.1.278")
				next = filepath.Join(filepath.Dir(original), "2.9.0")
				launcher = filepath.Join(base, "local/bin/claude")
			case "codex":
				original = filepath.Join(base, "standalone/releases/0.159.0-platform/bin/codex")
				next = filepath.Join(base, "standalone/releases/0.160.0-platform/bin/codex")
				launcher = filepath.Join(base, "standalone/current")
			case "pi":
				original = filepath.Join(base, "mise/installs/npm-earendil-works-pi-coding-agent/0.85.1/node_modules/.bin/pi")
				next = filepath.Join(base, "mise/installs/npm-earendil-works-pi-coding-agent/0.99.2/node_modules/.bin/pi")
				launcher = filepath.Join(base, "mise/installs/npm-earendil-works-pi-coding-agent/latest")
			}
			o.NativeBinary = original
			writeNativeFixture(t, original, "#!/bin/sh\nexit 99\n")
			root := installUpgradeFixture(t, h, o)
			pinnedSelection := ReceiptSelection{Harness: h, Root: root, StateDir: o.StateDir, NativeHome: o.NativeHome, NativeBinary: original, Binding: o.Binding}
			if c, err := InspectLaunch(pinnedSelection); err != nil || c.Executable != original {
				t.Fatal("manually pinned executable requires nonexistent updater", c, err)
			}
			writeNativeFixture(t, next, compatibleNativeFixture(t, h, root))
			if err := os.MkdirAll(filepath.Dir(launcher), 0700); err != nil {
				t.Fatal(err)
			}
			linkTarget := next
			if h == "codex" {
				linkTarget = filepath.Dir(filepath.Dir(next))
			}
			if h == "pi" {
				linkTarget = filepath.Dir(filepath.Dir(filepath.Dir(next)))
			}
			if err := os.Symlink(linkTarget, launcher); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(original); err != nil {
				t.Fatal(err)
			}
			s := ReceiptSelection{Harness: h, Root: root, StateDir: o.StateDir, NativeHome: o.NativeHome, NativeBinary: original, Binding: o.Binding}
			c, err := InspectLaunch(s)
			if err != nil {
				t.Fatal(err)
			}
			if c.Executable != next {
				t.Fatalf("did not follow same-installation locator: %s", c.Executable)
			}
			if err := ValidateLaunchNative(context.Background(), h, c); err != nil {
				t.Fatal(err)
			}
			profile := Profile{StateDir: o.StateDir, NativeHome: o.NativeHome, NativeBinary: original}
			switch h {
			case "codex":
				if r := Doctor(context.Background(), profile); !r.Healthy {
					t.Fatal("doctor failed legacy recovery", r)
				}
			case "pi":
				if r := DoctorPi(context.Background(), profile); !r.Healthy {
					t.Fatal("doctor failed legacy recovery", r)
				}
			default:
				if r := DoctorClaude(context.Background(), profile); !r.Healthy {
					t.Fatal("doctor failed legacy recovery", r)
				}
			}
			var repairOptions Options
			switch h {
			case "codex":
				p, err := PrepareRepair(RepairInput{Root: root})
				if err != nil {
					t.Fatal("repair failed legacy recovery", err)
				}
				repairOptions = p.Options
			case "pi":
				p, err := PreparePiRepair(RepairInput{Root: root})
				if err != nil {
					t.Fatal("repair failed legacy recovery", err)
				}
				repairOptions = p.Options
			default:
				p, err := PrepareClaudeRepair(RepairInput{Root: root})
				if err != nil {
					t.Fatal("repair failed legacy recovery", err)
				}
				repairOptions = p.Options
			}
			if repairOptions.NativeBinary != next || repairOptions.NativeLauncher == "" {
				t.Fatal("repair lost updater locator", repairOptions)
			}
			// Redirecting a legacy locator must fail even if its stale original still exists.
			writeNativeFixture(t, original, compatibleNativeFixture(t, h, root))
			if err := os.Remove(launcher); err != nil {
				t.Fatal(err)
			}
			foreign := filepath.Join(base, "foreign")
			writeNativeFixture(t, foreign, "#!/bin/sh\nexit 0\n")
			if err := os.Symlink(foreign, launcher); err != nil {
				t.Fatal(err)
			}
			if _, err := InspectLaunch(s); err == nil {
				t.Fatal("foreign legacy locator accepted")
			}
		})
	}
}

func TestNativeLauncherPreservedAndApplyRaceRefused(t *testing.T) {
	for _, h := range []string{"codex", "pi", "claude-code"} {
		t.Run(h, func(t *testing.T) {
			o := fixture(t)
			base := filepath.Dir(o.Binary)
			target := filepath.Join(base, "versions/one/agent")
			next := filepath.Join(base, "versions/two/agent")
			launcher := filepath.Join(base, "current")
			writeNativeFixture(t, target, "#!/bin/sh\nexit 99\n")
			writeNativeFixture(t, next, "#!/bin/sh\nexit 99\n")
			if err := os.Symlink(filepath.Dir(target), launcher); err != nil {
				t.Fatal(err)
			}
			o.NativeBinary = filepath.Join(launcher, "agent")
			var options Options
			var digest string
			var applyChanged func() error
			switch h {
			case "codex":
				p, err := Prepare(o)
				if err != nil {
					t.Fatal(err)
				}
				options, digest = p.Options, p.NativeSHA256
				applyChanged = func() error {
					_, err := apply(context.Background(), p, (&fakeNative{plan: p}).run, noProbe)
					return err
				}
			case "pi":
				p, err := PreparePi(PiOptions{Options: o})
				if err != nil {
					t.Fatal(err)
				}
				options, digest = p.Options, p.NativeSHA256
				applyChanged = func() error { _, err := applyPi(context.Background(), p, (&fakePi{}).run, noPiProbe); return err }
			default:
				p, err := PrepareClaude(ClaudeOptions{Options: o})
				if err != nil {
					t.Fatal(err)
				}
				options, digest = p.Options, p.NativeSHA256
				applyChanged = func() error {
					_, err := applyClaude(context.Background(), ClaudeApplyInput{Plan: p}, (&fakeClaude{}).run, noClaudeProbe)
					return err
				}
			}
			if options.NativeLauncher != o.NativeBinary || options.NativeBinary != target {
				t.Fatal("stable path lost", options)
			}

			if err := os.Remove(launcher); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Dir(next), launcher); err != nil {
				t.Fatal(err)
			}
			if err := verifyNativePreview(options, digest); err == nil {
				t.Fatal("retargeting between preview/apply accepted")
			}
			if err := applyChanged(); err == nil {
				t.Fatal("apply accepted changed launcher")
			}
		})
	}
}

func TestUntestedVersionsDoNotRequireInstallerCapabilitiesToLaunch(t *testing.T) {
	for _, h := range []string{"pi", "claude-code"} {
		t.Run(h, func(t *testing.T) {
			check := piNativeVersion
			version := "0.99.2-beta.1+build.7"
			if h == "claude-code" {
				check = claudeNativeVersion
				version = "2.2.0-beta.1+build.7 (Claude Code)"
			}
			run := func(_ context.Context, _ Options, args ...string) ([]byte, error) {
				if args[0] != "--version" {
					t.Fatal("version check required unrelated capability", args)
				}
				return []byte(version), nil
			}
			if err := check(context.Background(), Options{}, run); err != nil {
				t.Fatal("valid untested version rejected", err)
			}
		})
	}
}

func TestInstallerCapabilitiesAreScopedToRequiredMutationSurfaces(t *testing.T) {
	run := func(_ context.Context, _ Options, args ...string) ([]byte, error) {
		if args[0] == "--help" {
			return []byte("install remove"), nil
		}
		return []byte("--scope --json --keep-data"), nil
	}
	for _, h := range []string{"pi", "claude-code"} {
		if err := nativeInstallCapabilities(context.Background(), h, Options{}, run, true); err != nil {
			t.Fatal("unrelated flags incorrectly required", h, err)
		}
	}
	missing := func(ctx context.Context, o Options, args ...string) ([]byte, error) {
		raw, err := run(ctx, o, args...)
		return []byte(strings.ReplaceAll(string(raw), "--keep-data", "")), err
	}
	if err := nativeInstallCapabilities(context.Background(), "claude-code", Options{}, missing, false); err != nil {
		t.Fatal("fresh install required uninstall-only flag", err)
	}
	if err := nativeInstallCapabilities(context.Background(), "claude-code", Options{}, missing, true); err == nil || !strings.Contains(err.Error(), "--keep-data") {
		t.Fatal("missing data-preserving uninstall flag accepted", err)
	}
}

func TestMissingClaudeInstallerCapabilityRefusesBeforeNativeMutation(t *testing.T) {
	o := fixture(t)
	p, err := PrepareClaude(ClaudeOptions{Options: o})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeClaude{}
	if _, err := applyClaude(context.Background(), ClaudeApplyInput{Plan: p}, f.run, noClaudeProbe); err != nil {
		t.Fatal(err)
	}
	o.Generation = "next"
	next, err := PrepareClaude(ClaudeOptions{Options: o})
	if err != nil {
		t.Fatal(err)
	}
	before, err := inspectClaudeSettings(o.NativeHome)
	if err != nil {
		t.Fatal(err)
	}
	run := func(ctx context.Context, o Options, args ...string) ([]byte, error) {
		if args[len(args)-1] == "--help" {
			return []byte("--scope --json"), nil
		}
		if args[0] != "--version" {
			t.Fatal("native mutation occurred before capability rejection", args)
		}
		return f.run(ctx, o, args...)
	}
	if _, err := applyClaude(context.Background(), ClaudeApplyInput{Plan: next, SessionsStopped: true}, run, noClaudeProbe); err == nil || !strings.Contains(err.Error(), "--keep-data") {
		t.Fatal("missing installer capability accepted", err)
	}
	after, err := inspectClaudeSettings(o.NativeHome)
	if err != nil || after.SHA256 != before.SHA256 {
		t.Fatal("capability failure changed native settings", err)
	}
	// An existing connection needs no uninstall command to launch successfully.
	o.NativeBinary = p.NativeBinary
	writeNativeFixture(t, o.NativeBinary, "#!/bin/sh\ncase \"$*\" in --version) echo '2.2.0-beta.1+build.7 (Claude Code)' ;; *) exit 92 ;; esac\n")
	selected := ReceiptSelection{Harness: "claude-code", Root: p.Root, StateDir: o.StateDir, NativeHome: o.NativeHome, NativeBinary: o.NativeBinary, Binding: o.Binding}
	c, err := InspectLaunch(selected)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateLaunchNative(context.Background(), "claude-code", c); err != nil {
		t.Fatal("installer-only incompatibility blocked intact launch", err)
	}
}

func TestDoctorRefusesNativeMutationDuringCapabilityInspection(t *testing.T) {
	for _, h := range []string{"codex", "pi", "claude-code"} {
		t.Run(h, func(t *testing.T) {
			o := fixture(t)
			o.NativeBinary = filepath.Join(filepath.Dir(o.Binary), "agent")
			writeNativeFixture(t, o.NativeBinary, "#!/bin/sh\nexit 99\n")
			root := installUpgradeFixture(t, h, o)
			script := compatibleNativeFixture(t, h, root)
			script = strings.Replace(script, "#!/bin/sh\n", "#!/bin/sh\nprintf '#changed\\n' >> \"$0\"\n", 1)
			writeNativeFixture(t, o.NativeBinary, script)
			profile := Profile{StateDir: o.StateDir, NativeHome: o.NativeHome, NativeBinary: o.NativeBinary}
			var healthy bool
			var checks []Check
			switch h {
			case "codex":
				r := Doctor(context.Background(), profile)
				healthy, checks = r.Healthy, r.Checks
			case "pi":
				r := DoctorPi(context.Background(), profile)
				healthy, checks = r.Healthy, r.Checks
			default:
				r := DoctorClaude(context.Background(), profile)
				healthy, checks = r.Healthy, r.Checks
			}
			failedStability := false
			for _, c := range checks {
				if c.Name == "native-stability" && c.Status == "fail" {
					failedStability = true
				}
			}
			if healthy || !failedStability {
				t.Fatal("changed native reported stable", checks)
			}
		})
	}
}

func TestSavedStableLauncherSurvivesRetargetAndRemovedOriginal(t *testing.T) {
	for _, h := range []string{"codex", "pi", "claude-code"} {
		t.Run(h, func(t *testing.T) {
			o := fixture(t)
			base := filepath.Dir(o.Binary)
			old := filepath.Join(base, "native-v1/agent")
			next := filepath.Join(base, "native-v2/agent")
			launcher := filepath.Join(base, "my-agent")
			writeNativeFixture(t, old, "#!/bin/sh\nexit 99\n")
			if err := os.Symlink(old, launcher); err != nil {
				t.Fatal(err)
			}
			o.NativeBinary = launcher
			root := installUpgradeFixture(t, h, o)
			selection := ReceiptSelection{Harness: h, Root: root, StateDir: o.StateDir, NativeHome: o.NativeHome, NativeBinary: launcher, Binding: o.Binding}
			before, err := InspectLaunch(selection)
			if err != nil {
				t.Fatal(err)
			}
			if before.NativeLauncher != launcher || before.Executable != old {
				t.Fatal("explicit stable launcher not saved", before)
			}
			writeNativeFixture(t, next, compatibleNativeFixture(t, h, root))
			if err := os.Remove(launcher); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(next, launcher); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(old); err != nil {
				t.Fatal(err)
			}
			after, err := InspectLaunch(selection)
			if err != nil {
				t.Fatal(err)
			}
			if after.Executable != next || after.BindingSHA256 != before.BindingSHA256 || after.Options != before.Options {
				t.Fatal("upgrade modified immutable connection identity", after)
			}
			if err := ValidateLaunchNative(context.Background(), h, after); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOlderRuntimeLauncherFailureHasUpdateGuidance(t *testing.T) {
	o := fixture(t)
	target := o.NativeBinary
	o.NativeBinary = filepath.Join(filepath.Dir(target), "native-link")
	if err := os.Symlink(target, o.NativeBinary); err != nil {
		t.Fatal(err)
	}
	// The fixture runtime refuses all calls, representing an unsupported retained schema.
	for _, h := range []string{"codex", "pi", "claude-code"} {
		t.Run(h, func(t *testing.T) {
			var err error
			switch h {
			case "codex":
				_, err = PrepareViaRuntime(context.Background(), o)
			case "pi":
				_, err = PreparePiViaRuntime(context.Background(), PiOptions{Options: o})
			default:
				_, err = PrepareClaudeViaRuntime(context.Background(), ClaudeOptions{Options: o})
			}
			if err == nil || !strings.Contains(err.Error(), "select a current Mandalore runtime") {
				t.Fatal("older runtime boundary lacks actionable update guidance", err)
			}
		})
	}
}
