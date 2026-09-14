package distribution

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"time"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

type InstallResult struct {
	Phase              string `json:"phase"`
	PlanSHA256         string `json:"plan_sha256"`
	Prefix             string `json:"prefix"`
	Launcher           string `json:"launcher"`
	Runtime            string `json:"runtime"`
	PreviousRuntime    string `json:"previous_runtime,omitempty"`
	Pending            string `json:"pending,omitempty"`
	Connections        string `json:"connections"`
	Installed          bool   `json:"installed"`
	AlreadyCurrent     bool   `json:"already_current"`
	DestinationChanged bool   `json:"destination_changed"`
	Notice             string `json:"notice"`
}

type pendingInstall struct {
	FormatVersion   int               `json:"format_version"`
	Plan            InstallPlan       `json:"plan"`
	PlanSHA256      string            `json:"plan_sha256"`
	PreviousReceipt string            `json:"previous_receipt"`
	Directories     []PathObservation `json:"directories"`
}

const MaxPendingInstallBytes = 256 << 10

// ParsePendingInstallPlan recovers only the exact recorded plan. It performs no
// writes and makes no claim that the current filesystem still matches. ApplyInstall
// must independently re-read the pending record and validate retained bytes,
// directory identities and the expected old/new launcher and receipt states.
func ParsePendingInstallPlan(raw []byte) (InstallPlan, error) {
	var pending pendingInstall
	if err := strictjson.Decode(raw, &pending, MaxPendingInstallBytes); err != nil || pending.FormatVersion != 1 {
		return InstallPlan{}, errors.New("invalid pending installation record")
	}
	b, err := json.Marshal(pending.Plan)
	if err != nil {
		return InstallPlan{}, errors.New("invalid recorded installation plan")
	}
	p, err := ParseInstallPlan(b)
	if err != nil {
		return InstallPlan{}, err
	}
	if pending.PlanSHA256 != installPlanKey(p) {
		return InstallPlan{}, errors.New("pending record does not match its exact plan digest")
	}
	return p, nil
}

func installPlanKey(p InstallPlan) string { b, _ := json.Marshal(p); return Digest(b) }

func ApplyInstall(ctx context.Context, p InstallPlan) (InstallResult, error) {
	return applyInstall(ctx, p, NewReleaseClient(), verifyInstallRuntime, nil)
}

// The private verifier/phase observer are deterministic test seams, not runtime
// extension points. The public entrypoint always runs the production verifier.
func applyInstall(ctx context.Context, p InstallPlan, client *ReleaseClient, verify func(context.Context, string, InstallPlan) error, after func(string) error) (result InstallResult, err error) {
	result = InstallResult{Phase: "preflight", PlanSHA256: installPlanKey(p), Prefix: p.Prefix, Launcher: p.Launcher, Runtime: p.Runtime, PreviousRuntime: p.Observed.LauncherTarget, Connections: "unchanged", Notice: installNotice}
	b, _ := json.Marshal(p)
	if _, err = ParseInstallPlan(b); err != nil {
		return result, err
	}
	if p.OS != runtime.GOOS || p.Arch != runtime.GOARCH {
		return result, errors.New("reviewed plan targets another machine platform")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err = ctx.Err(); err != nil {
		return result, err
	}
	advance := func(phase string) error {
		result.Phase = phase
		if err := ctx.Err(); err != nil {
			return err
		}
		if after != nil {
			return after(phase)
		}
		return nil
	}

	// An interrupted exact activation is distinct from a fresh plan. It resumes
	// only retained verified bytes under the same lock and recorded directories.
	state := cliState(p.Prefix)
	if _, e := os.Lstat(filepath.Join(state, "pending.json")); e == nil {
		result.Pending = filepath.Join(state, "pending.json")
		result.Phase = "pending-inspection"
		for _, path := range []string{p.Prefix, filepath.Join(p.Prefix, "bin"), filepath.Join(p.Prefix, "lib"), state} {
			if _, err = observeDirectory(path); err != nil {
				return result, err
			}
		}
		unlock, created, e := lockInstallState(state)
		result.DestinationChanged = created
		if e != nil {
			return result, e
		}
		defer unlock()
		pending, raw, e := readPendingInstall(p)
		if e != nil {
			return result, e
		}
		result.Pending = filepath.Join(state, "pending.json")
		if err = checkPendingInstall(pending, raw); err != nil {
			return result, err
		}
		if err = verify(ctx, p.Runtime, p); err != nil {
			return result, err
		}
		return finishInstall(ctx, pending, raw, result, advance)
	} else if !os.IsNotExist(e) {
		return result, errors.New("cannot inspect pending installation state")
	}

	// Successful exact-plan replay needs no network or source directory. Its
	// receipt and retained bytes, not version strings, prove the finished action.
	if observation, e := observeInstallation(p.Prefix, p.OS, p.Arch); e == nil && observation.Current == p.Source.Manifest.SHA256 {
		raw, e := readInstallFile(filepath.Join(state, "receipt.json"), MaxManifestBytes)
		var receipt cliReceipt
		if e == nil && strictjson.Decode(raw, &receipt, MaxManifestBytes) == nil && receipt.LastPlan == result.PlanSHA256 {
			retained, e := retainedManifest(p.Prefix, observation.Current, p.OS, p.Arch)
			if e != nil || !reflect.DeepEqual(retained, p.Source.Manifest) {
				return result, errors.New("completed plan no longer matches retained bytes")
			}
			result.Phase, result.Installed, result.AlreadyCurrent = "complete", true, true
			return result, nil
		}
	}
	fresh, err := planInstall(ctx, p.InstallOptions, client)
	if err != nil {
		return result, err
	}
	if !reflect.DeepEqual(fresh, p) {
		return result, errors.New("installation plan is stale; inspect a fresh plan before applying")
	}
	if p.Observed.Current == p.Source.Manifest.SHA256 {
		result.Phase, result.Installed, result.AlreadyCurrent = "complete", true, true
		return result, nil
	}
	stage, cleanup, err := ownedInstallTemp("", "mandalore-install-")
	if err != nil {
		return result, err
	}
	defer cleanup()
	result.Phase = "staging"
	if err = stageInstallSource(ctx, p, client, stage); err != nil {
		return result, err
	}
	if err = os.Chmod(filepath.Join(stage, "mandalore"), 0700); err != nil {
		return result, err
	}
	if err = verify(ctx, filepath.Join(stage, "mandalore"), p); err != nil {
		return result, err
	}
	// A trusted version probe must not be able to change the verified bytes and
	// then have its old pre-execution digest reused for activation.
	if err = verifyInstallPair(stage, p); err != nil {
		return result, err
	}
	if err = advance("verified"); err != nil {
		return result, err
	}
	fresh, err = planInstall(ctx, p.InstallOptions, client)
	if err != nil {
		return result, err
	}
	if !reflect.DeepEqual(fresh, p) {
		return result, errors.New("source or destination changed during staging; inspect a fresh plan")
	}

	expected := append([]PathObservation(nil), p.Observed.Directories...)
	// Create only selected prefix/parents and shared bin/lib directories. They
	// remain user-owned; no cleanup ever recursively removes the prefix.
	for i := 0; i < 3; i++ {
		if !expected[i].Exists {
			if err = makeInstallDirectory(expected[i].Path, &result); err != nil {
				return result, err
			}
			expected[i], err = observeDirectory(expected[i].Path)
			if err != nil {
				return result, err
			}
		}
	}
	if err = checkInstallObservations(expected); err != nil {
		return result, err
	}
	initial := !p.Observed.Directories[3].Exists
	work := state
	if initial {
		var cleanupState func()
		work, cleanupState, err = ownedInstallTemp(filepath.Join(p.Prefix, "lib"), ".mandalore-stage-")
		if err != nil {
			return result, err
		}
		defer cleanupState()
		result.DestinationChanged = true
	}
	unlock, created, err := lockInstallState(work)
	result.DestinationChanged = result.DestinationChanged || created
	if err != nil {
		return result, err
	}
	defer unlock()
	if err = advance("lock-acquired"); err != nil {
		return result, err
	}
	if err = checkInstallObservations(expected); err != nil {
		return result, err
	}
	previous, err := checkOldInstallState(p)
	if err != nil {
		return result, err
	}
	beforeRetention := append([]PathObservation(nil), expected...)
	if initial {
		target := filepath.Join(work, "releases", "sha256-"+p.Source.Manifest.SHA256, p.OS+"_"+p.Arch)
		if err = os.MkdirAll(target, 0700); err != nil {
			return result, err
		}
		if err = copyInstallPair(ctx, stage, target, p); err != nil {
			return result, err
		}
		for i := 3; i < len(expected); i++ {
			rel, e := filepath.Rel(state, expected[i].Path)
			if e != nil {
				return result, e
			}
			d, e := observeDirectory(filepath.Join(work, rel))
			if e != nil {
				return result, e
			}
			d.Path = expected[i].Path
			expected[i] = d
		}
	} else if !p.RuntimeRetained {
		for i := 4; i < 6; i++ {
			if !expected[i].Exists {
				if err = os.Mkdir(expected[i].Path, 0700); err != nil {
					return result, err
				}
				result.DestinationChanged = true
				if err = syncInstallDirectory(filepath.Dir(expected[i].Path)); err != nil {
					return result, err
				}
				expected[i], err = observeDirectory(expected[i].Path)
				if err != nil {
					return result, err
				}
			}
		}
		targetStage, clean, e := ownedInstallTemp(filepath.Dir(filepath.Dir(p.Runtime)), ".runtime-")
		if e != nil {
			return result, e
		}
		result.DestinationChanged = true
		defer clean()
		if err = copyInstallPair(ctx, stage, targetStage, p); err != nil {
			return result, err
		}
		if err = checkInstallObservations(expected); err != nil {
			return result, err
		}
		if err = publishCandidate(targetStage, filepath.Dir(p.Runtime)); err != nil {
			return result, errors.New("retained target appeared or could not be published; no target replaced")
		}
		if err = syncInstallDirectory(filepath.Dir(filepath.Dir(p.Runtime))); err != nil {
			return result, err
		}
		expected[6], err = observeDirectory(filepath.Dir(p.Runtime))
		if err != nil {
			return result, err
		}
	}
	phase := "runtime-retained"
	if initial {
		phase = "runtime-staged"
	}
	if err = advance(phase); err != nil {
		return result, err
	}
	pending := pendingInstall{FormatVersion: 1, Plan: p, PlanSHA256: result.PlanSHA256, PreviousReceipt: string(previous), Directories: expected}
	raw, _ := json.Marshal(pending)
	if len(raw) > MaxPendingInstallBytes {
		return result, errors.New("pending activation exceeds its receipt budget")
	}
	if initial {
		if err = writeInstallFile(filepath.Join(work, "pending.json"), raw, 0600); err != nil {
			return result, err
		}
		for _, path := range []string{filepath.Join(work, "releases", "sha256-"+p.Source.Manifest.SHA256), filepath.Join(work, "releases"), work} {
			if err = syncInstallDirectory(path); err != nil {
				return result, err
			}
		}
		if err = checkInstallObservations(beforeRetention); err != nil {
			return result, err
		}
		if _, err = checkOldInstallState(p); err != nil {
			return result, err
		}
		if err = publishCandidate(work, state); err != nil {
			return result, errors.New("installation state appeared or could not be published; no directory replaced")
		}
		result.Pending = filepath.Join(state, "pending.json")
		if err = syncInstallDirectory(filepath.Dir(state)); err != nil {
			return result, err
		}
	} else {
		if err = checkInstallObservations(expected); err != nil {
			return result, err
		}
		if _, err = checkOldInstallState(p); err != nil {
			return result, err
		}
		pendingPath := filepath.Join(state, "pending.json")
		err = atomicInstallJSON(pendingPath, raw, false)
		if sameInstallBytes(pendingPath, raw) {
			result.DestinationChanged = true
			result.Pending = pendingPath
		}
		if err != nil {
			return result, err
		}
	}
	if err = advance("pending-recorded"); err != nil {
		return result, err
	}
	return finishInstall(ctx, pending, raw, result, advance)
}

func makeInstallDirectory(path string, result *InstallResult) error {
	parent := filepath.Dir(path)
	if _, err := os.Lstat(parent); os.IsNotExist(err) {
		if err := makeInstallDirectory(parent, result); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := os.Mkdir(path, 0700); err != nil {
		return errors.New("planned directory appeared or could not be created; inspect a fresh plan")
	}
	result.DestinationChanged = true
	return syncInstallDirectory(parent)
}

func checkInstallObservations(expected []PathObservation) error {
	for _, want := range expected {
		got, err := observeDirectory(want.Path)
		if err != nil || got != want {
			return errors.New("installation directory identities changed; refusing to follow or replace them")
		}
	}
	return nil
}

func installLauncherTarget(path string) (string, error) {
	st, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if st.Mode()&os.ModeSymlink == 0 {
		return "", errors.New("launcher is not an owned symlink; no replacement authorized")
	}
	return os.Readlink(path)
}

func checkOldInstallState(p InstallPlan) ([]byte, error) {
	target, err := installLauncherTarget(p.Launcher)
	if err != nil || target != p.Observed.LauncherTarget {
		return nil, errors.New("launcher changed after preview")
	}
	path := filepath.Join(cliState(p.Prefix), "receipt.json")
	if p.Observed.ReceiptSHA256 == "" {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			return nil, errors.New("unexpected ownership receipt appeared")
		}
		return nil, nil
	}
	b, err := readInstallFile(path, MaxManifestBytes)
	if err != nil || Digest(b) != p.Observed.ReceiptSHA256 {
		return nil, errors.New("ownership receipt changed after preview")
	}
	if _, err := retainedManifest(p.Prefix, p.Observed.Current, p.OS, p.Arch); err != nil {
		return nil, err
	}
	return b, nil
}

func readPendingInstall(p InstallPlan) (pendingInstall, []byte, error) {
	raw, err := readInstallFile(filepath.Join(cliState(p.Prefix), "pending.json"), MaxPendingInstallBytes)
	if err != nil {
		return pendingInstall{}, nil, err
	}
	var pending pendingInstall
	if err := strictjson.Decode(raw, &pending, MaxPendingInstallBytes); err != nil || pending.FormatVersion != 1 || pending.PlanSHA256 != installPlanKey(p) || !reflect.DeepEqual(pending.Plan, p) {
		return pendingInstall{}, nil, errors.New("pending activation does not match this exact reviewed plan; inspect before retrying")
	}
	return pending, raw, nil
}

func checkPendingInstall(pending pendingInstall, raw []byte) error {
	p := pending.Plan
	if pending.FormatVersion != 1 || pending.PlanSHA256 != installPlanKey(p) || len(pending.Directories) != len(p.Observed.Directories) {
		return errors.New("invalid pending activation identity")
	}
	for i, d := range pending.Directories {
		if d.Path != p.Observed.Directories[i].Path || !d.Exists {
			return errors.New("invalid pending destination observations")
		}
	}
	if err := checkInstallObservations(pending.Directories); err != nil {
		return err
	}
	if !sameInstallBytes(filepath.Join(cliState(p.Prefix), "pending.json"), raw) {
		return errors.New("pending activation record changed")
	}
	if (len(pending.PreviousReceipt) == 0 && p.Observed.ReceiptSHA256 != "") || (len(pending.PreviousReceipt) > 0 && Digest([]byte(pending.PreviousReceipt)) != p.Observed.ReceiptSHA256) {
		return errors.New("pending activation does not preserve the reviewed previous receipt")
	}
	retained, err := retainedManifest(p.Prefix, p.Source.Manifest.SHA256, p.OS, p.Arch)
	if err != nil || !reflect.DeepEqual(retained, p.Source.Manifest) {
		return errors.New("pending runtime differs from approved identity")
	}
	if p.Observed.Current != "" {
		if _, err := retainedManifest(p.Prefix, p.Observed.Current, p.OS, p.Arch); err != nil {
			return err
		}
	}
	return nil
}

func finishInstall(ctx context.Context, pending pendingInstall, raw []byte, result InstallResult, advance func(string) error) (InstallResult, error) {
	p := pending.Plan
	state := cliState(p.Prefix)
	if err := checkPendingInstall(pending, raw); err != nil {
		return result, err
	}
	path := filepath.Join(state, "receipt.json")
	next := nextInstallReceipt(p, pending.PlanSHA256)
	oldReceipt, newReceipt := sameInstallBytes(path, []byte(pending.PreviousReceipt)), sameInstallBytes(path, next)
	target, err := installLauncherTarget(p.Launcher)
	if err != nil || (!oldReceipt && !newReceipt) || (target != p.Runtime && target != p.Observed.LauncherTarget) || (newReceipt && target != p.Runtime) {
		return result, errors.New("launcher/receipt state conflicts with pending activation; no blind recovery")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	result.Pending = filepath.Join(state, "pending.json")
	if target != p.Runtime {
		tmp := filepath.Join(filepath.Dir(p.Launcher), ".mandalore-link-"+rand.Text())
		if err := os.Symlink(p.Runtime, tmp); err != nil {
			return result, err
		}
		defer os.Remove(tmp)
		if err := checkPendingInstall(pending, raw); err != nil {
			return result, err
		}
		if current, e := installLauncherTarget(p.Launcher); e != nil || current != p.Observed.LauncherTarget || !sameInstallBytes(path, []byte(pending.PreviousReceipt)) {
			return result, errors.New("activation inputs changed immediately before launcher switch")
		}
		if p.Observed.LauncherTarget == "" {
			err = publishCandidate(tmp, p.Launcher)
		} else {
			err = os.Rename(tmp, p.Launcher)
		}
		if err != nil {
			return result, err
		}
		result.DestinationChanged = true
		result.Phase = "launcher-activated"
		if err = syncInstallDirectory(filepath.Dir(p.Launcher)); err != nil {
			return result, err
		}
	}
	result.Phase = "launcher-activated"
	if err := advance("launcher-activated"); err != nil {
		return result, err
	}
	if err := checkPendingInstall(pending, raw); err != nil {
		return result, err
	}
	if target, e := installLauncherTarget(p.Launcher); e != nil || target != p.Runtime {
		return result, errors.New("launcher changed before receipt publication")
	}
	if !newReceipt {
		if !sameInstallBytes(path, []byte(pending.PreviousReceipt)) {
			return result, errors.New("previous receipt changed before publication")
		}
		result.DestinationChanged = true
		if err := atomicInstallJSON(path, next, len(pending.PreviousReceipt) > 0); err != nil {
			return result, err
		}
	}
	result.Phase = "receipt-recorded"
	if err := advance("receipt-recorded"); err != nil {
		return result, err
	}
	if err := checkPendingInstall(pending, raw); err != nil {
		return result, err
	}
	if target, e := installLauncherTarget(p.Launcher); e != nil || target != p.Runtime || !sameInstallBytes(path, next) {
		return result, errors.New("activated launcher or receipt failed final verification")
	}
	if err := os.Remove(result.Pending); err != nil {
		return result, err
	}
	result.DestinationChanged = true
	result.Pending = ""
	if err := syncInstallDirectory(state); err != nil {
		return result, err
	}
	if _, err := observeInstallation(p.Prefix, p.OS, p.Arch); err != nil {
		return result, err
	}
	result.Phase, result.Installed = "complete", true
	return result, nil
}
