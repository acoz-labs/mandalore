// promote-candidate is guarded repository-maintainer tooling, not a memory tool.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/acoz-labs/mandalore/internal/distribution"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	"golang.org/x/sys/unix"
)

type inspector interface {
	InspectCandidate(context.Context, distribution.CandidateSelection) (distribution.CandidateTransport, error)
}
type publisher interface {
	Publish(context.Context, string, string) (distribution.PublicationReceipt, error)
}
type services struct {
	inspector inspector
	publisher publisher
	authorize func(context.Context, distribution.ParsedManifest) error
}

type promotion struct {
	Phase            string                           `json:"phase"`
	Transport        *distribution.CandidateTransport `json:"transport,omitempty"`
	StagingDirectory string                           `json:"staging_directory,omitempty"`
	Publication      *distribution.PublicationReceipt `json:"publication,omitempty"`
}

func regular(path string, limit int64) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, errors.New("promotion input is missing, redirected or unreadable")
	}
	f := os.NewFile(uintptr(fd), path)
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() < 1 || st.Size() > limit {
		f.Close()
		return nil, errors.New("promotion input type or size is invalid")
	}
	return f, nil
}

func run(ctx context.Context, args []string, s services) (result promotion, err error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Minute)
	defer cancel()
	f := flag.NewFlagSet("promote-candidate", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	receipt := f.String("receipt", "", "Inspected candidate transport JSON")
	archive := f.String("archive", "", "Retained raw Actions ZIP")
	identity := f.String("identity", "", "Exact independently accepted candidate identity")
	parent := f.String("staging-parent", "", "Existing scratch directory for fresh private staging")
	if err := f.Parse(args); err != nil || f.NArg() != 0 || *receipt == "" || *archive == "" || *identity == "" || *parent == "" {
		return result, errors.New("promotion requires receipt, archive, identity and staging-parent")
	}
	file, err := regular(*receipt, 16384)
	if err != nil {
		return result, err
	}
	raw, err := io.ReadAll(io.LimitReader(file, 16385))
	file.Close()
	if err != nil {
		return result, errors.New("cannot read promotion transport receipt")
	}
	var transport distribution.CandidateTransport
	if err := strictjson.Decode(raw, &transport, 16384); err != nil {
		return result, errors.New("invalid promotion transport receipt")
	}
	refresh := func() error {
		fresh, err := s.inspector.InspectCandidate(ctx, transport.CandidateSelection)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(fresh, transport) {
			return errors.New("retained candidate transport changed; re-inspect rather than replacing accepted bytes")
		}
		return nil
	}
	if err := refresh(); err != nil {
		return result, err
	}
	result.Phase = "transport-verified"
	result.Transport = &transport
	input, err := regular(*archive, distribution.MaxCandidateArchiveBytes)
	if err != nil {
		return result, err
	}
	defer input.Close()
	st, err := input.Stat()
	if err != nil {
		return result, errors.New("cannot inspect retained archive")
	}
	directory, m, err := distribution.StageCandidateArchive(ctx, input, st.Size(), transport, *identity, *parent)
	result.StagingDirectory = directory
	if err != nil {
		return result, err
	}
	result.Phase = "candidate-staged"
	if err := refresh(); err != nil {
		return result, err
	}
	// These guards are mandatory in this command, not an optional workflow hint.
	if err := s.authorize(ctx, m); err != nil {
		return result, err
	}
	result.Phase = "authorized"
	if err := refresh(); err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	published, err := s.publisher.Publish(ctx, directory, *identity)
	result.Publication = &published
	if err != nil {
		return result, err
	}
	if published.Phase != "publication-verified" || published.Identity != *identity || published.SourceCommit != transport.SourceCommit || published.URL != "https://github.com/acoz-labs/mandalore/releases/tag/"+url.PathEscape(m.Manifest.Tag) || published.PendingOperation != "" || published.ReleaseID <= 0 || len(published.Assets) != 8 {
		return result, errors.New("publisher did not verify the selected complete release")
	}
	result.Phase = "publication-verified"
	return result, nil
}

// authorize binds the verified candidate into both existing release guards.
// No caller flag skips either check. Child output is bounded and not echoed;
// operators can run the named read-only guard directly for its diagnostic output.
func authorize(ctx context.Context, parsed distribution.ParsedManifest) error {
	if os.Getenv("GITHUB_REPOSITORY") != "acoz-labs/mandalore" || os.Getenv("GH_TOKEN") == "" {
		return errors.New("promotion requires the official repository context and an explicit GH_TOKEN")
	}
	m := parsed.Manifest
	env := []string{}
	replace := map[string]string{"RELEASE_SHA": m.SourceCommit, "RELEASE_VERSION": m.Version, "RELEASE_ARTIFACT": parsed.Identity(), "STAGING_REQUIRED": "false", "GH_DEBUG": "", "GH_HOST": "github.com", "GITHUB_SERVER_URL": "https://github.com"}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, ok := replace[key]; !ok && key != "RELEASE_POLICY_READ_TOKEN" {
			env = append(env, entry)
		}
	}
	for key, value := range replace {
		env = append(env, key+"="+value)
	}
	for _, args := range [][]string{{"bin/release-gate", "--require-acceptance", m.SourceCommit}, {"bin/finalize-release", "artifact", "--preflight"}} {
		command := exec.CommandContext(ctx, args[0], args[1:]...)
		command.Env = env
		command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		command.Cancel = func() error {
			if command.Process == nil {
				return os.ErrProcessDone
			}
			err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
			if err == syscall.ESRCH {
				return os.ErrProcessDone
			}
			return err
		}
		command.WaitDelay = 2 * time.Second
		// No pipe buffers or raw provider output are retained in the promotion receipt.
		command.Stdout = io.Discard
		command.Stderr = io.Discard
		if err := command.Run(); err != nil {
			return fmt.Errorf("publication authority refused or could not be verified by %s; run that read-only guard directly to diagnose", args[0])
		}
	}
	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Promotion requires an explicitly supplied GH_TOKEN.")
		os.Exit(1)
	}
	s := services{inspector: distribution.NewCandidateClient(token), publisher: distribution.NewPublisher(token, os.Getenv("RELEASE_POLICY_READ_TOKEN")), authorize: authorize}
	result, err := run(ctx, os.Args[1:], s)
	if result.Phase != "" {
		if encodeErr := json.NewEncoder(os.Stdout).Encode(result); encodeErr != nil {
			fmt.Fprintln(os.Stderr, "Cannot record promotion result; remote effects may exist.")
			os.Exit(1)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Promotion failed:", err)
		os.Exit(1)
	}
}
