package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/install"
)

func menuTrial(t *testing.T, script string, args ...string) (string, int) {
	t.Helper()
	var out bytes.Buffer
	code := run(context.Background(), append([]string{"menu", "--plain"}, args...), strings.NewReader(script), &out, &out)
	return out.String(), code
}

func TestMenuExitAndIncompleteInputDoNotConsent(t *testing.T) {
	for _, script := range []string{"", "8\n", ":back\n", "1\n", "garbage\n8\n"} {
		out, code := menuTrial(t, script)
		if code != 0 || !strings.Contains(out, "Opening this menu changes nothing") || strings.Contains(out, "\x1b") {
			t.Fatal(code, out)
		}
	}
}

func TestMenuCreatePreviewDefaultNoAndEOF(t *testing.T) {
	for _, consent := range []string{"\n8\n", "2", ":back\n8\n"} {
		dir := t.TempDir()
		root, path := filepath.Join(dir, "signet"), filepath.Join(dir, "binding.json")
		script := strings.Join([]string{"1", root, "Synthetic", "Test machine", "Test actor", path}, "\n") + "\n" + consent
		out, code := menuTrial(t, script)
		if code != 0 || !strings.Contains(out, "Review local setup") {
			t.Fatal(code, out)
		}
		for _, p := range []string{root, path} {
			if _, err := os.Lstat(p); !os.IsNotExist(err) {
				t.Fatal("cancel created state", p, out)
			}
		}
	}
}

func TestMenuCreatesBindsAndInspectsWithOneReader(t *testing.T) {
	dir := t.TempDir()
	root, path := filepath.Join(dir, "signet"), filepath.Join(dir, "binding.json")
	script := strings.Join([]string{"1", root, "Synthetic", "Test machine", "Test actor", path, "2", "3", "8", ""}, "\n")
	out, code := menuTrial(t, script)
	if code != 0 || !strings.Contains(out, "[PASS] Local signet ready") || !strings.Contains(out, "[PASS] Signet structure") || !strings.Contains(out, "not configured") {
		t.Fatal(code, out)
	}
	if _, err := binding.Open(path, "test"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(root, ".git")); err != nil || !info.IsDir() {
		t.Fatal("Git not initialized", err)
	}
}

func TestMenuBindingCollisionPrecedesCreate(t *testing.T) {
	dir := t.TempDir()
	root, path := filepath.Join(dir, "signet"), filepath.Join(dir, "binding.json")
	if err := os.WriteFile(path, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	script := strings.Join([]string{"1", root, "Synthetic", "Test machine", "Test actor", path, "8", ""}, "\n")
	out, code := menuTrial(t, script)
	if code != 1 || !strings.Contains(out, "binding already exists") {
		t.Fatal(code, out)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatal("orphan signet")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "preserve" {
		t.Fatal("binding replaced")
	}
}

func TestMenuCancelledContextCannotApply(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	if code := run(ctx, []string{"menu", "--plain"}, strings.NewReader("1\n"), &out, &out); code != 130 {
		t.Fatal(code, out.String())
	}
}

func TestMenuPartialGitFailurePreservesReadyBinding(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir) // no Git; creation and binding are still real operations
	root, path := filepath.Join(dir, "signet"), filepath.Join(dir, "binding.json")
	script := strings.Join([]string{"1", root, "Synthetic", "Test machine", "Test actor", path, "2", "8", ""}, "\n")
	out, code := menuTrial(t, script)
	if code != 1 || !strings.Contains(out, "Partial setup: signet and binding are ready") || strings.Contains(out, "[PASS] Local signet ready") {
		t.Fatal(code, out)
	}
	if _, err := binding.Open(path, "test"); err != nil {
		t.Fatal("lost completed step", err)
	}
}

type failedMenuWriter struct{}

func (failedMenuWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestMenuCannotApplyIfPreviewCannotBeDisplayed(t *testing.T) {
	dir := t.TempDir()
	root, path := filepath.Join(dir, "signet"), filepath.Join(dir, "binding.json")
	script := strings.Join([]string{"1", root, "Synthetic", "Test machine", "Test actor", path, "2", "8", ""}, "\n")
	if code := run(context.Background(), []string{"menu", "--plain"}, strings.NewReader(script), failedMenuWriter{}, io.Discard); code != 1 {
		t.Fatal(code)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatal("applied without visible preview")
	}
}

func TestMenuOverlongInputStopsWithoutReadingFollowingConsent(t *testing.T) {
	out, code := menuTrial(t, strings.Repeat("x", 8192)+"\n8\n")
	if code != 1 || !strings.Contains(out, "answer exceeds") {
		t.Fatal(code, out)
	}
}

func TestMenuExistingSignetGetsIndependentBindingWithoutGitInit(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "signet")
	v, code := cli(t, []string{"signet", "create", "--repository", root, "--name", "Synthetic", "--device-label", "Original"}, "")
	if code != 0 {
		t.Fatal(v)
	}
	first, second := filepath.Join(dir, "first.json"), filepath.Join(dir, "second.json")
	a, err := binding.Bind(root, first, "Original", "Synthetic")
	if err != nil {
		t.Fatal(err)
	}
	script := strings.Join([]string{"2", root, "Second machine", "Synthetic", second, "2", "8", ""}, "\n")
	out, code := menuTrial(t, script)
	if code != 0 || !strings.Contains(out, "[PASS] Local signet connected") {
		t.Fatal(code, out)
	}
	b, err := binding.Open(second, "test")
	if err != nil || b.ID() != a.SignetID {
		t.Fatal("bank identity changed", err)
	}
	data, _ := os.ReadFile(second)
	if strings.Contains(string(data), a.DeviceID) {
		t.Fatal("device was reused")
	}
	if _, err := os.Lstat(filepath.Join(root, ".git")); !os.IsNotExist(err) {
		t.Fatal("connect initialized Git")
	}
}

func TestMenuConnectionPreviewDoesNotRunArtifact(t *testing.T) {
	dir := t.TempDir()
	root, path := filepath.Join(dir, "signet"), filepath.Join(dir, "binding.json")
	v, code := cli(t, []string{"signet", "create", "--repository", root, "--name", "Synthetic", "--device-label", "Original"}, "")
	if code != 0 {
		t.Fatal(v)
	}
	if _, err := binding.Bind(root, path, "Test", "Synthetic"); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "do-not-execute")
	if err := os.WriteFile(binary, []byte("not executable code"), 0700); err != nil {
		t.Fatal(err)
	}
	state, native := filepath.Join(dir, "state"), filepath.Join(dir, "native")
	out, code := menuTrial(t, "5\n\n\n8\n", "--binding", path, "--binary", binary, "--native-binary", binary, "--state-dir", state, "--native-home", native)
	if code != 0 || !strings.Contains(out, "Review Codex connection") || !strings.Contains(out, "Native profile") || !strings.Contains(out, "Runtime SHA256") {
		t.Fatal(code, out)
	}
	for _, p := range []string{state, native} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Fatal("preview wrote files", p)
		}
	}
}

func TestMenuReportsNativePartialPhase(t *testing.T) {
	var out bytes.Buffer
	m := &menu{out: &out}
	v := api.Failure("connection.failed", "test failure", true)
	v.Error.ConnectionResult = &install.Result{Phase: "marketplace-registered", PreviousRoot: "/synthetic/previous", Connection: install.Plan{Root: "/synthetic/target"}}
	err := m.outcome("connection", v)
	if err == nil || !strings.Contains(err.Error(), "No automatic rollback") || !strings.Contains(out.String(), "marketplace-registered") || !strings.Contains(out.String(), "/synthetic/previous") {
		t.Fatal(err, out.String())
	}
}

func TestMenuNestedStatusUsesReadableFieldsNotRawJSON(t *testing.T) {
	var out bytes.Buffer
	m := &menu{out: &out}
	m.jsonBlock("Status", map[string]any{"notice": "A readable status explanation.", "last_attempt": map[string]any{"state": "pending", "delivered": false}})
	text := out.String()
	if !strings.Contains(text, "Status · last attempt") || !strings.Contains(text, "pending") || strings.Contains(text, "{\"") || strings.Contains(text, "notice:") {
		t.Fatal(text)
	}
}
