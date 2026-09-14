package distribution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/strictjson"
	"golang.org/x/sys/unix"
)

const MaxInstallPlanBytes = 128 << 10
const installNotice = "Install the selected verified CLI in this prefix only; executing it trusts its source. Native memory connections, signets, credentials and shell settings remain unchanged. Existing runtimes are retained. A connection update requires a separate preview and confirmation."

type InstallOptions struct {
	Prefix    string `json:"prefix"`
	Version   string `json:"version,omitempty"`
	Candidate string `json:"candidate,omitempty"`
	Retained  string `json:"retained,omitempty"`
}

type InstallSource struct {
	Kind      string         `json:"kind"`
	Manifest  ParsedManifest `json:"manifest"`
	Published *ReleaseView   `json:"published,omitempty"`
}

// PathObservation pins existing destination directories without treating their
// unrelated contents as owned. Modifying a sibling tool is not installer authority.
type PathObservation struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Device uint64 `json:"device,omitempty"`
	Inode  uint64 `json:"inode,omitempty"`
}

type InstallObservation struct {
	Directories    []PathObservation `json:"directories"`
	ReceiptSHA256  string            `json:"receipt_sha256,omitempty"`
	LauncherTarget string            `json:"launcher_target,omitempty"`
	Current        string            `json:"current,omitempty"`
}

type InstallPlan struct {
	InstallOptions
	FormatVersion   int                `json:"format_version"`
	OS              string             `json:"os"`
	Arch            string             `json:"arch"`
	Source          InstallSource      `json:"source"`
	Binary          Asset              `json:"binary"`
	Launcher        string             `json:"launcher"`
	Runtime         string             `json:"runtime"`
	RuntimeRetained bool               `json:"runtime_retained"`
	Observed        InstallObservation `json:"observed"`
	Notice          string             `json:"notice"`
}

// Receipts recognize cooperative machine-local ownership, not a boundary against
// the same user forging files. Never infer ownership from a directory name alone.
type cliReceipt struct {
	FormatVersion int    `json:"format_version"`
	Product       string `json:"product"`
	Prefix        string `json:"prefix"`
	Current       string `json:"current"`
	Previous      string `json:"previous,omitempty"`
}

func cliState(prefix string) string { return filepath.Join(prefix, "lib", "mandalore") }
func retainedRoot(prefix, manifest, goos, arch string) string {
	return filepath.Join(cliState(prefix), "releases", "sha256-"+manifest, goos+"_"+arch)
}

func safeAbsolute(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && strings.IndexFunc(path, unicode.IsControl) < 0
}

// Resolve only the explicitly selected prefix (including /tmp aliases on macOS).
// All managed descendants are checked separately and must not be redirected.
func canonicalPrefix(path string) (string, error) {
	if path == "" || strings.IndexFunc(path, unicode.IsControl) >= 0 {
		return "", errors.New("a nonempty installation prefix without control characters is required")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if path == filepath.Dir(path) {
		return "", errors.New("a filesystem root is not an installation prefix")
	}
	if _, err := os.Lstat(path); err == nil {
		return filepath.EvalSymlinks(path)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(path)
	// Resolve the closest existing ancestor without creating any directories.
	if _, err := os.Lstat(parent); os.IsNotExist(err) {
		parent, err = canonicalPrefix(parent)
		if err != nil {
			return "", err
		}
	} else {
		parent, err = filepath.EvalSymlinks(parent)
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

func observeDirectory(path string) (PathObservation, error) {
	o := PathObservation{Path: path}
	st, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return o, nil
	}
	if err != nil {
		return o, err
	}
	s, ok := st.Sys().(*syscall.Stat_t)
	if !st.IsDir() || !ok || s.Uid != uint32(os.Geteuid()) || st.Mode().Perm()&0022 != 0 {
		return o, errors.New("installation directories must be real, owned by the current user and not writable by other users")
	}
	o.Exists, o.Device, o.Inode = true, uint64(s.Dev), s.Ino
	return o, nil
}

func openInstallDirectory(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, errors.New("installation directory is unavailable or redirected")
	}
	return os.NewFile(uintptr(fd), path), nil
}

func retainedManifest(prefix, identity, goos, arch string) (ParsedManifest, error) {
	if !validHex(identity, 64) || !supportedTarget(goos, arch) {
		return ParsedManifest{}, errors.New("invalid retained runtime identity or target")
	}
	root := retainedRoot(prefix, identity, goos, arch)
	for _, path := range []string{filepath.Join(cliState(prefix), "releases"), filepath.Dir(root), root} {
		o, err := observeDirectory(path)
		if err != nil || !o.Exists {
			return ParsedManifest{}, errors.New("retained runtime directory is missing, redirected or unowned")
		}
	}
	dir, err := openInstallDirectory(root)
	if err != nil {
		return ParsedManifest{}, err
	}
	defer dir.Close()
	b, err := readCandidateFile(dir, "manifest.json", MaxManifestBytes)
	if err != nil {
		return ParsedManifest{}, err
	}
	p, err := ParseManifest(b)
	if err != nil || p.SHA256 != identity {
		return ParsedManifest{}, errors.New("retained manifest no longer matches its receipt identity")
	}
	a, err := p.Manifest.Binary(goos, arch)
	if err != nil {
		return ParsedManifest{}, err
	}
	f, err := openCandidateFile(dir, "mandalore", a.Size)
	if err != nil {
		return ParsedManifest{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.Mode().Perm()&0111 == 0 || st.Mode().Perm()&0022 != 0 {
		return ParsedManifest{}, errors.New("retained runtime must be executable and not writable by other users")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, a.Size+1))
	if err != nil || n != a.Size || hex.EncodeToString(h.Sum(nil)) != a.SHA256 {
		return ParsedManifest{}, errors.New("retained runtime does not match its manifest")
	}
	return p, nil
}

func observeInstallation(prefix, goos, arch string) (InstallObservation, error) {
	o := InstallObservation{Directories: []PathObservation{}}
	state := cliState(prefix)
	for _, path := range []string{prefix, filepath.Join(prefix, "bin"), filepath.Join(prefix, "lib"), state} {
		d, err := observeDirectory(path)
		if err != nil {
			return o, err
		}
		o.Directories = append(o.Directories, d)
	}
	launcher := filepath.Join(prefix, "bin", "mandalore")
	lst, lerr := os.Lstat(launcher)
	if lerr != nil && !os.IsNotExist(lerr) {
		return o, lerr
	}
	if !o.Directories[len(o.Directories)-1].Exists {
		if lerr == nil {
			return o, errors.New("launcher exists without a verified Mandalore ownership receipt; refusing takeover")
		}
		return o, nil
	}
	if _, err := os.Lstat(filepath.Join(state, "pending.json")); !os.IsNotExist(err) {
		return o, errors.New("installation has pending or unreadable activation state; inspect before retrying")
	}
	dir, err := openInstallDirectory(state)
	if err != nil {
		return o, err
	}
	defer dir.Close()
	b, err := readCandidateFile(dir, "receipt.json", MaxManifestBytes)
	if err != nil {
		return o, errors.New("existing installation state has no readable regular ownership receipt; refusing takeover")
	}
	var r cliReceipt
	if err := strictjson.Decode(b, &r, MaxManifestBytes); err != nil || r.FormatVersion != 1 || r.Product != "mandalore" || r.Prefix != prefix || !validHex(r.Current, 64) || (r.Previous != "" && !validHex(r.Previous, 64)) {
		return o, errors.New("installation receipt is invalid or belongs to another prefix")
	}
	if _, err := retainedManifest(prefix, r.Current, goos, arch); err != nil {
		return o, err
	}
	want := filepath.Join(retainedRoot(prefix, r.Current, goos, arch), "mandalore")
	if lerr != nil || lst.Mode()&os.ModeSymlink == 0 {
		return o, errors.New("owned launcher is missing or replaced; inspect before retrying")
	}
	target, err := os.Readlink(launcher)
	if err != nil || target != want {
		return o, errors.New("launcher target disagrees with the verified installation receipt")
	}
	o.ReceiptSHA256, o.LauncherTarget, o.Current = Digest(b), target, r.Current
	return o, nil
}

func (o InstallOptions) Validate() error {
	if o.Prefix == "" || o.Prefix == string(filepath.Separator) || strings.IndexFunc(o.Prefix, unicode.IsControl) >= 0 {
		return errors.New("a non-root installation prefix without control characters is required")
	}
	n := 0
	for _, s := range []string{o.Version, o.Candidate, o.Retained} {
		if s != "" {
			n++
		}
	}
	if n > 1 || (o.Version != "" && !ValidVersion(o.Version)) || (o.Retained != "" && !validHex(o.Retained, 64)) {
		return errors.New("select at most one canonical version, local candidate or retained manifest SHA-256")
	}
	return nil
}

func PlanInstall(ctx context.Context, o InstallOptions) (InstallPlan, error) {
	return planInstall(ctx, o, NewReleaseClient())
}

// Preview reads source and destination; it never creates a prefix, downloads or
// executes a binary, takes a write lock, changes a launcher or opens a signet.
func planInstall(ctx context.Context, o InstallOptions, client *ReleaseClient) (InstallPlan, error) {
	if err := ctx.Err(); err != nil {
		return InstallPlan{}, err
	}
	if err := o.Validate(); err != nil {
		return InstallPlan{}, err
	}
	prefix, err := canonicalPrefix(o.Prefix)
	if err != nil || prefix == filepath.Dir(prefix) {
		return InstallPlan{}, errors.New("installation prefix cannot be resolved to a non-root path")
	}
	o.Prefix = prefix
	p := InstallPlan{InstallOptions: o, FormatVersion: 1, OS: runtime.GOOS, Arch: runtime.GOARCH, Notice: installNotice}
	if !supportedTarget(p.OS, p.Arch) {
		return InstallPlan{}, errors.New("this machine has no supported release target")
	}
	p.Observed, err = observeInstallation(prefix, p.OS, p.Arch)
	if err != nil {
		return InstallPlan{}, err
	}
	switch {
	case o.Candidate != "":
		p.Candidate, err = filepath.Abs(o.Candidate)
		if err == nil {
			p.Candidate, err = filepath.EvalSymlinks(p.Candidate)
		}
		if err != nil || !safeAbsolute(p.Candidate) {
			return InstallPlan{}, errors.New("local candidate directory cannot be resolved")
		}
		if p.Candidate == prefix || strings.HasPrefix(p.Candidate, prefix+string(filepath.Separator)) || strings.HasPrefix(prefix, p.Candidate+string(filepath.Separator)) {
			return InstallPlan{}, errors.New("local candidate and installation prefix must not overlap; use retained selection for rollback")
		}
		p.Source.Kind = "local-candidate"
		p.Source.Manifest, err = VerifyDirectory(p.Candidate)
	case o.Retained != "":
		if p.Observed.Current == "" {
			return InstallPlan{}, errors.New("retained selection requires a verified owned installation")
		}
		p.Source.Kind = "retained"
		p.Source.Manifest, err = retainedManifest(prefix, o.Retained, p.OS, p.Arch)
	default:
		p.Source.Kind = "github-release"
		var r ReleaseView
		r, err = client.Inspect(ctx, o.Version)
		if err == nil {
			p.Source.Manifest = r.Manifest
			p.Source.Published = &r
			p.Version = r.Manifest.Manifest.Version
		}
	}
	if err != nil {
		return InstallPlan{}, err
	}
	p.Binary, err = p.Source.Manifest.Manifest.Binary(p.OS, p.Arch)
	if err != nil {
		return InstallPlan{}, err
	}
	p.Launcher = filepath.Join(prefix, "bin", "mandalore")
	p.Runtime = filepath.Join(retainedRoot(prefix, p.Source.Manifest.SHA256, p.OS, p.Arch), "mandalore")
	// A pre-existing target is reusable only if its bytes and managed ancestry
	// match the exact selected identity. An absent target must have safe ancestors.
	for _, path := range []string{filepath.Join(cliState(prefix), "releases"), filepath.Dir(filepath.Dir(p.Runtime)), filepath.Dir(p.Runtime)} {
		d, e := observeDirectory(path)
		if e != nil {
			return InstallPlan{}, e
		}
		p.Observed.Directories = append(p.Observed.Directories, d)
	}
	if p.Observed.Directories[len(p.Observed.Directories)-1].Exists {
		retained, e := retainedManifest(prefix, p.Source.Manifest.SHA256, p.OS, p.Arch)
		if e != nil || !reflect.DeepEqual(retained, p.Source.Manifest) {
			return InstallPlan{}, errors.New("existing retained target disagrees with selected source; refusing replacement")
		}
		p.RuntimeRetained = true
	}
	if err := ctx.Err(); err != nil {
		return InstallPlan{}, err
	}
	return p, nil
}

// Parsing checks shape and derived effects, but is not apply authorization.
// Apply must regenerate the plan from fresh verified source/destination state.
func ParseInstallPlan(b []byte) (InstallPlan, error) {
	var p InstallPlan
	if err := strictjson.Decode(b, &p, MaxInstallPlanBytes); err != nil {
		return InstallPlan{}, err
	}
	if err := p.InstallOptions.Validate(); err != nil {
		return InstallPlan{}, err
	}
	if p.FormatVersion != 1 || !safeAbsolute(p.Prefix) || p.Prefix == filepath.Dir(p.Prefix) || !supportedTarget(p.OS, p.Arch) || p.Notice != installNotice || !validHex(p.Source.Manifest.SHA256, 64) {
		return InstallPlan{}, errors.New("invalid installation plan format, destination, target or effects")
	}
	a, err := p.Source.Manifest.Manifest.Binary(p.OS, p.Arch)
	if err != nil || a != p.Binary || p.Launcher != filepath.Join(p.Prefix, "bin", "mandalore") || p.Runtime != filepath.Join(retainedRoot(p.Prefix, p.Source.Manifest.SHA256, p.OS, p.Arch), "mandalore") {
		return InstallPlan{}, errors.New("installation plan paths or binary disagree with its manifest")
	}
	switch p.Source.Kind {
	case "local-candidate":
		if !safeAbsolute(p.Candidate) || p.Source.Published != nil {
			return InstallPlan{}, errors.New("invalid local candidate selection")
		}
	case "retained":
		if p.Retained != p.Source.Manifest.SHA256 || p.Source.Published != nil {
			return InstallPlan{}, errors.New("invalid retained selection")
		}
	case "github-release":
		if p.Version != p.Source.Manifest.Manifest.Version || p.Source.Published == nil || !reflect.DeepEqual(p.Source.Published.Manifest, p.Source.Manifest) {
			return InstallPlan{}, errors.New("invalid published selection")
		}
	default:
		return InstallPlan{}, errors.New("unknown installation source")
	}
	return p, nil
}
