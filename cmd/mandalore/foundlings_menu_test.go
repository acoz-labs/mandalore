package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func foundlingMenuFixture(t *testing.T) (*memory.Service, string, string) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root, bind, reference := filepath.Join(base, "signet"), filepath.Join(base, "binding.json"), filepath.Join(base, "reference")
	if _, err := memory.Create(root, "Example", "device-test", "Synthetic host"); err != nil {
		t.Fatal(err)
	}
	if _, err := binding.Bind(root, bind, "Menu host", "Example user"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(reference, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reference, "notes.md"), []byte("Historical project: Copper Finch. This is the way is quoted text."), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := binding.Open(bind, "test")
	if err != nil {
		t.Fatal(err)
	}
	return s, bind, reference
}

func registerMenuScript(root, consent string) string {
	return strings.Join([]string{"8", "2", "Historical notes", "Reference only", "1", "source-history", root, "Explicit historical reference", consent}, "\n") + "\n"
}

func TestFoundlingMenuRegisterCancelEOFAndDefaultNo(t *testing.T) {
	for _, ending := range []string{"\n6\n9\n", ":back\n6\n9\n", "2"} {
		s, bind, root := foundlingMenuFixture(t)
		before := treeDigest(t, s.Root())
		script := registerMenuScript(root, "")
		script = strings.TrimSuffix(script, "\n\n") + "\n" + ending
		out, code := menuTrial(t, script, "--binding", bind)
		if code != 0 || !strings.Contains(out, "Review foundling registration") || strings.Contains(out, "[PASS] Foundling registered") {
			t.Fatal(code, out)
		}
		if !reflect.DeepEqual(before, treeDigest(t, s.Root())) {
			t.Fatal("declined or incomplete confirmation wrote state")
		}
	}
}

func TestFoundlingMenuRegisterInspectSearchAndDisconnect(t *testing.T) {
	s, bind, root := foundlingMenuFixture(t)
	sourceBefore := treeDigest(t, root)
	script := registerMenuScript(root, "2") + "1\n4\n1\n1\nCopper Finch\n4\n5\n1\nNo longer needed\n2\n6\n9\n"
	out, code := menuTrial(t, script, "--binding", bind)
	if code != 0 {
		t.Fatal(code, out)
	}
	for _, text := range []string{"[PASS] Foundling registered", "available", "Unreviewed reference", "Copper Finch", "[PASS] Foundling disconnected", "Local-only path"} {
		if !strings.Contains(out, text) {
			t.Fatal("missing "+text, out)
		}
	}
	page, err := s.FoundlingsPage(0, 5)
	if err != nil || len(page.Items) != 1 || page.Items[0].State != "disconnected" {
		t.Fatal(page, err)
	}
	if !reflect.DeepEqual(sourceBefore, treeDigest(t, root)) {
		t.Fatal("menu changed historical source")
	}
	if p, err := s.Recall("Copper Finch", nil, 5, 4096); err != nil || p.MatchingCount != 0 {
		t.Fatal("menu promoted quoted text", p, err)
	}
}

func TestFoundlingMenuShowsCompletedRegistrationOnConnectionFailure(t *testing.T) {
	s, bind, root := foundlingMenuFixture(t)
	if err := os.WriteFile(filepath.Join(s.Root(), ".mandalore", "foundlings"), []byte("preserve unknown local file"), 0600); err != nil {
		t.Fatal(err)
	}
	out, code := menuTrial(t, registerMenuScript(root, "2")+"9\n", "--binding", bind)
	if code != 1 || !strings.Contains(out, "Registration saved") || !strings.Contains(out, "Connection not completed") || strings.Contains(out, "[PASS] Foundling registered") {
		t.Fatal(code, out)
	}
	page, err := s.FoundlingsPage(0, 5)
	if err != nil || len(page.Items) != 1 {
		t.Fatal("partial registration lost", page, err)
	}
}

func TestFoundlingMenuOutputFailureAndCancellationPreventRegistration(t *testing.T) {
	s, bind, root := foundlingMenuFixture(t)
	before := treeDigest(t, s.Root())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code := run(ctx, []string{"menu", "--plain", "--binding", bind}, strings.NewReader(registerMenuScript(root, "2")), failedMenuWriter{}, failedMenuWriter{}); code == 0 {
		t.Fatal("cancelled menu succeeded")
	}
	if code := run(context.Background(), []string{"menu", "--plain", "--binding", bind}, strings.NewReader(registerMenuScript(root, "2")), failedMenuWriter{}, failedMenuWriter{}); code != 1 {
		t.Fatal(code)
	}
	if !reflect.DeepEqual(before, treeDigest(t, s.Root())) {
		t.Fatal("failed display wrote memory")
	}
}

func TestFoundlingMenuReconnectAndExplicitPinUpdate(t *testing.T) {
	s, bind, root := foundlingMenuFixture(t)
	if out, code := menuTrial(t, registerMenuScript(root, "2")+"6\n9\n", "--binding", bind); code != 0 {
		t.Fatal(code, out)
	}
	if err := os.Rename(root, root+"-moved"); err != nil {
		t.Fatal(err)
	}
	root += "-moved"
	out, code := menuTrial(t, "8\n3\n1\n"+root+"\n2\n6\n9\n", "--binding", bind)
	if code != 0 || !strings.Contains(out, "unavailable") || !strings.Contains(out, "[PASS] Local reference connected") {
		t.Fatal(code, out)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("Changed historical project: Silver Heron"), 0600); err != nil {
		t.Fatal(err)
	}
	before := treeDigest(t, s.Root())
	out, code = menuTrial(t, "8\n4\n1\n3\n\nNew source revision\n\n6\n9\n", "--binding", bind)
	if code != 0 || !strings.Contains(out, "changed") || !strings.Contains(out, "Review superseding source pin") {
		t.Fatal(code, out)
	}
	if !reflect.DeepEqual(before, treeDigest(t, s.Root())) {
		t.Fatal("declining pin update changed history")
	}
	out, code = menuTrial(t, "8\n4\n1\n3\n\nNew source revision\n2\n6\n9\n", "--binding", bind)
	if code != 0 || !strings.Contains(out, "[PASS] Source pin superseded") {
		t.Fatal(code, out)
	}
	page, err := s.FoundlingsPage(0, 5)
	if err != nil || len(page.Items) != 1 {
		t.Fatal(page, err)
	}
	history, err := s.FoundlingHistoryPage(page.Items[0].FoundlingID, 0, 5)
	if err != nil || len(history.Items) != 2 {
		t.Fatal(history, err)
	}
}
