package readiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assessmentInput(t *testing.T) Input {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(root, "missing-path"))
	return Input{Harness: "pi", StateDir: filepath.Join(root, "PRIVATE_STATE"), NativeHome: filepath.Join(root, "PRIVATE_PROFILE"), NativeBinary: filepath.Join(root, "PRIVATE_NATIVE"), Binding: filepath.Join(root, "PRIVATE_BINDING"), IncludePrompt: true}
}

func TestAssessMissingSetupIsUsefulWithoutProviderOrBank(t *testing.T) {
	in := assessmentInput(t)
	got, err := Assess(context.Background(), in)
	if err != nil || got.SchemaVersion != 1 || !got.Complete || got.Prompt == "" || got.NextAction.Code != "select-dependencies" {
		t.Fatal(got, err)
	}
	if strings.Contains(got.Prompt, "PRIVATE_") || strings.Contains(got.Prompt, filepath.Dir(in.Binding)) {
		t.Fatal("private path entered agent prompt", got.Prompt)
	}
	if got.Retained != nil {
		t.Fatal("inferred a retained generation")
	}
	if _, err := os.Stat(in.StateDir); !os.IsNotExist(err) {
		t.Fatal("created missing installation", err)
	}
	raw, err := json.Marshal(got)
	if err != nil || len(raw) > 60<<10 {
		t.Fatal("report is unexpectedly large", len(raw), err)
	}
	if got.Toolkit.Version != "" || got.Toolkit.SourceCommit != "" {
		t.Fatal("fabricated process identity without trusted context")
	}
}

func TestAssessKnownBuildContextAndPartialBinding(t *testing.T) {
	in := assessmentInput(t)
	if err := os.WriteFile(in.Binding, []byte("PRIVATE_INVALID_METADATA"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := WithBuild(context.Background(), "0.0.0-dev", strings.Repeat("a", 40))
	got, err := Assess(ctx, in)
	if err != nil || got.Complete || got.NextAction.Code != "inspect-selection" || got.Toolkit.SourceCommit != strings.Repeat("a", 40) {
		t.Fatal(got, err)
	}
	if strings.Contains(got.Prompt, "PRIVATE_") {
		t.Fatal("raw metadata entered prompt")
	}
}

func TestAssessCancellationReturnsNoReportOrPrompt(t *testing.T) {
	in := assessmentInput(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := Assess(ctx, in)
	if !errors.Is(err, context.Canceled) || got.SchemaVersion != 0 || got.Prompt != "" {
		t.Fatal("cancelled assessment returned report", got, err)
	}
}

func TestArtifactSupportRequiresKnownBytesAndMatchingPlatform(t *testing.T) {
	catalog, err := loadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	e := catalog.Evidence[0]
	if got := artifactSupport(catalog, e.Identities.RuntimeSHA256, e.Platform); got != "supported" {
		t.Fatal("known artifact lost its declared target", got)
	}
	other := e.Platform
	other.OS = "unsupported-synthetic"
	if got := artifactSupport(catalog, e.Identities.RuntimeSHA256, other); got != "unsupported" {
		t.Fatal("known incompatible artifact not identified", got)
	}
	if got := artifactSupport(catalog, strings.Repeat("f", 64), e.Platform); got != "unknown" {
		t.Fatal("unknown artifact architecture guessed", got)
	}
}

func TestNextActionDoesNotSendMissingConnectionStraightToNativeExecution(t *testing.T) {
	components := []Component{observedComponent("binding", "verified-static", "binding-metadata-consistent", true), observedComponent("native-profile", "missing", "directory-missing", true)}
	if got := nextAction(components); got.Code != "inspect-connection" {
		t.Fatal("missing profile skipped setup review", got)
	}
}

func TestUntestedRequirementsStayHarnessSpecific(t *testing.T) {
	in := assessmentInput(t)
	for _, harness := range []string{"codex", "pi", "claude-code"} {
		in.Harness = harness
		r, err := Assess(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		hasNode := strings.Contains(strings.Join(r.Untested, " "), "Node version")
		if hasNode != (harness == "pi") {
			t.Fatal("Node requirement attached to wrong harness", harness, r.Untested)
		}
	}
}

func TestAssessmentKeepsProcessMetadataWhenExecutablePathIsReplaced(t *testing.T) {
	in := assessmentInput(t)
	path := filepath.Join(filepath.Dir(in.Binding), "toolkit")
	original := []byte("original process image fixture")
	replacement := []byte("unrelated replacement image fixture")
	if err := os.WriteFile(path, original, 0700); err != nil {
		t.Fatal(err)
	}
	ctx := WithBuild(context.Background(), "0.0.0-test", strings.Repeat("a", 40))
	// Model a stable os.Executable result independently of the host's path
	// behavior after rename. This tests the actual assessment pipeline, not
	// native loader behavior or a promise of loaded-image attestation.
	executable := func() (string, error) { return path, nil }
	before, err := assess(ctx, in, executable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, path+"-previous"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, replacement, 0700); err != nil {
		t.Fatal(err)
	}
	after, err := assess(ctx, in, executable)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(replacement)
	if after.Toolkit.OnDisk.SHA256 != hex.EncodeToString(want[:]) || before.Toolkit.OnDisk.SHA256 == after.Toolkit.OnDisk.SHA256 {
		t.Fatal("replacement was not represented as the current on-disk bytes", before.Toolkit, after.Toolkit)
	}
	if before.Toolkit.Package != after.Toolkit.Package || after.Toolkit.SourceCommit != strings.Repeat("a", 40) || after.Toolkit.Version != "0.0.0-test" {
		t.Fatal("disk replacement rewrote process metadata", before.Toolkit, after.Toolkit)
	}
	for _, c := range after.Components {
		if c.Evidence == "verified" {
			t.Fatal("source/version labels certified unknown replacement", c)
		}
	}
	if !strings.Contains(after.Notice, "On-disk fingerprints do not attest loaded programs") || !strings.Contains(strings.Join(after.Untested, " "), "loaded-image identity") {
		t.Fatal("loaded-image limitation disappeared", after.Notice, after.Untested)
	}
}
