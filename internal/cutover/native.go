// Native harness commands for finding and disabling 1.x connections, moved
// from the 1.x installer (#163). Every command runs the selected executable
// with bounded output and a timeout.

package cutover

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

func nativeClaude(ctx context.Context, o Options, args ...string) ([]byte, error) {
	return execute(ctx, o.NativeBinary, filepath.Dir(o.Binding), environment(map[string]string{"CLAUDE_CONFIG_DIR": o.NativeHome}), nil, args...)
}

// hermesNativeArguments prevents Hermes's sticky active_profile from redirecting
// a selected root. Named profile homes already bypass that selection in Hermes.
func hermesNativeArguments(home string, args []string) []string {
	if filepath.Base(filepath.Dir(home)) != "profiles" {
		return append([]string{"--profile", "default"}, args...)
	}
	return append([]string{}, args...)
}

func nativeHermes(ctx context.Context, o Options, args ...string) ([]byte, error) {
	return execute(ctx, o.NativeBinary, filepath.Dir(o.Binding), environment(map[string]string{"HERMES_HOME": o.NativeHome, "PYTHONDONTWRITEBYTECODE": "1"}), nil, hermesNativeArguments(o.NativeHome, args)...)
}

type runner func(context.Context, Options, ...string) ([]byte, error)

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

// execute discards all output from a failed command.
func execute(ctx context.Context, binary, dir string, env []string, input io.Reader, args ...string) ([]byte, error) {
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
	// Stderr is never retained. An ExitError must not hide a simultaneous
	// output-copy limit failure.
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

func nativePi(ctx context.Context, o Options, args ...string) ([]byte, error) {
	return execute(ctx, o.NativeBinary, filepath.Dir(o.Binding), environment(map[string]string{"PI_CODING_AGENT_DIR": o.NativeHome}), nil, args...)
}

const maxNativeBinary = 512 << 20

// Options selects one native profile and its harness executable. Commands run
// beside Binding, a path in the profile that need not exist.
type Options struct {
	NativeHome   string
	NativeBinary string
	Binding      string
}

func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// canonical resolves an absent path through real existing ancestors without
// creating anything. Dangling symlinks cannot become installation destinations.
func canonical(path string) (string, error) {
	if !filepath.IsAbs(path) || strings.IndexFunc(path, unicode.IsControl) >= 0 {
		return "", errors.New("absolute paths without control characters required")
	}
	path = filepath.Clean(path)
	if _, err := os.Lstat(path); err == nil {
		return filepath.EvalSymlinks(path)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if filepath.Dir(path) == path {
		return "", errors.New("no existing path ancestor")
	}
	parent, err := canonical(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

func digestLimit(path string, limit int64) (string, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	// Refuse FIFOs/devices before open, which can block even without reading.
	if !st.Mode().IsRegular() || st.Size() < 1 || st.Size() > limit {
		return "", errors.New("executable must be a nonempty regular file within its size limit")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err = f.Stat()
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() || st.Size() < 1 || st.Size() > limit {
		return "", errors.New("executable must be a nonempty regular file within its size limit")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, limit+1))
	if err != nil || n > limit {
		return "", errors.New("runtime hashing failed or exceeded size limit")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func readRegular(path string, limit int64) ([]byte, error) {
	resolved, err := canonical(path)
	if err != nil || resolved != path {
		return nil, errors.New("managed file is redirected")
	}
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > limit {
		return nil, errors.New("managed file type or size is invalid")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, errors.New("managed file read exceeded its limit")
	}
	return b, nil
}
