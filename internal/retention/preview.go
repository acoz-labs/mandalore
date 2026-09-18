package retention

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/exportreport"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

type Request struct {
	BindingPath string    `json:"binding_path"`
	Selection   Selection `json:"selection"`
	Policy      Policy    `json:"policy"`
}
type Plan struct {
	Version         int                  `json:"version"`
	SignetID        string               `json:"signet_id"`
	BindingPath     string               `json:"binding_path"`
	Source          string               `json:"source"`
	SourceIdentity  string               `json:"source_identity"`
	BindingSHA256   string               `json:"binding_sha256"`
	SourceSHA256    string               `json:"source_sha256"`
	PolicySHA256    string               `json:"policy_sha256"`
	SelectionSHA256 string               `json:"selection_sha256"`
	Policy          Policy               `json:"policy"`
	Selection       Selection            `json:"selection"`
	Records         []Record             `json:"records"`
	Journals        []Journal            `json:"journals"`
	Sources         []SourceRelationship `json:"sources"`
	Notice          string               `json:"notice"`
}

var ErrPreview = errors.New("cannot prepare bounded retention review; inspect explicit binding, selection and policy; no files were written or synchronized")

func digest(value any) string {
	b, _ := json.Marshal(value)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

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
	if err = validatePolicy(in.Policy); err != nil {
		return
	}
	path, b, pin, err := readBinding(in.BindingPath)
	if err != nil {
		return
	}
	guard := binding.Guard{SHA256: pin, SignetID: b.SignetID}
	s, err := binding.OpenGuarded(path, "retention-preview", guard)
	if err != nil {
		return
	}
	root, err := filepath.EvalSymlinks(s.Root())
	if err != nil {
		return
	}
	before, err := os.Lstat(root)
	if err != nil {
		return
	}
	stat, ok := before.Sys().(*syscall.Stat_t)
	if !ok || !before.IsDir() {
		return out, ErrPreview
	}
	snapshot, err := exportreport.ReadReportSnapshot(ctx, root)
	if err != nil {
		return
	}
	if snapshot.Signet.ID != b.SignetID {
		return out, ErrPreview
	}
	records, journals, sources, err := project(ctx, snapshot, in.Selection, in.Policy)
	if err != nil {
		return
	}
	out = Plan{Version: 1, SignetID: b.SignetID, BindingPath: path, Source: root, SourceIdentity: fmt.Sprintf("%d:%d", stat.Dev, stat.Ino), BindingSHA256: pin, SourceSHA256: snapshot.Digest, PolicySHA256: digest(in.Policy), SelectionSHA256: digest(in.Selection), Policy: in.Policy, Selection: in.Selection, Records: records, Journals: journals, Sources: sources,
		Notice: "Sensitive metadata-only review, not authorization or an apply plan. No expiry, withdrawal, deletion, checkpoint or sync occurred. Age requires every current content head to precede the absolute cutoff. Visibility filters apply to records only; journals are separately selected and have no inferred record relationships. Source links may be shared; registration IDs are attribution, not verified external availability. Git history, original foundling files, offline copies and prior model context persist and are not enumerated here. Concurrent external edits are not an atomic snapshot."}
	encoded, e := json.Marshal(out)
	if e != nil || len(encoded) > 32768 {
		return Plan{}, ErrPreview
	}
	// Confirm a stable bounded view and the same pinned connection without taking
	// a writer lock (which would itself create local state). No repair or fetch.
	again, err := exportreport.ReadReportSnapshot(ctx, root)
	if err != nil {
		return
	}
	if again.Digest != out.SourceSHA256 {
		return Plan{}, ErrPreview
	}
	current, err := binding.OpenGuarded(path, "retention-preview", guard)
	if err != nil {
		return
	}
	currentRoot, err := filepath.EvalSymlinks(current.Root())
	if err != nil || currentRoot != root {
		return Plan{}, ErrPreview
	}
	after, err := os.Lstat(root)
	if err != nil || !os.SameFile(before, after) {
		return Plan{}, ErrPreview
	}
	if err = ctx.Err(); err != nil {
		return
	}
	return out, nil
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
