package foundlings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func canonFixture(t *testing.T) (*Manager, *memory.Service, memory.FoundlingRegistration, memory.Revision, string) {
	t.Helper()
	makeService := func(name string) *memory.Service {
		s, e := memory.Create(filepath.Join(t.TempDir(), name), name, "device-test", "Synthetic test")
		if e != nil {
			t.Fatal(e)
		}
		v, e := memory.OpenService(s.Root, memory.Authorship{DeviceID: "device-test", Actor: "Synthetic actor", Harness: "test"})
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	source := makeService("source")
	upgradeCanonFixture(t, source)
	primary := makeService("primary")
	rev, e := source.Remember(memory.Write{Kind: "fact", Summary: "Current color", Body: "Synthetic color blue", Basis: "observation", Reason: "fixture"})
	if e != nil {
		t.Fatal(e)
	}
	gitTest(t, source.Root(), "init", "-b", "main")
	gitTest(t, source.Root(), "config", "user.name", "Synthetic")
	gitTest(t, source.Root(), "config", "user.email", "synthetic@example.invalid")
	gitTest(t, source.Root(), "add", ".")
	gitTest(t, source.Root(), "commit", "-m", "Initial")
	pin := strings.TrimSpace(gitTest(t, source.Root(), "rev-parse", "HEAD"))
	reg, e := primary.WriteFoundling(memory.FoundlingWrite{Name: "Canon fixture", Description: "Synthetic external signet", Mode: "canon", Branch: "main", SourceSignetID: source.ID(), Source: memory.FoundlingSource{Kind: "git", Locator: "https://github.com/example/synthetic.git"}, Pin: memory.SourcePin{Algorithm: "git-sha1", Value: pin}, State: "active", Reason: "Explicit registration"})
	if e != nil {
		t.Fatal(e)
	}
	m := New(primary)
	m.canonTransport = func(string) string { return source.Root() }
	return m, source, reg, rev, source.Root()
}
func gitTest(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	b, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("git %v: %s %v", args, b, e)
	}
	return string(b)
}
func refreshTest(t *testing.T, m *Manager, session string) CanonReceipt {
	t.Helper()
	r, e := m.SessionRefresh(context.Background(), session)
	if e != nil || len(r) != 1 || r[0].State != "available" {
		t.Fatalf("refresh %v %v", r, e)
	}
	return r[0]
}
func recallTest(t *testing.T, m *Manager, id, session string) CanonRecallResult {
	t.Helper()
	r, e := m.CanonRecall(context.Background(), CanonRecallInput{SessionID: session, FoundlingID: id, Limit: 10, BudgetBytes: 8192})
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestCanonRefreshSessionIsolationAndReadOnly(t *testing.T) {
	m, source, reg, rev, root := canonFixture(t)
	before := fileTree(t, root)
	first := refreshTest(t, m, "session-one")
	if !reflect.DeepEqual(before, fileTree(t, root)) {
		t.Fatal("fetch changed source")
	}
	if got := recallTest(t, m, reg.FoundlingID, "session-one"); len(got.Memory.Current) != 1 || got.Memory.Current[0].ID != rev.ID {
		t.Fatal(got)
	}
	newer, e := source.Remember(memory.Write{Kind: "fact", Summary: "Current color", Body: "Synthetic color green", Basis: "observation", Reason: "Change", RecordID: rev.RecordID, Supersedes: []string{rev.ID}})
	if e != nil {
		t.Fatal(e)
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "Change")
	second := refreshTest(t, m, "session-two")
	if first.Pin == second.Pin {
		t.Fatal("branch advance not reflected")
	}
	if got := recallTest(t, m, reg.FoundlingID, "session-one"); got.Memory.Current[0].ID != rev.ID {
		t.Fatal("session drift", got)
	}
	if got := recallTest(t, m, reg.FoundlingID, "session-two"); len(got.Memory.Current) != 1 || got.Memory.Current[0].ID != newer.ID {
		t.Fatal(got)
	}
	registrations, e := m.memory.FoundlingHistoryPage(reg.FoundlingID, 0, 50)
	if e != nil || len(registrations.Items) != 1 {
		t.Fatal("fetch mutated registration", registrations, e)
	}
	localBefore := fileTree(t, m.memory.Root())
	if _, e = m.SessionStatus(context.Background(), "session-one"); e != nil {
		t.Fatal(e)
	}
	_ = recallTest(t, m, reg.FoundlingID, "session-one")
	if !reflect.DeepEqual(localBefore, fileTree(t, m.memory.Root())) {
		t.Fatal("read wrote cache")
	}
	if _, e = m.Search(context.Background(), SearchInput{FoundlingID: reg.FoundlingID, Query: "blue", Limit: 1}); e == nil {
		t.Fatal("canon accepted raw history search")
	}
}
func TestCanonOfflineAbsentBranchAndCancellation(t *testing.T) {
	m, _, reg, _, root := canonFixture(t)
	first := refreshTest(t, m, "online")
	m.canonTransport = func(string) string { return filepath.Join(t.TempDir(), "absent") }
	r, e := m.SessionRefresh(context.Background(), "offline")
	if e != nil || r[0].State != "stale" || r[0].Pin != first.Pin || r[0].Reason != "transport_unavailable" {
		t.Fatal(r, e)
	}
	cache, _ := m.canonRoot(false)
	if e = os.Remove(filepath.Join(cache, latestFile(reg.FoundlingID))); e != nil {
		t.Fatal(e)
	}
	r, e = m.SessionRefresh(context.Background(), "uncached")
	if e != nil || r[0].State != "unavailable" {
		t.Fatal(r, e)
	}
	m.canonTransport = func(string) string { return root }
	refreshTest(t, m, "online")
	gitTest(t, root, "branch", "-m", "different")
	r, e = m.SessionRefresh(context.Background(), "absent-branch")
	if e != nil || r[0].State != "stale" || r[0].Reason != "absent_branch" {
		t.Fatal(r, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	r, e = m.SessionRefresh(ctx, "cancelled")
	if !errors.Is(e, context.Canceled) || time.Since(start) > time.Second {
		t.Fatal(r, e)
	}
}

func TestCanonWithdrawalConflictsPromotionAndDisconnection(t *testing.T) {
	m, source, reg, rev, root := canonFixture(t)
	refreshTest(t, m, "initial")
	// Separate replicas create valid concurrent heads.
	replica := filepath.Join(t.TempDir(), "replica")
	if e := os.CopyFS(replica, os.DirFS(root)); e != nil {
		t.Fatal(e)
	}
	other, e := memory.OpenService(replica, memory.Authorship{DeviceID: "device-test", Actor: "Synthetic actor", Harness: "test"})
	if e != nil {
		t.Fatal(e)
	}
	for i, color := range []string{"green", "red"} {
		writer := source
		if i == 1 {
			writer = other
		}
		_, e := writer.Remember(memory.Write{Kind: "fact", Summary: "Current color", Body: color, Basis: "observation", Reason: "Concurrent branch", RecordID: rev.RecordID, Supersedes: []string{rev.ID}})
		if e != nil {
			t.Fatal(e)
		}
	}

	for _, dir := range []string{"memory/records", "memory/sources"} {
		e = filepath.WalkDir(filepath.Join(replica, dir), func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(replica, p)
			b, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			dest := filepath.Join(root, rel)
			if e = os.MkdirAll(filepath.Dir(dest), 0700); e != nil {
				return e
			}
			return os.WriteFile(dest, b, 0600)
		})
		if e != nil {
			t.Fatal(e)
		}
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "Conflict")
	refreshTest(t, m, "conflict")
	got := recallTest(t, m, reg.FoundlingID, "conflict")
	if len(got.Memory.Current) != 0 || len(got.Memory.Conflicts) != 1 || len(got.Memory.Conflicts[0].HeadIDs) != 2 {
		t.Fatal(got)
	}
	in := CanonRecallInput{SessionID: "conflict", FoundlingID: reg.FoundlingID, RegistrationID: reg.ID}
	heads, receipt, e := m.CanonHeads(context.Background(), in, rev.RecordID)
	if e != nil || len(heads) != 2 {
		t.Fatal(heads, e)
	}
	promoted, e := m.CanonPromote(context.Background(), CanonPromotionInput{SessionID: "conflict", FoundlingID: reg.FoundlingID, RegistrationID: reg.ID, RecordID: rev.RecordID, RevisionID: heads[0].ID, ContentSHA256: CanonRevisionDigest(heads[0]), Write: memory.Write{Kind: "fact", Summary: "Reviewed external claim", Body: "A deliberately selected conflicting claim", Basis: "import", Reason: "Explicit synthetic promotion"}})
	if e != nil {
		t.Fatal(e)
	}
	if promoted.Evidence.Basis != "import" {
		t.Fatal(promoted)
	}
	// Withdrawal is observed only after explicit refresh; previous session remains pinned.
	_, e = source.ChangeVisibility(context.Background(), "withdraw", memory.VisibilityWrite{RecordID: rev.RecordID, ContentHeads: got.Memory.Conflicts[0].HeadIDs, VisibilityHeads: []string{}, Reason: "Synthetic withdrawal"})
	if e != nil {
		t.Fatal(e)
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "Withdraw")
	refreshTest(t, m, "withdrawn")
	hidden := recallTest(t, m, reg.FoundlingID, "withdrawn")
	if len(hidden.Memory.Current) != 0 || len(hidden.Memory.Conflicts) != 0 {
		t.Fatal(hidden)
	}
	heads, _, e = m.CanonHeads(context.Background(), CanonRecallInput{SessionID: "withdrawn", FoundlingID: reg.FoundlingID}, rev.RecordID)
	if e != nil || len(heads) != 0 {
		t.Fatal(heads, e)
	}
	if receipt.Pin == hidden.Reference.Pin {
		t.Fatal("refresh unchanged")
	}
	_, e = m.memory.WriteFoundling(memory.FoundlingWrite{FoundlingID: reg.FoundlingID, Name: reg.Name, Description: reg.Description, Mode: reg.Mode, Branch: reg.Branch, SourceSignetID: reg.SourceSignetID, Source: reg.Source, Pin: reg.Pin, State: "disconnected", Supersedes: []string{reg.ID}, Reason: "Disconnect"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = m.CanonRecall(context.Background(), in); e == nil {
		t.Fatal("disconnected old snapshot readable")
	}
	p, e := m.memory.Recall("", nil, 10, 8192)
	if e != nil || len(p.Current) != 1 {
		t.Fatal("disconnect erased explicit promotion", p, e)
	}
}
func TestCanonInvalidSourceWithholdsAndCacheTampering(t *testing.T) {
	m, _, reg, _, root := canonFixture(t)
	refreshTest(t, m, "before")
	path := filepath.Join(root, "signet.json")
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	b = []byte(strings.Replace(string(b), reg.SourceSignetID, memory.NewID("signet"), 1))
	if e = os.WriteFile(path, b, 0600); e != nil {
		t.Fatal(e)
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "Wrong identity")
	r, e := m.SessionRefresh(context.Background(), "wrong")
	if e != nil || r[0].State != "unavailable" || r[0].Reason != "wrong_source_identity" {
		t.Fatal(r, e)
	}
	cache, _ := m.canonRoot(false)
	var old CanonReceipt
	if e = canonRead(cache, sessionFile("before", reg.FoundlingID), &old); e != nil {
		t.Fatal(e)
	}
	snapshot := filepath.Join(cache, "commit-"+old.Pin.Value+"-"+old.Digest)
	if e = os.WriteFile(filepath.Join(snapshot, "signet.json"), b, 0600); e != nil {
		t.Fatal(e)
	}
	status, e := m.SessionStatus(context.Background(), "before")
	if e != nil || status[0].State != "unavailable" || status[0].Reason != "invalid_snapshot" {
		t.Fatal(status, e)
	}
}
func TestLegacyOnlySessionDoesNotCreateCanonState(t *testing.T) {
	m, _, _ := connectedFixture(t)
	before := fileTree(t, m.memory.Root())
	r, e := m.SessionRefresh(context.Background(), "")
	if e != nil || len(r) != 0 {
		t.Fatal(r, e)
	}
	r, e = m.SessionStatus(context.Background(), "")
	if e != nil || len(r) != 0 {
		t.Fatal(r, e)
	}
	if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) {
		t.Fatal("legacy startup mutated local state")
	}
}

func upgradeCanonFixture(t *testing.T, s *memory.Service) {
	t.Helper()
	path := filepath.Join(s.Root(), "signet.json")
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var manifest memory.Signet
	if e = json.Unmarshal(b, &manifest); e != nil {
		t.Fatal(e)
	}
	u := memory.UpgradeRecord{Version: 1, ID: "upgrade-test", SignetID: s.ID(), From: 1, To: 2, OriginalManifestSHA256: hash(b), PortableSHA256: strings.Repeat("b", 64), BaseHead: strings.Repeat("c", 40), RecordedAt: "2026-09-18T12:00:00Z", Authorship: memory.Authorship{DeviceID: "device-test", Actor: "Synthetic actor", Harness: "test"}}
	dir := filepath.Join(s.Root(), "provenance/upgrades")
	if e = os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	data, _ := json.Marshal(u)
	if e = os.WriteFile(filepath.Join(dir, u.ID+".json"), data, 0600); e != nil {
		t.Fatal(e)
	}
	manifest.Version = 2
	data, _ = json.Marshal(manifest)
	if e = os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
	reopened, e := memory.OpenService(s.Root(), memory.Authorship{DeviceID: "device-test", Actor: "Synthetic actor", Harness: "test"})
	if e != nil {
		t.Fatal(e)
	}
	*s = *reopened
}

func TestCanonRewriteRestoreAndUnsupportedFormat(t *testing.T) {
	m, source, reg, rev, root := canonFixture(t)
	first := refreshTest(t, m, "one")
	withdrawal, e := source.ChangeVisibility(context.Background(), "withdraw", memory.VisibilityWrite{RecordID: rev.RecordID, ContentHeads: []string{rev.ID}, VisibilityHeads: []string{}, Reason: "withdraw"})
	if e != nil {
		t.Fatal(e)
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "withdraw")
	refreshTest(t, m, "two")
	_, e = source.ChangeVisibility(context.Background(), "restore", memory.VisibilityWrite{RecordID: rev.RecordID, ContentHeads: []string{rev.ID}, VisibilityHeads: []string{withdrawal.EventID}, Reason: "restore"})
	if e != nil {
		t.Fatal(e)
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "restore")
	refreshTest(t, m, "three")
	if p := recallTest(t, m, reg.FoundlingID, "three"); len(p.Memory.Current) != 1 {
		t.Fatal(p)
	}
	// A non-fast-forward upstream rewrite is observed as another immutable pin.
	gitTest(t, root, "reset", "--hard", first.Pin.Value)
	rewritten := refreshTest(t, m, "four")
	if rewritten.Pin != first.Pin {
		t.Fatal(rewritten)
	}
	if p := recallTest(t, m, reg.FoundlingID, "two"); len(p.Memory.Current) != 0 {
		t.Fatal("rewrite changed previous session")
	}
	path := filepath.Join(root, "signet.json")
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var data map[string]any
	if e = json.Unmarshal(b, &data); e != nil {
		t.Fatal(e)
	}
	data["schema_version"] = 999
	b, _ = json.Marshal(data)
	if e = os.WriteFile(path, b, 0600); e != nil {
		t.Fatal(e)
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "unsupported")
	r, e := m.SessionRefresh(context.Background(), "unsupported")
	if e != nil || r[0].State != "unavailable" || r[0].Reason != "invalid_source_format" {
		t.Fatal(r, e)
	}
}
func TestCanonFindsRegistrationBeyondFirstPageAndNoSourceExecution(t *testing.T) {
	m, _, reg, _, root := canonFixture(t)
	store, e := memory.Open(m.memory.Root())
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 55; i++ {
		legacy := reg
		legacy.Version = 1
		legacy.Mode = ""
		legacy.Branch = ""
		legacy.SourceSignetID = ""
		legacy.ID = memory.NewID("registration")
		legacy.FoundlingID = fmt.Sprintf("aaa-legacy-%03d", i)
		legacy.Source = memory.FoundlingSource{Kind: "local", Locator: "fixture-history"}
		legacy.Pin = memory.SourcePin{Algorithm: "sha256", Value: strings.Repeat("a", 64)}
		if e = store.PutFoundlingRegistration(legacy); e != nil {
			t.Fatal(e)
		}
	}
	first, e := m.memory.FoundlingsPage(0, 50)
	if e != nil {
		t.Fatal(e)
	}
	for _, entry := range first.Items {
		if entry.FoundlingID == reg.FoundlingID {
			t.Fatal("fixture canon must be beyond first page")
		}
	}
	marker := filepath.Join(t.TempDir(), "executed")
	// Neither a source's executable hooks nor tracked code execute on fetch.
	if e := os.WriteFile(filepath.Join(root, ".git/hooks/post-upload-pack"), []byte("#!/bin/sh\ntouch "+marker+"\n"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "script.sh"), []byte("#!/bin/sh\ntouch "+marker+"\n"), 0700); e != nil {
		t.Fatal(e)
	}
	gitTest(t, root, "add", "script.sh")
	gitTest(t, root, "commit", "-m", "Tracked inert code")
	receipt := refreshTest(t, m, "page")
	if receipt.FoundlingID != reg.FoundlingID {
		t.Fatal(receipt)
	}
	if _, e := os.Stat(marker); !os.IsNotExist(e) {
		t.Fatal("source content executed", e)
	}
}

func TestCanonConcurrentSessionRefreshAndBoundedSummary(t *testing.T) {
	m, _, reg, _, _ := canonFixture(t)
	type result struct {
		receipts []CanonReceipt
		err      error
	}
	done := make(chan result, 2)
	for _, session := range []string{"parallel-a", "parallel-b"} {
		go func(session string) { r, e := m.SessionRefresh(context.Background(), session); done <- result{r, e} }(session)
	}
	for i := 0; i < 2; i++ {
		r := <-done
		if r.err != nil || len(r.receipts) != 1 || r.receipts[0].State != "available" {
			t.Fatal(r)
		}
	}
	a := recallTest(t, m, reg.FoundlingID, "parallel-a")
	b := recallTest(t, m, reg.FoundlingID, "parallel-b")
	if a.Reference.Pin != b.Reference.Pin || a.Reference.SessionID == b.Reference.SessionID {
		t.Fatal(a, b)
	}
	receipts := make([]CanonReceipt, 100)
	for i := range receipts {
		receipts[i] = a.Reference
		receipts[i].Name = strings.Repeat("界", 80)
	}
	summary := CanonSummary(receipts)
	if len(summary) > 1200 || !json.Valid([]byte(summary)) || !strings.Contains(summary, `"truncated":true`) {
		t.Fatal(summary)
	}
}
func TestCanonTransportCancellationTerminatesProcessGroup(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\nsleep 30 &\nwait\n"
	if e := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, e := canonGit(ctx, t.TempDir(), false, "fetch")
	if !errors.Is(e, context.DeadlineExceeded) || time.Since(start) > 2*time.Second {
		t.Fatal(e, time.Since(start))
	}
}

func TestCanonFetchDeadlineReservesVerifiedStaleFallback(t *testing.T) {
	m, _, reg, _, _ := canonFixture(t)
	first := refreshTest(t, m, "seed")
	realGit, e := exec.LookPath("git")
	if e != nil {
		t.Fatal(e)
	}
	bin := t.TempDir()
	// Init remains real. The actual fetch subprocess reaches its deadline, so the
	// regression cannot pass merely by classifying an immediate mocked error.
	script := "#!/bin/sh\nfor arg do\nif [ \"$arg\" = fetch ]; then\nsleep 30 &\nwait\nexit $?\nfi\ndone\nexec '" + strings.ReplaceAll(realGit, "'", "'\\''") + "' \"$@\"\n"
	if e = os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	start := time.Now()
	receipts, e := m.SessionRefresh(context.Background(), "timed-out")
	elapsed := time.Since(start)
	if e != nil || len(receipts) != 1 || receipts[0].State != "stale" || receipts[0].Reason != "timeout" || receipts[0].Pin != first.Pin || receipts[0].RefreshedAt != first.RefreshedAt {
		t.Fatal(receipts, e)
	}
	if elapsed < 1800*time.Millisecond || elapsed >= 3*time.Second {
		t.Fatalf("fetch and stale validation outside shared budget: %s", elapsed)
	}
	if got := recallTest(t, m, reg.FoundlingID, "timed-out"); got.Reference.State != "stale" || len(got.Memory.Current) != 1 {
		t.Fatal(got)
	}
	// Parent cancellation is not the reserved internal fetch deadline: it must
	// withhold even a previously verified snapshot rather than read after cancel.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	timer := time.AfterFunc(150*time.Millisecond, cancel)
	defer timer.Stop()
	receipts, e = m.SessionRefresh(ctx, "cancelled-during-fetch")
	if e != nil || len(receipts) != 1 || receipts[0].State != "unavailable" || receipts[0].Reason != "cancelled" {
		t.Fatal(receipts, e)
	}
}
