package razorcrest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

func lifecycleGit(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "GIT_") {
			cmd.Env = append(cmd.Env, v)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture Git %v: %v: %s", args, err, b)
	}
	return strings.TrimSpace(string(b))
}
func lifecycleJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

type lifecycleSave struct {
	Saved    api.Receipt `json:"saved"`
	Delivery struct {
		OK     bool               `json:"ok"`
		Result *signetsync.Status `json:"result"`
	} `json:"delivery"`
}

func lifecycleReceipt(t *testing.T, v api.Envelope) lifecycleSave {
	t.Helper()
	if !v.OK {
		t.Fatalf("save failed: %+v", v.Error)
	}
	var r lifecycleSave
	if err := json.Unmarshal(lifecycleJSON(t, v.Result), &r); err != nil {
		t.Fatal(err)
	}
	if !r.Saved.DurableLocally || r.Saved.ID == "" {
		t.Fatalf("missing durable receipt: %+v", r)
	}
	return r
}
func lifecyclePrincipal() Principal {
	return Principal{Subject: "test-subject", Read: true, Write: true}
}

func TestLifecycleFailedDeliveryRestartAndRetry(t *testing.T) {
	s, _, _ := serviceFixture(t, true)
	s.config.Synchronization = true
	if out := s.api.Call(context.Background(), "memory_git_init", []byte(`{}`)); !out.OK {
		t.Fatal(out)
	}
	remote := filepath.Join(t.TempDir(), "origin.git")
	lifecycleGit(t, "init", "--bare", "--initial-branch=main", remote)
	lifecycleGit(t, "-C", s.memory.Root(), "remote", "add", "origin", remote)
	if out := s.delivery(context.Background()); !out.OK {
		t.Fatal(out)
	}
	unavailable := remote + ".offline"
	if err := os.Rename(remote, unavailable); err != nil {
		t.Fatal(err)
	}
	args := lifecycleJSON(t, rememberArguments())
	first := lifecycleReceipt(t, s.call(context.Background(), lifecyclePrincipal(), "memory_remember", args))
	if first.Delivery.Result != nil && first.Delivery.Result.Delivered {
		t.Fatal("unavailable remote claimed delivery")
	}
	records, err := s.memory.History(first.Saved.RecordID)
	if err != nil || len(records) != 1 || records[0].ID != first.Saved.ID {
		t.Fatalf("failed delivery lost local save: %v %v", records, err)
	}
	restarted, err := New(s.config)
	if err != nil {
		t.Fatal(err)
	}
	second := lifecycleReceipt(t, restarted.call(context.Background(), lifecyclePrincipal(), "memory_remember", args))
	if second.Saved.ID != first.Saved.ID {
		t.Fatal("restart retry published a new revision")
	}
	if err := os.Rename(unavailable, remote); err != nil {
		t.Fatal(err)
	}
	delivered := lifecycleReceipt(t, restarted.call(context.Background(), lifecyclePrincipal(), "memory_remember", args))
	if !delivered.Delivery.OK || delivered.Delivery.Result == nil || !delivered.Delivery.Result.Delivered || delivered.Saved.ID != first.Saved.ID {
		t.Fatalf("recovery delivery failed: %+v", delivered)
	}
	remoteHead := lifecycleGit(t, "--git-dir="+remote, "rev-parse", "main")
	if remoteHead != delivered.Delivery.Result.Head {
		t.Fatal("delivery receipt did not identify remote HEAD")
	}
	path := "memory/records/" + first.Saved.RecordID + "/" + first.Saved.ID + ".json"
	published := lifecycleGit(t, "--git-dir="+remote, "show", "main:"+path)
	var revision memory.Revision
	if err := json.Unmarshal([]byte(published), &revision); err != nil || revision.ID != first.Saved.ID {
		t.Fatalf("remote memory not published: %v", err)
	}
	history, err := restarted.memory.History(first.Saved.RecordID)
	if err != nil || len(history) != 1 {
		t.Fatalf("retries duplicated history: %v %v", history, err)
	}
}

func TestLifecycleConcurrentCorrectionsRetainConflictingHeads(t *testing.T) {
	s, _, _ := serviceFixture(t, true)
	first := lifecycleReceipt(t, s.call(context.Background(), lifecyclePrincipal(), "memory_remember", lifecycleJSON(t, rememberArguments())))
	results := make(chan api.Envelope, 2)
	for i := range 2 {
		args := rememberArguments()
		args["request_id"] = fmt.Sprintf("correction-request-%04d", i)
		record := args["record"].(map[string]any)
		record["record_id"], record["supersedes"], record["body"] = first.Saved.RecordID, []string{first.Saved.ID}, fmt.Sprintf("Independent correction %d", i)
		raw := lifecycleJSON(t, args)
		go func() { results <- s.call(context.Background(), lifecyclePrincipal(), "memory_remember", raw) }()
	}
	ids := map[string]bool{}
	for range 2 {
		r := lifecycleReceipt(t, <-results)
		ids[r.Saved.ID] = true
	}
	packet, err := s.memory.Recall("", nil, 10, 8192)
	if err != nil || len(packet.Current) != 0 || len(packet.Conflicts) != 1 || len(packet.Conflicts[0].HeadIDs) != 2 {
		t.Fatalf("concurrent claims not retained as conflict: %+v %v", packet, err)
	}
	for _, id := range packet.Conflicts[0].HeadIDs {
		if !ids[id] {
			t.Fatal("unexpected conflict head", id)
		}
	}
}

func TestLifecycleCancellationPreventsPublication(t *testing.T) {
	s, _, _ := serviceFixture(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := s.call(ctx, lifecyclePrincipal(), "memory_remember", lifecycleJSON(t, rememberArguments()))
	if out.OK || out.Error == nil || out.Error.Code != "operation.cancelled" {
		t.Fatalf("cancelled save accepted: %+v", out)
	}
	packet, err := s.memory.Recall("", nil, 10, 8192)
	if err != nil || packet.MatchingCount != 0 || packet.ConflictCount != 0 {
		t.Fatalf("cancelled save published memory: %+v %v", packet, err)
	}
	files, err := filepath.Glob(filepath.Join(s.memory.Root(), ".mandalore", "operation-*.json"))
	if err != nil || len(files) != 0 {
		t.Fatalf("cancelled save created intent: %v %v", files, err)
	}
}

func TestLifecycleCanonSessionsSubjectIsolationAndExpiry(t *testing.T) {
	s, _, _ := serviceFixture(t, true)
	p := lifecyclePrincipal()
	opened := s.call(context.Background(), p, "razor_session_open", []byte(`{}`))
	if !opened.OK {
		t.Fatal(opened)
	}
	var session struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(lifecycleJSON(t, opened.Result), &session); err != nil || session.SessionID == "" {
		t.Fatal("missing session", err)
	}
	raw := lifecycleJSON(t, map[string]string{"session_id": session.SessionID})
	if status := s.call(context.Background(), p, "foundling_status", raw); !status.OK {
		t.Fatal("owner denied own session", status)
	}
	foreign := p
	foreign.Subject = "another-subject"
	for _, name := range []string{"foundling_status", "foundling_refresh", "foundling_canon_recall"} {
		denied := s.call(context.Background(), foreign, name, raw)
		if denied.OK || denied.Error == nil || denied.Error.Code != "access.denied" {
			t.Fatalf("foreign subject accessed %s: %+v", name, denied)
		}
	}
	current := s.sessions[session.SessionID]
	current.expires = time.Now().Add(-time.Second)
	s.sessions[session.SessionID] = current
	denied := s.call(context.Background(), p, "foundling_status", raw)
	if denied.OK || denied.Error == nil || denied.Error.Code != "access.denied" {
		t.Fatal("expired session accepted", denied)
	}
}
