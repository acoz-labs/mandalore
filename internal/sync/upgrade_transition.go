package signetsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

type UpgradeTransition struct {
	Source           UpgradeSource
	OriginalManifest []byte
	Manifest         []byte
	Record           memory.UpgradeRecord
}

type UpgradeTransitionState struct {
	Published bool `json:"published"`
	Activated bool `json:"activated"`
}

type upgradeChange struct {
	path               string
	receipt, old, next []byte
	state              UpgradeTransitionState
}

// InspectUpgradeTransition permits exactly the prepared receipt and the reviewed
// manifest replacement, not a generic dirty-worktree exception. It validates the
// original immutable Git candidate in an owned temporary directory, then compares
// every original live file against its blob. It never adopts, repairs or edits
// source data. Use under the caller's shared writer lock before mutation/recovery.
func (s *Synchronizer) InspectUpgradeTransition(ctx context.Context, in UpgradeTransition) (UpgradeTransitionState, error) {
	if err := ctx.Err(); err != nil {
		return UpgradeTransitionState{}, err
	}
	var old, next memory.Signet
	if strictjson.Decode(in.OriginalManifest, &old, 4<<20) != nil || strictjson.Decode(in.Manifest, &next, 4<<20) != nil {
		return UpgradeTransitionState{}, ErrUpgradeSource
	}
	if old.Version != 1 || next.Version != 2 || old.ID != in.Source.SignetID || old.ID != s.store.Signet.ID || next.ID != old.ID || next.Name != old.Name {
		return UpgradeTransitionState{}, ErrUpgradeSource
	}
	h := sha256.Sum256(in.OriginalManifest)
	if hex.EncodeToString(h[:]) != in.Source.ManifestSHA256 {
		return UpgradeTransitionState{}, ErrUpgradeSource
	}
	u := in.Record
	if u.SignetID != old.ID || u.BaseHead != in.Source.Head || u.OriginalManifestSHA256 != in.Source.ManifestSHA256 || u.PortableSHA256 != in.Source.PortableSHA256 {
		return UpgradeTransitionState{}, ErrUpgradeSource
	}
	if err := memory.ValidateUpgradeRecords(next, []memory.UpgradeRecord{u}, func(string) error { return s.store.ValidateAuthorship(u.Authorship) }); err != nil {
		return UpgradeTransitionState{}, ErrUpgradeSource
	}
	if err := s.boundary(ctx); err != nil {
		return UpgradeTransitionState{}, err
	}
	if err := s.validateCandidate(ctx, in.Source.Head); err != nil {
		return UpgradeTransitionState{}, err
	}
	b, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return UpgradeTransitionState{}, ErrUpgradeSource
	}
	change := &upgradeChange{path: "provenance/upgrades/" + u.ID + ".json", receipt: append(b, '\n'), old: in.OriginalManifest, next: in.Manifest}
	actual, err := s.upgradeSource(ctx, change)
	if err != nil {
		return change.state, err
	}
	if actual != in.Source {
		return change.state, ErrUpgradeSource
	}
	return change.state, nil
}
