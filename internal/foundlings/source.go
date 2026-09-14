// Package foundlings owns read-only historical source connections and retrieval.
// Portable knowledge/registration semantics remain in the memory engine.
package foundlings

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/acoz-labs/mandalore/internal/memory"
)

const MaxFileBytes = 4 << 20
const MaxSourceBytes = 64 << 20
const MaxFiles = 10000
const maxEntries = 30000

var ErrChanged = errors.New("reference content or identity changed; inspect and explicitly repin or reconnect before using it")
var ErrUnavailable = errors.New("reference is unavailable or unsupported; inspect the selected local source")

type Observation struct {
	Root          string                 `json:"local_root"`
	Source        memory.FoundlingSource `json:"source"`
	Pin           memory.SourcePin       `json:"pin"`
	ContentSHA256 string                 `json:"content_sha256"`
	Files         int                    `json:"eligible_files"`
	Bytes         int                    `json:"eligible_bytes"`
	Excluded      int                    `json:"excluded_entries"`
	Notice        string                 `json:"notice"`
}

type snapshot struct {
	View      Observation
	Documents map[string][]byte
}

func hash(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }

func documentDigest(documents map[string][]byte) string {
	names := make([]string, 0, len(documents))
	for name := range documents {
		names = append(names, name)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		fmt.Fprintf(h, "%d:%s:%d:", len(name), name, len(documents[name]))
		_, _ = h.Write(documents[name])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func canonicalDirectory(root string) (string, error) {
	if !filepath.IsAbs(root) || len(root) > 4096 || strings.IndexFunc(root, unicode.IsControl) >= 0 || !utf8.ValidString(root) {
		return "", errors.New("explicit absolute source directory without control characters required")
	}
	root = filepath.Clean(root)
	st, err := os.Lstat(root)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return "", ErrUnavailable
	}
	return filepath.EvalSymlinks(root)
}

func relative(name string) bool {
	return fs.ValidPath(name) && name != "." && len(name) <= 2048 && !strings.ContainsAny(name, "\\:") && strings.IndexFunc(name, unicode.IsControl) < 0 && utf8.ValidString(name)
}

func excluded(name string) bool {
	for _, part := range strings.Split(name, "/") {
		switch part {
		case ".git", ".mandalore", ".my-friday", ".DS_Store", "node_modules", ".venv", "__pycache__":
			return true
		}
	}
	return false
}

func eligible(name string) bool {
	if excluded(name) || strings.HasPrefix(path.Base(name), ".") {
		return false
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".md", ".markdown", ".txt", ".json":
		return true
	}
	return false
}

// Observe verifies a stable selected text snapshot and returns only routing and
// integrity metadata. File contents stay out of previews and registration data.
func Observe(ctx context.Context, source memory.FoundlingSource, root string) (Observation, error) {
	s, err := observe(ctx, source, root)
	if err != nil {
		return Observation{}, err
	}
	return s.View, nil
}

func observe(ctx context.Context, source memory.FoundlingSource, root string) (*snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	placeholder := memory.SourcePin{Algorithm: "sha256", Value: strings.Repeat("0", 64)}
	if source.Kind == "git" {
		placeholder = memory.SourcePin{Algorithm: "git-sha1", Value: strings.Repeat("0", 40)}
	}
	if err := memory.ValidateFoundlingIdentity(source, placeholder); err != nil {
		return nil, err
	}
	root, err := canonicalDirectory(root)
	if err != nil {
		return nil, err
	}
	first, err := observeOnce(ctx, source, root)
	if err != nil {
		return nil, err
	}
	second, err := observeOnce(ctx, source, root)
	if err != nil {
		return nil, err
	}
	if first.View != second.View {
		return nil, ErrChanged
	}
	return second, nil
}

func observeOnce(ctx context.Context, source memory.FoundlingSource, root string) (*snapshot, error) {
	s := &snapshot{View: Observation{Root: root, Source: source, Notice: "Selected reference text only, not current guidance. No source execution, fetch or writes. Excluded metadata/code/transcript formats are not evidence; remote freshness is not tested."}, Documents: map[string][]byte{}}
	if source.Kind == "git" {
		return observeGit(ctx, s)
	}
	before, err := os.Lstat(root)
	if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return nil, ErrUnavailable
	}
	confined, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer confined.Close()
	entries := 0
	var walk func(string, int) error
	walk = func(name string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries++
		if entries > maxEntries || depth > 64 {
			return errors.New("reference directory enumeration exceeds its bounds")
		}
		if name != "." && !relative(name) {
			return errors.New("unsupported reference-relative path")
		}
		st, err := confined.Lstat(name)
		if err != nil {
			return err
		}
		if st.Mode()&os.ModeSymlink != 0 || (!st.IsDir() && !st.Mode().IsRegular()) {
			return errors.New("reference contains a symlink or special file")
		}
		if excluded(name) {
			s.View.Excluded++
			return nil
		}
		if !st.IsDir() {
			if !eligible(name) {
				s.View.Excluded++
				return nil
			}
			data, err := readRegular(confined, name)
			if err != nil {
				return err
			}
			return s.add(name, data)
		}
		dir, err := confined.OpenFile(name, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return err
		}
		defer dir.Close()
		for {
			children, readErr := dir.ReadDir(128)
			for _, child := range children {
				if err := walk(path.Join(name, child.Name()), depth+1); err != nil {
					return err
				}
			}
			if readErr == io.EOF {
				return nil
			}
			if readErr != nil {
				return readErr
			}
		}
	}
	if err := walk(".", 0); err != nil {
		return nil, err
	}
	after, err := os.Lstat(root)
	if err != nil || !os.SameFile(before, after) {
		return nil, ErrChanged
	}
	s.View.ContentSHA256 = documentDigest(s.Documents)
	s.View.Pin = memory.SourcePin{Algorithm: "sha256", Value: s.View.ContentSHA256}
	return s, nil
}

func readRegular(root *os.Root, name string) ([]byte, error) {
	if !relative(name) {
		return nil, errors.New("invalid reference locator")
	}
	// Check every component, including Git-selected paths not visited by a walk.
	parts := strings.Split(name, "/")
	for n := 1; n < len(parts); n++ {
		st, err := root.Lstat(strings.Join(parts[:n], "/"))
		if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return nil, ErrUnavailable
		}
	}
	st, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > MaxFileBytes {
		return nil, errors.New("reference file must be regular and at most 4 MiB")
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(st, opened) {
		return nil, ErrChanged
	}
	b, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxFileBytes {
		return nil, errors.New("reference file exceeds 4 MiB")
	}
	if !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
		return nil, errors.New("reference text must be UTF-8 without binary NUL bytes")
	}
	return b, nil
}

func (s *snapshot) add(name string, data []byte) error {
	if len(s.Documents) >= MaxFiles || len(data) > MaxSourceBytes-s.View.Bytes {
		return errors.New("reference selection exceeds 10000 files or 64 MiB")
	}
	s.Documents[name] = data
	s.View.Files++
	s.View.Bytes += len(data)
	return nil
}
