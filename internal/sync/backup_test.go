package signetsync

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPushOnlyDeliversWithoutFetchingAndRefusesRemoteAhead(t *testing.T) {
	a, first, b, second, remote := remoteFixture(t)
	ctx := context.Background()
	remember(t, first, "Copper Finch")
	if _, err := a.Checkpoint(ctx); err != nil {
		t.Fatal(err)
	}
	head, err := a.Push(ctx, remote)
	if err != nil || head == "" || gitTest(t, a.store.Root, "ls-remote", remote, "refs/heads/main")[:40] != head {
		t.Fatalf("push %q: %v", head, err)
	}
	if ahead, err := a.RemoteAhead(ctx, remote); err != nil || ahead {
		t.Fatalf("own head reported ahead: %v %v", ahead, err)
	}
	// Another writer pushes; the first checkout must not overwrite it.
	remember(t, second, "Second writer")
	if out, err := b.Sync(ctx, 10*time.Second); err != nil || !out.Delivered {
		t.Fatal(out, err)
	}
	if ahead, err := a.RemoteAhead(ctx, remote); err != nil || !ahead {
		t.Fatalf("remote ahead not detected: %v %v", ahead, err)
	}
	remember(t, first, "Local only")
	if _, err := a.Checkpoint(ctx); err != nil {
		t.Fatal(err)
	}
	// The remote commit is not local yet: the last pushed head tells.
	if state, err := a.RemoteState(ctx, remote, head); err != nil || state != RemoteDiverged {
		t.Fatalf("divergence not detected: %q %v", state, err)
	}
	if _, err := a.Push(ctx, remote); !errors.Is(err, ErrRemoteAhead) {
		t.Fatalf("push over remote-ahead: %v", err)
	}
	before := gitTest(t, a.store.Root, "rev-parse", "HEAD")
	if _, err := a.FastForward(ctx, remote); !errors.Is(err, ErrDiverged) {
		t.Fatalf("divergent fast-forward: %v", err)
	}
	if gitTest(t, a.store.Root, "rev-parse", "HEAD") != before {
		t.Fatal("refused reconcile moved main")
	}
	// Fetched, the remote commit is local and ancestry decides.
	if state, err := a.RemoteState(ctx, remote, ""); err != nil || state != RemoteDiverged {
		t.Fatalf("fetched divergence: %q %v", state, err)
	}
}

func TestFastForwardReconcilesARemoteAheadCheckout(t *testing.T) {
	a, _, b, second, remote := remoteFixture(t)
	ctx := context.Background()
	remember(t, second, "Second writer")
	if out, err := b.Sync(ctx, 10*time.Second); err != nil || !out.Delivered {
		t.Fatal(out, err)
	}
	if state, err := a.RemoteState(ctx, remote, ""); err != nil || state != RemoteAheadOf {
		t.Fatalf("remote ahead not detected: %q %v", state, err)
	}
	out, err := a.FastForward(ctx, remote)
	if err != nil || out.State != "synchronized" || out.Head != out.RemoteHead {
		t.Fatalf("fast-forward %+v: %v", out, err)
	}
	if ahead, err := a.RemoteAhead(ctx, remote); err != nil || ahead {
		t.Fatalf("still ahead after reconcile: %v %v", ahead, err)
	}
	if err := a.store.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Push(ctx, "relative/path"); err == nil {
		t.Fatal("ambiguous remote accepted")
	}
}
