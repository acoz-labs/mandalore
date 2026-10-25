package testfixture

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/formatupgrade"
	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

// LegacySignet is a synthetic format 2 signet as 1.x devices hold it: a bare
// file:// remote and one clone per device, each with its own binding outside
// the clone, all in a new test directory.
type LegacySignet struct {
	SignetID, Remote, RemoteDir string
	Clones                      map[string]LegacyClone // by device ID
}

type LegacyClone struct {
	Root, Binding string
}

// NewLegacySignet enrolls every device, upgrades the signet to format 2,
// pushes it from the first device and clones it for the rest.
func NewLegacySignet(t testing.TB, devices ...string) LegacySignet {
	t.Helper()
	ctx := context.Background()
	base := t.TempDir()
	clone := func(device string) LegacyClone {
		return LegacyClone{Root: filepath.Join(base, device, "signet"), Binding: filepath.Join(base, device, "binding.json")}
	}
	first := clone(devices[0])
	store, err := memory.Create(first.Root, "Synthetic legacy", devices[0], "Synthetic "+devices[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, device := range devices[1:] {
		if err := store.AddDevice(memory.Device{Version: 1, ID: device, Label: "Synthetic " + device}); err != nil {
			t.Fatal(err)
		}
	}
	l := LegacySignet{SignetID: store.Signet.ID, RemoteDir: filepath.Join(base, "remote.git"), Clones: map[string]LegacyClone{}}
	l.Remote = "file://" + l.RemoteDir
	g, err := signetsync.Open(store.Root, store.Signet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	writeBinding(t, first.Binding, binding.Binding{Version: 1, SignetID: l.SignetID, Root: store.Root, DeviceID: devices[0], Actor: "Synthetic actor"})
	plan, err := formatupgrade.Preview(ctx, formatupgrade.Request{BindingPath: first.Binding})
	if err != nil {
		t.Fatal(err)
	}
	if r, err := formatupgrade.Apply(ctx, formatupgrade.ApplyRequest{Plan: plan, StoppedWriters: true}); err != nil || !r.Activated {
		t.Fatal(r, err)
	}
	if g, err = signetsync.Open(store.Root, store.Signet.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Checkpoint(ctx); err != nil {
		t.Fatal(err)
	}
	Git(t, base, "init", "--bare", "--initial-branch=main", l.RemoteDir)
	Git(t, store.Root, "remote", "add", "origin", l.Remote)
	if _, err := g.Push(ctx, l.Remote); err != nil {
		t.Fatal(err)
	}
	l.Clones[devices[0]] = first
	for _, device := range devices[1:] {
		c := clone(device)
		Git(t, base, "clone", "--quiet", l.Remote, c.Root)
		writeBinding(t, c.Binding, binding.Binding{Version: 1, SignetID: l.SignetID, Root: c.Root, DeviceID: device, Actor: "Synthetic actor"})
		l.Clones[device] = c
	}
	return l
}

func writeBinding(t testing.TB, path string, b binding.Binding) {
	t.Helper()
	raw, _ := json.Marshal(b)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
}

// LegacyService opens a clone as its 1.x device would.
func (c LegacyClone) Service(t testing.TB) *memory.Service {
	t.Helper()
	s, err := binding.Open(c.Binding, "codex")
	if err != nil {
		t.Fatal(err)
	}
	return s
}
