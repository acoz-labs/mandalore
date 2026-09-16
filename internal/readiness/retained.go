package readiness

import (
	"context"
	"os"
	"path/filepath"

	"github.com/acoz-labs/mandalore/internal/install"
)

type retainedDeclaration struct {
	RuntimeSHA256  string `json:"runtime_sha256"`
	NativeSHA256   string `json:"native_sha256"`
	BindingSHA256  string `json:"binding_sha256"`
	PackageSHA256  string `json:"package_sha256"`
	PackageVersion string `json:"package_version"`
}

type retainedObservation struct {
	Setup    string               `json:"setup"`
	Code     string               `json:"code"`
	Complete bool                 `json:"complete"`
	Runtime  fileDigest           `json:"runtime"`
	Declared *retainedDeclaration `json:"declared,omitempty"`
}

type fileHasher func(context.Context, string, int64, bool) (fileDigest, error)

func retainedProblem(err error) (retainedObservation, error) {
	m, err := metadataProblem(err, false)
	if err != nil {
		return retainedObservation{}, err
	}
	if m.Code == "binding-target-missing" {
		m.Code = "retained-target-missing"
	}
	return retainedObservation{Setup: m.Setup, Code: m.Code}, nil
}

func inspectRetained(ctx context.Context, s install.ReceiptSelection, native fileDigest, bound metadataObservation, read metadataReader, fingerprint fileHasher) (retainedObservation, error) {
	if err := ctx.Err(); err != nil {
		return retainedObservation{}, err
	}
	if s.Root == "" {
		return retainedObservation{Setup: "unknown", Code: "retained-unselected", Complete: true}, nil
	}
	name, limit := "connection.json", int64(32<<10)
	if s.Harness == "pi" {
		name, limit = "receipt.json", 64<<10
	}
	raw, err := read(ctx, filepath.Join(s.Root, name), limit)
	if err != nil {
		return retainedProblem(err)
	}
	m, err := install.ProjectReceiptMetadata(s, raw)
	if err != nil {
		return retainedObservation{Setup: "inconsistent", Code: "retained-receipt-or-selection-invalid"}, nil
	}
	if bound.Setup != "verified-static" || native.SHA256 == "" || !native.Executable || native.Size == 0 {
		return retainedObservation{Setup: "unknown", Code: "retained-inputs-unverified"}, nil
	}
	if bound.SHA256 != m.BindingSHA256 || bound.signetID != m.SignetID || native.SHA256 != m.NativeSHA256 || native.Path != s.NativeBinary {
		return retainedObservation{Setup: "inconsistent", Code: "retained-selection-changed"}, nil
	}
	if err := ctx.Err(); err != nil {
		return retainedObservation{}, err
	}
	// The pure ownership projection must succeed BEFORE following this path.
	// Receipt identities remain declared; only runtime bytes are measured here.
	runtime, err := fingerprint(ctx, m.Runtime, 128<<20, false)
	if err != nil {
		if os.IsNotExist(err) {
			return retainedObservation{Setup: "inconsistent", Code: "retained-runtime-missing"}, nil
		}
		return retainedProblem(err)
	}
	r := retainedObservation{Setup: "inconsistent", Code: "retained-runtime-changed", Runtime: runtime, Declared: &retainedDeclaration{
		RuntimeSHA256: m.RuntimeSHA256, NativeSHA256: m.NativeSHA256, BindingSHA256: m.BindingSHA256, PackageSHA256: m.PackageSHA256, PackageVersion: m.PackageVersion,
	}}
	if runtime.SHA256 != m.RuntimeSHA256 || runtime.Size == 0 || !runtime.Executable {
		return r, nil
	}
	if err := ctx.Err(); err != nil {
		return retainedObservation{}, err
	}
	r.Setup, r.Code, r.Complete = "verified-static", "retained-runtime-and-selection-consistent", true
	return r, nil
}
