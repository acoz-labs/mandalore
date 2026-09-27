package install

import (
	"context"
	"errors"
	"os"

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
}

func InspectLaunch(s ReceiptSelection) (LaunchConnection, error) {
	var c LaunchConnection
	for _, p := range []string{s.Root, s.StateDir, s.NativeHome, s.NativeBinary, s.Binding} {
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
		if e = verifyPiBindingNative(p); e != nil {
			return c, e
		}
		c = LaunchConnection{p.Options, p.Root, p.SignetID, p.BindingSHA256, p.Runtime, p.ReadOnly}
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
		if e = verifyClaudeBindingNative(p); e != nil {
			return c, e
		}
		c = LaunchConnection{p.Options, p.Root, p.SignetID, p.BindingSHA256, p.Runtime, p.ReadOnly}
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
		if d, e := digestLimit(s.NativeBinary, maxNativeBinary); e != nil || d != p.NativeSHA256 {
			return c, errors.New("native executable changed")
		}
		c = LaunchConnection{p.Options, p.Root, p.SignetID, p.BindingSHA256, p.Runtime, false}
	default:
		return c, errors.New("unsupported agent; choose codex, pi or claude-code")
	}
	if c.StateDir != s.StateDir || c.NativeHome != s.NativeHome || c.NativeBinary != s.NativeBinary || c.Binding != s.Binding {
		return LaunchConnection{}, errors.New("connection belongs to a different launch selection")
	}
	if _, e := binding.OpenGuarded(c.Binding, s.Harness, binding.Guard{SHA256: c.BindingSHA256, SignetID: c.SignetID}); e != nil {
		return LaunchConnection{}, e
	}
	if st, e := os.Stat(c.NativeBinary); e != nil || st.Mode()&0111 == 0 {
		return LaunchConnection{}, errors.New("selected native executable is unavailable or not executable")
	}
	return c, nil
}

func ValidateLaunchNative(ctx context.Context, harness string, c LaunchConnection) error {
	p := Profile{StateDir: c.StateDir, NativeHome: c.NativeHome, NativeBinary: c.NativeBinary}
	// Native checks inherit profile selection through existing scoped adapters.
	switch harness {
	case "pi":
		r := DoctorPi(ctx, p)
		if r.Healthy && r.Connection != nil && r.Connection.Root == c.Root {
			return nil
		}
	case "claude-code":
		r := DoctorClaude(ctx, p)
		if r.Healthy && r.Connection != nil && r.Connection.Root == c.Root {
			return nil
		}
	case "codex":
		r, e := loadReceipt(c.Root)
		if e != nil {
			break
		}
		ms, ps, e := inventory(ctx, c.Options, native)
		if e != nil {
			break
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
