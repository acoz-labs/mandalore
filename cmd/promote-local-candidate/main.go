// promote-local-candidate publishes retained local bytes after portable acceptance.
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
	"strings"
	"syscall"
	"time"

	"github.com/acoz-labs/mandalore/internal/distribution"
)

type publisher interface {
	Publish(context.Context, string, string) (distribution.PublicationReceipt, error)
}

type services struct {
	verify    func(string) (distribution.ParsedManifest, error)
	authorize func(context.Context, distribution.ParsedManifest, int, string) error
	publisher publisher
}

func run(ctx context.Context, args []string, s services) (distribution.PublicationReceipt, error) {
	var result distribution.PublicationReceipt
	f := flag.NewFlagSet("promote-local-candidate", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	directory := f.String("directory", "", "Retained candidate directory")
	identity := f.String("identity", "", "Exact accepted product identity")
	issue := f.Int("issue", 0, "Accepted delivery issue")
	actor := f.String("expected-actor", "", "Authenticated independent maintainer")
	verifyOnly := f.Bool("verify-only", false, "Verify retained bytes without credentials or publication")
	if err := f.Parse(args); err != nil || f.NArg() != 0 || *directory == "" || *identity == "" || (!*verifyOnly && (*issue < 1 || *actor == "")) {
		return result, errors.New("directory, identity, issue and expected-actor are required")
	}
	parsed, err := s.verify(*directory)
	if err != nil {
		return result, err
	}
	if parsed.Identity() != *identity {
		return result, errors.New("local candidate differs from accepted product identity")
	}
	if *verifyOnly {
		return distribution.PublicationReceipt{Phase: "candidate-verified", Identity: parsed.Identity(), SourceCommit: parsed.Manifest.SourceCommit}, nil
	}
	// The manifest digest binds every payload file. The portable gate also binds
	// independent acceptance, exact source, issue specification and implementation.
	if err := s.authorize(ctx, parsed, *issue, *actor); err != nil {
		return result, err
	}
	again, err := s.verify(*directory)
	if err != nil || again.Identity() != parsed.Identity() {
		return result, errors.New("candidate changed during authorization")
	}
	result, err = s.publisher.Publish(ctx, *directory, parsed.Identity())
	if err != nil {
		return result, err
	}
	wantURL := "https://github.com/acoz-labs/mandalore/releases/tag/" + url.PathEscape(parsed.Manifest.Tag)
	if result.Phase != "publication-verified" || result.Identity != parsed.Identity() || result.SourceCommit != parsed.Manifest.SourceCommit || result.URL != wantURL || result.ReleaseID <= 0 || result.PendingOperation != "" || len(result.Assets) != 8 {
		return result, errors.New("publisher did not verify the complete accepted product")
	}
	return result, nil
}

func authorize(ctx context.Context, parsed distribution.ParsedManifest, issue int, actor string) error {
	if os.Getenv("GH_TOKEN") == "" {
		return errors.New("explicit maintainer credential required")
	}
	command := exec.CommandContext(ctx, "bin/sdlc-release", "gate", "--repo", "acoz-labs/mandalore",
		"--issue", fmt.Sprint(issue), "--sha", parsed.Manifest.SourceCommit,
		"--artifact", "sha256:"+parsed.SHA256, "--expected-actor", actor)
	for _, value := range os.Environ() {
		name, _, _ := strings.Cut(value, "=")
		if name != "RELEASE_POLICY_READ_TOKEN" && name != "GH_DEBUG" && name != "GH_HOST" {
			command.Env = append(command.Env, value)
		}
	}
	command.Env = append(command.Env, "GH_HOST=github.com", "GH_DEBUG=")
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
	command.Stdout, command.Stderr = io.Discard, io.Discard
	if err := command.Run(); err != nil {
		return errors.New("portable acceptance gate refused publication; run bin/sdlc-release gate directly for diagnosis")
	}
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	token := os.Getenv("GH_TOKEN")
	result, err := run(ctx, os.Args[1:], services{verify: distribution.VerifyDirectory,
		authorize: authorize, publisher: distribution.NewPublisher(token, os.Getenv("RELEASE_POLICY_READ_TOKEN"))})
	if result.Phase != "" {
		if encodeErr := json.NewEncoder(os.Stdout).Encode(result); encodeErr != nil {
			fmt.Fprintln(os.Stderr, "Cannot retain publication receipt; inspect remote effects.")
			os.Exit(1)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Local promotion failed:", err)
		os.Exit(1)
	}
}
