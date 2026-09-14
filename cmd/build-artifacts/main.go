// build-artifacts builds local candidates only. It cannot publish a GitHub release.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/acoz-labs/mandalore/internal/distribution"
)

func main() {
	f := flag.NewFlagSet("build-artifacts", flag.ExitOnError)
	source := f.String("source", ".", "Clean source repository root")
	output := f.String("output", "", "New candidate directory outside the source tree")
	f.Parse(os.Args[1:])
	if *output == "" || f.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Usage: bin/build-artifacts --output NEW-DIRECTORY [--source REPOSITORY]")
		os.Exit(2)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	p, err := distribution.Build(ctx, distribution.BuildOptions{Source: *source, Output: *output})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Candidate build failed:", err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(p); err != nil {
		os.Exit(1)
	}
}
