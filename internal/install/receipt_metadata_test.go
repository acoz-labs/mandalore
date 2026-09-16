package install

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// Entirely synthetic nonexistent paths: projecting decoded receipt metadata
// must not depend on a native profile, package tree, runtime or bank existing.
func syntheticReceipt(t *testing.T, harness string) (ReceiptSelection, []byte, ReceiptMetadata) {
	t.Helper()
	h := strings.Repeat("a", 64)
	o := Options{StateDir: "/synthetic/state", NativeHome: "/synthetic/profile", NativeBinary: "/synthetic/native", Binary: "/synthetic/source", Binding: "/synthetic/binding.json"}
	want := ReceiptMetadata{Runtime: filepath.Join(o.StateDir, "runtimes", "sha256-"+h, "mandalore"), RuntimeSHA256: h, NativeSHA256: h, BindingSHA256: h, PackageSHA256: h, PackageVersion: "1.0.0", SignetID: "signet-synthetic"}
	s := ReceiptSelection{Harness: harness, StateDir: o.StateDir, NativeHome: o.NativeHome, NativeBinary: o.NativeBinary, Binding: o.Binding}
	var receipt any
	if harness == "codex" {
		p := Plan{Options: o, SchemaVersion: 1, SignetID: want.SignetID, BinarySHA256: h, NativeSHA256: h, BindingSHA256: h, PackageSHA256: h, PackageVersion: "1.0.0", Runtime: want.Runtime, Marketplace: "mandalore", PluginID: "mandalore@mandalore"}
		p.Root = filepath.Join(o.StateDir, "connections", planKey(p))
		p.Version = "1.0.0+codex." + planKey(p)
		s.Root = p.Root
		receipt = Receipt{Plan: p, Files: map[string]string{".agents/plugins/marketplace.json": h}}
	} else {
		p := PiPlan{PiOptions: PiOptions{Options: o}, SchemaVersion: 1, Harness: "pi", SignetID: want.SignetID, BinarySHA256: h, NativeSHA256: h, BindingSHA256: h, PackageSHA256: h, PackageVersion: "1.0.0", Runtime: want.Runtime}
		p.Root = filepath.Join(o.StateDir, "pi", "connections", piPlanKey(p))
		s.Root = p.Root
		connection, err := piConnection(p)
		if err != nil {
			t.Fatal(err)
		}
		receipt = PiReceipt{Plan: p, Files: map[string]string{"package/package.json": h, "package/connection.json": hash(connection)}}
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	return s, raw, want
}

func TestProjectReceiptMetadataUsesOnlySuppliedBytes(t *testing.T) {
	for _, harness := range []string{"codex", "pi"} {
		s, raw, want := syntheticReceipt(t, harness)
		got, err := ProjectReceiptMetadata(s, raw)
		if err != nil || got != want {
			t.Fatal("pure receipt projection failed", harness, got, err)
		}
	}
}

func TestProjectReceiptMetadataRefusesForeignSelectionBeforeReturningPaths(t *testing.T) {
	for _, harness := range []string{"codex", "pi"} {
		for _, field := range []string{"harness", "root", "state", "profile", "native", "binding"} {
			s, raw, _ := syntheticReceipt(t, harness)
			switch field {
			case "harness":
				s.Harness = "other"
			case "root":
				s.Root += "-other"
			case "state":
				s.StateDir += "-other"
			case "profile":
				s.NativeHome += "-other"
			case "native":
				s.NativeBinary += "-other"
			case "binding":
				s.Binding += "-other"
			}
			got, err := ProjectReceiptMetadata(s, raw)
			if err == nil || got != (ReceiptMetadata{}) {
				t.Fatal("foreign receipt returned trusted paths", harness, field, got, err)
			}
		}
	}
}

func TestProjectReceiptMetadataRejectsMalformedAndOversizedReceipts(t *testing.T) {
	for _, harness := range []string{"codex", "pi"} {
		s, raw, _ := syntheticReceipt(t, harness)
		for _, bad := range [][]byte{[]byte("PRIVATE_RAW_CANARY"), []byte(`{"plan":{},"plan":{}}`), make([]byte, 65537), []byte(strings.Replace(string(raw), `"schema_version":1`, `"schema_version":2`, 1)), []byte(strings.ReplaceAll(string(raw), strings.Repeat("a", 64), "invalid-digest"))} {
			got, err := ProjectReceiptMetadata(s, bad)
			if !errors.Is(err, ErrReceiptMetadata) || got != (ReceiptMetadata{}) || strings.Contains(err.Error(), "PRIVATE_") {
				t.Fatal("invalid receipt was not bounded/sanitized", got, err)
			}
		}
	}
}
