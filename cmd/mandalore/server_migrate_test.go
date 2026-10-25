package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

// migrateCLI runs a migrate command and returns its exit code and stdout.
func migrateCLI(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var out, errout bytes.Buffer
	code := run(context.Background(), append([]string{"server", "migrate"}, args...), strings.NewReader(""), &out, &errout)
	return code, out.String()
}

// preflight runs preflight as the clone's device and saves the receipt.
func preflight(t *testing.T, c testfixture.LegacyClone) (int, string) {
	t.Helper()
	t.Setenv("MANDALORE_BINDING", c.Binding)
	code, out := migrateCLI(t, "preflight", "--clone", c.Root)
	path := filepath.Join(t.TempDir(), "receipt.json")
	if err := os.WriteFile(path, []byte(out), 0600); err != nil {
		t.Fatal(err)
	}
	return code, path
}

// serverConfig writes a configuration with no signets yet, on a loopback port
// nothing listens on.
func migrateConfig(t *testing.T) (path, stateDir, root string) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listen := l.Addr().String()
	l.Close()
	base := t.TempDir()
	stateDir, root = filepath.Join(base, "state"), filepath.Join(base, "signets", "primary")
	raw, _ := json.Marshal(map[string]any{"listen": listen, "public_url": "https://memory.example.com", "state_dir": stateDir, "owner": map[string]any{"name": "Example"}, "signets": []any{}})
	path = filepath.Join(base, "config.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path, stateDir, root
}

// treeHash covers every path, mode and byte under root except RETIRED.md.
func treeHash(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == filepath.Join(root, "RETIRED.md") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		h.Write([]byte(rel + "\x00"))
		if d.Type().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			h.Write(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// writable lists paths outside .git that keep a write bit.
func writable(t *testing.T, root string) []string {
	t.Helper()
	found := []string{}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if path == filepath.Join(root, ".git") {
			return filepath.SkipDir
		}
		if info, err := d.Info(); err == nil && info.Mode().Perm()&0222 != 0 {
			found = append(found, path)
		}
		return nil
	})
	return found
}

// restoreWrite lets t.TempDir remove retired clones afterwards.
func restoreWrite(t *testing.T, root string) {
	t.Cleanup(func() {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err == nil {
				_ = os.Chmod(path, 0700)
			}
			return nil
		})
	})
}

func result(t *testing.T, out string) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("not JSON: %s", out)
	}
	if r, ok := v["result"].(map[string]any); ok {
		return r
	}
	return v
}

// Acceptance 1, 2 and 4: preflight, init, retire and verify on devices A, B
// and C, where B starts with an unpushed commit.
func TestMigrateCutoverAcceptance(t *testing.T) {
	ctx := context.Background()
	l := testfixture.NewLegacySignet(t, "device-a", "device-b", "device-c")
	a, b, c := l.Clones["device-a"], l.Clones["device-b"], l.Clones["device-c"]
	if _, err := b.Service(t).Remember(memory.Write{Kind: "fact", Summary: "Unpushed on B", Body: "Only device B has this.", Basis: "observation", Reason: "Fixture"}); err != nil {
		t.Fatal(err)
	}
	gb, err := signetsync.Open(b.Root, l.SignetID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gb.Checkpoint(ctx); err != nil {
		t.Fatal(err)
	}

	// Acceptance 1: A clean, B blocked, C clean.
	codeA, receiptA := preflight(t, a)
	codeB, receiptB := preflight(t, b)
	codeC, receiptC := preflight(t, c)
	if codeA != 0 || codeB != 1 || codeC != 0 {
		t.Fatalf("preflight exit codes A=%d B=%d C=%d, want 0 1 0", codeA, codeB, codeC)
	}
	raw, _ := os.ReadFile(receiptB)
	if r := result(t, string(raw)); r["clean"] != false || r["device_id"] != "device-b" || r["unpushed_commits"] != float64(1) {
		t.Fatalf("B receipt: %v", r)
	}

	// Acceptance 2: init refuses while B is blocked, then succeeds.
	config, stateDir, root := migrateConfig(t)
	before, _ := os.ReadFile(config)
	init := func(receipts ...string) (int, string) {
		args := []string{"init", "--config", config, "--remote", l.Remote, "--slug", "primary", "--root", root}
		for _, r := range receipts {
			args = append(args, "--preflight", r)
		}
		return migrateCLI(t, args...)
	}
	if code, out := init(receiptA, receiptB, receiptC); code == 0 || !strings.Contains(out, "device-b is blocked") {
		t.Fatalf("init accepted a blocked device: %d %s", code, out)
	}
	if after, _ := os.ReadFile(config); !bytes.Equal(before, after) {
		t.Fatal("refused init changed the configuration")
	}
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("refused init left a checkout")
	}
	if s, err := gb.Sync(ctx, 30*time.Second); err != nil || !s.Delivered {
		t.Fatal(s, err)
	}
	if codeB, receiptB = preflight(t, b); codeB != 0 {
		t.Fatalf("B still blocked after pushing: %d", codeB)
	}
	code, out := init(receiptA, receiptB, receiptC)
	if code != 0 {
		t.Fatalf("init: %d %s", code, out)
	}
	if m := result(t, out); m["pushed"] != true || len(m["devices"].([]any)) != 3 {
		t.Fatalf("init result: %v", m)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "signets", "primary", "migration.json")); err != nil {
		t.Fatal("init did not record the migration in server state")
	}

	for _, clone := range []testfixture.LegacyClone{a, b, c} {
		restoreWrite(t, clone.Root)
		hash := treeHash(t, clone.Root)
		if code, out := migrateCLI(t, "retire", "--clone", clone.Root, "--server", "https://memory.example.com"); code != 0 {
			t.Fatalf("retire: %d %s", code, out)
		}
		if treeHash(t, clone.Root) != hash {
			t.Fatal("retire changed clone content")
		}
		if w := writable(t, clone.Root); len(w) > 0 {
			t.Fatalf("retire left writable paths: %v", w)
		}
		note, err := os.ReadFile(filepath.Join(clone.Root, "RETIRED.md"))
		if err != nil || !strings.Contains(string(note), "https://memory.example.com") || !strings.Contains(string(note), time.Now().UTC().Format("2006-01-02")) {
			t.Fatalf("RETIRED.md: %s %v", note, err)
		}
	}

	// Acceptance 4: verify passes, and a 1.x write in a retired clone fails
	// with a permission error.
	var listen struct{ Listen string }
	raw, _ = os.ReadFile(config)
	_ = json.Unmarshal(raw, &listen)
	serving, err := net.Listen("tcp", listen.Listen)
	if err != nil {
		t.Fatal(err)
	}
	code, out = migrateCLI(t, "verify", "--config", config, "--slug", "primary")
	serving.Close()
	if code == 0 || !strings.Contains(out, "listening") {
		t.Fatalf("verify wrote while the server was listening: %d %s", code, out)
	}
	code, out = migrateCLI(t, "verify", "--config", config, "--slug", "primary")
	if code != 0 {
		t.Fatalf("verify: %d %s", code, out)
	}
	if v := result(t, out); v["head"] != v["remote_head"] || v["canary_record_id"] == "" {
		t.Fatalf("verify result: %v", v)
	}
	if testfixture.Git(t, l.RemoteDir, "rev-parse", "main") != result(t, out)["head"] {
		t.Fatal("verify's canary is not the remote head")
	}
	if os.Geteuid() == 0 {
		t.Log("running as root, which ignores file modes: the permission error is not observable here")
		return
	}
	_, err = a.Service(t).Remember(memory.Write{Kind: "fact", Summary: "Late 1.x write", Body: "Must fail.", Basis: "observation", Reason: "Fixture"})
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("1.x write in a retired clone: %v, want a permission error", err)
	}
}

// Acceptance 3: --device-lost lets init proceed without that device's
// preflight and records the reason in server state only.
func TestMigrateInitDeviceLost(t *testing.T) {
	l := testfixture.NewLegacySignet(t, "device-a", "device-b", "device-c")
	_, receiptA := preflight(t, l.Clones["device-a"])
	_, receiptB := preflight(t, l.Clones["device-b"])
	config, stateDir, root := migrateConfig(t)
	args := []string{"init", "--config", config, "--remote", l.Remote, "--slug", "primary", "--root", root, "--preflight", receiptA, "--preflight", receiptB}
	if code, out := migrateCLI(t, args...); code == 0 || !strings.Contains(out, "device-c") {
		t.Fatalf("init proceeded without C: %d %s", code, out)
	}
	const reason = "Synthetic laptop retired before the cutover"
	code, out := migrateCLI(t, append(args, "--device-lost", "device-c", "--reason", reason)...)
	if code != 0 {
		t.Fatalf("init: %d %s", code, out)
	}
	state, err := os.ReadFile(filepath.Join(stateDir, "signets", "primary", "migration.json"))
	if err != nil || !strings.Contains(string(state), reason) || !strings.Contains(string(state), `"evidence": "lost"`) {
		t.Fatalf("lost reason not in server state: %s %v", state, err)
	}
	history := testfixture.Git(t, root, "log", "-p", "--all")
	if strings.Contains(history, reason) {
		t.Fatal("lost reason leaked into the signet history")
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if data, _ := os.ReadFile(path); d.Type().IsRegular() && bytes.Contains(data, []byte(reason)) {
			t.Fatalf("lost reason leaked into the signet: %s", path)
		}
		return nil
	})
	var c struct{ Signets []map[string]any }
	raw, _ := os.ReadFile(config)
	if json.Unmarshal(raw, &c) != nil || len(c.Signets) != 1 || c.Signets[0]["root"] != root || c.Signets[0]["signet_id"] != l.SignetID {
		t.Fatalf("config entry: %s", raw)
	}
	for _, bad := range [][]string{
		{"--device-lost", "device-c"},
		{"--reason", reason},
		{"--device-lost", "device-c", "--reason", reason, "--reason", "twice"},
	} {
		if code, _ := migrateCLI(t, append([]string{"init", "--config", config, "--remote", l.Remote, "--slug", "other", "--root", root + "-other"}, bad...)...); code != 2 {
			t.Errorf("%v: exit %d, want a usage error", bad, code)
		}
	}
}

func TestMigrateInitRejectsForeignOrForgedReceipts(t *testing.T) {
	l := testfixture.NewLegacySignet(t, "device-a")
	other := testfixture.NewLegacySignet(t, "device-a")
	config, _, root := migrateConfig(t)
	_, foreign := preflight(t, other.Clones["device-a"])
	_, receipt := preflight(t, l.Clones["device-a"])
	raw, _ := os.ReadFile(receipt)
	var envelope map[string]any
	_ = json.Unmarshal(raw, &envelope)
	envelope["result"].(map[string]any)["head"] = strings.Repeat("0", 40)
	forged := filepath.Join(t.TempDir(), "forged.json")
	raw, _ = json.Marshal(envelope)
	if err := os.WriteFile(forged, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{foreign: "another signet", forged: "not on the remote", config: "not the JSON output"} {
		code, out := migrateCLI(t, "init", "--config", config, "--remote", l.Remote, "--slug", "primary", "--root", root, "--preflight", path)
		if code == 0 || !strings.Contains(out, want) {
			t.Errorf("init with %s: %d %s, want %q", filepath.Base(path), code, out, want)
		}
	}
	if code, out := migrateCLI(t, "init", "--config", config, "--remote", l.Remote, "--slug", "primary", "--root", root, "--device-lost", "device-z", "--reason", "Never enrolled"); code == 0 || !strings.Contains(out, "device-z") {
		t.Errorf("init accepted an unenrolled lost device: %d %s", code, out)
	}
}

// Acceptance 5: a clone with a pending .mandalore intent is blocked.
func TestMigratePreflightBlocksPendingIntents(t *testing.T) {
	l := testfixture.NewLegacySignet(t, "device-a", "device-b")
	a, b := l.Clones["device-a"], l.Clones["device-b"]
	if code, _ := preflight(t, a); code != 0 {
		t.Fatalf("clean clone blocked: %d", code)
	}
	// B writes an intent; copying it to A without its event leaves A with an
	// intent whose publication never happened.
	if _, err := b.Service(t).AppendJournalIdempotent("request-synthetic", "note", "Interrupted write"); err != nil {
		t.Fatal(err)
	}
	intents, _ := filepath.Glob(filepath.Join(b.Root, ".mandalore", "operation-*.json"))
	if len(intents) != 1 {
		t.Fatal(intents)
	}
	data, _ := os.ReadFile(intents[0])
	if err := os.WriteFile(filepath.Join(a.Root, ".mandalore", filepath.Base(intents[0])), data, 0600); err != nil {
		t.Fatal(err)
	}
	code, path := preflight(t, a)
	raw, _ := os.ReadFile(path)
	r := result(t, string(raw))
	if code != 1 || r["clean"] != false || len(r["pending_intents"].([]any)) != 1 || len(r["uncommitted"].([]any)) != 0 {
		t.Fatalf("pending intent not blocked: %d %v", code, r)
	}
}

func TestMigratePreflightOutputAndErrors(t *testing.T) {
	l := testfixture.NewLegacySignet(t, "device-a")
	a := l.Clones["device-a"]
	t.Setenv("MANDALORE_BINDING", a.Binding)
	code, out := migrateCLI(t, "preflight", "--clone", a.Root, "--format", "text")
	if code != 0 || !strings.Contains(out, "device device-a: clean") {
		t.Fatalf("text preflight: %d %s", code, out)
	}
	t.Setenv("MANDALORE_BINDING", filepath.Join(t.TempDir(), "missing.json"))
	if code, out := migrateCLI(t, "preflight", "--clone", a.Root); code != 0 || result(t, out)["device_id"] != nil {
		t.Fatalf("preflight without a binding: %d %s", code, out)
	}
	if code, out := migrateCLI(t, "preflight", "--clone", t.TempDir()); code != 2 || !json.Valid([]byte(out)) {
		t.Fatalf("preflight of a non-signet: %d %s", code, out)
	}
	if err := os.MkdirAll(filepath.Join(a.Root, "memory/records/record-broken"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a.Root, "memory/records/record-broken/revision-broken.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if code, _ := migrateCLI(t, "preflight", "--clone", a.Root); code != 2 {
		t.Fatalf("preflight of an invalid signet: exit %d, want 2", code)
	}
	for _, args := range [][]string{{}, {"preflight"}, {"preflight", "--clone", a.Root, "--format", "yaml"}, {"preflight", "--clone", a.Root, "--config", "x"},
		{"retire", "--clone", a.Root}, {"retire", "--clone", a.Root, "--server", "https://memory.example.com", "--native", "codex=relative,/bin/false"},
		{"verify", "--slug", "primary"}, {"rollback"}} {
		if code, out := migrateCLI(t, args...); code != 2 || !json.Valid([]byte(out)) {
			t.Errorf("%v: %d %s", args, code, out)
		}
	}
}
