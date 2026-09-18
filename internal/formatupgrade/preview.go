// Package formatupgrade owns explicit same-signet upgrades, never automatic
// activation during ordinary memory access, startup or synchronization.
package formatupgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

type Request struct {
	BindingPath string `json:"binding_path"`
}

type Plan struct {
	Version                int                      `json:"version"`
	BindingPath            string                   `json:"binding_path"`
	BindingSHA256          string                   `json:"binding_sha256"`
	Root                   string                   `json:"root"`
	Source                 signetsync.UpgradeSource `json:"source"`
	FromVersion            int                      `json:"from_version"`
	ToVersion              int                      `json:"to_version"`
	RequiresStoppedWriters bool                     `json:"requires_stopped_writers"`
	Effects                []string                 `json:"effects"`
	Notice                 string                   `json:"notice"`
}

var ErrPreview = errors.New("cannot preview upgrade; inspect the explicit binding and a valid checkpointed format1 signet; no upgrade or synchronization was performed")

func Preview(ctx context.Context, in Request) (out Plan, err error) {
	defer func() {
		if err != nil {
			out = Plan{}
			if ctx.Err() != nil {
				err = ctx.Err()
			} else {
				err = ErrPreview
			}
		}
	}()
	if err = ctx.Err(); err != nil {
		return
	}
	path, b, pin, err := readBinding(in.BindingPath)
	if err != nil {
		return
	}
	guard := binding.Guard{SHA256: pin, SignetID: b.SignetID}
	service, err := binding.OpenGuarded(path, "format-upgrade", guard)
	if err != nil {
		return
	}
	root, err := filepath.EvalSymlinks(service.Root())
	if err != nil {
		return
	}
	sy, err := signetsync.Open(root, b.SignetID)
	if err != nil {
		return
	}
	source, err := sy.UpgradeSource(ctx)
	if err != nil {
		return
	}
	// Re-open exactly the pinned binding after the source read. A changed alias
	// or binding cannot substitute another checkout even with the same signet ID.
	again, err := binding.OpenGuarded(path, "format-upgrade", guard)
	if err != nil {
		return
	}
	currentRoot, err := filepath.EvalSymlinks(again.Root())
	if err != nil || currentRoot != root {
		return out, ErrPreview
	}
	if err = ctx.Err(); err != nil {
		return
	}
	return Plan{Version: 1, BindingPath: path, BindingSHA256: pin, Root: root, Source: source, FromVersion: 1, ToVersion: 2, RequiresStoppedWriters: true,
		Effects: []string{"append immutable upgrade evidence", "atomically activate format2 in the same signet", "preserve existing memory, journals, provenance and Git history", "leave checkpoint and delivery for separate operations"},
		Notice:  "Sensitive review metadata only. Apply requires explicit acknowledgement that affected writers are stopped. Older clients refuse upgraded banks; offline copies and prior model context cannot be revoked. No automatic upgrade, rollback, downgrade or erasure."}, nil
}

func readBinding(path string) (string, binding.Binding, string, error) {
	var b binding.Binding
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || len(path) > 4096 || !utf8.ValidString(path) || strings.IndexFunc(path, func(r rune) bool { return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) }) >= 0 {
		return "", b, "", ErrPreview
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", b, "", err
	}
	path = filepath.Join(parent, filepath.Base(path))
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Size() > 16384 {
		return "", b, "", ErrPreview
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return "", b, "", err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return "", b, "", ErrPreview
	}
	data, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil {
		return "", b, "", err
	}
	if err := strictjson.Decode(data, &b, 16384); err != nil {
		return "", b, "", err
	}
	h := sha256.Sum256(data)
	return path, b, hex.EncodeToString(h[:]), nil
}
