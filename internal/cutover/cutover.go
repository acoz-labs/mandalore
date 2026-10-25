// Package cutover moves 1.x clones to the single 2.0 server: preflight checks
// a clone is safe to retire, and retire stops its 1.x writers without
// deleting anything.
package cutover

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

const ReceiptKind = "mandalore.migrate.preflight"

// Receipt is a clone's preflight. Saved to a file, it is the evidence
// `server migrate init --preflight` accepts for the clone's device.
type Receipt struct {
	Version  int    `json:"schema_version"`
	Kind     string `json:"kind"`
	Clean    bool   `json:"clean"`
	SignetID string `json:"signet_id"`
	Format   int    `json:"format"`
	// DeviceID comes from the local binding when it names this clone.
	DeviceID string `json:"device_id,omitempty"`
	Clone    string `json:"clone"`
	signetsync.CloneState
	PendingIntents []string `json:"pending_intents"`
	Blockers       []string `json:"blockers"`
	CheckedAt      string   `json:"checked_at"`
}

// Preflight inspects a clone without writing to it. An error means it could
// not be inspected; a receipt with blockers means it is not safe to retire.
// bindingPath may be empty or missing; then the device is not reported.
func Preflight(ctx context.Context, clone, bindingPath string) (Receipt, error) {
	r := Receipt{Version: 1, Kind: ReceiptKind, PendingIntents: []string{}, Blockers: []string{}, CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	store, err := memory.Open(clone)
	if err != nil {
		return r, errors.New("clone is not a readable signet")
	}
	r.SignetID, r.Format, r.Clone = store.Signet.ID, store.Signet.Version, store.Root
	if err := store.Validate(); err != nil {
		return r, fmt.Errorf("signet validation failed: %w", err)
	}
	g, err := signetsync.Open(store.Root, store.Signet.ID)
	if err != nil {
		return r, err
	}
	if r.CloneState, err = g.InspectClone(ctx); err != nil {
		return r, errors.New("clone is not a standalone Git checkout on main with no operation in progress")
	}
	if r.PendingIntents, err = store.PendingIntents(); err != nil {
		return r, errors.New("pending write intents are unreadable")
	}
	if bindingPath != "" {
		if b, err := binding.Read(bindingPath); err == nil && b.SignetID == r.SignetID && sameDir(b.Root, store.Root) {
			r.DeviceID = b.DeviceID
		}
	}
	block := func(format string, args ...any) { r.Blockers = append(r.Blockers, fmt.Sprintf(format, args...)) }
	if len(r.Uncommitted) > 0 {
		block("%d uncommitted file(s): run a 1.x sync", len(r.Uncommitted))
	}
	if r.Unpushed > 0 {
		block("%d unpushed commit(s): run a 1.x sync", r.Unpushed)
	}
	if len(r.PendingIntents) > 0 {
		block("%d pending write intent(s) in .mandalore: interrupted razor-crest writes whose entry was never published; retry each from its client with the same request_id, or save the entry it holds elsewhere and move the file out of .mandalore (see docs/migration-2.0.md)", len(r.PendingIntents))
	}
	switch r.Remote {
	case "none":
		block("no origin remote: this clone's memory has never been backed up")
	case "unreachable":
		block("origin unreachable: cannot confirm the memory is pushed")
	case signetsync.RemoteDiverged:
		block("diverged from origin: run a 1.x sync to reconcile")
	case signetsync.RemoteMoved:
		block("origin has commits this clone has not fetched: run a 1.x sync, then preflight again")
	}
	if r.Format < 2 {
		block("format 1: upgrade to format 2 first with runbook step 1 in docs/migration-2.0.md (stop every 1.x writer, signet_upgrade_preview, signet_upgrade_apply with stopped_writers: true using the 1.x binary, then sync); verify withdraws its canary, which needs format 2")
	}
	r.Clean = len(r.Blockers) == 0
	return r, nil
}

func sameDir(a, b string) bool {
	a, errA := filepath.EvalSymlinks(a)
	b, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && a == b
}
