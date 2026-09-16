package readiness

import (
	"context"
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
	for _, harness := range []string{"codex", "pi"} {
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
