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
// bounded escalation. Native Hermes commands themselves retain their 30s bounds.
func executeHermesDelegate(ctx context.Context, binary, dir string, input []byte, args ...string) ([]byte, error) {
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
		return nil, errors.New("selected Hermes runtime exceeded its output limit")
	}
	if err != nil {
		if ctx.Err() != nil {
			return out.buffer.Bytes(), ctx.Err()
		}
		return out.buffer.Bytes(), errors.New("selected Hermes runtime exited unsuccessfully; raw stderr suppressed")
	}
	return out.buffer.Bytes(), nil
}

// Preview failures have no mutation receipt. Preserve a bounded typed reason,
// never raw stdout/stderr or terminal controls, and do not mask cancellation.
func hermesPreviewFailure(raw []byte, fallback error) error {
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
	return errors.New("selected Hermes runtime refused preview [" + reply.Error.Code + "]: " + reply.Error.Message)
}

func PrepareHermesViaRuntime(ctx context.Context, o HermesOptions) (HermesPlan, error) {
	if err := ctx.Err(); err != nil {
		return HermesPlan{}, err
	}
	if err := prepareNative("hermes", &o.Options); err != nil {
		return HermesPlan{}, err
	}
	var err error
	for _, path := range []*string{&o.StateDir, &o.NativeHome, &o.NativeBinary, &o.Binary, &o.Binding} {
		*path, err = canonical(*path)
		if err != nil {
			return HermesPlan{}, err
		}
	}
	if o.RecoverFrom != "" {
		o.RecoverFrom, err = canonical(o.RecoverFrom)
		if err != nil {
			return HermesPlan{}, err
		}
	}
	want, err := digest(o.Binary)
	if err != nil {
		return HermesPlan{}, err
	}
	input, _ := json.Marshal(o)
	if len(input) > 32768 {
		return HermesPlan{}, errors.New("Hermes options exceed the input limit")
	}
	raw, err := executeHermesDelegate(ctx, o.Binary, filepath.Dir(o.Binding), input, "call", "hermes_connection_plan", "--read-only")
	if err != nil {
		return HermesPlan{}, nativeDelegationFailure(o.Options, hermesPreviewFailure(raw, err))
	}
	var reply struct {
		Protocol int             `json:"protocol_version"`
		OK       bool            `json:"ok"`
		Result   json.RawMessage `json:"result"`
	}
	if decodeNative(raw, &reply) != nil || reply.Protocol != 1 || !reply.OK {
		return HermesPlan{}, nativeDelegationFailure(o.Options, hermesPreviewFailure(raw, errors.New("selected runtime returned an invalid Hermes preview")))
	}
	var p HermesPlan
	if strictjson.Decode(reply.Result, &p, 32768) != nil {
		return HermesPlan{}, errors.New("selected Hermes plan has invalid fields or exceeds its limit")
	}
	if got, e := digest(o.Binary); e != nil || got != want {
		return HermesPlan{}, errors.New("selected runtime changed during Hermes preview")
	}
	if p.HermesOptions != o || p.BinarySHA256 != want || p.SchemaVersion != 1 || p.Harness != "hermes" || p.PackageVersion == "" || len(p.PackageVersion) > 128 {
		return HermesPlan{}, errors.New("Hermes preview is not bound to the selected inputs")
	}
	decoded, e := hex.DecodeString(p.PackageSHA256)
	if e != nil || len(decoded) != 32 || strings.ToLower(p.PackageSHA256) != p.PackageSHA256 {
		return HermesPlan{}, errors.New("invalid selected Hermes package identity")
	}
	if err := verifyHermesBindingNative(p); err != nil {
		return HermesPlan{}, err
	}
	if p.Runtime != filepath.Join(o.StateDir, "runtimes", "sha256-"+want, "mandalore") || p.Root != filepath.Join(o.StateDir, "hermes", "connections", hermesPlanKey(p)) {
		return HermesPlan{}, errors.New("selected Hermes runtime returned inconsistent generated paths")
	}
	for _, path := range []string{p.Runtime, p.Root} {
		if real, e := canonical(path); e != nil || real != path {
			return HermesPlan{}, errors.New("selected Hermes managed destination is redirected")
		}
	}
	settings, err := inspectHermesSettings(o.NativeHome)
	if err != nil {
		return HermesPlan{}, err
	}
	if settings.Exists != p.SettingsExists || settings.SHA256 != p.SettingsSHA256 {
		return HermesPlan{}, errors.New("Hermes native settings changed during preview")
	}
	previous, err := hermesSelectedRegistration(settings, o.StateDir)
	if err != nil {
		return HermesPlan{}, err
	}
	if o.RecoverFrom != "" {
		if previous != "" && previous != o.RecoverFrom {
			return HermesPlan{}, errors.New("Hermes repair preview selects a different native registration")
		}
		previous = o.RecoverFrom
	}
	if p.PreviousRoot != previous {
		return HermesPlan{}, errors.New("Hermes preview has inconsistent prior registration")
	}
	if previous != "" {
		if _, err := ownedHermes(previous, o.StateDir, o.NativeHome, o.RecoverFrom != ""); err != nil {
			return HermesPlan{}, err
		}
		data, err := readRegular(filepath.Join(previous, "receipt.json"), 65536)
		if err != nil || hash(data) != p.PreviousReceiptSHA256 {
			return HermesPlan{}, errors.New("prior Hermes ownership changed during preview")
		}
	} else if p.PreviousReceiptSHA256 != "" {
		return HermesPlan{}, errors.New("Hermes preview declares unknown prior ownership")
	}
	return p, nil
}

func ApplyHermesViaRuntime(ctx context.Context, p HermesPlan) (HermesResult, error) {
	if err := ctx.Err(); err != nil {
		return HermesResult{}, err
	}
	fresh, err := PrepareHermesViaRuntime(ctx, p.HermesOptions)
	if err != nil {
		return HermesResult{}, err
	}
	if fresh != p {
		return HermesResult{}, errors.New("selected Hermes plan is stale; inspect and preview again")
	}
	input, _ := json.Marshal(p)
	if len(input) > 32768 {
		return HermesResult{}, errors.New("Hermes plan exceeds the input limit")
	}
	raw, runErr := executeHermesDelegate(ctx, p.Binary, filepath.Dir(p.Binding), input, "call", "hermes_connection_apply")
	var reply struct {
		Protocol int             `json:"protocol_version"`
		OK       bool            `json:"ok"`
		Result   json.RawMessage `json:"result"`
		Error    struct {
			Connection json.RawMessage `json:"hermes_connection_result"`
		} `json:"error"`
	}
	if decodeNative(raw, &reply) != nil || reply.Protocol != 1 {
		return HermesResult{}, errors.New("selected Hermes runtime returned no usable apply receipt; inspect native state before retrying")
	}
	data := reply.Result
	if !reply.OK {
		data = reply.Error.Connection
	}
	var result HermesResult
	if strictjson.Decode(data, &result, 65536) != nil || result.Connection != p {
		return HermesResult{}, errors.New("selected Hermes apply receipt does not match the reviewed plan; inspect before retrying")
	}
	if runErr != nil {
		return result, runErr
	}
	if !reply.OK {
		return result, errors.New("selected Hermes runtime did not complete installation; inspect the retained phase")
	}
	if !result.Installed || !result.RequiresFreshSession || result.Phase != "verified" || result.Uncertain {
		return result, errors.New("selected Hermes runtime did not verify a complete connection")
	}
	return result, nil
}

// Repair executes only an intact runtime already named by the owned receipt.
// Its embedded package, not this menu process's version, determines the repair.
func PrepareHermesRepairViaRuntime(ctx context.Context, in RepairInput) (HermesPlan, error) {
	if err := ctx.Err(); err != nil {
		return HermesPlan{}, err
	}
	root, err := canonical(in.Root)
	if err != nil {
		return HermesPlan{}, err
	}
	in.Root = root
	r, err := loadHermesReceipt(root)
	if err != nil {
		return HermesPlan{}, errors.New("no intact Hermes ownership receipt; inspect before reconnecting")
	}
	if _, err := ownedHermes(root, r.Plan.StateDir, r.Plan.NativeHome, true); err != nil {
		return HermesPlan{}, err
	}
	b, err := readRegular(r.Plan.Binding, 32768)
	if err != nil || hash(b) != r.Plan.BindingSHA256 {
		return HermesPlan{}, errors.New("Hermes binding changed; repair cannot select another signet or writer")
	}
	o := r.Plan.HermesOptions
	o.Binary = r.Plan.Runtime
	if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
		o.Binary = r.Plan.Binary
		if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
			return HermesPlan{}, errors.New("no intact runtime remains; preview reconnect with a trusted artifact")
		}
	}
	if in.NativeBinary != "" {
		_, err = canonical(in.NativeBinary)
		if err != nil {
			return HermesPlan{}, err
		}
		o.NativeBinary = in.NativeBinary
		o.NativeLauncher = ""
	}
	if err := prepareNative("hermes", &o.Options); err != nil {
		return HermesPlan{}, err
	}
	input, _ := json.Marshal(in)
	if len(input) > 32768 {
		return HermesPlan{}, errors.New("Hermes repair input exceeds limit")
	}
	raw, err := executeHermesDelegate(ctx, o.Binary, filepath.Dir(o.Binding), input, "call", "hermes_connection_repair_plan", "--read-only")
	if err != nil {
		return HermesPlan{}, nativeDelegationFailure(o.Options, hermesPreviewFailure(raw, err))
	}
	var reply struct {
		Protocol int             `json:"protocol_version"`
		OK       bool            `json:"ok"`
		Result   json.RawMessage `json:"result"`
	}
	if decodeNative(raw, &reply) != nil || reply.Protocol != 1 || !reply.OK {
		return HermesPlan{}, nativeDelegationFailure(o.Options, hermesPreviewFailure(raw, errors.New("retained runtime could not preview Hermes repair; inspect its ownership and inputs")))
	}
	var p HermesPlan
	if strictjson.Decode(reply.Result, &p, 32768) != nil {
		return HermesPlan{}, errors.New("invalid retained Hermes repair plan")
	}
	if p.Generation == "" || p.Generation == r.Plan.Generation {
		return HermesPlan{}, errors.New("Hermes repair did not select a fresh generation")
	}
	o.Generation, o.RecoverFrom = p.Generation, root
	if p.HermesOptions != o || p.BindingSHA256 != r.Plan.BindingSHA256 || p.SignetID != r.Plan.SignetID || p.BinarySHA256 != r.Plan.BinarySHA256 || p.PackageSHA256 != r.Plan.PackageSHA256 || p.PackageVersion != r.Plan.PackageVersion {
		return HermesPlan{}, nativeDelegationFailure(o.Options, errors.New("retained Hermes repair changed owned connection identity or access mode"))
	}
	fresh, err := PrepareHermesViaRuntime(ctx, o)
	if err != nil {
		return HermesPlan{}, err
	}
	if fresh != p {
		return HermesPlan{}, errors.New("Hermes repair preview changed; inspect before retrying")
	}
	return p, nil
}
