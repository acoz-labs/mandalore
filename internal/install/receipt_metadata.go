package install

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode"
)

var ErrReceiptMetadata = errors.New("selected ownership receipt is invalid or belongs to a different selection")

// ReceiptSelection is the operator's explicit selection, not paths adopted from
// untrusted receipt fields. Its paths must already be canonical observations.
type ReceiptSelection struct {
	Harness, Root, StateDir, NativeHome, NativeBinary, Binding string
}

// ReceiptMetadata contains declared identities, NOT measured or trusted bytes.
// No package tree, native profile, binding or runtime is read by the projection.
type ReceiptMetadata struct {
	Runtime, RuntimeSHA256, NativeSHA256, BindingSHA256 string
	PackageSHA256, PackageVersion, SignetID             string
}

// ProjectReceiptMetadata reuses the native ownership parsers without invoking
// inventory, setup, repair or execution. No filesystem or network operation is
// performed; the caller must bound the receipt read and inspect selected bytes
// separately. A valid self-consistent receipt is not publisher trust.
func ProjectReceiptMetadata(s ReceiptSelection, raw []byte) (ReceiptMetadata, error) {
	for _, path := range []string{s.Root, s.StateDir, s.NativeHome, s.NativeBinary, s.Binding} {
		if !receiptPath(path) {
			return ReceiptMetadata{}, ErrReceiptMetadata
		}
	}
	var m ReceiptMetadata
	var o Options
	switch s.Harness {
	case "codex":
		r, err := decodeReceipt(s.Root, raw)
		if err != nil {
			return ReceiptMetadata{}, ErrReceiptMetadata
		}
		p := r.Plan
		o = p.Options
		m = ReceiptMetadata{Runtime: p.Runtime, RuntimeSHA256: p.BinarySHA256, NativeSHA256: p.NativeSHA256, BindingSHA256: p.BindingSHA256, PackageSHA256: p.PackageSHA256, PackageVersion: p.PackageVersion, SignetID: p.SignetID}
	case "pi":
		r, err := decodePiReceipt(s.Root, raw)
		if err != nil {
			return ReceiptMetadata{}, ErrReceiptMetadata
		}
		p := r.Plan
		o = p.Options
		m = ReceiptMetadata{Runtime: p.Runtime, RuntimeSHA256: p.BinarySHA256, NativeSHA256: p.NativeSHA256, BindingSHA256: p.BindingSHA256, PackageSHA256: p.PackageSHA256, PackageVersion: p.PackageVersion, SignetID: p.SignetID}
	default:
		return ReceiptMetadata{}, ErrReceiptMetadata
	}
	if o.StateDir != s.StateDir || o.NativeHome != s.NativeHome || o.NativeBinary != s.NativeBinary || o.Binding != s.Binding {
		return ReceiptMetadata{}, ErrReceiptMetadata
	}
	if !receiptPath(o.Binary) || !receiptPath(m.Runtime) || len(m.PackageVersion) < 1 || len(m.PackageVersion) > 128 || strings.IndexFunc(m.PackageVersion, unicode.IsControl) >= 0 {
		return ReceiptMetadata{}, ErrReceiptMetadata
	}
	return m, nil
}

func receiptPath(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && len(path) <= 4096 && strings.IndexFunc(path, unicode.IsControl) < 0
}
