package distribution

import (
	"archive/tar"
	"bytes"
	"context"
	"debug/buildinfo"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"
)

const PinnedGo = "go1.26.4"
const maxSourceBytes = 64 << 20

type BuildOptions struct{ Source, Output string }

type buildOutput struct {
	data  bytes.Buffer
	limit int
}

func (b *buildOutput) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.data.Len() {
		return 0, errors.New("build process output limit exceeded")
	}
	return b.data.Write(p)
}

func runBuildTool(ctx context.Context, dir string, env []string, limit int, binary string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir, cmd.Env = dir, env
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
	var out buildOutput
	out.limit = limit
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("build process cancelled: %w", ctx.Err())
		}
		return nil, errors.New("build process failed or exceeded its output limit; raw output suppressed")
	}
	return out.data.Bytes(), nil
}

// unpackSource writes only a new owned directory. Reject links, special files,
// duplicate names, hidden Git state and oversized expansions before extraction.
func unpackSource(data []byte, root string) error {
	if len(data) > maxSourceBytes {
		return errors.New("source export exceeds size limit")
	}
	r := tar.NewReader(bytes.NewReader(data))
	seen := map[string]bool{}
	var total int64
	for {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.New("invalid source archive")
		}
		if h.Typeflag == tar.TypeXGlobalHeader {
			continue
		} // Git's commit comment; not a file.
		name := strings.TrimSuffix(h.Name, "/")
		if !fs.ValidPath(name) || name == "." || len(name) > 1024 || strings.Contains(name, "\\") || strings.IndexFunc(name, unicode.IsControl) >= 0 || seen[name] || len(seen) >= 8192 {
			return errors.New("invalid source export name or inventory")
		}
		for _, part := range strings.Split(name, "/") {
			if part == ".git" {
				return errors.New("Git state must not enter source export")
			}
		}
		seen[name] = true
		path := filepath.Join(root, filepath.FromSlash(name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0700); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if h.Size < 0 || h.Size > maxSourceBytes-total {
				return errors.New("source export expansion exceeds limit")
			}
			total += h.Size
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				return err
			}
			mode := os.FileMode(0600)
			if h.Mode&0111 != 0 {
				mode = 0700
			}
			f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			n, copyErr := io.Copy(f, io.LimitReader(r, h.Size+1))
			closeErr := f.Close()
			if copyErr != nil || closeErr != nil || n != h.Size {
				return errors.New("source export write failed")
			}
		default:
			return errors.New("source export must not contain links or special files")
		}
	}
	if len(seen) == 0 {
		return errors.New("source export is empty")
	}
	return nil
}

func buildFlags(version, commit string) string {
	return "-buildid= -X main.version=" + version + " -X main.sourceCommit=" + commit
}

// VerifyBuiltBinary reads Go metadata without executing a possibly foreign target.
// It checks target/toolchain settings, not publisher authority or native behavior.
// Go omits linker flags with -trimpath. Source/version provenance comes from the
// controlled source export/compile and manifest, with runtime reporting tested
// separately; do not pretend absent linker flags are inspectable evidence.
func VerifyBuiltBinary(path string, m Manifest, a Asset) error {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return errors.New("release CLI is not an inspectable Go executable")
	}
	if info.GoVersion != m.GoVersion || info.Path != "github.com/acoz-labs/mandalore/cmd/mandalore" || info.Main.Path != "github.com/acoz-labs/mandalore" {
		return errors.New("release CLI build provenance is inconsistent")
	}
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	for key, want := range map[string]string{"GOOS": a.OS, "GOARCH": a.Arch, "CGO_ENABLED": "0", "-trimpath": "true", "-buildmode": "exe", "-compiler": "gc"} {
		if settings[key] != want {
			return fmt.Errorf("release CLI build setting %s is inconsistent", key)
		}
	}
	for _, dep := range info.Deps {
		if dep.Replace != nil {
			return errors.New("release CLI must not use replaced modules")
		}
	}
	return nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Build produces only a local candidate. Source must be clean and explicitly
// selected, output must be absent and outside it, and compilation uses a private
// export/cache with sealed Go settings. No Git tag, release or native install.
func Build(ctx context.Context, o BuildOptions) (ParsedManifest, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	source, err := filepath.Abs(o.Source)
	if err != nil {
		return ParsedManifest{}, err
	}
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		return ParsedManifest{}, err
	}
	output, err := filepath.Abs(o.Output)
	if err != nil {
		return ParsedManifest{}, err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(output))
	if err != nil {
		return ParsedManifest{}, err
	}
	output = filepath.Join(parent, filepath.Base(output))
	if within(source, output) || within(output, source) {
		return ParsedManifest{}, errors.New("candidate output and source must not overlap")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return ParsedManifest{}, errors.New("candidate output must be absent")
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return ParsedManifest{}, errors.New("Git is required for exact-source builds")
	}
	gitEnv := []string{"PATH=" + os.Getenv("PATH"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C"}
	gitRun := func(limit int, args ...string) ([]byte, error) {
		base := []string{"-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-c", "core.attributesFile=/dev/null", "-c", "maintenance.auto=false", "-c", "gc.auto=0"}
		return runBuildTool(ctx, source, gitEnv, limit, git, append(base, args...)...)
	}
	status, err := gitRun(1<<20, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil || len(status) != 0 {
		return ParsedManifest{}, errors.New("release build requires a clean committed source tree")
	}
	top, err := gitRun(4096, "rev-parse", "--show-toplevel")
	if err != nil || strings.TrimSpace(string(top)) != source {
		return ParsedManifest{}, errors.New("select the source repository root explicitly")
	}
	sha, err := gitRun(128, "rev-parse", "--verify", "HEAD")
	commit := strings.TrimSpace(string(sha))
	if err != nil || !validHex(commit, 40) {
		return ParsedManifest{}, errors.New("source commit is invalid")
	}
	goBinary, err := exec.LookPath("go")
	if err != nil {
		return ParsedManifest{}, errors.New("Go 1.26.4 is required")
	}
	stage, err := os.MkdirTemp(parent, ".mandalore-build-")
	if err != nil {
		return ParsedManifest{}, err
	}
	defer os.RemoveAll(stage) // Only this invocation's newly created export/cache.
	work, payload := filepath.Join(stage, "source"), filepath.Join(stage, "payload")
	for _, dir := range []string{work, payload, filepath.Join(stage, "gotmp")} {
		if err := os.Mkdir(dir, 0700); err != nil {
			return ParsedManifest{}, err
		}
	}
	goEnv := []string{"PATH=" + os.Getenv("PATH"), "LC_ALL=C", "GOENV=off", "GOTOOLCHAIN=local", "GOWORK=off", "CGO_ENABLED=0", "GOTELEMETRY=off",
		"GOCACHE=" + filepath.Join(stage, "cache"), "GOMODCACHE=" + filepath.Join(stage, "modules"), "GOPATH=" + filepath.Join(stage, "gopath"), "GOTMPDIR=" + filepath.Join(stage, "gotmp"),
		"GOPROXY=https://proxy.golang.org", "GOSUMDB=sum.golang.org"}
	v, err := runBuildTool(ctx, work, goEnv, 256, goBinary, "env", "GOVERSION")
	if err != nil || strings.TrimSpace(string(v)) != PinnedGo {
		return ParsedManifest{}, errors.New("release builder requires exactly Go 1.26.4")
	}
	archive, err := gitRun(maxSourceBytes, "archive", "--format=tar", commit)
	if err != nil {
		return ParsedManifest{}, err
	}
	if err := unpackSource(archive, work); err != nil {
		return ParsedManifest{}, err
	}
	versionBytes, err := os.ReadFile(filepath.Join(work, "VERSION"))
	releaseVersion := strings.TrimSpace(string(versionBytes))
	if err != nil || !ValidVersion(releaseVersion) {
		return ParsedManifest{}, errors.New("tracked VERSION must contain a canonical release version")
	}
	p, err := PreparePlugin(os.DirFS(filepath.Join(work, "plugins", "codex")), releaseVersion)
	if err != nil {
		return ParsedManifest{}, err
	}
	for name, data := range p.Files {
		if err := os.WriteFile(filepath.Join(work, "plugins", "codex", filepath.FromSlash(name)), data, pluginMode(name)); err != nil {
			return ParsedManifest{}, err
		}
	}
	// The source export is bounded, tracked and symlink-free. Stamp the Pi
	// manifest here as well, so every target embeds this release's identity.
	piManifest := filepath.Join(work, "plugins", "pi", "package", "package.json")
	piSource, err := os.ReadFile(piManifest)
	if err != nil {
		return ParsedManifest{}, err
	}
	piStamped, err := preparePiManifest(piSource, releaseVersion)
	if err != nil {
		return ParsedManifest{}, err
	}
	if err := os.WriteFile(piManifest, piStamped, 0600); err != nil {
		return ParsedManifest{}, err
	}
	m := Manifest{FormatVersion: 1, Product: "mandalore", Version: releaseVersion, Tag: "v" + releaseVersion, SourceCommit: commit, GoVersion: PinnedGo, ProtocolVersion: 1, SignetReadVersions: []int{1, 2}, SignetWriteVersions: []int{1, 2}, PluginSHA256: p.SHA256}
	for _, target := range [][2]string{{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}} {
		a := Asset{Kind: "cli", OS: target[0], Arch: target[1], Name: "mandalore_" + releaseVersion + "_" + target[0] + "_" + target[1]}
		file := filepath.Join(payload, a.Name)
		env := append(append([]string{}, goEnv...), "GOOS="+a.OS, "GOARCH="+a.Arch)
		_, err := runBuildTool(ctx, work, env, 1<<20, goBinary, "build", "-mod=readonly", "-modcacherw", "-trimpath", "-buildvcs=false", "-ldflags", buildFlags(releaseVersion, commit), "-o", file, "./cmd/mandalore")
		if err != nil {
			return ParsedManifest{}, fmt.Errorf("compile %s/%s: %w", a.OS, a.Arch, err)
		}
		if err := VerifyBuiltBinary(file, m, a); err != nil {
			return ParsedManifest{}, err
		}
		b, err := os.ReadFile(file)
		if err != nil || len(b) > MaxBinaryBytes {
			return ParsedManifest{}, errors.New("built executable exceeds limit or cannot be read")
		}
		a.Size, a.SHA256 = int64(len(b)), Digest(b)
		m.Assets = append(m.Assets, a)
	}
	bootstrap, err := os.ReadFile(filepath.Join(work, "packaging", "install.sh"))
	if err != nil {
		return ParsedManifest{}, errors.New("tracked bootstrap is missing")
	}
	for _, item := range []struct {
		kind, name string
		data       []byte
	}{{"codex-plugin", "mandalore_" + releaseVersion + "_codex.zip", p.Archive}, {"bootstrap", "install.sh", bootstrap}} {
		if err := os.WriteFile(filepath.Join(payload, item.name), item.data, 0600); err != nil {
			return ParsedManifest{}, err
		}
		m.Assets = append(m.Assets, Asset{Kind: item.kind, Name: item.name, Size: int64(len(item.data)), SHA256: Digest(item.data)})
	}
	manifest, err := EncodeManifest(m)
	if err != nil {
		return ParsedManifest{}, err
	}
	sums, err := Checksums(manifest)
	if err != nil {
		return ParsedManifest{}, err
	}
	for name, b := range map[string][]byte{"manifest.json": manifest, "SHA256SUMS": sums} {
		if err := os.WriteFile(filepath.Join(payload, name), b, 0600); err != nil {
			return ParsedManifest{}, err
		}
	}
	verified, err := VerifyDirectory(payload)
	if err != nil {
		return ParsedManifest{}, err
	}
	status, err = gitRun(1<<20, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil || len(status) != 0 {
		return ParsedManifest{}, errors.New("source changed during build; candidate not published")
	}
	finalSHA, err := gitRun(128, "rev-parse", "--verify", "HEAD")
	if err != nil || strings.TrimSpace(string(finalSHA)) != commit {
		return ParsedManifest{}, errors.New("source commit changed during build")
	}
	if err := ctx.Err(); err != nil {
		return ParsedManifest{}, err
	}
	if err := publishCandidate(payload, output); err != nil {
		return ParsedManifest{}, errors.New("candidate output appeared or could not be published; no existing directory was overwritten")
	}
	return verified, nil
}
