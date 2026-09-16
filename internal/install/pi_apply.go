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
	piplugin "github.com/acoz-labs/mandalore/plugins/pi"
)

type PiResult struct {
	Connection           PiPlan `json:"connection"`
	Installed            bool   `json:"installed"`
	AlreadyCurrent       bool   `json:"already_current"`
	RequiresFreshSession bool   `json:"requires_fresh_session"`
	Phase                string `json:"phase"`
	Attempt              string `json:"attempt,omitempty"`
	Uncertain            bool   `json:"native_effects_uncertain"`
	Notice               string `json:"notice"`
}

type piProbe func(context.Context, PiPlan) error

// PiNativeVersion is the exact implemented native contract, not a version range.
const PiNativeVersion = "0.85.1"

func nativePi(ctx context.Context, o Options, args ...string) ([]byte, error) {
	return execute(ctx, o.NativeBinary, filepath.Dir(o.Binding), environment(map[string]string{"PI_CODING_AGENT_DIR": o.NativeHome}), nil, args...)
}

func piNativeVersion(ctx context.Context, o Options, run runner) error {
	raw, err := run(ctx, o, "--version")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(raw)) != PiNativeVersion {
		return errors.New("unsupported Pi native contract; this adapter is verified against Pi 0.85.1")
	}
	return nil
}

func probePiRuntime(ctx context.Context, p PiPlan) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	call := func(args []string, input string, out any) error {
		raw, err := execute(ctx, p.Runtime, filepath.Dir(p.Binding), environment(nil), strings.NewReader(input), args...)
		if err != nil {
			return err
		}
		if decodeNative(raw, out) != nil {
			return errors.New("invalid retained Pi runtime response")
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
		return errors.New("retained Pi runtime has incompatible protocol or machine target")
	}
	var info struct {
		Protocol int           `json:"protocol_version"`
		OK       bool          `json:"ok"`
		Result   piplugin.Info `json:"result"`
	}
	if err := call([]string{"call", "pi_package_inspect"}, "{}", &info); err != nil {
		return err
	}
	if !info.OK || info.Protocol != 1 || info.Result.Name != "mandalore" || info.Result.HarnessProtocol != 1 || info.Result.SHA256 != p.PackageSHA256 || info.Result.Version != p.PackageVersion {
		return errors.New("retained runtime embeds a different Pi package; use the selected runtime to prepare the connection")
	}
	var local struct {
		Protocol int  `json:"protocol_version"`
		OK       bool `json:"ok"`
	}
	if err := call([]string{"call", "memory_context", "--binding", p.Binding, "--binding-sha256", p.BindingSHA256, "--signet-id", p.SignetID, "--harness", "pi", "--read-only"}, "{}", &local); err != nil {
		return err
	}
	if !local.OK || local.Protocol != 1 {
		return errors.New("retained runtime could not open the pinned Pi binding")
	}
	return nil
}

func verifyPiBindingNative(p PiPlan) error {
	if _, err := binding.OpenGuarded(p.Binding, "installation", binding.Guard{SHA256: p.BindingSHA256, SignetID: p.SignetID}); err != nil {
		return err
	}
	got, err := digestLimit(p.NativeBinary, maxNativeBinary)
	if err != nil || got != p.NativeSHA256 {
		return errors.New("selected native Pi executable changed after preview")
	}
	return nil
}

func piLock(state string) (func(), error) {
	if err := realDirectory(state); err != nil {
		return nil, err
	}
	path := filepath.Join(state, ".install-lock")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, errors.New("installation is locked; inspect the owner before any stale-lock recovery")
	}
	st, statErr := f.Stat()
	closeErr := f.Close()
	unlock := func() {
		if now, e := os.Lstat(path); e == nil && st != nil && os.SameFile(st, now) {
			_ = os.Remove(path)
		}
	}
	if statErr != nil || closeErr != nil {
		unlock()
		return nil, errors.New("cannot establish Pi installation lock")
	}
	return unlock, nil
}

func ApplyPi(ctx context.Context, p PiPlan) (PiResult, error) {
	return applyPi(ctx, p, nativePi, probePiRuntime)
}

func applyPi(ctx context.Context, p PiPlan, run runner, probe piProbe) (result PiResult, resultErr error) {
	result = PiResult{Connection: p, RequiresFreshSession: true, Phase: "preflight"}
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
			result.Notice = "Connection incomplete. Inspect native inventory and retained phase receipts before retrying; a new repair preview may be required. No rollback, authentication change or synchronization was attempted."
			if result.Attempt != "" {
				raw, _ := json.MarshalIndent(result, "", "  ")
				_ = writeNew(filepath.Join(result.Attempt, "failed.json"), raw)
			}
		}
	}()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if p.SchemaVersion != 1 || p.Harness != "pi" || p.Root != filepath.Join(p.StateDir, "pi", "connections", piPlanKey(p)) {
		return result, errors.New("invalid Pi installation plan")
	}
	if err := verifyPiBindingNative(p); err != nil {
		return result, err
	}
	current, err := inspectPiSettings(p.NativeHome)
	if err != nil {
		return result, err
	}
	selected, err := piSelectedRegistration(current, p.StateDir)
	if err != nil {
		return result, err
	}
	if selected == p.Root {
		r, err := ownedPi(p.Root, p.StateDir, p.NativeHome, false)
		if err != nil || r.Plan != p {
			return result, errors.New("installed Pi generation differs from this plan")
		}
		if err := piNativeVersion(ctx, p.Options, run); err != nil {
			return result, err
		}
		if err := probe(ctx, p); err != nil {
			return result, err
		}
		result.Installed, result.AlreadyCurrent, result.Phase = true, true, "verified"
		result.Notice = "This exact Pi connection is already registered. Active-session loading and model access are not tested."
		return result, nil
	}
	fresh, err := PreparePi(p.PiOptions)
	if err != nil {
		return result, err
	}
	if fresh != p {
		return result, errors.New("Pi installation inputs or native inventory changed after preview")
	}
	unlock, err := piLock(p.StateDir)
	if err != nil {
		return result, err
	}
	defer unlock()
	fresh, err = PreparePi(p.PiOptions)
	if err != nil || fresh != p {
		return result, errors.New("Pi installation inputs changed while acquiring the lock")
	}
	if err := piNativeVersion(ctx, p.Options, run); err != nil {
		return result, err
	}
	if err := stageRuntime(p.runtimePlan()); err != nil {
		return result, err
	}
	if err := probe(ctx, p); err != nil {
		return result, err
	}
	if err := publishPiBundle(p); err != nil {
		return result, err
	}
	attempts := filepath.Join(p.StateDir, "pi", "attempts")
	if err := realDirectory(attempts); err != nil {
		return result, err
	}
	result.Attempt, err = os.MkdirTemp(attempts, piPlanKey(p)+"-")
	if err != nil {
		return result, err
	}
	if err := record("staged"); err != nil {
		return result, err
	}
	// Native startup can have effects; re-observe settings before changing them.
	before, err := inspectPiSettings(p.NativeHome)
	if err != nil {
		return result, err
	}
	if before.Exists != p.SettingsExists || before.SHA256 != p.SettingsSHA256 {
		return result, errors.New("Pi profile changed during preparation; inspect and preview again")
	}
	if err := verifyPiBindingNative(p); err != nil {
		return result, err
	}
	previousPath := ""
	if selected != "" {
		previousPath = filepath.Join(selected, "package")
	}
	if previousPath != "" {
		if _, err := ownedPi(selected, p.StateDir, p.NativeHome, p.RecoverFrom != ""); err != nil {
			return result, err
		}
		result.Uncertain = true
		if err := record("remove-started"); err != nil {
			return result, err
		}
		if _, err := run(ctx, p.Options, "remove", previousPath); err != nil {
			return result, err
		}
		after, err := inspectPiSettings(p.NativeHome)
		if err != nil {
			return result, err
		}
		root, err := piSelectedRegistration(after, p.StateDir)
		if err != nil || root != "" || !piSettingsPreserved(before, after, previousPath, "") {
			return result, errors.New("native removal changed unexpected Pi state; preserve it for inspection")
		}
		before = after
		result.Uncertain = false
		if err := record("registration-removed"); err != nil {
			return result, err
		}
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	again, err := inspectPiSettings(p.NativeHome)
	if err != nil {
		return result, err
	}
	if again.Exists != before.Exists || again.SHA256 != before.SHA256 {
		return result, errors.New("Pi profile changed before installation; inspect before retrying")
	}
	if err := verifyPiBindingNative(p); err != nil {
		return result, err
	}
	result.Uncertain = true
	if err := record("install-started"); err != nil {
		return result, err
	}
	newPath := filepath.Join(p.Root, "package")
	if _, err := run(ctx, p.Options, "install", newPath); err != nil {
		return result, err
	}
	after, err := inspectPiSettings(p.NativeHome)
	if err != nil {
		return result, err
	}
	root, err := piSelectedRegistration(after, p.StateDir)
	if err != nil || root != p.Root || !piSettingsPreserved(before, after, "", newPath) {
		return result, errors.New("native installation did not preserve the selected Pi profile; inspect before retrying")
	}
	if _, err := ownedPi(p.Root, p.StateDir, p.NativeHome, false); err != nil {
		return result, err
	}
	if err := verifyPiBindingNative(p); err != nil {
		return result, err
	}
	result.Installed, result.Uncertain = true, false
	result.Notice = "Connected for fresh Pi sessions. Existing native tools, authentication and unrelated profile settings are preserved; memory, synchronization and existing-session context were not changed."
	if err := record("verified"); err != nil {
		return result, err
	}
	return result, nil
}
