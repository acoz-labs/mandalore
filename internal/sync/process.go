// Package signetsync owns Git mechanics; the memory engine owns record semantics.
package signetsync

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const commandOutputLimit = 1 << 20

var ErrBoundary = errors.New("synchronization requires the intended standalone signet on main with no in-progress Git operation")
var ErrDirty = errors.New("unsafe or partially staged work requires explicit reconciliation; files and index preserved")
var ErrHistory = errors.New("durable evidence or signet identity was rewritten; append superseding records instead")

type commandError struct {
	operation string
	exitCode  int
	cause     error
}

func (e *commandError) Error() string {
	return "Git " + e.operation + " failed; inspect local synchronization state"
}
func (e *commandError) Unwrap() error { return e.cause }

// Drop repository/config/identity redirection from the caller, but retain native
// credential-helper environment and SSH agent access. We never persist secrets.
func cleanEnvironment() []string {
	env := []string{}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "GIT_") || key == "SSH_ASKPASS" {
			continue
		}
		env = append(env, entry)
	}
	return append(env, "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "GIT_ATTR_NOSYSTEM=1", "GIT_NO_LAZY_FETCH=1", "GIT_NO_REPLACE_OBJECTS=1", "GIT_ALLOW_PROTOCOL=file:ssh:https", "GIT_SSH_COMMAND=ssh -o BatchMode=yes -o StrictHostKeyChecking=yes")
}

type limitedOutput struct {
	buffer bytes.Buffer
	limit  int
	cancel context.CancelFunc
}

func (b *limitedOutput) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.buffer.Len() {
		b.cancel()
		return 0, errors.New("Git output limit exceeded")
	}
	return b.buffer.Write(p)
}

func (b *limitedOutput) String() string { return b.buffer.String() }

func (s *Synchronizer) git(parent context.Context, args ...string) (string, error) {
	out, err := s.gitOutput(parent, commandOutputLimit, args...)
	return strings.TrimSpace(out), err
}

func (s *Synchronizer) gitOutput(parent context.Context, limit int, args ...string) (string, error) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	base := []string{"-C", s.store.Root, "--git-dir=" + s.store.Root + "/.git", "--work-tree=" + s.store.Root,
		"-c", "core.bare=false", "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false",
		"-c", "core.attributesFile=/dev/null", "-c", "core.askPass=", "-c", "credential.interactive=false",
		"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false", "-c", "protocol.ext.allow=never",
		"-c", "user.name=Mandalore", "-c", "user.email=mandalore@localhost"}
	if args[0] == "commit-tree" {
		name, nameErr := s.git(ctx, "config", "--local", "--get", "user.name")
		email, emailErr := s.git(ctx, "config", "--local", "--get", "user.email")
		if nameErr != nil && !isExit(nameErr, 1) {
			return "", nameErr
		}
		if emailErr != nil && !isExit(emailErr, 1) {
			return "", emailErr
		}
		if (nameErr == nil) != (emailErr == nil) {
			return "", ErrBoundary
		}
		if nameErr == nil {
			if name == "" || email == "" || len(name) > 256 || len(email) > 256 || strings.ContainsAny(name+email, "\x00\r\n") {
				return "", ErrBoundary
			}
			base = append(base, "-c", "user.name="+name, "-c", "user.email="+email)
		}
	}
	cmd := exec.CommandContext(ctx, "git", append(base, args...)...)
	cmd.Env = cleanEnvironment()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = time.Second
	stdout := &limitedOutput{limit: limit, cancel: cancel}
	stderr := &limitedOutput{limit: 65536, cancel: cancel}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	if err != nil {
		code := -1
		if cmd.ProcessState != nil {
			code = cmd.ProcessState.ExitCode()
		}
		cause := err
		if parent.Err() != nil {
			cause = parent.Err()
		}
		return "", &commandError{operation: args[0], exitCode: code, cause: cause}
	}
	return stdout.String(), nil
}

func isExit(err error, code int) bool {
	var e *commandError
	return errors.As(err, &e) && e.exitCode == code
}
