package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/readiness"
)

func assessmentMenuFixture(t *testing.T) (string, []string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(root, "missing"))
	return root, []string{"--state-dir", filepath.Join(root, "state"), "--native-home", filepath.Join(root, "profile"), "--native-binary", filepath.Join(root, "native"), "--binding", filepath.Join(root, "binding")}
}

func TestArmorerAssessmentDefaultWorksWithoutSetupOrChanges(t *testing.T) {
	root, args := assessmentMenuFixture(t)
	// Armorer, default assessment, Codex, assess, default report Back,
	// selection Back, Armorer Back, Exit.
	out, code := menuTrial(t, "6\n\n1\n\n\n3\n3\n10\n", args...)
	if code != 0 || !strings.Contains(out, "Machine readiness") || !strings.Contains(out, "Next step") || !strings.Contains(out, "Support") || !strings.Contains(out, "Evidence") || !strings.Contains(out, "missing") {
		t.Fatal(code, out)
	}
	if strings.Contains(out, "Help me assess Mandalore") || strings.Contains(out, "Run these native checks?") {
		t.Fatal("default assessment launched extra journey", out)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("assessment menu created state", entries, err)
	}
}

func TestAssessmentPromptRequiresExplicitView(t *testing.T) {
	_, args := assessmentMenuFixture(t)
	out, code := menuTrial(t, "6\n1\n2\n1\n3\n1\n3\n3\n10\n", args...)
	if code != 0 || !strings.Contains(out, "Help me assess Mandalore's pi memory connection") || !strings.Contains(out, "not permission to install") {
		t.Fatal(code, out)
	}
}

func TestAssessmentEditorChangesOneFieldAndClearsOptionalRoot(t *testing.T) {
	root, _ := assessmentMenuFixture(t)
	var out bytes.Buffer
	m := &menu{ctx: context.Background(), in: bufio.NewReader(strings.NewReader("4\n" + filepath.Join(root, "new-binding") + "\n5\n:none\n6\n")), out: &out}
	in := readiness.Input{Harness: "pi", Binding: filepath.Join(root, "old-binding"), StateDir: filepath.Join(root, "state"), NativeHome: filepath.Join(root, "profile"), NativeBinary: filepath.Join(root, "native"), ConnectionRoot: filepath.Join(root, "generation")}
	if err := m.editAssessment(&in); err != nil {
		t.Fatal(err, out.String())
	}
	if in.Binding != filepath.Join(root, "new-binding") || in.ConnectionRoot != "" || in.NativeHome != filepath.Join(root, "profile") {
		t.Fatal("unselected fields changed", in)
	}
}

func TestNativeInspectionDisclosureRequiresExplicitCompleteConsent(t *testing.T) {
	for _, harness := range []string{"codex", "pi"} {
		for _, script := range []string{"\n", ":back\n", "2", "2\n"} {
			var out bytes.Buffer
			called := 0
			profile := install.Profile{StateDir: "/synthetic/state", NativeHome: "/synthetic/profile", NativeBinary: "/synthetic/native"}
			m := &menu{ctx: context.Background(), in: bufio.NewReader(strings.NewReader(script)), out: &out, nativeInspect: func(gotHarness string, gotProfile install.Profile) api.Envelope {
				called++
				if gotHarness != harness || gotProfile != profile {
					t.Fatal("changed approved native selection")
				}
				if harness == "pi" {
					return api.Success(install.PiReport{})
				}
				return api.Success(install.Report{})
			}}
			err := m.nativeInspection(harness, profile)
			if script == "2\n" {
				if err != nil || called != 1 {
					t.Fatal("explicit handoff failed", err, called)
				}
			} else if called != 0 || err == nil {
				t.Fatal("incomplete/default consent dispatched native inspection", err, called)
			}
			if !strings.Contains(out.String(), "logs or cache") || !strings.Contains(out.String(), "/synthetic/native") {
				t.Fatal("native effects/selection not disclosed", out.String())
			}
		}
	}
}

func TestAssessmentReportBackDoesNotDispatchNative(t *testing.T) {
	var out bytes.Buffer
	m := &menu{ctx: context.Background(), in: bufio.NewReader(strings.NewReader(":back\n")), out: &out, nativeInspect: func(string, install.Profile) api.Envelope {
		t.Fatal("Back dispatched native check")
		return api.Envelope{}
	}}
	err := m.assessmentReport(readiness.Report{Complete: true, NextAction: readiness.Action{Summary: "Synthetic next step"}})
	if err != nil && !errors.Is(err, console.ErrBack) {
		t.Fatal(err)
	}
}

func TestCancelledAssessmentReportDoesNotRenderSuccessOrPrompt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	m := &menu{ctx: ctx, in: bufio.NewReader(strings.NewReader("")), out: &out}
	err := m.assessmentReport(readiness.Report{Complete: true, Prompt: "SYNTHETIC_PROMPT"})
	if !errors.Is(err, context.Canceled) || out.Len() != 0 {
		t.Fatal("cancelled report rendered content", err, out.String())
	}
}

func TestAssessmentInvalidEditedSelectionCanBeCorrectedWithoutLeavingJourney(t *testing.T) {
	root, args := assessmentMenuFixture(t)
	// Edit binding to a relative path, assess (rejected), edit the same field
	// back to an absolute path, assess successfully, then return and exit.
	script := "6\n1\n1\n2\n4\nrelative\n6\n1\n2\n4\n" + filepath.Join(root, "fixed-binding") + "\n6\n1\n1\n3\n3\n10\n"
	out, code := menuTrial(t, script, args...)
	if code != 0 || !strings.Contains(out, "Selection needs attention") || !strings.Contains(out, "Machine readiness") {
		t.Fatal("selection correction lost the journey", code, out)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("editing selection changed filesystem", entries, err)
	}
}

func TestAssessmentNativeFailurePreservesSnapshotAndDoesNotRetry(t *testing.T) {
	var out bytes.Buffer
	calls := 0
	m := &menu{ctx: context.Background(), in: bufio.NewReader(strings.NewReader("4\n2\n1\n")), out: &out, nativeInspect: func(string, install.Profile) api.Envelope {
		calls++
		return api.Failure("connection.failed", "Synthetic native failure", false)
	}}
	r := readiness.Report{Harness: "pi", Complete: true, NextAction: readiness.Action{Summary: "Synthetic next step"}, Selection: readiness.Selection{NativeBinary: readiness.SelectedPath{Path: "/synthetic/native"}, NativeHome: readiness.SelectedPath{Path: "/synthetic/profile"}}}
	if err := m.assessmentReport(r); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !m.failed || !strings.Contains(out.String(), "Assessment snapshot unchanged") || !strings.Contains(out.String(), "Synthetic native failure") {
		t.Fatal("native failure hidden or retried", calls, out.String())
	}
	if !r.Complete {
		t.Fatal("native result rewrote earlier assessment")
	}
}
