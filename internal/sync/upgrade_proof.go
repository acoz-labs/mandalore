package signetsync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

func manifestTransition(old, next []byte) bool {
	var a, b memory.Signet
	return strictjson.Decode(old, &a, 4<<20) == nil && strictjson.Decode(next, &b, 4<<20) == nil && a.Version == 1 && b.Version == 2 && a.ID == b.ID && a.Name == b.Name
}

func (s *Synchronizer) permittedManifestChange(ctx context.Context, revisions []string) error {
	if s.store.Signet.Version != 2 || len(revisions) < 1 || len(revisions) > 2 {
		return ErrHistory
	}
	old, err := s.gitOutput(ctx, 4<<20, "show", revisions[0]+":signet.json")
	if err != nil {
		return err
	}
	var next []byte
	if len(revisions) == 2 {
		raw, err := s.gitOutput(ctx, 4<<20, "show", revisions[1]+":signet.json")
		if err != nil {
			return err
		}
		next = []byte(raw)
	} else {
		info, err := os.Lstat(filepath.Join(s.store.Root, "signet.json"))
		if err != nil || !info.Mode().IsRegular() || info.Size() > 4<<20 {
			return ErrHistory
		}
		next, err = os.ReadFile(filepath.Join(s.store.Root, "signet.json"))
		if err != nil {
			return err
		}
	}
	if !manifestTransition([]byte(old), next) {
		return ErrHistory
	}
	// Exact upgrade proofs are checked on the bounded immutable candidate before
	// a checkpoint ref advances or any remote tree is adopted.
	return nil
}

func protectedEvidence(path string) bool {
	return strings.HasPrefix(path, "memory/") || strings.HasPrefix(path, "provenance/") || strings.HasPrefix(path, "foundlings/registrations/")
}

func inventoryDigest(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		data := files[name]
		fmt.Fprintf(h, "%d:%s:%d:", len(name), name, len(data))
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Check every claimed original checkpoint, not just the metadata schema. Bases
// must be format1 commits; recursive validation therefore cannot follow an
// upgrade chain. Limit aggregate base bytes even when many receipts are present.
func (s *Synchronizer) validateUpgradeProofs(ctx context.Context, files map[string][]byte, anchors []string) error {
	cache := map[string]map[string][]byte{}
	var total int
	for path, data := range files {
		if !strings.HasPrefix(path, "provenance/upgrades/") || !strings.HasSuffix(path, ".json") {
			continue
		}
		var u memory.UpgradeRecord
		if strictjson.Decode(data, &u, 4<<20) != nil {
			return ErrHistory
		}
		base, ok := cache[u.BaseHead]
		if !ok {
			ancestor := false
			for _, anchor := range anchors {
				if anchor == "" {
					continue
				}
				if _, err := s.git(ctx, "merge-base", "--is-ancestor", u.BaseHead, anchor); err == nil {
					ancestor = true
					break
				}
			}
			if !ancestor {
				return ErrHistory
			}
			kind, err := s.git(ctx, "cat-file", "-t", u.BaseHead)
			if err != nil || kind != "commit" {
				return ErrHistory
			}
			base, err = s.candidateFiles(ctx, u.BaseHead)
			if err != nil {
				return err
			}
			var manifest memory.Signet
			if strictjson.Decode(base["signet.json"], &manifest, 4<<20) != nil || manifest.Version != 1 {
				return ErrHistory
			}
			for name, b := range base {
				total += len(name) + len(b)
				if total > candidateArchiveLimit {
					return ErrHistory
				}
				if strings.HasPrefix(name, "provenance/upgrades/") && strings.HasSuffix(name, ".json") {
					return ErrHistory
				}
			}
			if err := s.validateCandidate(ctx, u.BaseHead); err != nil {
				return err
			}
			cache[u.BaseHead] = base
		}
		h := sha256.Sum256(base["signet.json"])
		if hex.EncodeToString(h[:]) != u.OriginalManifestSHA256 || inventoryDigest(base) != u.PortableSHA256 || !manifestTransition(base["signet.json"], files["signet.json"]) {
			return ErrHistory
		}
		for name, b := range base {
			actual, exists := files[name]
			if protectedEvidence(name) && (!exists || !bytes.Equal(actual, b)) {
				return ErrHistory
			}
		}
	}
	return nil
}
