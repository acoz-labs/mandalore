// candidate-transport is maintainer workflow tooling, not a memory MCP operation.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"syscall"

	"github.com/acoz-labs/mandalore/internal/distribution"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	"golang.org/x/sys/unix"
)

type inspector interface {
	InspectCandidate(context.Context, distribution.CandidateSelection) (distribution.CandidateTransport, error)
}

func regular(path string, limit int64) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, errors.New("candidate input is missing, redirected or unreadable")
	}
	f := os.NewFile(uintptr(fd), path)
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() < 1 || st.Size() > limit {
		f.Close()
		return nil, errors.New("candidate input type or size is invalid")
	}
	return f, nil
}

func run(ctx context.Context, args []string, out io.Writer, c inspector) error {
	if len(args) == 0 || (args[0] != "inspect" && args[0] != "verify") {
		return errors.New("choose inspect or verify")
	}
	f := flag.NewFlagSet("candidate-transport", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var s distribution.CandidateSelection
	f.StringVar(&s.SourceCommit, "source", "", "Exact source SHA")
	f.Int64Var(&s.RunID, "run-id", 0, "Successful candidate workflow run")
	f.Int64Var(&s.ArtifactID, "artifact-id", 0, "Retained Actions artifact ID")
	receipt := f.String("receipt", "", "Previously inspected transport JSON (verify only)")
	archive := f.String("archive", "", "Downloaded raw ZIP (verify only)")
	expected := f.String("expected-identity", "", "Required accepted candidate identity (optional for first nomination)")
	if err := f.Parse(args[1:]); err != nil || f.NArg() != 0 {
		return errors.New("invalid candidate-transport arguments")
	}
	if args[0] == "inspect" {
		if *receipt != "" || *archive != "" || *expected != "" {
			return errors.New("inspect does not accept downloaded inputs")
		}
		if err := s.Validate(); err != nil {
			return err
		}
		t, err := c.InspectCandidate(ctx, s)
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(t)
	}
	if s.SourceCommit != "" || s.RunID != 0 || s.ArtifactID != 0 || *receipt == "" || *archive == "" {
		return errors.New("verify requires exactly a receipt and archive, not a replacement selection")
	}
	r, err := regular(*receipt, 16384)
	if err != nil {
		return err
	}
	raw, err := io.ReadAll(io.LimitReader(r, 16385))
	r.Close()
	if err != nil {
		return errors.New("cannot read transport receipt")
	}
	var t distribution.CandidateTransport
	if err := strictjson.Decode(raw, &t, 16384); err != nil {
		return errors.New("invalid transport receipt")
	}
	fresh, err := c.InspectCandidate(ctx, t.CandidateSelection)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(fresh, t) {
		return errors.New("candidate transport changed; re-inspect instead of replacing accepted bytes")
	}
	a, err := regular(*archive, distribution.MaxCandidateArchiveBytes)
	if err != nil {
		return err
	}
	defer a.Close()
	st, err := a.Stat()
	if err != nil {
		return err
	}
	p, err := distribution.VerifyCandidateArchive(ctx, a, st.Size(), t)
	if err != nil {
		return err
	}
	if *expected != "" && *expected != p.Identity() {
		return errors.New("verified candidate does not match accepted identity")
	}
	fresh, err = c.InspectCandidate(ctx, t.CandidateSelection)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(fresh, t) {
		return errors.New("candidate transport changed during verification")
	}
	return json.NewEncoder(out).Encode(struct {
		Identity  string                          `json:"identity"`
		Transport distribution.CandidateTransport `json:"transport"`
		Manifest  distribution.ParsedManifest     `json:"manifest"`
	}{p.Identity(), t, p})
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout, distribution.NewCandidateClient(os.Getenv("GH_TOKEN"))); err != nil {
		fmt.Fprintln(os.Stderr, "Candidate verification failed:", err)
		os.Exit(1)
	}
}
