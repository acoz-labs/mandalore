package sessionsync

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The helper is a real disposable process: killing it must release both locks
// even while its separately grouped Git child remains alive.
func TestSessionLockInheritanceHelper(t *testing.T) {
	raw := os.Getenv("MANDALORE_LOCK_HELPER")
	if raw == "" {
		return
	}
	var input struct {
		Path      string
		Selection Selection
	}
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		t.Fatal(err)
	}
	c, err := Open(input.Path, input.Selection)
	if err != nil {
		t.Fatal(err)
	}
	attempt := c.Attempt(context.Background(), Boundary{Kind: "turn", SessionID: "killed-parent", EventKey: "one"})
	t.Fatalf("parent must be killed during controlled transport wait: %+v", attempt)
}

func TestKilledSessionReleasesBothLocksWhileGitChildLives(t *testing.T) {
	c, s, _ := fixture(t)
	save(t, s, "Durable before forced parent termination")
	dir := filepath.Dir(c.policyPath)
	wrapper := filepath.Join(dir, "wrapper")
	if err := os.Mkdir(wrapper, 0700); err != nil {
		t.Fatal(err)
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "git-child.pid")
	quote := func(v string) string { return "'" + strings.ReplaceAll(v, "'", "'\\''") + "'" }
	script := "#!/bin/sh\nfor arg in \"$@\"; do\n if [ \"$arg\" = ls-remote ]; then\n  printf '%s\\n' \"$$\" > " + quote(marker) + "\n  exec /bin/sleep 60\n fi\ndone\nexec " + quote(realGit) + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(wrapper, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(struct {
		Path      string
		Selection Selection
	}{c.policyPath, c.selected})
	log, err := os.Create(filepath.Join(dir, "helper.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	cmd := exec.Command(os.Args[0], "-test.run=^TestSessionLockInheritanceHelper$")
	cmd.Stdout, cmd.Stderr = log, log
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "PATH=") && !strings.HasPrefix(value, "MANDALORE_LOCK_HELPER=") {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env, "PATH="+wrapper+string(os.PathListSeparator)+os.Getenv("PATH"), "MANDALORE_LOCK_HELPER="+string(input))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	childPID := 0
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(marker); err == nil {
			childPID, _ = strconv.Atoi(strings.TrimSpace(string(raw)))
			if childPID > 1 {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	if childPID <= 1 {
		t.Fatal("helper never reached the controlled post-checkpoint Git wait")
	}
	// Only the known disposable child is cleaned up, and only after assertions.
	defer syscall.Kill(-childPID, syscall.SIGKILL)
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	waited = true
	if err := syscall.Kill(childPID, 0); err != nil {
		t.Fatal("Git child did not survive parent termination", err)
	}
	for _, name := range []string{"session-sync.lock", "write.lock"} {
		f, err := os.OpenFile(filepath.Join(s.Root(), ".mandalore", name), os.O_RDWR, 0600)
		if err != nil {
			t.Fatal(err)
		}
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		}
		_ = f.Close()
		if err != nil {
			t.Errorf("orphan Git child retained %s after parent SIGKILL: %v", name, err)
		}
	}
	if t.Failed() {
		return
	}
	recovered := c.Attempt(context.Background(), Boundary{Kind: "turn", SessionID: "next-foreground", EventKey: "two"})
	if recovered.Error != nil || recovered.Status == nil || !recovered.Status.Delivered {
		t.Fatalf("next foreground did not deliver saved content: %+v", recovered)
	}
	if err := syscall.Kill(childPID, 0); err != nil {
		t.Fatal("recovery passed only after controlled child exited", err)
	}
	packet, err := s.Recall("Durable before forced parent termination", nil, 3, 4096)
	if err != nil || packet.MatchingCount != 1 {
		t.Fatal("saved content lost or duplicated", packet, err)
	}
}
