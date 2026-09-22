package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/formatupgrade"
	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/sessionsync"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func hash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func sessionGit(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "GIT_") {
			cmd.Env = append(cmd.Env, v)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture git: %v: %s", err, raw)
	}
	return strings.TrimSpace(string(raw))
}
func sessionFixture(t *testing.T) (*API, *memory.Service, *signetsync.Synchronizer) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "signet")
	if _, err := memory.Create(root, "Synthetic", "device-fixture", "Fixture"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "binding.json")
	b, err := binding.Bind(root, path, "Fixture", "Synthetic actor")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	guard := binding.Guard{SHA256: hash(raw), SignetID: b.SignetID}
	runtime := filepath.Join(dir, "runtime")
	if err := os.WriteFile(runtime, []byte("synthetic runtime"), 0700); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(dir, "state")
	if err := os.Mkdir(state, 0700); err != nil {
		t.Fatal(err)
	}
	p := sessionsync.Policy{SchemaVersion: 1, Mode: sessionsync.EnabledMode, Binding: path, BindingSHA256: guard.SHA256, SignetID: b.SignetID, Runtime: runtime, RuntimeSHA256: hash([]byte("synthetic runtime")), StateDir: state}
	raw, _ = json.Marshal(p)
	policyPath := filepath.Join(state, "policy.json")
	if err := os.WriteFile(policyPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := sessionsync.Open(policyPath, sessionsync.Selection{PolicySHA256: hash(raw), Binding: path, Guard: guard, Runtime: runtime})
	if err != nil {
		t.Fatal(err)
	}
	s, err := binding.OpenGuarded(path, "test", guard)
	if err != nil {
		t.Fatal(err)
	}
	g, err := signetsync.Open(root, s.ID())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	remote := filepath.Join(dir, "remote.git")
	sessionGit(t, "init", "--bare", remote)
	sessionGit(t, "-C", root, "remote", "add", "origin", remote)
	a, err := NewSession(s, c)
	if err != nil {
		t.Fatal(err)
	}
	return a, s, g
}

func TestSessionPrimitivePublishesAndLegacyStaysLocal(t *testing.T) {
	a, s, g := sessionFixture(t)
	data := []byte(`{"kind":"fact","summary":"Teal","body":"Preview status is teal","basis":"user-direction","reason":"Confirmed"}`)
	local := New(s, false).Call(context.Background(), "memory_remember", data)
	if !local.OK || local.SessionSync != nil || local.Result.(Receipt).Synchronization != "not-requested" {
		t.Fatal(local)
	}
	out := a.Call(context.Background(), "memory_remember", data)
	if !out.OK || out.SessionSync == nil || out.SessionSync.Status == nil || !out.SessionSync.Status.Delivered || !out.Result.(Receipt).DurableLocally {
		t.Fatal(out)
	}
	status, err := g.Inspect(context.Background())
	if err != nil || status.Dirty {
		t.Fatal(status, err)
	}
	remote := sessionGit(t, "--git-dir="+filepath.Join(filepath.Dir(s.Root()), "remote.git"), "rev-parse", "refs/heads/main")
	if remote != out.SessionSync.Status.Head {
		t.Fatal("receipt is not remote head")
	}
	before := sessionGit(t, "-C", s.Root(), "rev-parse", "HEAD")
	denied := New(s, true).Call(context.Background(), "memory_remember", data)
	if denied.OK || denied.SessionSync != nil {
		t.Fatal(denied)
	}
	if before != sessionGit(t, "-C", s.Root(), "rev-parse", "HEAD") {
		t.Fatal("read-only mutated")
	}
}
func TestSessionCombinedJournalAndFailureReceipts(t *testing.T) {
	a, s, _ := sessionFixture(t)
	out := a.Call(context.Background(), "memory_journal_append_and_sync", []byte(`{"entry":{"kind":"outcome","summary":"Synthetic useful result"}}`))
	if !out.OK || out.SessionSync == nil || !out.SessionSync.Status.Delivered {
		t.Fatal(out)
	}
	combined := out.Result.(SaveAndSyncResult)
	if !combined.Saved.DurableLocally || !combined.Delivery.OK || combined.Delivery.Error != nil {
		t.Fatal(combined)
	}
	entries, err := s.Journal("", 100)
	if err != nil || len(entries) != 1 {
		t.Fatal(entries, err)
	}
	sessionGit(t, "-C", s.Root(), "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
	out = a.Call(context.Background(), "memory_remember_and_sync", []byte(`{"record":{"kind":"fact","summary":"Amber","body":"Synthetic amber","basis":"user-direction","reason":"Confirmed"}}`))
	if !out.OK || out.SessionSync == nil || out.SessionSync.Status.Delivered {
		t.Fatal(out)
	}
	combined = out.Result.(SaveAndSyncResult)
	if !combined.Saved.DurableLocally {
		t.Fatal(combined)
	}
	if !combined.Delivery.OK && (combined.Delivery.Result != nil || combined.Delivery.Error == nil) {
		t.Fatal("malformed failed delivery", combined)
	}
}
func TestSessionInvalidInputDoesNotSynchronize(t *testing.T) {
	a, s, _ := sessionFixture(t)
	out := a.Call(context.Background(), "memory_remember", []byte(`{"kind":"project","summary":"Bad","body":"Synthetic","basis":"user-direction","reason":"Confirmed"}`))
	if out.OK || out.SessionSync != nil {
		t.Fatal(out)
	}
	if _, err := os.Stat(filepath.Join(s.Root(), ".mandalore", "session-sync.json")); !os.IsNotExist(err) {
		t.Fatal("invalid write triggered transport", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out = a.Call(ctx, "memory_journal_append", []byte(`{"kind":"outcome","summary":"Cancelled"}`))
	if out.OK || out.SessionSync != nil {
		t.Fatal(out)
	}
}
func TestSessionEffectsAndMutationBoundary(t *testing.T) {
	a, _, _ := sessionFixture(t)
	for _, op := range a.Catalog() {
		if sessionMutation(op.Name) && !op.Network {
			t.Fatal(op.Name, "missing network effect")
		}
		if op.Name == "memory_recall" && (op.Network || strings.Contains(op.Description, "call memory_sync")) {
			t.Fatal(op.Description)
		}
	}
	for _, name := range []string{"foundling_connect", "memory_git_init", "memory_checkpoint", "memory_recall", "memory_scopes", "foundling_list"} {
		if sessionMutation(name) {
			t.Fatal("local operation classified as portable save", name)
		}
	}
	for _, name := range []string{"memory_remember", "memory_journal_append", "memory_remember_and_sync", "memory_journal_append_and_sync", "memory_withdraw", "memory_restore", "foundling_register", "foundling_disconnect", "foundling_promote"} {
		if !sessionMutation(name) {
			t.Fatal("missing portable mutation", name)
		}
	}
}
func TestWriteSchemaHintsMatchCanonicalEnums(t *testing.T) {
	raw, err := os.ReadFile("../memory/schemas/memory-revision.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var canonical struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if json.Unmarshal(raw, &canonical) != nil {
		t.Fatal("schema")
	}
	var evidence struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	json.Unmarshal(canonical.Properties["evidence"], &evidence)
	for _, op := range Catalog() {
		if op.Name != "memory_remember" && op.Name != "memory_remember_and_sync" {
			continue
		}
		schema := op.InputSchema
		if op.Name == "memory_remember_and_sync" {
			schema = schema.Properties["record"]
		}
		for _, field := range []string{"kind", "basis", "sensitivity", "volatility", "confidence"} {
			raw := canonical.Properties[field]
			if field == "basis" || field == "confidence" {
				raw = evidence.Properties[field]
			}
			var values struct {
				Enum []string `json:"enum"`
			}
			json.Unmarshal(raw, &values)
			if len(values.Enum) == 0 {
				t.Fatal(field)
			}
			for _, value := range values.Enum {
				if !strings.Contains(schema.Properties[field].Description, value) {
					t.Fatal(op.Name, field, value)
				}
			}
		}
	}
}

func TestSessionPartialRegistrationStillDeliversPublishedReceipt(t *testing.T) {
	a, _, _ := sessionFixture(t)
	root, preview := referenceFixture(t, a)
	if err := os.WriteFile(filepath.Join(a.service.Root(), ".mandalore", "foundlings"), []byte("preserve unknown local mapping"), 0600); err != nil {
		t.Fatal(err)
	}
	out := foundlingCall(t, a, "foundling_register", FoundlingRegisterInput{Name: "Historical notes", Description: "Reference only", Source: preview.Source, Pin: preview.Pin, Root: root, Reason: "Explicit selection"})
	if out.OK || out.Error == nil || !out.Error.WriteMayHaveOccurred || out.Error.FoundlingResult == nil || out.Error.FoundlingResult.Registration == nil {
		t.Fatal(out)
	}
	if out.SessionSync == nil || out.SessionSync.Status == nil || !out.SessionSync.Status.Delivered {
		t.Fatal("partial published registration was not delivered", out)
	}
}

func TestSessionFoundlingMutationsDeliver(t *testing.T) {
	a, _, _ := sessionFixture(t)
	root, preview := referenceFixture(t, a)
	registration := sessionFoundlingCall(t, a, "foundling_register", FoundlingRegisterInput{Name: "Historical notes", Description: "Reference only", Source: preview.Source, Pin: preview.Pin, Root: root, Reason: "Explicit selection"})
	if !registration.OK {
		t.Fatal(registration)
	}
	r := registration.Result.(FoundlingMutationResult)
	if r.Registration == nil || r.Connection == nil || !r.Connection.Connected {
		t.Fatal(r)
	}
	id := r.Registration.FoundlingID
	list := a.Call(context.Background(), "foundling_list", []byte(`{}`))
	if !list.OK || len(list.Result.(memory.Page[memory.FoundlingSummary]).Items) != 1 {
		t.Fatal(list)
	}
	inspection := sessionFoundlingCall(t, a, "foundling_inspect", FoundlingSelector{FoundlingID: id})
	if !inspection.OK || inspection.Result.(foundlings.Inspection).State != "available" {
		t.Fatal(inspection)
	}
	search := sessionFoundlingCall(t, a, "foundling_search", FoundlingSearchInput{FoundlingID: id, Query: "Copper Finch"})
	if !search.OK || len(search.Result.(foundlings.SearchResult).Items) != 1 {
		t.Fatal(search)
	}
	hit := search.Result.(foundlings.SearchResult).Items[0]
	read := sessionFoundlingCall(t, a, "foundling_read", FoundlingReadInput{FoundlingID: id, RegistrationID: r.Registration.ID, Locator: hit.Origin.RelativeLocator})
	if !read.OK || !read.Result.(foundlings.Excerpt).Complete {
		t.Fatal(read)
	}
	promote := foundlings.PromotionInput{FoundlingID: id, RegistrationID: r.Registration.ID, Locator: hit.Origin.RelativeLocator, ContentSHA256: hit.Origin.ContentSHA256, Write: memory.Write{Kind: "decision", Summary: "Project name", Body: "Silver Heron is current; Copper Finch was historical.", Basis: "import", Reason: "Adapt to current user direction"}}
	saved := sessionFoundlingCall(t, a, "foundling_promote", promote)
	if !saved.OK || !saved.Result.(Receipt).DurableLocally {
		t.Fatal(saved)
	}
	a.ReadOnly = true
	for _, name := range []string{"foundling_register", "foundling_connect", "foundling_disconnect", "foundling_promote"} {
		out := a.Call(context.Background(), name, []byte(`{}`))
		if out.OK || out.Error.Code != "operation.read_only" || out.Error.WriteMayHaveOccurred {
			t.Fatal(name, out)
		}
	}
	a.ReadOnly = false
	off := sessionFoundlingCall(t, a, "foundling_disconnect", FoundlingDisconnectInput{FoundlingID: id, RegistrationID: r.Registration.ID, Reason: "Disconnect, retain learned knowledge"})
	if !off.OK {
		t.Fatal(off)
	}
	refused := sessionFoundlingCall(t, a, "foundling_promote", promote)
	if refused.OK || refused.Error.Code != "foundling.changed" || refused.Error.WriteMayHaveOccurred {
		t.Fatal(refused)
	}
	if recall := a.Call(context.Background(), "memory_recall", []byte(`{"query":"Silver Heron"}`)); !recall.OK || recall.Result.(memory.RecallPacket).MatchingCount != 1 {
		t.Fatal(recall)
	}
}

func sessionFoundlingCall(t *testing.T, a *API, name string, in any) Envelope {
	t.Helper()
	out := foundlingCall(t, a, name, in)
	if out.OK && sessionMutation(name) {
		assertSessionDelivered(t, a, out)
	}
	return out
}
func assertSessionDelivered(t *testing.T, a *API, out Envelope) {
	t.Helper()
	if out.SessionSync == nil || out.SessionSync.Status == nil || !out.SessionSync.Status.Delivered {
		t.Fatal("missing automatic delivery", out)
	}
	remote := sessionGit(t, "--git-dir="+filepath.Join(filepath.Dir(a.service.Root()), "remote.git"), "rev-parse", "refs/heads/main")
	if remote != out.SessionSync.Status.Head {
		t.Fatal("remote differs from delivery receipt", remote, out.SessionSync)
	}
}
func TestSessionVisibilityMutationsDeliver(t *testing.T) {
	a, s, syncer := sessionFixture(t)
	r, err := s.Remember(memory.Write{Kind: "fact", Summary: "Synthetic record", Body: "Teal convention", Basis: "user-direction", Reason: "Confirmed"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := syncer.Checkpoint(context.Background()); err != nil {
		t.Fatal(err)
	}
	bindingPath := filepath.Join(filepath.Dir(s.Root()), "binding.json")
	p := inlineCall(t, New(nil, true), "signet_upgrade_preview", formatupgrade.Request{BindingPath: bindingPath}).Result.(formatupgrade.Plan)
	activated := inlineCall(t, New(nil, false), "signet_upgrade_apply", formatupgrade.ApplyRequest{Plan: p, StoppedWriters: true})
	if !activated.OK {
		t.Fatal(activated)
	}
	s, err = binding.Open(bindingPath, "test")
	if err != nil {
		t.Fatal(err)
	}
	a, err = NewSession(s, a.session)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"memory_withdraw", "memory_restore"} {
		h := inlineCall(t, a, "memory_visibility_history", HistoryInput{RecordID: r.RecordID}).Result.(memory.VisibilityHistory)
		out := inlineCall(t, a, name, memory.VisibilityWrite{RecordID: r.RecordID, ContentHeads: h.State.ContentHeads, VisibilityHeads: h.State.VisibilityHeads, Reason: "Explicit decision"})
		assertSessionDelivered(t, a, out)
		receipt := out.Result.(memory.VisibilityReceipt)
		if !receipt.DurableLocally || receipt.Synchronization == "not-requested" {
			t.Fatal(receipt)
		}
	}
}
