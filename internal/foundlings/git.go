package foundlings

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

const maxGitOutput = 8 << 20
const gitTimeout = 10 * time.Second

// These commands inspect local objects only. In particular they never invoke
// transport, checkout, status, attributes, filters or user-supplied programs.
func gitEnvironment() []string {
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.HasPrefix(key, "GIT_") && key != "SSH_ASKPASS" {
			env = append(env, entry)
		}
	}
	return append(env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "GIT_ATTR_NOSYSTEM=1", "GIT_NO_LAZY_FETCH=1", "GIT_NO_REPLACE_OBJECTS=1", "GIT_ALLOW_PROTOCOL=", "GIT_CONFIG_SYSTEM=/dev/null")
}

type cappedOutput struct {
	// Do not embed bytes.Buffer: its promoted ReadFrom would let io.Copy bypass
	// Write and therefore bypass the size limit.
	buffer   bytes.Buffer
	limit    int
	cancel   context.CancelFunc
	overflow bool
}

func (b *cappedOutput) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.buffer.Len() {
		b.overflow = true
		b.cancel()
		return 0, errors.New("reference Git output exceeds bounds")
	}
	return b.buffer.Write(p)
}

func sourceGit(parent context.Context, root, input string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, gitTimeout)
	defer cancel()
	base := []string{"-C", root, "--git-dir=" + filepath.Join(root, ".git"), "--work-tree=" + root,
		"-c", "core.bare=false", "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false",
		"-c", "core.attributesFile=/dev/null", "-c", "core.askPass=", "-c", "credential.interactive=false",
		"-c", "protocol.allow=never", "-c", "protocol.ext.allow=never"}
	cmd := exec.CommandContext(ctx, "git", append(base, args...)...)
	cmd.Env = gitEnvironment()
	cmd.Stdin = strings.NewReader(input)
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
	out := &cappedOutput{limit: maxGitOutput, cancel: cancel}
	stderr := &cappedOutput{limit: 65536, cancel: cancel}
	cmd.Stdout, cmd.Stderr = out, stderr
	err := cmd.Run()
	if out.overflow || stderr.overflow {
		return "", errors.New("reference Git output exceeds bounds")
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		return "", errors.New("local reference Git inspection failed; check source identity and local objects")
	}
	return out.buffer.String(), nil
}

func gitIdentity(ctx context.Context, s *snapshot) (memory.SourcePin, error) {
	root := s.View.Root
	st, err := os.Lstat(filepath.Join(root, ".git"))
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return memory.SourcePin{}, ErrUnavailable
	}
	origin, err := sourceGit(ctx, root, "", "config", "--local", "--no-includes", "--null", "--get-all", "remote.origin.url")
	if err != nil {
		return memory.SourcePin{}, err
	}
	if origin != s.View.Source.Locator+"\x00" {
		return memory.SourcePin{}, ErrChanged
	}
	format, err := sourceGit(ctx, root, "", "rev-parse", "--show-object-format")
	if err != nil {
		return memory.SourcePin{}, err
	}
	head, err := sourceGit(ctx, root, "", "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return memory.SourcePin{}, err
	}
	pin := memory.SourcePin{Algorithm: "git-" + strings.TrimSpace(format), Value: strings.TrimSpace(head)}
	if err := memory.ValidateFoundlingIdentity(s.View.Source, pin); err != nil {
		return memory.SourcePin{}, err
	}
	return pin, nil
}

type treeFile struct {
	name, object string
	size         int
}

func observeGit(ctx context.Context, s *snapshot) (*snapshot, error) {
	before, err := os.Lstat(s.View.Root)
	if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return nil, ErrUnavailable
	}
	pin, err := gitIdentity(ctx, s)
	if err != nil {
		return nil, err
	}
	tree, err := sourceGit(ctx, s.View.Root, "", "ls-tree", "-rz", "--full-tree", pin.Value)
	if err != nil {
		return nil, err
	}
	entries := strings.Split(tree, "\x00")
	if entries[len(entries)-1] != "" || len(entries)-1 > maxEntries {
		return nil, errors.New("invalid or oversized reference tree")
	}
	files := []treeFile{}
	var objects strings.Builder
	for _, entry := range entries[:len(entries)-1] {
		meta, name, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 || !relative(name) {
			return nil, errors.New("invalid reference tree entry")
		}
		if !eligible(name) {
			s.View.Excluded++
			continue
		}
		if fields[1] != "blob" || (fields[0] != "100644" && fields[0] != "100755") {
			return nil, errors.New("reference text must be a regular tracked file")
		}
		if len(files) >= MaxFiles {
			return nil, errors.New("reference selection exceeds 10000 files")
		}
		if len(fields[2]) != len(pin.Value) || strings.Trim(fields[2], "0123456789abcdef") != "" {
			return nil, errors.New("invalid Git blob identity")
		}
		files = append(files, treeFile{name: name, object: fields[2]})
		objects.WriteString(fields[2] + "\n")
	}
	// Batch object inspection proves the selected blobs are locally available.
	// Without it, hashing worktree bytes could accidentally conceal missing objects.
	if len(files) > 0 {
		checked, err := sourceGit(ctx, s.View.Root, objects.String(), "cat-file", "--batch-check")
		if err != nil {
			return nil, err
		}
		lines := strings.Split(checked, "\n")
		if len(lines) != len(files)+1 || lines[len(lines)-1] != "" {
			return nil, ErrUnavailable
		}
		total := 0
		for i, line := range lines[:len(files)] {
			fields := strings.Fields(line)
			if len(fields) != 3 || fields[0] != files[i].object || fields[1] != "blob" {
				return nil, ErrUnavailable
			}
			size, err := strconv.Atoi(fields[2])
			if err != nil || size < 0 || size > MaxFileBytes || size > MaxSourceBytes-total {
				return nil, errors.New("reference object size exceeds bounds")
			}
			files[i].size = size
			total += size
		}
	}
	root, err := os.OpenRoot(s.View.Root)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer root.Close()
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		data, err := readRegular(root, file.name)
		if err != nil {
			return nil, err
		}
		if len(data) != file.size || blobHash(pin.Algorithm, data) != file.object {
			return nil, ErrChanged
		}
		if err := s.add(file.name, data); err != nil {
			return nil, err
		}
	}
	afterPin, err := gitIdentity(ctx, s)
	if err != nil {
		return nil, err
	}
	after, err := os.Lstat(s.View.Root)
	if err != nil || !os.SameFile(before, after) || pin != afterPin {
		return nil, ErrChanged
	}
	s.View.Pin = pin
	s.View.ContentSHA256 = documentDigest(s.Documents)
	s.View.Notice += " Git evidence covers eligible tracked text matching local HEAD only; untracked files are not enumerated or counted and full checkout cleanliness is not claimed."
	return s, nil
}

func blobHash(algorithm string, data []byte) string {
	header := fmt.Sprintf("blob %d\x00", len(data))
	if algorithm == "git-sha1" {
		h := sha1.New() // Git object identity, not a new cryptographic trust scheme.
		_, _ = h.Write([]byte(header))
		_, _ = h.Write(data)
		return hex.EncodeToString(h.Sum(nil))
	}
	h := sha256.New()
	_, _ = h.Write([]byte(header))
	_, _ = h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
