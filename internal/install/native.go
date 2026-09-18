package install

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

type runner func(context.Context, Options, ...string) ([]byte, error)
type probeFunc func(context.Context, Plan) error

// Do not promote bytes.Buffer.ReadFrom: io.Copy could otherwise bypass Write.
type boundedOutput struct {
	buffer   bytes.Buffer
	exceeded bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.buffer.Len()+len(p) > 1<<20 {
		b.exceeded = true
		return 0, errors.New("process output exceeded limit")
	}
	return b.buffer.Write(p)
}

func execute(ctx context.Context, binary, dir string, env []string, input io.Reader, args ...string) ([]byte, error) {
	raw, err := executeBounded(ctx, binary, dir, env, input, args...)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// Only typed delegation may inspect bounded stdout after an ordinary nonzero
// exit. Existing native commands still discard all failed raw output above.
func executeBounded(ctx context.Context, binary, dir string, env []string, input io.Reader, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir, cmd.Env, cmd.Stdin = dir, env, input
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = time.Second
	var out boundedOutput
	cmd.Stdout = &out
	// Stderr is never retained. Typed callers may decode bounded failed stdout,
	// but an ExitError must not hide a simultaneous output-copy limit failure.
	err := cmd.Run()
	if out.exceeded {
		return nil, errors.New("selected process exceeded its output limit; raw output suppressed")
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("selected process cancelled: %w", ctx.Err())
		}
		var exited *exec.ExitError
		if errors.As(err, &exited) {
			return out.buffer.Bytes(), errors.New("selected process exited unsuccessfully")
		}
		return nil, errors.New("selected process failed, timed out or exceeded its output limit; raw output suppressed")
	}
	return out.buffer.Bytes(), nil
}

func environment(overrides map[string]string) []string {
	env := []string{}
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if _, replaced := overrides[key]; !replaced {
			env = append(env, item)
		}
	}
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}

func native(ctx context.Context, o Options, args ...string) ([]byte, error) {
	return execute(ctx, o.NativeBinary, filepath.Dir(o.Binding), environment(map[string]string{"CODEX_HOME": o.NativeHome}), nil, args...)
}

func probeRuntime(ctx context.Context, p Plan) error {
	env := environment(map[string]string{"MANDALORE_BINDING": p.Binding})
	raw, err := execute(ctx, p.Runtime, filepath.Dir(p.Binding), env, nil, "version")
	if err != nil {
		return err
	}
	var v struct {
		OK     bool `json:"ok"`
		Result struct {
			Name         string `json:"name"`
			Protocol     int    `json:"protocol_version"`
			HookProtocol int    `json:"codex_hook_protocol"`
			OS           string `json:"os"`
			Arch         string `json:"arch"`
		} `json:"result"`
	}
	if decodeNative(raw, &v) != nil || !v.OK || v.Result.Name != "mandalore" || v.Result.Protocol != 1 || v.Result.HookProtocol != 1 || v.Result.OS != runtime.GOOS || v.Result.Arch != runtime.GOARCH {
		return errors.New("selected runtime does not declare a compatible Mandalore, hook and machine interface")
	}
	raw, err = execute(ctx, p.Runtime, filepath.Dir(p.Binding), env, strings.NewReader(`{"hook_event_name":"SessionStart"}`), "codex-memory-hook")
	if err != nil {
		return err
	}
	var hook struct {
		Warning string `json:"systemMessage"`
		Context struct {
			Event string `json:"hookEventName"`
			Text  string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if len(raw) > 16384 || decodeNative(raw, &hook) != nil || hook.Warning != "" || hook.Context.Event != "SessionStart" || hook.Context.Text == "" {
		return errors.New("selected runtime hook probe did not attach the selected binding; no native registration changed")
	}
	return nil
}

// Validate duplicate keys/bounds while allowing new unrelated native fields.
func decodeNative(raw []byte, result any) error {
	var object map[string]any
	if err := strictjson.Decode(raw, &object, 1<<20); err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}

type marketplace struct {
	Name   string `json:"name"`
	Root   string `json:"root"`
	Source struct {
		Type string `json:"sourceType"`
		Path string `json:"source"`
	} `json:"marketplaceSource"`
}
type plugin struct {
	ID      string `json:"pluginId"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
	Source  struct {
		Type string `json:"sourceType"`
		Path string `json:"source"`
	} `json:"marketplaceSource"`
}

func inventory(ctx context.Context, o Options, run runner) ([]marketplace, []plugin, error) {
	raw, err := run(ctx, o, "plugin", "marketplace", "list", "--json")
	if err != nil {
		return nil, nil, fmt.Errorf("selected Codex profile inventory (plugin marketplace list --json): %w", err)
	}
	var ms struct {
		Items []marketplace `json:"marketplaces"`
	}
	if decodeNative(raw, &ms) != nil || ms.Items == nil {
		return nil, nil, errors.New("unsupported native marketplace response")
	}
	raw, err = run(ctx, o, "plugin", "list", "--json")
	if err != nil {
		return nil, nil, fmt.Errorf("selected Codex profile inventory (plugin list --json): %w", err)
	}
	var ps struct {
		Items []plugin `json:"installed"`
	}
	if decodeNative(raw, &ps) != nil || ps.Items == nil {
		return nil, nil, errors.New("unsupported native plugin response")
	}
	return ms.Items, ps.Items, nil
}

func cacheRoot(p Plan) string {
	return filepath.Join(p.NativeHome, "plugins/cache/mandalore/mandalore", p.Version)
}

func verifyCache(p Plan, allowMissing bool) error {
	r, err := loadReceipt(p.Root)
	if err != nil {
		return err
	}
	files := map[string]string{}
	for name, hash := range r.Files {
		if strings.HasPrefix(name, "plugins/mandalore/") {
			files[strings.TrimPrefix(name, "plugins/mandalore/")] = hash
		}
	}
	root := cacheRoot(p)
	resolved, err := canonical(root)
	if err != nil || resolved != root {
		return errors.New("native cache is redirected; preserve it")
	}
	if _, err := os.Lstat(root); os.IsNotExist(err) && allowMissing {
		return nil
	}
	return verifyTree(root, files, allowMissing)
}

func owned(root string, p Plan, allowMissing bool) (Receipt, error) {
	r, err := loadReceipt(root)
	if err != nil || r.Plan.StateDir != p.StateDir || r.Plan.NativeHome != p.NativeHome || filepath.Dir(root) != filepath.Join(p.StateDir, "connections") {
		return Receipt{}, errors.New("marketplace is owned by another source; nothing replaced")
	}
	if err := verifyTree(root, r.Files, allowMissing); err != nil {
		return Receipt{}, err
	}
	if err := verifyCache(r.Plan, true); err != nil {
		return Receipt{}, err
	}
	if _, err := os.Lstat(r.Plan.Runtime); err == nil {
		if h, e := digest(r.Plan.Runtime); e != nil || h != r.Plan.BinarySHA256 {
			return Receipt{}, errors.New("previous runtime edited; preserve it before recovery")
		}
	} else if !os.IsNotExist(err) || !allowMissing {
		return Receipt{}, errors.New("previous runtime missing or unreadable; preview repair")
	}
	return r, nil
}

func checkCollision(ctx context.Context, p Plan, run runner) (string, error) {
	ms, ps, err := inventory(ctx, p.Options, run)
	if err != nil {
		return "", err
	}
	previous := ""
	for _, m := range ms {
		if m.Name != p.Marketplace {
			continue
		}
		if previous != "" || m.Source.Type != "local" || m.Source.Path != m.Root {
			return "", errors.New("ambiguous or foreign marketplace; nothing replaced")
		}
		if _, err := owned(m.Root, p, p.Generation != ""); err != nil {
			return "", err
		}
		previous = m.Root
	}
	seen := false
	for _, v := range ps {
		if v.Enabled && (v.Name == "my-friday-memory" || v.Name == "my-friday" || v.Name == "mandalore" && v.ID != p.PluginID) {
			return "", errors.New("another active memory integration exists; resolve ownership explicitly before connecting")
		}
		if v.ID != p.PluginID {
			continue
		}
		if seen || v.Source.Type != "local" {
			return "", errors.New("ambiguous or foreign native plugin")
		}
		seen = true
		r, err := owned(v.Source.Path, p, p.Generation != "")
		if err != nil || r.Plan.Version != v.Version {
			return "", errors.New("native plugin source/version is not a verified managed connection")
		}
	}
	return previous, nil
}

type Result struct {
	Connection           Plan   `json:"connection"`
	Installed            bool   `json:"installed"`
	RequiresFreshSession bool   `json:"requires_fresh_session"`
	Phase                string `json:"phase"`
	PreviousRoot         string `json:"previous_marketplace_root,omitempty"`
	Notice               string `json:"notice"`
}

func Apply(ctx context.Context, p Plan) (Result, error) { return apply(ctx, p, native, probeRuntime) }

func apply(ctx context.Context, p Plan, run runner, probe probeFunc) (result Result, resultErr error) {
	return applyAcknowledged(ctx, p, false, run, probe)
}

// ApplyAcknowledged accepts an assertion about this invocation only. It is never
// stored in a plan or receipt and is not proof of automatic session detection.
func ApplyAcknowledged(ctx context.Context, p Plan, sessionsStopped bool) (Result, error) {
	return applyAcknowledged(ctx, p, sessionsStopped, native, probeRuntime)
}

func applyAcknowledged(ctx context.Context, p Plan, sessionsStopped bool, run runner, probe probeFunc) (result Result, resultErr error) {
	result = Result{Connection: p, RequiresFreshSession: true, Phase: "preflight"}
	defer func() {
		if resultErr != nil && result.Phase != "deferred" {
			result.Notice = "Installation is incomplete. Retained copies and completed phases are preserved; inspect before retrying. No automatic rollback or authentication changes were attempted."
		}
	}()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	current, err := Prepare(p.Options)
	if err != nil {
		return result, err
	}
	if current != p {
		return result, errors.New("installation inputs changed after preview; preview again")
	}
	if err := realDirectory(p.StateDir); err != nil {
		return result, err
	}
	lock := filepath.Join(p.StateDir, ".install-lock")
	f, err := os.OpenFile(lock, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return result, errors.New("installation locked; inspect before removing a stale lock")
	}
	st, statErr := f.Stat()
	closeErr := f.Close()
	defer func() {
		if now, err := os.Lstat(lock); err == nil && st != nil && os.SameFile(st, now) {
			_ = os.Remove(lock)
		}
	}()
	if statErr != nil || closeErr != nil {
		return result, errors.New("cannot establish installation lock")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	// Codex resolves CODEX_HOME before even read-only inventory. Prepare only
	// on explicit apply, after plan revalidation and locking; preserve existing
	// profile contents and permissions, including on a later partial failure.
	if err := realDirectory(p.NativeHome); err != nil {
		return result, fmt.Errorf("cannot prepare selected Codex profile: %w", err)
	}
	previous, err := checkCollision(ctx, p, run)
	if err != nil {
		return result, err
	}
	result.PreviousRoot = previous
	verified, occupied, err := replacementState(ctx, p, run)
	if err != nil {
		return result, err
	}
	if verified {
		result.Installed, result.Phase = true, "verified"
		result.Notice = "The exact connection is already installed and verified; no native registration or cache was changed. This does not verify active-session context, hook trust or live MCP."
		return result, nil
	}
	if occupied && !sessionsStopped {
		result.Phase = "deferred"
		result.Notice = "Connection replacement deferred; native registration and cache were not changed. Exit all Codex sessions using this profile (idle is insufficient), then explicitly acknowledge sessions_stopped for this apply. Restart sessions and review hook trust afterward."
		return result, errors.New("connection replacement requires explicit stopped-session acknowledgement")
	}
	if err := stageRuntime(p); err != nil {
		return result, err
	}
	if err := probe(ctx, p); err != nil {
		return result, err
	}
	if err := publishBundle(p); err != nil {
		return result, err
	}
	if err := verifyCache(p, true); err != nil {
		return result, err
	}
	result.Phase = "staged"
	again, err := checkCollision(ctx, p, run)
	if err != nil {
		return result, err
	}
	if again != previous {
		return result, errors.New("native registration changed during preparation; inspect and preview again")
	}
	if previous != "" && previous != p.Root {
		if _, err := run(ctx, p.Options, "plugin", "marketplace", "remove", p.Marketplace, "--json"); err != nil {
			return result, err
		}
		result.Phase = "registration-removed"
	}
	if _, err := run(ctx, p.Options, "plugin", "marketplace", "add", p.Root); err != nil {
		return result, err
	}
	result.Phase = "marketplace-registered"
	if _, err := run(ctx, p.Options, "plugin", "add", p.PluginID, "--json"); err != nil {
		return result, err
	}
	result.Phase = "plugin-installed"
	ms, ps, err := inventory(ctx, p.Options, run)
	if err != nil {
		return result, err
	}
	marketOK, pluginOK := false, false
	for _, m := range ms {
		if m.Name == p.Marketplace && m.Root == p.Root && m.Source.Path == p.Root && m.Source.Type == "local" {
			marketOK = true
		}
	}
	for _, v := range ps {
		if v.ID == p.PluginID && v.Enabled && v.Version == p.Version && v.Source.Path == p.Root && v.Source.Type == "local" {
			pluginOK = true
		}
	}
	if !marketOK || !pluginOK {
		return result, errors.New("native installation did not report the selected source/version enabled")
	}
	if err := verifyCache(p, false); err != nil {
		return result, err
	}
	result.Installed, result.Phase = true, "verified"
	result.Notice = "Connected for fresh Codex sessions. Native plugin cache may have been replaced; restart affected sessions and review native hook trust and MCP startup. Signet contents, synchronization, authentication and shell startup files were not changed. Explicit MANDALORE_BIN/BINDING overrides still take precedence."
	return result, nil
}

// Call only after ownership/collision checks. Inventory alone is not sufficient
// to call a repeat application a no-op: retained source and cache must verify.
func replacementState(ctx context.Context, p Plan, run runner) (verified, occupied bool, err error) {
	ms, ps, err := inventory(ctx, p.Options, run)
	if err != nil {
		return false, false, err
	}
	marketOK, pluginOK := false, false
	for _, m := range ms {
		if m.Name == p.Marketplace {
			occupied = true
			marketOK = m.Root == p.Root && m.Source.Path == p.Root && m.Source.Type == "local"
		}
	}
	for _, v := range ps {
		if v.ID == p.PluginID {
			occupied = true
			pluginOK = v.Enabled && v.Version == p.Version && v.Source.Path == p.Root && v.Source.Type == "local"
		}
	}
	if marketOK && pluginOK {
		if _, e := owned(p.Root, p, false); e == nil && verifyCache(p, false) == nil {
			return true, occupied, nil
		}
	}
	return false, occupied, nil
}
