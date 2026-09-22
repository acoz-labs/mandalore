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
	claudeplugin "github.com/acoz-labs/mandalore/plugins/claude-code"
)

type ClaudeResult struct {
	Connection           ClaudePlan `json:"connection"`
	Installed            bool       `json:"installed"`
	AlreadyCurrent       bool       `json:"already_current"`
	RequiresFreshSession bool       `json:"requires_fresh_session"`
	Phase                string     `json:"phase"`
	Attempt              string     `json:"attempt,omitempty"`
	Uncertain            bool       `json:"native_effects_uncertain"`
	Notice               string     `json:"notice"`
}

type claudeProbe func(context.Context, ClaudePlan) error

// ClaudeNativeVersion is the exact implemented native contract, not a version range.
const ClaudeNativeVersion = "2.1.278"

func nativeClaude(ctx context.Context, o Options, args ...string) ([]byte, error) {
	return execute(ctx, o.NativeBinary, filepath.Dir(o.Binding), environment(map[string]string{"CLAUDE_CONFIG_DIR": o.NativeHome}), nil, args...)
}

func claudeNativeVersion(ctx context.Context, o Options, run runner) error {
	raw, err := run(ctx, o, "--version")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(raw)) != ClaudeNativeVersion+" (Claude Code)" {
		return errors.New("unsupported Claude native contract; this adapter is verified against Claude 2.1.278")
	}
	return nil
}

func probeClaudeRuntime(ctx context.Context, p ClaudePlan) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	call := func(args []string, input string, out any) error {
		raw, err := execute(ctx, p.Runtime, filepath.Dir(p.Binding), environment(nil), strings.NewReader(input), args...)
		if err != nil {
			return err
		}
		if decodeNative(raw, out) != nil {
			return errors.New("invalid retained Claude runtime response")
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
		return errors.New("retained Claude runtime has incompatible protocol or machine target")
	}
	var info struct {
		Protocol int               `json:"protocol_version"`
		OK       bool              `json:"ok"`
		Result   claudeplugin.Info `json:"result"`
	}
	if err := call([]string{"call", "claude_code_package_inspect"}, "{}", &info); err != nil {
		return err
	}
	if !info.OK || info.Protocol != 1 || info.Result.Name != "mandalore" || info.Result.HarnessProtocol != 1 || info.Result.SHA256 != p.PackageSHA256 || info.Result.Version != p.PackageVersion {
		return errors.New("retained runtime embeds a different Claude package; use the selected runtime to prepare the connection")
	}
	var local struct {
		Protocol int  `json:"protocol_version"`
		OK       bool `json:"ok"`
	}
	if err := call([]string{"call", "memory_context", "--binding", p.Binding, "--binding-sha256", p.BindingSHA256, "--signet-id", p.SignetID, "--harness", "claude-code", "--read-only"}, "{}", &local); err != nil {
		return err
	}
	if !local.OK || local.Protocol != 1 {
		return errors.New("retained runtime could not open the pinned Claude binding")
	}
	return nil
}

func verifyClaudeBindingNative(p ClaudePlan) error {
	if _, err := binding.OpenGuarded(p.Binding, "installation", binding.Guard{SHA256: p.BindingSHA256, SignetID: p.SignetID}); err != nil {
		return err
	}
	got, err := digestLimit(p.NativeBinary, maxNativeBinary)
	if err != nil || got != p.NativeSHA256 {
		return errors.New("selected native Claude executable changed after preview")
	}
	return nil
}

func claudeLock(state string) (func(), error) {
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
		return nil, errors.New("cannot establish Claude installation lock")
	}
	return unlock, nil
}

type ClaudeApplyInput struct {
	Plan            ClaudePlan `json:"plan"`
	SessionsStopped bool       `json:"sessions_stopped,omitempty"`
}

func ApplyClaude(ctx context.Context, in ClaudeApplyInput) (ClaudeResult, error) {
	return applyClaude(ctx, in, nativeClaude, probeClaudeRuntime)
}

func applyClaude(ctx context.Context, in ClaudeApplyInput, run runner, probe claudeProbe) (result ClaudeResult, resultErr error) {
	p := in.Plan
	result = ClaudeResult{Connection: p, RequiresFreshSession: true, Phase: "preflight"}
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
	if p.SchemaVersion != 1 || p.Harness != "claude-code" || p.Root != filepath.Join(p.StateDir, "claude-code", "connections", claudePlanKey(p)) {
		return result, errors.New("invalid Claude installation plan")
	}
	if err := verifyClaudeBindingNative(p); err != nil {
		return result, err
	}
	current, err := inspectClaudeSettings(p.NativeHome)
	if err != nil {
		return result, err
	}
	selected, err := claudeSelectedRegistration(current, p.StateDir)
	if err != nil {
		return result, err
	}
	if selected == p.Root {
		if !current.Enabled {
			return result, errors.New("Claude memory plugin is disabled; explicitly preview reconnect or repair to re-enable it")
		}
		r, err := ownedClaude(p.Root, p.StateDir, p.NativeHome, false)
		if err != nil || r.Plan != p {
			return result, errors.New("installed Claude generation differs from this plan")
		}
		if err := verifyClaudeCache(current, p, false); err != nil {
			return result, err
		}
		if err := claudeNativeVersion(ctx, p.Options, run); err != nil {
			return result, err
		}
		if err := probe(ctx, p); err != nil {
			return result, err
		}
		result.Installed, result.AlreadyCurrent, result.Phase = true, true, "verified"
		result.Notice = "This exact Claude connection is already registered. Active-session loading and model access are not tested."
		return result, nil
	}
	if p.PreviousRoot != "" && !in.SessionsStopped {
		result.Phase = "deferred"
		result.Notice = "Exit all Claude Code sessions using this profile before replacing the connection, then apply with sessions_stopped=true. No changes were made."
		return result, nil
	}
	fresh, err := PrepareClaude(p.ClaudeOptions)
	if err != nil {
		return result, err
	}
	if fresh != p {
		return result, errors.New("Claude installation inputs or native inventory changed after preview")
	}
	targetCache := filepath.Join(p.NativeHome, "plugins", "cache", "mandalore", "mandalore", p.cacheVersion())
	if real, err := canonical(targetCache); err != nil || real != targetCache {
		return result, errors.New("Claude target cache is redirected")
	}
	if _, err := os.Lstat(targetCache); err == nil {
		return result, errors.New("Claude target cache already exists without this active registration; preserve it before recovery")
	} else if !os.IsNotExist(err) {
		return result, err
	}
	unlock, err := claudeLock(p.StateDir)
	if err != nil {
		return result, err
	}
	defer unlock()
	fresh, err = PrepareClaude(p.ClaudeOptions)
	if err != nil || fresh != p {
		return result, errors.New("Claude installation inputs changed while acquiring the lock")
	}
	if err := claudeNativeVersion(ctx, p.Options, run); err != nil {
		return result, err
	}
	if err := stageRuntime(p.runtimePlan()); err != nil {
		return result, err
	}
	if err := probe(ctx, p); err != nil {
		return result, err
	}
	if err := publishClaudeBundle(p); err != nil {
		return result, err
	}
	attempts := filepath.Join(p.StateDir, "claude-code", "attempts")
	if err := realDirectory(attempts); err != nil {
		return result, err
	}
	result.Attempt, err = os.MkdirTemp(attempts, claudePlanKey(p)+"-")
	if err != nil {
		return result, err
	}
	if err := record("staged"); err != nil {
		return result, err
	}
	// Native startup can have effects; re-observe settings before changing them.
	before, err := inspectClaudeSettings(p.NativeHome)
	if err != nil {
		return result, err
	}
	if before.Exists != p.SettingsExists || before.SHA256 != p.SettingsSHA256 {
		return result, errors.New("Claude profile changed during preparation; inspect and preview again")
	}
	if err := verifyClaudeBindingNative(p); err != nil {
		return result, err
	}
	previousPath := ""
	if selected != "" {
		previousPath = filepath.Join(selected, "package")
	}
	if previousPath != "" {
		if _, err := ownedClaude(selected, p.StateDir, p.NativeHome, p.RecoverFrom != ""); err != nil {
			return result, err
		}
		result.Uncertain = true
		if err := record("remove-started"); err != nil {
			return result, err
		}
		if before.Cache != "" {
			prior, err := loadClaudeReceipt(selected)
			if err != nil {
				return result, err
			}
			if err := verifyClaudeCache(before, prior.Plan, p.RecoverFrom != ""); err != nil {
				return result, err
			}
			if _, err := run(ctx, p.Options, "plugin", "uninstall", "mandalore@mandalore", "--scope", "user", "--json", "--keep-data"); err != nil {
				return result, err
			}
		}
		if _, err := run(ctx, p.Options, "plugin", "marketplace", "remove", "mandalore", "--scope", "user"); err != nil {
			return result, err
		}
		after, err := inspectClaudeSettings(p.NativeHome)
		if err != nil {
			return result, err
		}
		root, err := claudeSelectedRegistration(after, p.StateDir)
		if err != nil || root != "" || !claudeSettingsPreserved(before, after, previousPath, "") {
			return result, errors.New("native removal changed unexpected Claude state; preserve it for inspection")
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
	again, err := inspectClaudeSettings(p.NativeHome)
	if err != nil {
		return result, err
	}
	if again.Exists != before.Exists || again.SHA256 != before.SHA256 {
		return result, errors.New("Claude profile changed before installation; inspect before retrying")
	}
	if err := verifyClaudeBindingNative(p); err != nil {
		return result, err
	}
	result.Uncertain = true
	if err := record("install-started"); err != nil {
		return result, err
	}
	newPath := filepath.Join(p.Root, "package")
	if _, err := run(ctx, p.Options, "plugin", "marketplace", "add", p.Root, "--scope", "user"); err != nil {
		return result, err
	}
	if _, err := run(ctx, p.Options, "plugin", "install", "mandalore@mandalore", "--scope", "user", "--json"); err != nil {
		return result, err
	}
	after, err := inspectClaudeSettings(p.NativeHome)
	if err != nil {
		return result, err
	}
	root, err := claudeSelectedRegistration(after, p.StateDir)
	if err != nil || root != p.Root || !after.Enabled || after.Version != p.nativeVersion() || after.Cache != filepath.Join(p.NativeHome, "plugins", "cache", "mandalore", "mandalore", p.cacheVersion()) || !claudeSettingsPreserved(before, after, "", newPath) {
		return result, errors.New("native installation did not preserve the selected Claude profile; inspect before retrying")
	}
	if _, err := ownedClaude(p.Root, p.StateDir, p.NativeHome, false); err != nil {
		return result, err
	}
	if err := verifyClaudeBindingNative(p); err != nil {
		return result, err
	}
	if err := verifyClaudeCache(after, p, false); err != nil {
		return result, err
	}
	result.Installed, result.Uncertain = true, false
	result.Notice = "Connected for fresh Claude sessions. Existing native tools, authentication and unrelated profile settings are preserved; memory, synchronization and existing-session context were not changed."
	if err := record("verified"); err != nil {
		return result, err
	}
	return result, nil
}
