package install

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/binding"
	hermesplugin "github.com/acoz-labs/mandalore/plugins/hermes"
)

type HermesResult struct {
	Connection           HermesPlan `json:"connection"`
	Installed            bool       `json:"installed"`
	AlreadyCurrent       bool       `json:"already_current"`
	RequiresFreshSession bool       `json:"requires_fresh_session"`
	Phase                string     `json:"phase"`
	Attempt              string     `json:"attempt,omitempty"`
	Uncertain            bool       `json:"native_effects_uncertain"`
	Notice               string     `json:"notice"`
}

type hermesProbe func(context.Context, HermesPlan) error

// HermesNativeVersion records a tested baseline, not an exclusive supported version.
const HermesNativeVersion = "0.21.5"

func nativeHermes(ctx context.Context, o Options, args ...string) ([]byte, error) {
	return execute(ctx, o.NativeBinary, filepath.Dir(o.Binding), environment(map[string]string{"HERMES_HOME": o.NativeHome, "PYTHONDONTWRITEBYTECODE": "1"}), nil, args...)
}

func hermesNativeVersion(ctx context.Context, o Options, run runner) error {
	raw, err := run(ctx, o, "--version")
	if err != nil {
		return err
	}
	if !strings.Contains(string(raw), "Hermes") {
		return errors.New("incompatible Hermes version response; select a working Hermes installation")
	}
	return nil
}

func probeHermesRuntime(ctx context.Context, p HermesPlan) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	call := func(args []string, input string, out any) error {
		raw, err := execute(ctx, p.Runtime, filepath.Dir(p.Binding), environment(nil), strings.NewReader(input), args...)
		if err != nil {
			return err
		}
		if decodeNative(raw, out) != nil {
			return errors.New("invalid retained Hermes runtime response")
		}
		return nil
	}
	var version struct {
		Protocol int  `json:"protocol_version"`
		OK       bool `json:"ok"`
		Result   struct {
			Name     string `json:"name"`
			Protocol int    `json:"protocol_version"`
			OS       string `json:"os"`
			Arch     string `json:"arch"`
		} `json:"result"`
	}
	if err := call([]string{"version"}, "", &version); err != nil {
		return err
	}
	if !version.OK || version.Protocol != 1 || version.Result.Name != "mandalore" || version.Result.Protocol != 1 || version.Result.OS != runtime.GOOS || version.Result.Arch != runtime.GOARCH {
		return errors.New("retained Hermes runtime has incompatible protocol or machine target")
	}
	var info struct {
		Protocol int               `json:"protocol_version"`
		OK       bool              `json:"ok"`
		Result   hermesplugin.Info `json:"result"`
	}
	if err := call([]string{"call", "hermes_package_inspect"}, "{}", &info); err != nil {
		return err
	}
	if !info.OK || info.Protocol != 1 || info.Result.Name != "mandalore" || info.Result.HarnessProtocol != 1 || info.Result.SHA256 != p.PackageSHA256 || info.Result.Version != p.PackageVersion {
		return errors.New("retained runtime embeds a different Hermes package; use the selected runtime to prepare the connection")
	}
	var local struct {
		Protocol int  `json:"protocol_version"`
		OK       bool `json:"ok"`
	}
	if err := call([]string{"call", "memory_context", "--binding", p.Binding, "--binding-sha256", p.BindingSHA256, "--signet-id", p.SignetID, "--harness", "hermes", "--read-only"}, "{}", &local); err != nil {
		return err
	}
	if !local.OK || local.Protocol != 1 {
		return errors.New("retained runtime could not open the pinned Hermes binding")
	}
	return nil
}

func verifyHermesBindingNative(p HermesPlan) error {
	if _, err := binding.OpenGuarded(p.Binding, "installation", binding.Guard{SHA256: p.BindingSHA256, SignetID: p.SignetID}); err != nil {
		return err
	}
	if err := verifyNativePreview(p.Options, p.NativeSHA256); err != nil {
		return err
	}
	return nil
}

func ApplyHermes(ctx context.Context, p HermesPlan) (HermesResult, error) {
	return applyHermes(ctx, p, nativeHermes, probeHermesRuntime)
}

func applyHermes(ctx context.Context, p HermesPlan, run runner, probe hermesProbe) (result HermesResult, resultErr error) {
	result = HermesResult{Connection: p, RequiresFreshSession: true, Phase: "preflight"}
	record := func(phase string) error {
		result.Phase = phase
		if result.Attempt == "" {
			return nil
		}
		raw, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		return writeNew(filepath.Join(result.Attempt, phase+".json"), raw)
	}
	defer func() {
		if resultErr != nil {
			result.Notice = "Connection incomplete. Inspect native selection and retained phase receipts before retrying. Retained generations were preserved; no automatic rollback or synchronization was attempted."
			if result.Attempt != "" {
				raw, _ := json.MarshalIndent(result, "", "  ")
				_ = writeNew(filepath.Join(result.Attempt, "failed.json"), raw)
			}
		}
	}()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if p.SchemaVersion != 1 || p.Harness != "hermes" || p.Root != filepath.Join(p.StateDir, "hermes", "connections", hermesPlanKey(p)) {
		return result, errors.New("invalid Hermes installation plan")
	}
	if err := verifyHermesBindingNative(p); err != nil {
		return result, err
	}
	current, err := inspectHermesSettings(p.NativeHome)
	if err != nil {
		return result, err
	}
	if current.Root == p.Root && current.Enabled {
		r, err := ownedHermes(p.Root, p.StateDir, p.NativeHome, false)
		if err != nil || r.Plan != p {
			return result, errors.New("installed Hermes generation differs from plan")
		}
		if err := probe(ctx, p); err != nil {
			return result, err
		}
		result.Installed, result.AlreadyCurrent, result.Phase = true, true, "verified"
		result.Notice = "This exact Hermes connection is already registered. Active-session loading and model access are not tested."
		return result, nil
	}
	fresh, err := PrepareHermes(p.HermesOptions)
	if err != nil {
		return result, err
	}
	if fresh != p {
		return result, errors.New("Hermes installation inputs changed after preview")
	}
	unlock, err := piLock(p.StateDir) // Shared installation lock serializes all harnesses.
	if err != nil {
		return result, err
	}
	defer unlock()
	fresh, err = PrepareHermes(p.HermesOptions)
	if err != nil || fresh != p {
		return result, errors.New("Hermes installation inputs changed while acquiring lock")
	}
	if err := hermesNativeVersion(ctx, p.Options, run); err != nil {
		return result, err
	}
	for _, verb := range []string{"enable", "disable"} {
		raw, err := run(ctx, p.Options, "plugins", verb, "--help")
		if err != nil || !strings.Contains(string(raw), verb) {
			return result, errors.New("Hermes native plugin activation capability is unavailable")
		}
	}
	if err := stageRuntime(p.runtimePlan()); err != nil {
		return result, err
	}
	if err := probe(ctx, p); err != nil {
		return result, err
	}
	if err := publishHermesBundle(p); err != nil {
		return result, err
	}
	attempts := filepath.Join(p.StateDir, "hermes", "attempts")
	if err := realDirectory(attempts); err != nil {
		return result, err
	}
	result.Attempt, err = os.MkdirTemp(attempts, hermesPlanKey(p)+"-")
	if err != nil {
		return result, err
	}
	if err := record("staged"); err != nil {
		return result, err
	}
	before, err := inspectHermesSettings(p.NativeHome)
	if err != nil {
		return result, err
	}
	recoveringAbsent := before.Root == "" && p.RecoverFrom != "" && p.RecoverFrom == p.PreviousRoot
	if before.Exists != p.SettingsExists || before.SHA256 != p.SettingsSHA256 || (before.Root != p.PreviousRoot && !recoveringAbsent) {
		return result, errors.New("Hermes profile changed during preparation")
	}
	if p.PreviousRoot != "" {
		if _, err := ownedHermes(p.PreviousRoot, p.StateDir, p.NativeHome, p.RecoverFrom != ""); err != nil {
			return result, err
		}
	}
	if err := verifyHermesBindingNative(p); err != nil {
		return result, err
	}
	plugins := filepath.Join(p.NativeHome, "plugins")
	if err := realDirectory(plugins); err != nil {
		return result, err
	}
	result.Uncertain = true
	if err := record("registration-started"); err != nil {
		return result, err
	}
	// Discovery supports a local directory symlink. Retain immutable generations
	// outside the profile and replace only our owned registration atomically.
	link := filepath.Join(plugins, "mandalore")
	temporary, err := os.MkdirTemp(plugins, ".mandalore-")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(temporary)
	stagedLink := filepath.Join(temporary, "__link__") // Native discovery skips dunder entries.
	if err := os.Symlink(filepath.Join(p.Root, "package"), stagedLink); err != nil {
		return result, err
	}
	again, err := inspectHermesSettings(p.NativeHome)
	if err != nil || again.Root != before.Root || again.SHA256 != before.SHA256 {
		return result, errors.New("Hermes selection changed before activation")
	}
	if err := os.Rename(stagedLink, link); err != nil {
		return result, err
	}
	if err := record("registered"); err != nil {
		return result, err
	}
	if _, err := run(ctx, p.Options, "plugins", "enable", "mandalore"); err != nil {
		return result, err
	}
	after, err := inspectHermesSettings(p.NativeHome)
	if err != nil {
		return result, err
	}
	if after.Root != p.Root || !after.Enabled || !hermesSettingsPreserved(before, after) {
		return result, errors.New("Hermes native activation changed unexpected settings or did not enable Mandalore; inspect before retrying")
	}
	if _, err := ownedHermes(p.Root, p.StateDir, p.NativeHome, false); err != nil {
		return result, err
	}
	result.Installed, result.Uncertain = true, false
	result.Notice = "Connected for fresh Hermes sessions. Native memory and authentication remain configured independently. Restart long-running Hermes services to load this generation. Live model tools and synchronization were not tested."
	if err := record("verified"); err != nil {
		return result, err
	}
	return result, nil
}
