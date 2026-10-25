package cutover

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

// NativeProfile names one harness profile retire inspects.
type NativeProfile struct {
	Harness string `json:"harness"`
	Home    string `json:"native_home"`
	Binary  string `json:"native_binary"`
}

type Disabler func(context.Context, LegacyConnection) error

type Retirement struct {
	Clone    string `json:"clone"`
	SignetID string `json:"signet_id"`
	// Disabled lists what this run disabled. An interrupted earlier run's
	// disables are in its failure message; Clear lists every profile now
	// verified to have no enabled connection.
	Disabled []LegacyConnection `json:"disabled_this_run"`
	Clear    []NativeProfile    `json:"profiles_clear"`
	Note     string             `json:"note"`
	ReadOnly bool               `json:"read_only"`
	Notice   string             `json:"notice"`
}

// PartialError is a failure after the command already changed something.
// Written says what, so the operator can recover.
type PartialError struct {
	Err     error
	Written string
}

func (e *PartialError) Error() string { return e.Err.Error() + "; already written: " + e.Written }
func (e *PartialError) Unwrap() error { return e.Err }

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
	out.Clone, out.SignetID, out.Disabled, out.Clear = store.Root, store.Signet.ID, []LegacyConnection{}, []NativeProfile{}
	// states is each profile's real state, read back from its configuration
	// after every disable, so a stop names what is still enabled.
	states := []string{}
	// Once a disable has taken effect, a failure has changed native state.
	stopped := func(next int, err error) error {
		for _, p := range profiles[next:] {
			states = append(states, fmt.Sprintf("%s in %s: not reached", p.Harness, p.Home))
		}
		err = fmt.Errorf("%w; profiles: %s", err, strings.Join(states, "; "))
		if len(out.Disabled) == 0 {
			return err
		}
		done := []string{}
		for _, c := range out.Disabled {
			done = append(done, fmt.Sprintf("%s %s in %s", c.Harness, c.Target, c.Home))
		}
		return &PartialError{Err: err, Written: "disabled " + strings.Join(done, ", ") + "; the clone itself was not changed"}
	}
	for i, p := range profiles {
		found, err := FindLegacyConnections(ctx, p.Harness, p.Home, p.Binary)
		if err != nil {
			states = append(states, fmt.Sprintf("%s in %s: inventory failed", p.Harness, p.Home))
			return out, stopped(i+1, fmt.Errorf("%s profile inventory failed, and the clone was not changed: %w", p.Harness, err))
		}
		for _, c := range found {
			failed := disable(ctx, c)
			still, err := FindLegacyConnections(ctx, p.Harness, p.Home, p.Binary)
			state := "disabled"
			switch {
			case err != nil:
				state = "unknown, its configuration could not be reread"
			case slices.Contains(still, c) && failed != nil:
				state = "failed, still enabled"
			case slices.Contains(still, c):
				state = "unchanged, still enabled: its disable exited 0 but changed nothing"
			default:
				out.Disabled = append(out.Disabled, c)
				if failed != nil {
					state = "disabled, though its disable reported an error"
				}
			}
			states = append(states, fmt.Sprintf("%s %s in %s: %s", p.Harness, c.Target, p.Home, state))
			switch {
			case failed != nil:
				return out, stopped(i+1, fmt.Errorf("%s disable of %s failed, and the clone was not changed: %w", p.Harness, c.Target, failed))
			case err != nil:
				return out, stopped(i+1, fmt.Errorf("%s profile could not be reread after its disable, and the clone was not changed: %w", p.Harness, err))
			case state != "disabled":
				return out, stopped(i+1, fmt.Errorf("%s still shows an enabled Mandalore connection, and the clone was not changed", p.Harness))
			}
		}
		if still, err := FindLegacyConnections(ctx, p.Harness, p.Home, p.Binary); err != nil || len(still) > 0 {
			states = append(states, fmt.Sprintf("%s in %s: not confirmed clear", p.Harness, p.Home))
			return out, stopped(i+1, fmt.Errorf("%s still shows an enabled Mandalore connection, and the clone was not changed", p.Harness))
		}
		if len(found) == 0 {
			states = append(states, fmt.Sprintf("%s in %s: already clear", p.Harness, p.Home))
		}
		out.Clear = append(out.Clear, p)
	}
	out.Note = filepath.Join(store.Root, RetiredNote)
	note := fmt.Sprintf("# Retired 1.x clone\n\nRetired on %s. Memory for this signet (%s) is now served by the Mandalore 2.0 server at %s.\n\n"+
		"This clone is read-only and must never be written by 1.x again. It was not deleted. To roll back before the first 2.0 write, follow docs/migration-2.0.md in the Mandalore repository.\n",
		time.Now().UTC().Format("2006-01-02"), store.Signet.ID, serverURL)
	if _, err := os.Lstat(out.Note); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(out.Note, []byte(note), 0444); err != nil {
			return out, stopped(len(profiles), errors.New("cannot write RETIRED.md, and the clone was not changed"))
		}
	} else if err != nil {
		return out, stopped(len(profiles), err)
	}
	if err := readOnly(store.Root); err != nil {
		return out, &PartialError{Err: errors.New("cannot remove write permission from every file; rerun retire to finish"), Written: "RETIRED.md, and some files are already read-only"}
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
