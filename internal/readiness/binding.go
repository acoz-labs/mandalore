package readiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// No names, actor labels, device labels, remembered content or raw errors leave
// this projection. Complete is observation completeness, not store health.
type metadataObservation struct {
	signetID string
	Setup    string `json:"setup"`
	Code     string `json:"code"`
	Complete bool   `json:"complete"`
	SHA256   string `json:"sha256,omitempty"`
}

type metadataReader func(context.Context, string, int64) ([]byte, error)

func metadataProblem(err error, missingBinding bool) (metadataObservation, error) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return metadataObservation{}, err
	}
	if missingBinding && os.IsNotExist(err) {
		return metadataObservation{Setup: "missing", Code: "binding-missing", Complete: true}, nil
	}
	code := "metadata-unreadable"
	setup := "unknown"
	switch {
	case os.IsNotExist(err):
		code, setup = "binding-target-missing", "inconsistent"
	case errors.Is(err, errMetadataUnsafe):
		code, setup = "metadata-unsafe", "inconsistent"
	case errors.Is(err, errMetadataLimit):
		code, setup = "metadata-oversized", "inconsistent"
	case errors.Is(err, errMetadataChanged):
		code, setup = "metadata-changed", "inconsistent"
	}
	return metadataObservation{Setup: setup, Code: code}, nil
}

func inconsistentMetadata(code string) (metadataObservation, error) {
	return metadataObservation{Setup: "inconsistent", Code: code}, nil
}

func inspectBinding(ctx context.Context, path, harness string, read metadataReader) (metadataObservation, error) {
	if err := ctx.Err(); err != nil {
		return metadataObservation{}, err
	}
	raw, err := read(ctx, path, 16<<10)
	if err != nil {
		return metadataProblem(err, true)
	}
	var b binding.Binding
	if strictjson.Decode(raw, &b, 16<<10) != nil || b.Version != 1 ||
		!filepath.IsAbs(b.Root) || len(b.Root) > 4096 || strings.IndexFunc(b.Root, unicode.IsControl) >= 0 ||
		memory.ValidateDeviceID(b.DeviceID) != nil || (harness != "codex" && harness != "pi") {
		return inconsistentMetadata("binding-invalid")
	}
	sum := sha256.Sum256(raw)
	root := filepath.Clean(b.Root)
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return metadataProblem(err, false)
	}
	if resolved != root {
		return inconsistentMetadata("signet-root-redirected")
	}
	info, err := os.Lstat(root)
	if err != nil {
		return metadataProblem(err, false)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return inconsistentMetadata("signet-root-invalid")
	}
	rel, err := filepath.Rel(root, filepath.Clean(path))
	if err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return inconsistentMetadata("binding-inside-signet")
	}
	for _, marker := range []string{"agent.json", "bank.json"} {
		if err := ctx.Err(); err != nil {
			return metadataObservation{}, err
		}
		_, err := os.Lstat(filepath.Join(root, marker))
		if err == nil {
			return inconsistentMetadata("signet-legacy-format")
		}
		if !os.IsNotExist(err) {
			return metadataProblem(err, false)
		}
	}
	manifest, err := read(ctx, filepath.Join(root, "signet.json"), 4<<20)
	if err != nil {
		return metadataProblem(err, false)
	}
	var s memory.Signet
	if strictjson.Decode(manifest, &s, 4<<20) != nil {
		return inconsistentMetadata("signet-manifest-invalid")
	}
	if s.Version != memory.FormatVersion {
		return inconsistentMetadata("signet-format-unsupported")
	}
	if s.ID != b.SignetID {
		return inconsistentMetadata("binding-signet-mismatch")
	}
	device, err := read(ctx, filepath.Join(root, "provenance", "devices", b.DeviceID+".json"), 4<<20)
	if err != nil {
		return metadataProblem(err, false)
	}
	var d memory.Device
	if strictjson.Decode(device, &d, 4<<20) != nil {
		return inconsistentMetadata("device-metadata-invalid")
	}
	if memory.ValidateConnectionMetadata(s, d, memory.Authorship{DeviceID: b.DeviceID, Actor: b.Actor, Harness: harness}) != nil {
		return inconsistentMetadata("binding-metadata-invalid")
	}
	if err := ctx.Err(); err != nil {
		return metadataObservation{}, err
	}
	return metadataObservation{signetID: b.SignetID, Setup: "verified-static", Code: "binding-metadata-consistent", Complete: true, SHA256: hex.EncodeToString(sum[:])}, nil
}
