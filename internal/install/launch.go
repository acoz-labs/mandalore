package install

import (
	"context"
	"errors"
	"fmt"

	"github.com/acoz-labs/mandalore/internal/binding"
)

// LaunchConnection validates a selected retained connection without running native
// code, installing, synchronizing, or changing authorization. Codex's native
// registration is additionally checked by ValidateLaunchNative at actual launch.
type LaunchConnection struct {
	Options
	Root          string `json:"connection_root"`
	SignetID      string `json:"signet_id"`
	BindingSHA256 string `json:"binding_sha256"`
	Runtime       string `json:"runtime"`
	ReadOnly      bool   `json:"read_only"`
	Executable    string `json:"executable"`
	NativeSHA256  string `json:"native_sha256"`
	NativeLocator string `json:"native_locator"`
}

func InspectLaunch(s ReceiptSelection) (LaunchConnection, error) {
	var c LaunchConnection
	for _, p := range []string{s.Root, s.StateDir, s.NativeHome, s.Binding} {
		actual, err := canonical(p)
		if err != nil || actual != p {
			return c, errors.New("launch paths must be explicit canonical absolute paths")
		}
	}
	switch s.Harness {
	case "pi":
		r, e := ownedPi(s.Root, s.StateDir, s.NativeHome, false)
		if e != nil {
			return c, e
		}
		p := r.Plan
		settings, e := inspectPiSettings(s.NativeHome)
		if e != nil {
			return c, e
		}
		root, e := piSelectedRegistration(settings, s.StateDir)
		if e != nil || root != s.Root {
			return c, errors.New("Pi registration differs from selected connection")
		}
		c = LaunchConnection{p.Options, p.Root, p.SignetID, p.BindingSHA256, p.Runtime, p.ReadOnly, "", "", ""}
	case "claude-code":
		r, e := ownedClaude(s.Root, s.StateDir, s.NativeHome, false)
		if e != nil {
			return c, e
		}
		p := r.Plan
		settings, e := inspectClaudeSettings(s.NativeHome)
		if e != nil {
			return c, e
		}
		root, e := claudeSelectedRegistration(settings, s.StateDir)
		if e != nil || root != s.Root || !settings.Enabled {
			return c, errors.New("Claude registration is missing, disabled or differs from selection")
		}
		if e = verifyClaudeCache(settings, p, false); e != nil {
			return c, e
		}
		c = LaunchConnection{p.Options, p.Root, p.SignetID, p.BindingSHA256, p.Runtime, p.ReadOnly, "", "", ""}
	case "codex":
		r, e := loadReceipt(s.Root)
		if e != nil {
			return c, e
		}
		p := r.Plan
		if _, e = owned(s.Root, Plan{Options: Options{StateDir: s.StateDir, NativeHome: s.NativeHome}}, false); e != nil {
			return c, e
		}
		if e = verifyCache(p, false); e != nil {
			return c, e
		}
		c = LaunchConnection{p.Options, p.Root, p.SignetID, p.BindingSHA256, p.Runtime, false, "", "", ""}
	default:
		return c, errors.New("unsupported agent; choose codex, pi or claude-code")
	}
	if c.StateDir != s.StateDir || c.NativeHome != s.NativeHome || (c.NativeBinary != s.NativeBinary && c.NativeLauncher != s.NativeBinary) || c.Binding != s.Binding {
		return LaunchConnection{}, errors.New("connection belongs to a different launch selection")
	}
	if _, e := binding.OpenGuarded(c.Binding, s.Harness, binding.Guard{SHA256: c.BindingSHA256, SignetID: c.SignetID}); e != nil {
		return LaunchConnection{}, e
	}
	var err error
	c.Executable, c.NativeSHA256, c.NativeLocator, err = nativeSnapshot(s.Harness, c.Options)
	if err != nil {
		return LaunchConnection{}, err
	}
	return c, nil
}

func ValidateLaunchNative(ctx context.Context, harness string, c LaunchConnection) (resultErr error) {
	defer func() {
		if resultErr == nil {
			resultErr = verifyNativeObservation(harness, c.Options, c.Executable, c.NativeSHA256, c.NativeLocator)
		}
	}()
	target, digest, locator, err := nativeSnapshot(harness, c.Options)
	if err != nil {
		return err
	}
	if target != c.Executable || digest != c.NativeSHA256 || locator != c.NativeLocator {
		return errors.New("native executable changed during launch; retry to validate the current installation")
	}
	p := Profile{StateDir: c.StateDir, NativeHome: c.NativeHome, NativeBinary: c.Executable}
	// Native checks inherit profile selection through existing scoped adapters.
	switch harness {
	case "pi":
		r := DoctorPi(ctx, p)
		if r.Healthy && r.Connection != nil && r.Connection.Root == c.Root {
			return nil
		}
		return launchCheckFailure(r.Checks)
	case "claude-code":
		r := DoctorClaude(ctx, p)
		if r.Healthy && r.Connection != nil && r.Connection.Root == c.Root {
			return nil
		}
		return launchCheckFailure(r.Checks)
	case "codex":
		r, e := loadReceipt(c.Root)
		if e != nil {
			break
		}
		ms, ps, e := inventory(ctx, Options{NativeHome: c.NativeHome, NativeBinary: c.Executable, Binding: c.Binding}, native)
		if e != nil {
			return fmt.Errorf("Codex plugin inventory capability failed: %w", e)
		}
		markets, plugins := 0, 0
		for _, m := range ms {
			if m.Name == "mandalore" {
				if m.Root != c.Root || m.Source.Path != c.Root || m.Source.Type != "local" {
					return errors.New("Codex marketplace differs from launch selection")
				}
				markets++
			}
		}
		for _, v := range ps {
			if v.ID == r.Plan.PluginID && v.Enabled && v.Version == r.Plan.Version && v.Source.Type == "local" && v.Source.Path == c.Root {
				plugins++
			}
			if v.Enabled && (v.Name == "my-friday-memory" || v.Name == "my-friday" || v.Name == "mandalore" && v.ID != r.Plan.PluginID) {
				return errors.New("another native memory integration is active")
			}
		}
		if markets == 1 && plugins == 1 {
			return nil
		}
	}
	if ctx.Err() != nil {
		return errors.New("launch interrupted before native start; no configuration was changed")
	}
	return errors.New("native connection validation failed; inspect with mandalore connection armorer using the selected profile, then explicitly reconnect if needed")
}

func launchCheckFailure(checks []Check) error {
	for _, check := range checks {
		if check.Status == "fail" {
			return fmt.Errorf("native connection check %s failed: %s", check.Name, check.Detail)
		}
	}
	return errors.New("native registration differs from selected connection")
}
