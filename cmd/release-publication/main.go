// release-publication verifies product releases for the maintainer ledger.
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
	"os/signal"
	"syscall"

	"github.com/acoz-labs/mandalore/internal/distribution"
)

type verifier interface {
	VerifyPublication(context.Context, string, string) (distribution.PublicationReceipt, error)
}

func run(ctx context.Context, args []string, out io.Writer, c verifier) error {
	if len(args) == 0 || args[0] != "selection" && args[0] != "verify" {
		return errors.New("choose selection or verify")
	}
	f := flag.NewFlagSet("release-publication", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	version := f.String("version", "", "Exact product version")
	identity := f.String("identity", "", "Exact accepted candidate identity")
	if err := f.Parse(args[1:]); err != nil || f.NArg() != 0 {
		return errors.New("invalid release-publication arguments")
	}
	s, err := distribution.SelectPublication(*version, *identity)
	if err != nil {
		return err
	}
	if args[0] == "selection" {
		return json.NewEncoder(out).Encode(s)
	}
	r, err := c.VerifyPublication(ctx, s.Version, s.Identity)
	if err != nil {
		return err
	}
	if r.Phase != "publication-verified" || r.Identity != s.Identity || r.SourceCommit != s.SourceCommit || r.URL != "https://github.com/acoz-labs/mandalore/releases/tag/"+url.PathEscape(s.Tag) || r.PendingOperation != "" || r.ReleaseID <= 0 || len(r.Assets) != 8 {
		return errors.New("published verification returned an incomplete or different identity")
	}
	return json.NewEncoder(out).Encode(r)
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	// Public verification never reads GH_TOKEN or native CLI credential stores.
	if err := run(ctx, os.Args[1:], os.Stdout, distribution.NewReleaseClient()); err != nil {
		fmt.Fprintln(os.Stderr, "Publication verification failed:", err)
		os.Exit(1)
	}
}
