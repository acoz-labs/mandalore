package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Closing a terminal descriptor need not release an already blocked read.
// Keep that read blocked until the test explicitly releases it, independently
// of the context, so a passing test cannot borrow EOF as cancellation evidence.
type heldMenuInput struct {
	started  chan struct{}
	release  chan struct{}
	finished chan struct{}
}

func (r *heldMenuInput) Read(p []byte) (int, error) {
	close(r.started)
	<-r.release
	defer close(r.finished)
	return copy(p, "\n"), nil
}

func TestPlainMenuBlockedInputCancellation(t *testing.T) {
	for _, operation := range []string{"choice", "text", "confirmation"} {
		for _, partial := range []string{"", "2"} {
			t.Run(operation+"/partial="+partial, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				held := &heldMenuInput{make(chan struct{}), make(chan struct{}), make(chan struct{})}
				var out bytes.Buffer
				m := &menu{ctx: ctx, in: bufio.NewReader(io.MultiReader(strings.NewReader(partial), held)), out: &out}
				result := make(chan error, 1)
				go func() {
					var err error
					switch operation {
					case "choice":
						_, err = m.selectItem("Choose", []string{"No", "Yes"}, 0)
					case "text":
						_, err = m.input("Path", "unchanged")
					case "confirmation":
						err = m.confirm()
					}
					result <- err
				}()
				returned := false
				defer func() {
					close(held.release)
					<-held.finished
					if !returned {
						<-result
					}
				}()
				<-held.started
				cancel()
				select {
				case err := <-result:
					returned = true
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("cancelled input became an answer: %v", err)
					}
				case <-time.After(time.Second):
					t.Error("cancelled prompt waited for another input byte")
				}
			})
		}
	}
}

func TestPlainMenuCancellationAtPendingApplyDoesNotCreate(t *testing.T) {
	dir := t.TempDir()
	root, binding := filepath.Join(dir, "signet"), filepath.Join(dir, "binding.json")
	script := strings.Join([]string{"1", root, "Synthetic", "Test host", "Test actor", binding}, "\n") + "\n2"
	held := &heldMenuInput{make(chan struct{}), make(chan struct{}), make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out bytes.Buffer
	result := make(chan int, 1)
	go func() {
		result <- run(ctx, []string{"menu", "--plain", "--binding", binding}, io.MultiReader(strings.NewReader(script), held), &out, &out)
	}()
	returned := false
	defer func() {
		close(held.release)
		<-held.finished
		if !returned {
			<-result
		}
	}()
	<-held.started
	cancel()
	select {
	case code := <-result:
		returned = true
		if code != 130 || !strings.Contains(out.String(), "Apply these changes?") {
			t.Fatalf("unexpected cancellation result: %d %s", code, out.String())
		}
	case <-time.After(time.Second):
		t.Error("cancelled pending apply waited for a newline")
	}
	for _, path := range []string{root, binding} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatal("cancellation created state", path, err)
		}
	}
}

type cancellingMenuInput struct{ cancel context.CancelFunc }

func (r cancellingMenuInput) Read(p []byte) (int, error) {
	r.cancel()
	return copy(p, "2\n"), nil
}

func TestPlainMenuReadyApprovalCannotWinCancellation(t *testing.T) {
	for range 100 {
		ctx, cancel := context.WithCancel(context.Background())
		m := &menu{ctx: ctx, in: bufio.NewReader(cancellingMenuInput{cancel}), out: io.Discard}
		err := m.confirm()
		cancel()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ready approval escaped cancellation: %v", err)
		}
	}
}
