package sessionsync

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

func git(t *testing.T, args ...string) string {
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
func fixture(t *testing.T) (*Coordinator, *memory.Service, *signetsync.Synchronizer) {
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
	p := Policy{SchemaVersion: 1, Mode: EnabledMode, Binding: path, BindingSHA256: guard.SHA256, SignetID: b.SignetID, Runtime: runtime, RuntimeSHA256: hash([]byte("synthetic runtime")), StateDir: state}
	raw, _ = json.Marshal(p)
	policyPath := filepath.Join(state, "policy.json")
	if err := os.WriteFile(policyPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Open(policyPath, Selection{PolicySHA256: hash(raw), Binding: path, Guard: guard, Runtime: runtime})
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
	git(t, "init", "--bare", remote)
	git(t, "-C", root, "remote", "add", "origin", remote)
	return c, s, g
}
func save(t *testing.T, s *memory.Service, body string) {
	t.Helper()
	if _, err := s.Remember(memory.Write{Kind: "fact", Summary: body, Body: body, Basis: "user-direction", Reason: "Synthetic confirmed choice"}); err != nil {
		t.Fatal(err)
	}
}

func TestFirstAttemptRefreshAndStartupPair(t *testing.T) {
	c, s, g := fixture(t)
	save(t, s, "teal")
	var calls atomic.Int32
	run := func(ctx context.Context, d time.Duration) (signetsync.Status, error) {
		calls.Add(1)
		return g.Sync(ctx, d)
	}
	first := c.attempt(context.Background(), Boundary{Kind: "startup", SessionID: "native-one"}, run)
	if first.Error != nil || first.Status == nil || !first.Status.Delivered || !first.Attempted {
		t.Fatal("first attempt", first)
	}
	turn := Boundary{Kind: "turn", SessionID: "native-one", EventKey: "prompt-one"}
	paired := c.attempt(context.Background(), turn, run)
	if paired.Error != nil || !paired.Coalesced || calls.Load() != 1 {
		t.Fatal("startup pairing", paired, calls.Load())
	}
	again := c.attempt(context.Background(), turn, run)
	if !again.Coalesced || calls.Load() != 1 {
		t.Fatal("same event replay", again)
	}
	next := c.attempt(context.Background(), Boundary{Kind: "turn", SessionID: "native-one", EventKey: "prompt-two"}, run)
	if next.Error != nil || next.Coalesced || calls.Load() != 2 {
		t.Fatal("later turn skipped", next, calls.Load())
	}
}

func TestExpiredStartupAndPendingEventAreRetried(t *testing.T) {
	c, _, g := fixture(t)
	var calls atomic.Int32
	run := func(ctx context.Context, d time.Duration) (signetsync.Status, error) {
		calls.Add(1)
		return g.Sync(ctx, d)
	}
	first := c.attempt(context.Background(), Boundary{Kind: "startup", SessionID: "native"}, run)
	if first.Error != nil {
		t.Fatal(first)
	}
	path := filepath.Join(c.root, ".mandalore", "session-sync.json")
	raw, _ := os.ReadFile(path)
	var last receipt
	_ = json.Unmarshal(raw, &last)
	last.CompletedAt = time.Now().Add(-StartupPairWindow - time.Second)
	if err := writeReceipt(path, last); err != nil {
		t.Fatal(err)
	}
	if next := c.attempt(context.Background(), Boundary{Kind: "turn", SessionID: "native", EventKey: "one"}, run); next.Coalesced || calls.Load() != 2 {
		t.Fatal("expired startup suppressed turn", next)
	}
	git(t, "-C", c.root, "remote", "set-url", "origin", filepath.Join(filepath.Dir(c.root), "missing-remote"))
	turn := Boundary{Kind: "turn", SessionID: "native", EventKey: "offline"}
	pending := c.attempt(context.Background(), turn, run)
	if pending.Status == nil || pending.Status.Delivered {
		t.Fatal(pending)
	}
	count := calls.Load()
	next := c.attempt(context.Background(), turn, run)
	if next.Coalesced || calls.Load() != count+1 {
		t.Fatal("pending event suppressed retry", next)
	}
}

func TestOverlappingAttemptsCoalesceAcrossPoliciesAndLaterSaveDoesNot(t *testing.T) {
	c, s, g := fixture(t)
	save(t, s, "first")
	otherPolicy := c.policy
	otherPolicy.StateDir = filepath.Join(filepath.Dir(c.policy.StateDir), "other-state")
	if err := os.Mkdir(otherPolicy.StateDir, 0700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(otherPolicy)
	otherPath := filepath.Join(otherPolicy.StateDir, "policy.json")
	if err := os.WriteFile(otherPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	selection := c.selected
	selection.PolicySHA256 = hash(raw)
	other, err := Open(otherPath, selection)
	if err != nil {
		t.Fatal(err)
	}
	if c.selected.PolicySHA256 == other.selected.PolicySHA256 {
		t.Fatal("fixture policies must differ")
	}
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	run := func(ctx context.Context, d time.Duration) (signetsync.Status, error) {
		if calls.Add(1) == 1 {
			close(started)
			<-release
		}
		return g.Sync(ctx, d)
	}
	var results [5]Attempt
	var wg sync.WaitGroup
	wg.Add(5)
	go func() { defer wg.Done(); results[0] = c.attempt(context.Background(), Boundary{Kind: "turn"}, run) }()
	<-started
	for i := 1; i < 5; i++ {
		go func(i int) {
			defer wg.Done()
			results[i] = other.attempt(context.Background(), Boundary{Kind: "turn"}, run)
		}(i)
	}
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatal("overlapping attempts stormed", calls.Load(), results)
	}
	for _, r := range results {
		if r.Error != nil || r.Status == nil || !r.Status.Delivered {
			t.Fatal(r)
		}
	}
	save(t, s, "second")
	next := c.attempt(context.Background(), Boundary{Kind: "write"}, run)
	if next.Coalesced || calls.Load() != 2 || next.Status == nil || !next.Status.Delivered {
		t.Fatal("new save reused previous delivery", next)
	}
}

func TestPolicyRetargetAndCancellationNeverAuthorizeSync(t *testing.T) {
	for _, kind := range []string{"binding", "runtime", "policy", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			c, _, _ := fixture(t)
			var calls int
			run := func(context.Context, time.Duration) (signetsync.Status, error) {
				calls++
				return signetsync.Status{}, nil
			}
			ctx := context.Background()
			switch kind {
			case "binding":
				raw, _ := os.ReadFile(c.policy.Binding)
				_ = os.WriteFile(c.policy.Binding, append(raw, ' '), 0600)
			case "runtime":
				_ = os.WriteFile(c.policy.Runtime, []byte("changed"), 0700)
			case "policy":
				raw, _ := os.ReadFile(c.policyPath)
				_ = os.WriteFile(c.policyPath, append(raw, ' '), 0600)
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			result := c.attempt(ctx, Boundary{Kind: "turn"}, run)
			if result.Error == nil || calls != 0 || result.Attempted {
				t.Fatal("invalid selection authorized sync", result, calls)
			}
		})
	}
}

func TestSeparateCloneDoesNotReuseSameSignetReceipt(t *testing.T) {
	c, _, g := fixture(t)
	first := c.Attempt(context.Background(), Boundary{Kind: "turn", SessionID: "same", EventKey: "same"})
	if first.Error != nil {
		t.Fatal(first)
	}
	clone := filepath.Join(filepath.Dir(c.root), "clone")
	git(t, "clone", "--branch", "main", filepath.Join(filepath.Dir(c.root), "remote.git"), clone)
	if _, err := os.Stat(filepath.Join(clone, ".mandalore", "session-sync.json")); !os.IsNotExist(err) {
		t.Fatal("coordination receipt synchronized")
	}
	inspection, err := g.Inspect(context.Background())
	if err != nil || inspection.Dirty {
		t.Fatal(inspection, err)
	}
	bindingPath := filepath.Join(filepath.Dir(c.root), "clone-binding.json")
	b, err := binding.Bind(clone, bindingPath, "Clone", "Clone actor")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(bindingPath)
	p := c.policy
	p.Binding = bindingPath
	p.BindingSHA256 = hash(raw)
	raw, _ = json.Marshal(p)
	policyPath := filepath.Join(p.StateDir, "clone-policy.json")
	if err := os.WriteFile(policyPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	other, err := Open(policyPath, Selection{PolicySHA256: hash(raw), Binding: bindingPath, Guard: binding.Guard{SHA256: p.BindingSHA256, SignetID: b.SignetID}, Runtime: p.Runtime})
	if err != nil {
		t.Fatal(err)
	}
	if other.identity == c.identity {
		t.Fatal("different clone reused physical coordination identity")
	}
	result := other.Attempt(context.Background(), Boundary{Kind: "turn", SessionID: "same", EventKey: "same"})
	if result.Coalesced || !result.Attempted || result.Error != nil {
		t.Fatal("different clone reused freshness", result)
	}
}

func TestWaitingWriteCannotInheritEarlierCheckpoint(t *testing.T) {
	c, s, g := fixture(t)
	save(t, s, "before checkpoint")
	var calls atomic.Int32
	checkpointed := make(chan struct{})
	release := make(chan struct{})
	run := func(ctx context.Context, d time.Duration) (signetsync.Status, error) {
		result, err := g.Sync(ctx, d)
		if calls.Add(1) == 1 {
			close(checkpointed)
			<-release
		}
		return result, err
	}
	leader := make(chan Attempt, 1)
	go func() { leader <- c.attempt(context.Background(), Boundary{Kind: "turn"}, run) }()
	<-checkpointed
	save(t, s, "after checkpoint")
	waiter := make(chan Attempt, 1)
	go func() { waiter <- c.attempt(context.Background(), Boundary{Kind: "write"}, run) }()
	time.Sleep(50 * time.Millisecond)
	close(release)
	first, second := <-leader, <-waiter
	if first.Error != nil || second.Error != nil || second.Coalesced || !second.Attempted || second.Status == nil || !second.Status.Delivered || calls.Load() != 2 {
		t.Fatal(first, second, calls.Load())
	}
	if first.Status.Head == second.Status.Head {
		t.Fatal("later published content was absent from second delivery")
	}
}
