// Package migration converts an explicitly selected legacy memory-only bank.
// It never executes the predecessor or activates a writer.
package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

const MaxFileBytes = 4 << 20
const MaxSnapshotBytes = 64 << 20
const MaxFiles = 10000

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,127}$`)

type Options struct {
	Source       string `json:"source"`
	Output       string `json:"output"`
	DeviceLabel  string `json:"device_label"`
	Actor        string `json:"actor"`
	Binding      string `json:"legacy_binding,omitempty"`
	NativeHome   string `json:"native_home,omitempty"`
	NativeBinary string `json:"native_binary,omitempty"`
}

type Counts struct {
	Files             int `json:"files"`
	Directories       int `json:"directories"`
	Bytes             int `json:"bytes"`
	Devices           int `json:"devices"`
	Sources           int `json:"sources"`
	Revisions         int `json:"revisions"`
	Records           int `json:"records"`
	ConflictedRecords int `json:"conflicted_records"`
	Journal           int `json:"journal"`
	SourceChanges     int `json:"source_changes"`
}

type WriterObservations struct {
	Lock   string                  `json:"legacy_lock"`
	Native install.MemoryInventory `json:"native"`
	Notice string                  `json:"notice"`
}

type Plan struct {
	Options
	Version       int                `json:"schema_version"`
	SourceID      string             `json:"source_id"`
	SourceSHA256  string             `json:"source_sha256"`
	BindingSHA256 string             `json:"binding_sha256,omitempty"`
	Counts        Counts             `json:"counts"`
	Excluded      []string           `json:"excluded"`
	Writers       WriterObservations `json:"writers"`
	Notice        string             `json:"notice"`
}

type snapshot struct {
	original    map[string][]byte
	converted   map[string][]byte
	decoded     memory.Snapshot
	excluded    []string
	counts      Counts
	digest      string
	directories []string
}

func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func filesDigest(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		fmt.Fprintf(h, "%d:%s:%d:", len(name), name, len(files[name]))
		_, _ = h.Write(files[name])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func inside(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Resolve only ancestors, not a final symlink. Output's parent must already
// exist, so even a successful preflight creates no directories.
func canonical(path string) (string, error) {
	if !filepath.IsAbs(path) || len(path) > 4096 || strings.IndexFunc(path, unicode.IsControl) >= 0 {
		return "", errors.New("explicit absolute paths without control characters required")
	}
	path = filepath.Clean(path)
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	path = filepath.Join(parent, filepath.Base(path))
	if st, err := os.Lstat(path); err == nil && st.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("final path must not be a symlink")
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return path, nil
}

func normalize(o Options) (Options, error) {
	if strings.TrimSpace(o.DeviceLabel) == "" || strings.TrimSpace(o.Actor) == "" || len(o.DeviceLabel) > 256 || len(o.Actor) > 256 || strings.IndexFunc(o.DeviceLabel+o.Actor, unicode.IsControl) >= 0 {
		return o, errors.New("explicit one-line conversion device label and actor required, at most 256 bytes each")
	}
	for _, p := range []*string{&o.Source, &o.Output, &o.Binding, &o.NativeHome, &o.NativeBinary} {
		if *p == "" && p != &o.Source && p != &o.Output {
			continue
		}
		var err error
		*p, err = canonical(*p)
		if err != nil {
			return o, err
		}
	}
	if inside(o.Source, o.Output) || inside(o.Output, o.Source) {
		return o, errors.New("source and output must not overlap")
	}
	if o.Binding != "" && (inside(o.Source, o.Binding) || inside(o.Output, o.Binding)) {
		return o, errors.New("legacy binding must remain outside source and output")
	}
	if (o.NativeHome == "") != (o.NativeBinary == "") {
		return o, errors.New("native inventory requires both explicit profile and binary")
	}
	for _, p := range []string{o.NativeHome, o.NativeBinary} {
		if p != "" && (inside(o.Source, p) || inside(o.Output, p) || inside(p, o.Source) || inside(p, o.Output)) {
			return o, errors.New("native inventory paths must not overlap source or output")
		}
	}
	if _, err := os.Lstat(o.Output); !os.IsNotExist(err) {
		return o, errors.New("output already exists or cannot be inspected; choose a new bundle path")
	}
	return o, nil
}

// readFile never follows the final component and refuses special files before
// opening. O_NONBLOCK also prevents a raced FIFO from hanging a preflight.
func readFile(path string, limit int) ([]byte, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > int64(limit) {
		return nil, errors.New("nonregular or oversized file")
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(st, opened) || !opened.Mode().IsRegular() {
		return nil, errors.New("file changed while opening snapshot")
	}
	b, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(b) > limit {
		return nil, errors.New("file exceeds snapshot limit")
	}
	return b, nil
}

type sourceLock struct {
	file  *os.File
	path  string
	state string
	info  fs.FileInfo
}

func lockSource(root string) (*sourceLock, error) {
	st, err := os.Lstat(root)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("legacy source must be a real directory")
	}
	local := filepath.Join(root, ".my-friday")
	if st, err := os.Lstat(local); err == nil {
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("legacy local state must be a real directory")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	path := filepath.Join(local, "write.lock")
	lock := &sourceLock{path: path, state: "absent"}
	st, err = os.Lstat(path)
	if os.IsNotExist(err) {
		return lock, nil
	}
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, errors.New("legacy lock is not a regular file")
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	lock.file, lock.info, lock.state = os.NewFile(uintptr(fd), path), st, "available"
	if err := syscall.Flock(fd, syscall.LOCK_SH|syscall.LOCK_NB); err != nil {
		lock.close()
		return nil, errors.New("legacy writer lock busy; stop writers and preflight again")
	}
	opened, err := lock.file.Stat()
	if err != nil || !os.SameFile(st, opened) {
		lock.close()
		return nil, errors.New("legacy lock changed while opening")
	}
	if err := lock.check(); err != nil {
		lock.close()
		return nil, err
	}
	return lock, nil
}

func (l *sourceLock) close() {
	if l.file != nil {
		_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
		_ = l.file.Close()
	}
}
func (l *sourceLock) check() error {
	st, err := os.Lstat(l.path)
	if l.state == "absent" && os.IsNotExist(err) {
		return nil
	}
	if err != nil || l.info == nil || !os.SameFile(st, l.info) || !st.Mode().IsRegular() {
		return errors.New("legacy lock identity changed; preflight again")
	}
	return nil
}

func Preflight(ctx context.Context, options Options) (Plan, error) {
	o, err := normalize(options)
	if err != nil {
		return Plan{}, err
	}
	lock, err := lockSource(o.Source)
	if err != nil {
		return Plan{}, err
	}
	defer lock.close()
	p, _, err := inspect(ctx, o, lock)
	return p, err
}

func inspect(ctx context.Context, o Options, lock *sourceLock) (Plan, *snapshot, error) {
	p := Plan{Options: o, Version: 1}
	s, err := readSnapshot(ctx, o.Source)
	if err != nil {
		return p, nil, err
	}
	p.SourceID, p.SourceSHA256, p.Counts, p.Excluded = s.decoded.Signet.ID, s.digest, s.counts, s.excluded
	p.BindingSHA256, err = checkBinding(o, s.decoded)
	if err != nil {
		return p, nil, err
	}
	p.Writers = WriterObservations{Lock: lock.state, Native: install.MemoryInventory{Status: "not-tested", Plugins: []install.MemoryPlugin{}}, Notice: "Lock availability and installed plugins do not prove idle sessions, same-bank targeting or quiescence on other machines. Stop all writers before apply."}
	if o.NativeHome != "" {
		p.Writers.Native, err = install.InspectMemoryPlugins(ctx, o.NativeHome, o.NativeBinary)
		if err != nil {
			return p, nil, err
		}
	}
	again, err := readSnapshot(ctx, o.Source)
	if err != nil {
		return p, nil, err
	}
	if again.digest != s.digest || !sameStrings(again.excluded, s.excluded) {
		return p, nil, errors.New("source changed during preflight; stop writers and retry")
	}
	if err := lock.check(); err != nil {
		return p, nil, err
	}
	if digest, err := checkBinding(o, s.decoded); err != nil || digest != p.BindingSHA256 {
		return p, nil, errors.New("legacy binding changed during preflight")
	}
	p.Notice = "Supported source: My Friday memory-only bank.json format 1. Portable snapshot limits: 4 MiB/file, 64 MiB total, 10000 files. Output retains original portable bytes, not Git history, local state or an off-device backup. Preserve the untouched source. Apply does not activate a writer, initialize Git, copy credentials or synchronize."
	if len(marshal(p)) > 24<<10 {
		return p, nil, errors.New("migration plan exceeds 24 KiB report limit; reduce optional inventory or path lengths")
	}
	return p, s, nil
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func checkBinding(o Options, s memory.Snapshot) (string, error) {
	if o.Binding == "" {
		return "", nil
	}
	b, err := readFile(o.Binding, 16<<10)
	if err != nil {
		return "", err
	}
	var value struct {
		Version  int    `json:"schema_version"`
		BankID   string `json:"bank_id"`
		Root     string `json:"root"`
		DeviceID string `json:"device_id"`
		Actor    string `json:"actor"`
	}
	if err := strictjson.Decode(b, &value, 16<<10); err != nil {
		return "", errors.New("invalid legacy binding format")
	}
	root, err := canonical(value.Root)
	if err != nil || root != o.Source || value.Version != 1 || value.BankID != s.Signet.ID || strings.TrimSpace(value.Actor) == "" {
		return "", errors.New("legacy binding does not identify the selected source")
	}
	for _, d := range s.Devices {
		if d.ID == value.DeviceID {
			return hash(b), nil
		}
	}
	return "", errors.New("legacy binding device is missing from the selected source")
}

// Marshal helpers retain JSON numbers through json.RawMessage in conversion;
// typed validation results are never written back as imported source objects.
func marshal(value any) []byte {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return append(b, '\n')
}
