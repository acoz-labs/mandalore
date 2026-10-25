package cutover_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/cutover"
	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

// A format 1 clone is blocked with a pointer to the runbook's upgrade step,
// and that step is where the message says.
func TestPreflightFormatOneNamesTheRunbookStep(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store, err := memory.Create(filepath.Join(t.TempDir(), "signet"), "Synthetic format 1", "device-a", "Synthetic device-a")
	if err != nil {
		t.Fatal(err)
	}
	g, err := signetsync.Open(store.Root, store.Signet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	r, err := cutover.Preflight(context.Background(), store.Root, "")
	if err != nil || r.Clean || r.Format != 1 {
		t.Fatalf("preflight of a format 1 clone: %+v %v", r, err)
	}
	blocker := ""
	for _, b := range r.Blockers {
		if strings.HasPrefix(b, "format 1") {
			blocker = b
		}
	}
	for _, want := range []string{"runbook step 1 in docs/migration-2.0.md", "signet_upgrade_preview", "stopped_writers: true", "1.x binary", "sync"} {
		if !strings.Contains(blocker, want) {
			t.Errorf("format 1 blocker %q lacks %q", blocker, want)
		}
	}
	doc, err := os.ReadFile("../../docs/migration-2.0.md")
	if err != nil {
		t.Fatal(err)
	}
	runbook := string(doc)[strings.Index(string(doc), "## Runbook"):]
	if !strings.HasPrefix(runbook[strings.Index(runbook, "1. "):], "1. Upgrade a format 1 signet to format 2") ||
		strings.Index(runbook, "signet_upgrade_apply") > strings.Index(runbook, "Preflight every device") {
		t.Fatal("runbook step 1 is not the format upgrade before preflight")
	}
}

// An ignored canon cache whose full listing would exceed the Git output limit
// does not stop preflight, and an output-limit failure is reported as one.
func TestPreflightLargeIgnoredCanonCacheAndOutputLimitCause(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store, err := memory.Create(filepath.Join(t.TempDir(), "signet"), "Synthetic canon cache", "device-a", "Synthetic device-a")
	if err != nil {
		t.Fatal(err)
	}
	g, err := signetsync.Open(store.Root, store.Signet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(store.Root, ".mandalore", "canon", "source-"+strings.Repeat("a", 200))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for size, i := 0, 0; size <= 1<<20; i++ {
		name := fmt.Sprintf("%06d-%s.md", i, strings.Repeat("b", 200))
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
		size += len(".mandalore/canon//") + len(filepath.Base(dir)) + len(name) + 1
	}
	r, err := cutover.Preflight(context.Background(), store.Root, "")
	if err != nil || len(r.Uncommitted) != 0 || r.Remote != "none" {
		t.Fatalf("preflight with a large canon cache: %+v %v", r, err)
	}

	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte("#!/bin/sh\nwhile :; do printf 'synthetic-output\\n'; done\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	_, err = cutover.Preflight(context.Background(), store.Root, "")
	if err == nil || !strings.Contains(err.Error(), "output exceeded the 1048576-byte limit") || strings.Contains(err.Error(), "standalone") {
		t.Fatalf("output-limit cause not passed through: %v", err)
	}
}
