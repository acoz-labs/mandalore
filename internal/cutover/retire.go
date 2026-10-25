package cutover

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/memory"
)

// NativeProfile names one harness profile retire inspects.
type NativeProfile struct {
	Harness, Home, Binary string
}

type Disabler func(context.Context, install.LegacyConnection) error

type Retirement struct {
	Clone    string                     `json:"clone"`
	SignetID string                     `json:"signet_id"`
	Disabled []install.LegacyConnection `json:"disabled"`
	Note     string                     `json:"note"`
	ReadOnly bool                       `json:"read_only"`
	Notice   string                     `json:"notice"`
}

// RetiredNote is the file retire writes at the clone's root.
const RetiredNote = "RETIRED.md"

// Retire disables every enabled 1.x connection in the named profiles, writes
// RETIRED.md and removes write permission from the clone, keeping .git as it
// is. It deletes nothing, and rerunning it finishes an interrupted retire.
func Retire(ctx context.Context, clone, serverURL string, profiles []NativeProfile, disable Disabler) (Retirement, error) {
	var out Retirement
	u, err := url.Parse(serverURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
		return out, errors.New("server URL must be an http or https URL")
	}
	store, err := memory.Open(clone)
	if err != nil {
		return out, errors.New("clone is not a readable signet")
	}
	out.Clone, out.SignetID, out.Disabled = store.Root, store.Signet.ID, []install.LegacyConnection{}
	for _, p := range profiles {
		found, err := install.FindLegacyConnections(ctx, p.Harness, p.Home, p.Binary)
		if err != nil {
			return out, fmt.Errorf("%s profile inventory failed: %w", p.Harness, err)
		}
		for _, c := range found {
			if err := disable(ctx, c); err != nil {
				return out, fmt.Errorf("%s disable failed; the clone was not changed: %w", p.Harness, err)
			}
			out.Disabled = append(out.Disabled, c)
		}
		if still, err := install.FindLegacyConnections(ctx, p.Harness, p.Home, p.Binary); err != nil || len(still) > 0 {
			return out, fmt.Errorf("%s still shows an enabled Mandalore connection; the clone was not changed", p.Harness)
		}
	}
	out.Note = filepath.Join(store.Root, RetiredNote)
	note := fmt.Sprintf("# Retired 1.x clone\n\nRetired on %s. Memory for this signet (%s) is now served by the Mandalore 2.0 server at %s.\n\n"+
		"This clone is read-only and must never be written by 1.x again. It was not deleted. To roll back before the first 2.0 write, follow docs/migration-2.0.md in the Mandalore repository.\n",
		time.Now().UTC().Format("2006-01-02"), store.Signet.ID, serverURL)
	if _, err := os.Lstat(out.Note); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(out.Note, []byte(note), 0444); err != nil {
			return out, errors.New("cannot write RETIRED.md")
		}
	} else if err != nil {
		return out, err
	}
	if err := readOnly(store.Root); err != nil {
		return out, errors.New("cannot remove write permission from every file; rerun retire to finish")
	}
	out.ReadOnly = true
	out.Notice = "Retired. 1.x writes now fail with a permission error. Nothing was deleted."
	return out, nil
}

// readOnly is chmod -R a-w that leaves .git untouched.
func readOnly(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root && d.Name() == ".git" && filepath.Dir(path) == root {
			return filepath.SkipDir
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.Chmod(path, info.Mode().Perm()&^0222)
	})
}
