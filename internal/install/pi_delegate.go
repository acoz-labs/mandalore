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

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// A selected Mandalore runtime owns further native process groups. Give its
// signal handler time to cancel/reap them and emit a partial receipt before
// bounded escalation. Native Pi commands themselves retain their 30s bounds.
func executePiDelegate(ctx context.Context, binary, dir string, input []byte, args ...string) ([]byte, error) {
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
		return nil, errors.New("selected Pi runtime exceeded its output limit")
	}
	if err != nil {
		if ctx.Err() != nil {
			return out.buffer.Bytes(), ctx.Err()
		}
		return out.buffer.Bytes(), errors.New("selected Pi runtime exited unsuccessfully; raw stderr suppressed")
	}
	return out.buffer.Bytes(), nil
}

func PreparePiViaRuntime(ctx context.Context, o PiOptions) (PiPlan, error) {
	if err := ctx.Err(); err != nil {
		return PiPlan{}, err
	}
	var err error
	for _, path := range []*string{&o.StateDir, &o.NativeHome, &o.NativeBinary, &o.Binary, &o.Binding} {
		*path, err = canonical(*path)
		if err != nil {
			return PiPlan{}, err
		}
	}
	if o.RecoverFrom != "" {
		o.RecoverFrom, err = canonical(o.RecoverFrom)
		if err != nil {
			return PiPlan{}, err
		}
	}
	want, err := digest(o.Binary)
	if err != nil {
		return PiPlan{}, err
	}
	input, _ := json.Marshal(o)
	if len(input) > 32768 {
		return PiPlan{}, errors.New("Pi options exceed the input limit")
	}
	raw, err := executePiDelegate(ctx, o.Binary, filepath.Dir(o.Binding), input, "call", "pi_connection_plan", "--read-only")
	if err != nil {
		return PiPlan{}, err
	}
	var reply struct {
		Protocol int             `json:"protocol_version"`
		OK       bool            `json:"ok"`
		Result   json.RawMessage `json:"result"`
	}
	if decodeNative(raw, &reply) != nil || reply.Protocol != 1 || !reply.OK {
		return PiPlan{}, errors.New("selected runtime returned an invalid Pi preview")
	}
	var p PiPlan
	if strictjson.Decode(reply.Result, &p, 32768) != nil {
		return PiPlan{}, errors.New("selected Pi plan has invalid fields or exceeds its limit")
	}
	if got, e := digest(o.Binary); e != nil || got != want {
		return PiPlan{}, errors.New("selected runtime changed during Pi preview")
	}
	if p.PiOptions != o || p.BinarySHA256 != want || p.SchemaVersion != 1 || p.Harness != "pi" || p.PackageVersion == "" || len(p.PackageVersion) > 128 {
		return PiPlan{}, errors.New("Pi preview is not bound to the selected inputs")
	}
	decoded, e := hex.DecodeString(p.PackageSHA256)
	if e != nil || len(decoded) != 32 || strings.ToLower(p.PackageSHA256) != p.PackageSHA256 {
		return PiPlan{}, errors.New("invalid selected Pi package identity")
	}
	if err := verifyPiBindingNative(p); err != nil {
		return PiPlan{}, err
	}
	if p.Runtime != filepath.Join(o.StateDir, "runtimes", "sha256-"+want, "mandalore") || p.Root != filepath.Join(o.StateDir, "pi", "connections", piPlanKey(p)) {
		return PiPlan{}, errors.New("selected Pi runtime returned inconsistent generated paths")
	}
	for _, path := range []string{p.Runtime, p.Root} {
		if real, e := canonical(path); e != nil || real != path {
			return PiPlan{}, errors.New("selected Pi managed destination is redirected")
		}
	}
	settings, err := inspectPiSettings(o.NativeHome)
	if err != nil {
		return PiPlan{}, err
	}
	if settings.Exists != p.SettingsExists || settings.SHA256 != p.SettingsSHA256 {
		return PiPlan{}, errors.New("Pi native settings changed during preview")
	}
	previous, err := piSelectedRegistration(settings, o.StateDir)
	if err != nil {
		return PiPlan{}, err
	}
	if o.RecoverFrom != "" {
		if previous != "" && previous != o.RecoverFrom {
			return PiPlan{}, errors.New("Pi repair preview selects a different native registration")
		}
		previous = o.RecoverFrom
	}
	if p.PreviousRoot != previous {
		return PiPlan{}, errors.New("Pi preview has inconsistent prior registration")
	}
	if previous != "" {
		if _, err := ownedPi(previous, o.StateDir, o.NativeHome, o.RecoverFrom != ""); err != nil {
			return PiPlan{}, err
		}
		data, err := readRegular(filepath.Join(previous, "receipt.json"), 65536)
		if err != nil || hash(data) != p.PreviousReceiptSHA256 {
			return PiPlan{}, errors.New("prior Pi ownership changed during preview")
		}
	} else if p.PreviousReceiptSHA256 != "" {
		return PiPlan{}, errors.New("Pi preview declares unknown prior ownership")
	}
	return p, nil
}

func ApplyPiViaRuntime(ctx context.Context, p PiPlan) (PiResult, error) {
	if err := ctx.Err(); err != nil {
		return PiResult{}, err
	}
	fresh, err := PreparePiViaRuntime(ctx, p.PiOptions)
	if err != nil {
		return PiResult{}, err
	}
	if fresh != p {
		return PiResult{}, errors.New("selected Pi plan is stale; inspect and preview again")
	}
	input, _ := json.Marshal(p)
	if len(input) > 32768 {
		return PiResult{}, errors.New("Pi plan exceeds the input limit")
	}
	raw, runErr := executePiDelegate(ctx, p.Binary, filepath.Dir(p.Binding), input, "call", "pi_connection_apply")
	var reply struct {
		Protocol int             `json:"protocol_version"`
		OK       bool            `json:"ok"`
		Result   json.RawMessage `json:"result"`
		Error    struct {
			Connection json.RawMessage `json:"pi_connection_result"`
		} `json:"error"`
	}
	if decodeNative(raw, &reply) != nil || reply.Protocol != 1 {
		return PiResult{}, errors.New("selected Pi runtime returned no usable apply receipt; inspect native state before retrying")
	}
	data := reply.Result
	if !reply.OK {
		data = reply.Error.Connection
	}
	var result PiResult
	if strictjson.Decode(data, &result, 65536) != nil || result.Connection != p {
		return PiResult{}, errors.New("selected Pi apply receipt does not match the reviewed plan; inspect before retrying")
	}
	if runErr != nil {
		return result, runErr
	}
	if !reply.OK {
		return result, errors.New("selected Pi runtime did not complete installation; inspect the retained phase")
	}
	if !result.Installed || !result.RequiresFreshSession || result.Phase != "verified" || result.Uncertain {
		return result, errors.New("selected Pi runtime did not verify a complete connection")
	}
	return result, nil
}

// Repair executes only an intact runtime already named by the owned receipt.
// Its embedded package, not this menu process's version, determines the repair.
func PreparePiRepairViaRuntime(ctx context.Context, in RepairInput) (PiPlan, error) {
	if err := ctx.Err(); err != nil {
		return PiPlan{}, err
	}
	root, err := canonical(in.Root)
	if err != nil {
		return PiPlan{}, err
	}
	in.Root = root
	r, err := loadPiReceipt(root)
	if err != nil {
		return PiPlan{}, errors.New("no intact Pi ownership receipt; inspect before reconnecting")
	}
	if _, err := ownedPi(root, r.Plan.StateDir, r.Plan.NativeHome, true); err != nil {
		return PiPlan{}, err
	}
	b, err := readRegular(r.Plan.Binding, 32768)
	if err != nil || hash(b) != r.Plan.BindingSHA256 {
		return PiPlan{}, errors.New("Pi binding changed; repair cannot select another signet or writer")
	}
	o := r.Plan.PiOptions
	o.Binary = r.Plan.Runtime
	if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
		o.Binary = r.Plan.Binary
		if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
			return PiPlan{}, errors.New("no intact runtime remains; preview reconnect with a trusted artifact")
		}
	}
	if in.NativeBinary != "" {
		in.NativeBinary, err = canonical(in.NativeBinary)
		if err != nil {
			return PiPlan{}, err
		}
		o.NativeBinary = in.NativeBinary
	}
	input, _ := json.Marshal(in)
	if len(input) > 32768 {
		return PiPlan{}, errors.New("Pi repair input exceeds limit")
	}
	raw, err := executePiDelegate(ctx, o.Binary, filepath.Dir(o.Binding), input, "call", "pi_connection_repair_plan", "--read-only")
	if err != nil {
		return PiPlan{}, err
	}
	var reply struct {
		Protocol int             `json:"protocol_version"`
		OK       bool            `json:"ok"`
		Result   json.RawMessage `json:"result"`
	}
	if decodeNative(raw, &reply) != nil || reply.Protocol != 1 || !reply.OK {
		return PiPlan{}, errors.New("retained runtime could not preview Pi repair; inspect its ownership and inputs")
	}
	var p PiPlan
	if strictjson.Decode(reply.Result, &p, 32768) != nil {
		return PiPlan{}, errors.New("invalid retained Pi repair plan")
	}
	if p.Generation == "" || p.Generation == r.Plan.Generation {
		return PiPlan{}, errors.New("Pi repair did not select a fresh generation")
	}
	o.Generation, o.RecoverFrom = p.Generation, root
	if p.PiOptions != o || p.BindingSHA256 != r.Plan.BindingSHA256 || p.SignetID != r.Plan.SignetID || p.BinarySHA256 != r.Plan.BinarySHA256 || p.PackageSHA256 != r.Plan.PackageSHA256 || p.PackageVersion != r.Plan.PackageVersion {
		return PiPlan{}, errors.New("retained Pi repair changed owned connection identity or access mode")
	}
	fresh, err := PreparePiViaRuntime(ctx, o)
	if err != nil {
		return PiPlan{}, err
	}
	if fresh != p {
		return PiPlan{}, errors.New("Pi repair preview changed; inspect before retrying")
	}
	return p, nil
}
