package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/cutover"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

// migrateFixture is a one-device legacy signet with a clean preflight receipt
// and a server configuration with no signets, on a port nothing listens on.
func migrateFixture(t *testing.T) (testfixture.LegacySignet, MigrateInit) {
	t.Helper()
	l := testfixture.NewLegacySignet(t, "device-a")
	a := l.Clones["device-a"]
	r, err := cutover.Preflight(context.Background(), a.Root, a.Binding)
	if err != nil || !r.Clean {
		t.Fatal(r, err)
	}
	base := t.TempDir()
	receipt := filepath.Join(base, "receipt.json")
	raw, _ := json.Marshal(map[string]any{"protocol_version": 1, "ok": true, "result": r})
	if err := os.WriteFile(receipt, raw, 0600); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listen := ln.Addr().String()
	ln.Close()
	config := filepath.Join(base, "config.json")
	raw, _ = json.Marshal(map[string]any{"listen": listen, "public_url": "https://memory.example.com", "state_dir": filepath.Join(base, "state"), "owner": map[string]any{"name": "Example"}, "signets": []any{}})
	if err := os.WriteFile(config, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return l, MigrateInit{ConfigPath: config, Remote: l.Remote, Slug: "primary", Root: filepath.Join(base, "signets", "primary"), Preflights: []string{receipt}}
}

func remoteMain(t *testing.T, l testfixture.LegacySignet) string {
	t.Helper()
	return testfixture.Git(t, l.RemoteDir, "rev-parse", "main")
}

// An init that fails after pushing the enrollment says so, and the documented
// recovery (remove root, mark that server device lost) completes it.
func TestInitStopsHonestlyAfterPush(t *testing.T) {
	l, in := migrateFixture(t)
	before, _ := os.ReadFile(in.ConfigPath)
	writeConfig = func(string, Config) error { return errors.New("synthetic failure") }
	m, err := Init(context.Background(), in)
	writeConfig = func(path string, c Config) error { return writeState(path, c) }
	var partial *cutover.PartialError
	if !errors.As(err, &partial) {
		t.Fatalf("late init failure is not reported as a partial write: %v", err)
	}
	for _, want := range []string{in.Root, m.ServerDeviceID, "pushed to the remote as " + remoteMain(t, l)[:12], "configuration is unchanged"} {
		if m.ServerDeviceID == "" || !strings.Contains(err.Error(), want) {
			t.Errorf("message %q lacks %q", err, want)
		}
	}
	if after, _ := os.ReadFile(in.ConfigPath); string(after) != string(before) {
		t.Fatal("configuration changed")
	}
	// A plain rerun refuses: the orphaned server device is enrolled.
	if err := os.RemoveAll(in.Root); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), in); err == nil || !strings.Contains(err.Error(), m.ServerDeviceID) {
		t.Fatalf("rerun without marking the orphan: %v", err)
	}
	in.Lost = []LostDevice{{ID: m.ServerDeviceID, Reason: "Server enrollment from an init that stopped before writing the configuration"}}
	again, err := Init(context.Background(), in)
	if err != nil || !again.Pushed || again.ServerDeviceID == m.ServerDeviceID {
		t.Fatal(again, err)
	}
}

// A verify whose push fails reports the canary's state, and starting the
// server once, as the runbook says, pushes it so a rerun passes.
func TestVerifyStopsHonestlyAfterCanary(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores the read-only remote this test uses to fail the push")
	}
	l, in := migrateFixture(t)
	if _, err := Init(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(in.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	chmodTree(t, l.RemoteDir, 0555)
	_, err = Verify(context.Background(), c, "primary")
	chmodTree(t, l.RemoteDir, 0755)
	var partial *cutover.PartialError
	if !errors.As(err, &partial) || !strings.Contains(err.Error(), "withdrawn and committed, not confirmed pushed") {
		t.Fatalf("late verify failure: %v", err)
	}
	if _, err := Verify(context.Background(), c, "primary"); err == nil || errors.As(err, &partial) {
		t.Fatalf("rerun before the server pushed: %v", err)
	}
	s, err := newServer(c, io.Discard, fastTiming)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)
	g, err := signetsync.Open(in.Root, l.SignetID)
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, 5*time.Second, "server pushed the canary", func() bool {
		head, _ := g.Head(context.Background())
		return remoteMain(t, l) == head
	})
	cancel()
	if _, err := Verify(context.Background(), c, "primary"); err != nil {
		t.Fatal(err)
	}
}

func chmodTree(t *testing.T, root string, mode fs.FileMode) {
	t.Helper()
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_ = os.Chmod(path, mode)
		}
		return nil
	})
}
