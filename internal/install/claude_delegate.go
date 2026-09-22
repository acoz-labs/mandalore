package install

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// A selected Mandalore runtime owns further native process groups. Give its
// signal handler time to cancel/reap them and emit a partial receipt before
// bounded escalation. Native Claude commands themselves retain their 30s bounds.
func executeClaudeDelegate(ctx context.Context, binary, dir string, input []byte, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir, cmd.Env, cmd.Stdin = dir, environment(nil), bytes.NewReader(input)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = 2 * time.Second
	var out boundedOutput
	cmd.Stdout = &out // Raw stderr is discarded.
	err := cmd.Run()
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	if out.exceeded {
		return nil, errors.New("selected Claude runtime exceeded its output limit")
	}
	if err != nil {
		if ctx.Err() != nil {
			return out.buffer.Bytes(), ctx.Err()
		}
		return out.buffer.Bytes(), errors.New("selected Claude runtime exited unsuccessfully; raw stderr suppressed")
	}
	return out.buffer.Bytes(), nil
}

// Preview failures have no mutation receipt. Preserve a bounded typed reason,
// never raw stdout/stderr or terminal controls, and do not mask cancellation.
func claudePreviewFailure(raw []byte, fallback error) error {
	if errors.Is(fallback, context.Canceled) || errors.Is(fallback, context.DeadlineExceeded) {
		return fallback
	}
	var reply struct {
		Protocol int   `json:"protocol_version"`
		OK       *bool `json:"ok"`
		Error    struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if len(raw) > 65536 || decodeNative(raw, &reply) != nil || reply.Protocol != 1 || reply.OK == nil || *reply.OK {
		return fallback
	}
	if reply.Error.Code == "" || len(reply.Error.Code) > 128 || strings.TrimSpace(reply.Error.Message) == "" || len(reply.Error.Message) > 2048 || strings.IndexFunc(reply.Error.Code+reply.Error.Message, unicode.IsControl) >= 0 {
		return fallback
	}
	return errors.New("selected Claude runtime refused preview [" + reply.Error.Code + "]: " + reply.Error.Message)
}

func PrepareClaudeViaRuntime(ctx context.Context, o ClaudeOptions) (ClaudePlan, error) {
	if err := ctx.Err(); err != nil {
		return ClaudePlan{}, err
	}
	var err error
	for _, path := range []*string{&o.StateDir, &o.NativeHome, &o.NativeBinary, &o.Binary, &o.Binding} {
		*path, err = canonical(*path)
		if err != nil {
			return ClaudePlan{}, err
		}
	}
	if o.RecoverFrom != "" {
		o.RecoverFrom, err = canonical(o.RecoverFrom)
		if err != nil {
			return ClaudePlan{}, err
		}
	}
	want, err := digest(o.Binary)
	if err != nil {
		return ClaudePlan{}, err
	}
	input, _ := json.Marshal(o)
	if len(input) > 32768 {
		return ClaudePlan{}, errors.New("Claude options exceed the input limit")
	}
	raw, err := executeClaudeDelegate(ctx, o.Binary, filepath.Dir(o.Binding), input, "call", "claude_code_connection_plan", "--read-only")
	if err != nil {
		return ClaudePlan{}, claudePreviewFailure(raw, err)
	}
	var reply struct {
		Protocol int             `json:"protocol_version"`
		OK       bool            `json:"ok"`
		Result   json.RawMessage `json:"result"`
	}
	if decodeNative(raw, &reply) != nil || reply.Protocol != 1 || !reply.OK {
		return ClaudePlan{}, claudePreviewFailure(raw, errors.New("selected runtime returned an invalid Claude preview"))
	}
	var p ClaudePlan
	if strictjson.Decode(reply.Result, &p, 32768) != nil {
		return ClaudePlan{}, errors.New("selected Claude plan has invalid fields or exceeds its limit")
	}
	if got, e := digest(o.Binary); e != nil || got != want {
		return ClaudePlan{}, errors.New("selected runtime changed during Claude preview")
	}
	if p.ClaudeOptions != o || p.BinarySHA256 != want || p.SchemaVersion != 1 || p.Harness != "claude-code" || p.PackageVersion == "" || len(p.PackageVersion) > 128 {
		return ClaudePlan{}, errors.New("Claude preview is not bound to the selected inputs")
	}
	decoded, e := hex.DecodeString(p.PackageSHA256)
	if e != nil || len(decoded) != 32 || strings.ToLower(p.PackageSHA256) != p.PackageSHA256 {
		return ClaudePlan{}, errors.New("invalid selected Claude package identity")
	}
	if err := verifyClaudeBindingNative(p); err != nil {
		return ClaudePlan{}, err
	}
	if p.Runtime != filepath.Join(o.StateDir, "runtimes", "sha256-"+want, "mandalore") || p.Root != filepath.Join(o.StateDir, "claude-code", "connections", claudePlanKey(p)) {
		return ClaudePlan{}, errors.New("selected Claude runtime returned inconsistent generated paths")
	}
	for _, path := range []string{p.Runtime, p.Root} {
		if real, e := canonical(path); e != nil || real != path {
			return ClaudePlan{}, errors.New("selected Claude managed destination is redirected")
		}
	}
	settings, err := inspectClaudeSettings(o.NativeHome)
	if err != nil {
		return ClaudePlan{}, err
	}
	if settings.Exists != p.SettingsExists || settings.SHA256 != p.SettingsSHA256 {
		return ClaudePlan{}, errors.New("Claude native settings changed during preview")
	}
	previous, err := claudeSelectedRegistration(settings, o.StateDir)
	if err != nil {
		return ClaudePlan{}, err
	}
	if o.RecoverFrom != "" {
		if previous != "" && previous != o.RecoverFrom {
			return ClaudePlan{}, errors.New("Claude repair preview selects a different native registration")
		}
		previous = o.RecoverFrom
	}
	if p.PreviousRoot != previous {
		return ClaudePlan{}, errors.New("Claude preview has inconsistent prior registration")
	}
	if previous != "" {
		if _, err := ownedClaude(previous, o.StateDir, o.NativeHome, o.RecoverFrom != ""); err != nil {
			return ClaudePlan{}, err
		}
		data, err := readRegular(filepath.Join(previous, "receipt.json"), 65536)
		if err != nil || hash(data) != p.PreviousReceiptSHA256 {
			return ClaudePlan{}, errors.New("prior Claude ownership changed during preview")
		}
	} else if p.PreviousReceiptSHA256 != "" {
		return ClaudePlan{}, errors.New("Claude preview declares unknown prior ownership")
	}
	return p, nil
}

func ApplyClaudeViaRuntime(ctx context.Context, in ClaudeApplyInput) (ClaudeResult, error) {
	p := in.Plan
	if err := ctx.Err(); err != nil {
		return ClaudeResult{}, err
	}
	if d, err := digest(p.Binary); err != nil || d != p.BinarySHA256 {
		return ClaudeResult{}, errors.New("selected Claude runtime changed after preview")
	}
	settings, err := inspectClaudeSettings(p.NativeHome)
	if err != nil {
		return ClaudeResult{}, err
	}
	if settings.Root == p.Root {
		receipt, err := ownedClaude(p.Root, p.StateDir, p.NativeHome, false)
		if err != nil || receipt.Plan != p {
			return ClaudeResult{}, errors.New("registered Claude generation differs from reviewed plan")
		}
		if err := verifyClaudeBindingNative(p); err != nil {
			return ClaudeResult{}, err
		}
		if err := verifyClaudeCache(settings, p, false); err != nil {
			return ClaudeResult{}, err
		}
	} else {
		fresh, err := PrepareClaudeViaRuntime(ctx, p.ClaudeOptions)
		if err != nil {
			return ClaudeResult{}, err
		}
		if fresh != p {
			return ClaudeResult{}, errors.New("selected Claude plan is stale; inspect and preview again")
		}
	}
	input, _ := json.Marshal(in)
	if len(input) > 32768 {
		return ClaudeResult{}, errors.New("Claude plan exceeds the input limit")
	}
	raw, runErr := executeClaudeDelegate(ctx, p.Binary, filepath.Dir(p.Binding), input, "call", "claude_code_connection_apply")
	var reply struct {
		Protocol int             `json:"protocol_version"`
		OK       bool            `json:"ok"`
		Result   json.RawMessage `json:"result"`
		Error    struct {
			Connection json.RawMessage `json:"claude_code_connection_result"`
		} `json:"error"`
	}
	if decodeNative(raw, &reply) != nil || reply.Protocol != 1 {
		return ClaudeResult{}, errors.New("selected Claude runtime returned no usable apply receipt; inspect native state before retrying")
	}
	data := reply.Result
	if !reply.OK {
		data = reply.Error.Connection
	}
	var result ClaudeResult
	if strictjson.Decode(data, &result, 65536) != nil || result.Connection != p {
		return ClaudeResult{}, errors.New("selected Claude apply receipt does not match the reviewed plan; inspect before retrying")
	}
	if runErr != nil {
		return result, runErr
	}
	if !reply.OK {
		return result, errors.New("selected Claude runtime did not complete installation; inspect the retained phase")
	}
	if result.Phase == "deferred" && !result.Installed && !result.Uncertain {
		return result, nil
	}
	if !result.Installed || !result.RequiresFreshSession || result.Phase != "verified" || result.Uncertain {
		return result, errors.New("selected Claude runtime did not verify a complete connection")
	}
	return result, nil
}

// Repair executes only an intact runtime already named by the owned receipt.
// Its embedded package, not this menu process's version, determines the repair.
func PrepareClaudeRepairViaRuntime(ctx context.Context, in RepairInput) (ClaudePlan, error) {
	if err := ctx.Err(); err != nil {
		return ClaudePlan{}, err
	}
	root, err := canonical(in.Root)
	if err != nil {
		return ClaudePlan{}, err
	}
	in.Root = root
	r, err := loadClaudeReceipt(root)
	if err != nil {
		return ClaudePlan{}, errors.New("no intact Claude ownership receipt; inspect before reconnecting")
	}
	if _, err := ownedClaude(root, r.Plan.StateDir, r.Plan.NativeHome, true); err != nil {
		return ClaudePlan{}, err
	}
	b, err := readRegular(r.Plan.Binding, 32768)
	if err != nil || hash(b) != r.Plan.BindingSHA256 {
		return ClaudePlan{}, errors.New("Claude binding changed; repair cannot select another signet or writer")
	}
	o := r.Plan.ClaudeOptions
	o.Binary = r.Plan.Runtime
	if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
		o.Binary = r.Plan.Binary
		if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
			return ClaudePlan{}, errors.New("no intact runtime remains; preview reconnect with a trusted artifact")
		}
	}
	if in.NativeBinary != "" {
		in.NativeBinary, err = canonical(in.NativeBinary)
		if err != nil {
			return ClaudePlan{}, err
		}
		o.NativeBinary = in.NativeBinary
	}
	input, _ := json.Marshal(in)
	if len(input) > 32768 {
		return ClaudePlan{}, errors.New("Claude repair input exceeds limit")
	}
	raw, err := executeClaudeDelegate(ctx, o.Binary, filepath.Dir(o.Binding), input, "call", "claude_code_connection_repair_plan", "--read-only")
	if err != nil {
		return ClaudePlan{}, claudePreviewFailure(raw, err)
	}
	var reply struct {
		Protocol int             `json:"protocol_version"`
		OK       bool            `json:"ok"`
		Result   json.RawMessage `json:"result"`
	}
	if decodeNative(raw, &reply) != nil || reply.Protocol != 1 || !reply.OK {
		return ClaudePlan{}, claudePreviewFailure(raw, errors.New("retained runtime could not preview Claude repair; inspect its ownership and inputs"))
	}
	var p ClaudePlan
	if strictjson.Decode(reply.Result, &p, 32768) != nil {
		return ClaudePlan{}, errors.New("invalid retained Claude repair plan")
	}
	if p.Generation == "" || p.Generation == r.Plan.Generation {
		return ClaudePlan{}, errors.New("Claude repair did not select a fresh generation")
	}
	o.Generation, o.RecoverFrom = p.Generation, root
	if p.ClaudeOptions != o || p.BindingSHA256 != r.Plan.BindingSHA256 || p.SignetID != r.Plan.SignetID || p.BinarySHA256 != r.Plan.BinarySHA256 || p.PackageSHA256 != r.Plan.PackageSHA256 || p.PackageVersion != r.Plan.PackageVersion {
		return ClaudePlan{}, errors.New("retained Claude repair changed owned connection identity or access mode")
	}
	fresh, err := PrepareClaudeViaRuntime(ctx, o)
	if err != nil {
		return ClaudePlan{}, err
	}
	if fresh != p {
		return ClaudePlan{}, errors.New("Claude repair preview changed; inspect before retrying")
	}
	return p, nil
}
