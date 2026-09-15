package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
)

type bindingSwapWriter struct {
	buffer    bytes.Buffer
	onConfirm func()
}

type searchBoundaryWriter struct {
	buffer bytes.Buffer
	onPage func()
}

func (w *searchBoundaryWriter) String() string { return w.buffer.String() }

func (w *searchBoundaryWriter) Write(p []byte) (int, error) {
	n, err := w.buffer.Write(p)
	if w.onPage != nil && strings.Contains(w.String(), "Search results") {
		f := w.onPage
		w.onPage = nil
		f()
	}
	return n, err
}

func TestFoundlingMenuStaleNextPageDoesNotOfferBudgetRetry(t *testing.T) {
	s, bind, root := foundlingMenuFixture(t)
	for i := 0; i < 4; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("page-%d.md", i)), []byte("PageMarker historical note."), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if out, code := menuTrial(t, registerMenuScript(root, "2")+"6\n10\n", "--binding", bind); code != 0 {
		t.Fatal(code, out)
	}
	before := treeDigest(t, s.Root())
	out := &searchBoundaryWriter{onPage: func() {
		if err := os.WriteFile(filepath.Join(root, "page-3.md"), []byte("CHANGED-CONTENT-CANARY"), 0600); err != nil {
			t.Fatal(err)
		}
	}}
	code := run(context.Background(), []string{"menu", "--plain", "--binding", bind}, strings.NewReader("8\n4\n1\n1\nPageMarker\n1\n10\n"), out, out)
	if code != 1 || out.onPage != nil || !strings.Contains(out.String(), "reference content or identity changed") || strings.Contains(out.String(), "32768-byte") || strings.Contains(out.String(), "CHANGED-CONTENT-CANARY") {
		t.Fatal("stale next page was not refused without budget retry", code, out.String())
	}
	if !reflect.DeepEqual(before, treeDigest(t, s.Root())) {
		t.Fatal("stale page caused a signet change")
	}
}

func (w *bindingSwapWriter) String() string { return w.buffer.String() }

func (w *bindingSwapWriter) Write(p []byte) (int, error) {
	if w.onConfirm != nil && strings.Contains(string(p), "Apply these changes?") {
		f := w.onConfirm
		w.onConfirm = nil
		f()
	}
	return w.buffer.Write(p)
}

func TestFoundlingMenuPinsItsSelectedSignetThroughConfirmation(t *testing.T) {
	first, bind, source := foundlingMenuFixture(t)
	second, secondBind, _ := foundlingMenuFixture(t)
	other, err := os.ReadFile(secondBind)
	if err != nil {
		t.Fatal(err)
	}
	out := &bindingSwapWriter{onConfirm: func() {
		if err := os.WriteFile(bind, other, 0600); err != nil {
			t.Fatal(err)
		}
	}}
	// The first confirmation must keep its original bank. Back and re-entry
	// deliberately pick up the replacement binding for a second registration.
	script := registerMenuScript(source, "2") + "6\n" + registerMenuScript(source, "2") + "6\n10\n"
	code := run(context.Background(), []string{"menu", "--plain", "--binding", bind}, strings.NewReader(script), out, out)
	if code != 0 || out.onConfirm != nil {
		t.Fatal("confirmation was not exercised", code, out.String())
	}
	for _, target := range []struct {
		service *memory.Service
		count   int
	}{{first, 1}, {second, 1}} {
		p, err := target.service.FoundlingsPage(0, 5)
		if err != nil || len(p.Items) != target.count {
			t.Fatal("confirmation redirected to another signet", p, err)
		}
	}
}

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

func TestFoundlingMenuProgressivePagesAndExpandedRead(t *testing.T) {
	s, bind, root := foundlingMenuFixture(t)
	for i := 0; i < 12; i++ {
		text := fmt.Sprintf("PageMarker document %02d. ", i) + strings.Repeat("Historical context. ", 80) + fmt.Sprintf("END-NOTE-%02d", i)
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("page-%02d.md", i)), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if out, code := menuTrial(t, registerMenuScript(root, "2")+"6\n10\n", "--binding", bind); code != 0 {
		t.Fatal(code, out)
	}
	before, sourceBefore := treeDigest(t, s.Root()), treeDigest(t, root)
	// Three Next selections reach the fourth page; complete-page Back returns
	// through the ordinary reference actions. Then explicitly expand a read.
	out, code := menuTrial(t, "8\n4\n1\n1\nPageMarker\n1\n1\n1\n2\npage-11.md\n0\n8192\n4\n6\n10\n", "--binding", bind)
	if code != 0 || !strings.Contains(out, "page-11.md") || !strings.Contains(out, "END-NOTE-11") || strings.Count(out, "Next page") != 3 || !strings.Contains(out, "Read content bytes") {
		t.Fatal("progressive page/read journey failed", code, out)
	}
	if !reflect.DeepEqual(before, treeDigest(t, s.Root())) || !reflect.DeepEqual(sourceBefore, treeDigest(t, root)) {
		t.Fatal("menu retrieval wrote source or signet")
	}
	for _, size := range []string{"", "8193", "bad", ":back"} {
		out, code := menuTrial(t, "8\n4\n1\n2\npage-11.md\n0\n"+size+"\n4\n6\n10\n", "--binding", bind)
		if strings.Contains(out, "END-NOTE-11") || (size == "" && code != 0) || ((size == "8193" || size == "bad") && code != 1) {
			t.Fatal("read default/validation/cancellation failed", size, code, out)
		}
		if !reflect.DeepEqual(before, treeDigest(t, s.Root())) {
			t.Fatal("read validation/cancellation wrote state")
		}
	}
}

func TestFoundlingMenuBudgetRecoveryIsExplicit(t *testing.T) {
	for _, choice := range []string{"", "1", ":back"} {
		s, bind, root := foundlingMenuFixture(t)
		dir := root
		for i := 0; i < 4; i++ {
			dir = filepath.Join(dir, strings.Repeat("<", 180))
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "heavy.md"), []byte("HeavyMarker TOKEN-HEAVY "+strings.Repeat("\x01", 1024)), 0600); err != nil {
			t.Fatal(err)
		}
		if out, code := menuTrial(t, registerMenuScript(root, "2")+"6\n10\n", "--binding", bind); code != 0 {
			t.Fatal(code, out)
		}
		before := treeDigest(t, s.Root())
		out, code := menuTrial(t, "8\n4\n1\n1\nHeavyMarker\n"+choice+"\n4\n6\n10\n", "--binding", bind)
		if code != 0 || !strings.Contains(out, "32768") || (strings.Contains(out, "TOKEN-HEAVY") != (choice == "1")) {
			t.Fatal("budget recovery did not respect explicit choice", choice, code, out)
		}
		if !reflect.DeepEqual(before, treeDigest(t, s.Root())) {
			t.Fatal("budget recovery changed signet")
		}
	}
}

func registerMenuScript(root, consent string) string {
	return strings.Join([]string{"8", "2", "Historical notes", "Reference only", "1", "source-history", root, "Explicit historical reference", consent}, "\n") + "\n"
}

func TestFoundlingMenuRegisterCancelEOFAndDefaultNo(t *testing.T) {
	for _, ending := range []string{"\n6\n10\n", ":back\n6\n10\n", "2"} {
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
	script := registerMenuScript(root, "2") + "1\n4\n1\n1\nCopper Finch\n4\n5\n1\nNo longer needed\n2\n6\n10\n"
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
	out, code := menuTrial(t, registerMenuScript(root, "2")+"10\n", "--binding", bind)
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
	if out, code := menuTrial(t, registerMenuScript(root, "2")+"6\n10\n", "--binding", bind); code != 0 {
		t.Fatal(code, out)
	}
	if err := os.Rename(root, root+"-moved"); err != nil {
		t.Fatal(err)
	}
	root += "-moved"
	out, code := menuTrial(t, "8\n3\n1\n"+root+"\n2\n6\n10\n", "--binding", bind)
	if code != 0 || !strings.Contains(out, "unavailable") || !strings.Contains(out, "[PASS] Local reference connected") {
		t.Fatal(code, out)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("Changed historical project: Silver Heron"), 0600); err != nil {
		t.Fatal(err)
	}
	before := treeDigest(t, s.Root())
	out, code = menuTrial(t, "8\n4\n1\n3\n\nNew source revision\n\n6\n10\n", "--binding", bind)
	if code != 0 || !strings.Contains(out, "changed") || !strings.Contains(out, "Review superseding source pin") {
		t.Fatal(code, out)
	}
	if !reflect.DeepEqual(before, treeDigest(t, s.Root())) {
		t.Fatal("declining pin update changed history")
	}
	out, code = menuTrial(t, "8\n4\n1\n3\n\nNew source revision\n2\n6\n10\n", "--binding", bind)
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
