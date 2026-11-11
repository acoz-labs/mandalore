package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestOldReaderOnMovedRecord runs an older binary (MANDALORE_OLD_BINARY:
// bin/check-old-reader builds 2.2.1 from 95bbf17 and 2.1.2 from 79b378e)
// against signets that this build has moved a record in, on format 1 and
// format 2, and asserts each row of the old-reader table in
// docs/signet-format.md ("Record moves").
func TestOldReaderOnMovedRecord(t *testing.T) {
	old := os.Getenv("MANDALORE_OLD_BINARY")
	if old == "" {
		t.Skip("MANDALORE_OLD_BINARY is unset; run bin/check-old-reader")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	current := filepath.Join(t.TempDir(), "mandalore")
	if out, err := exec.CommandContext(ctx, "go", "build", "-o", current, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	for _, format := range []string{"format1", "format2"} {
		t.Run(format, func(t *testing.T) { oldReaderMoved(ctx, t, old, current, format) })
	}
}

var (
	movedFrom = map[string]any{"kind": "project", "id": "example-x"}
	movedTo   = map[string]any{"kind": "project", "id": "example-y"}
)

func oldReaderMoved(ctx context.Context, t *testing.T, old, current, format string) {
	e := env{t: t, ctx: ctx, home: t.TempDir()}
	base := t.TempDir()
	root := filepath.Join(base, "server")
	if err := os.CopyFS(root, os.DirFS("../../internal/memory/testdata/compat/format1")); err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		ID string `json:"id"`
	}
	raw, _ := os.ReadFile(filepath.Join(root, "signet.json"))
	_ = json.Unmarshal(raw, &manifest)
	devices, _ := os.ReadDir(filepath.Join(root, "provenance/devices"))
	device := ""
	for _, d := range devices {
		if strings.HasSuffix(d.Name(), ".json") {
			device = strings.TrimSuffix(d.Name(), ".json")
		}
	}
	binding := filepath.Join(base, "server-binding.json")
	e.ok("", current, "signet", "bind", "--repository", root, "--binding", binding, "--device-label", "Synthetic CLI", "--actor", "Synthetic actor")
	remote := filepath.Join(base, "remote.git")
	e.git(base, "init", "-q", "--bare", "--initial-branch=main", remote)
	e.git(root, "init", "-q", "--initial-branch=main")
	e.git(root, "add", ".")
	e.git(root, "commit", "-q", "-m", "Released fixture")
	if format == "format2" {
		plan, _ := json.Marshal(e.ok(`{"binding_path":"`+binding+`"}`, old, "call", "signet_upgrade_preview"))
		e.ok(`{"plan":`+string(plan)+`,"stopped_writers":true}`, old, "call", "signet_upgrade_apply")
		e.ok("", old, "memory", "checkpoint", "--binding", binding)
	}
	e.git(root, "remote", "add", "origin", remote)
	e.git(root, "push", "-q", "origin", "main")
	behind, ahead := filepath.Join(base, "behind"), filepath.Join(base, "ahead")
	e.git(base, "clone", "-q", remote, behind)
	e.git(base, "clone", "-q", remote, ahead)
	aheadBinding := filepath.Join(base, "ahead-binding.json")
	e.ok("", old, "signet", "bind", "--repository", ahead, "--binding", aheadBinding, "--device-label", "Synthetic old clone", "--actor", "Synthetic actor")
	e.ok(`{"kind":"fact","summary":"Local only","body":"Written by the old binary and not pushed","basis":"observation","reason":"Synthetic"}`, old, "memory", "remember", "--binding", aheadBinding)
	e.ok("", old, "memory", "checkpoint", "--binding", aheadBinding)

	// This build saves a record in X, corrects it, moves it to Y and backs up.
	address := freeAddress(t)
	config := loopbackConfig(t, base, address, map[string]any{"slug": "primary", "root": root, "signet_id": manifest.ID, "device_id": device, "remote": "file://" + remote})
	srv := e.serve(current, config, address)
	remember := func(id string, scope map[string]any, body, recordID string, supersedes ...string) map[string]any {
		record := map[string]any{"kind": "fact", "summary": "Synthetic moved record", "body": body, "basis": "user-direction", "reason": "Synthetic", "scope": scope}
		if recordID != "" {
			record["record_id"], record["supersedes"] = recordID, supersedes
		}
		return map[string]any{"request_id": id, "record": record}
	}
	receipt := func(text string) (id, recordID string) {
		var v struct {
			Result struct {
				ID       string `json:"id"`
				RecordID string `json:"record_id"`
			} `json:"result"`
		}
		if json.Unmarshal([]byte(text), &v) != nil || v.Result.ID == "" {
			t.Fatalf("receipt: %s", text)
		}
		return v.Result.ID, v.Result.RecordID
	}
	ok, text := srv.call(t, "memory_remember", remember("old-reader-save-00001", movedFrom, "MOVED_MARKER first", ""))
	if !ok {
		t.Fatal(text)
	}
	first, recordID := receipt(text)
	if ok, text = srv.call(t, "memory_remember", remember("old-reader-save-00002", movedFrom, "MOVED_MARKER second", recordID, first)); !ok {
		t.Fatal(text)
	}
	second, _ := receipt(text)
	moveArgs := map[string]any{"request_id": "old-reader-move-00001", "move": map[string]any{"record_id": recordID, "scope": movedTo, "expected_content_heads": []string{second}, "expected_move_heads": []string{}, "reason": "Synthetic move"}}
	if ok, text = srv.call(t, "memory_move", moveArgs); !ok {
		t.Fatal(text)
	}
	for deadline := time.Now().Add(15 * time.Second); !strings.Contains(e.git(base, "--git-dir="+remote, "ls-tree", "-r", "--name-only", "main"), "memory/moves/"); time.Sleep(100 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the move never reached the remote")
		}
	}
	srv.stop()
	name := filepath.Base(old)

	// Format 1 upgrade: the old preview refuses, and so does the old apply of
	// a plan this build previews.
	if format == "format1" {
		if out, _ := e.run(`{"binding_path":"`+binding+`"}`, old, "call", "signet_upgrade_preview"); strings.Contains(out, `"ok":true`) {
			t.Fatalf("%s upgrade preview: %s", name, out)
		}
		before := files(t, root)
		plan, _ := json.Marshal(e.ok(`{"binding_path":"`+binding+`"}`, current, "call", "signet_upgrade_preview"))
		out, _ := e.run(`{"plan":`+string(plan)+`,"stopped_writers":true}`, old, "call", "signet_upgrade_apply")
		if strings.Contains(out, `"ok":true`) {
			t.Fatalf("%s upgrade apply: %s", name, out)
		}
		sameFiles(t, name+" upgrade", before, files(t, root))
		_, err := os.Stat(filepath.Join(root, ".mandalore/format-upgrade"))
		t.Logf("%s upgrade preview and apply refuse; the signet stays format 1 (retained preparation: %v): %s", name, err == nil, strings.TrimSpace(out))
	}

	// The old server serves the record in its old scope.
	srv = e.serve(old, config, address)
	list, err := srv.session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	prime := false
	for _, tool := range list.Tools {
		if tool.Name == "memory_move" {
			t.Fatalf("%s lists memory_move", name)
		}
		prime = prime || tool.Name == "memory_prime"
	}
	reads := []struct {
		name   string
		args   map[string]any
		want   string
		absent string
	}{
		{"memory_recall", map[string]any{"scope": movedFrom}, "MOVED_MARKER second", ""},
		{"memory_recall", map[string]any{"scope": movedTo}, `"current":[]`, "MOVED_MARKER"},
		{"memory_scopes", map[string]any{"limit": 50}, "example-x", "example-y"},
		{"memory_history", map[string]any{"record_id": recordID}, "MOVED_MARKER first", "move-"},
		{"memory_context", map[string]any{"event": "start"}, "example-x", "example-y"},
	}
	if prime {
		reads = append(reads, struct {
			name   string
			args   map[string]any
			want   string
			absent string
		}{"memory_prime", map[string]any{"scope": movedFrom}, `"summary":"Synthetic moved record"`, ""})
	}
	for _, r := range reads {
		ok, text := srv.call(t, r.name, r.args)
		if !ok || !strings.Contains(text, r.want) || r.absent != "" && strings.Contains(text, r.absent) {
			t.Fatalf("%s %s %v: %s", name, r.name, r.args, text)
		}
	}
	before := files(t, root)
	ok, text = srv.call(t, "memory_remember", map[string]any{"request_id": "old-reader-move-00001", "record": map[string]any{"kind": "fact", "summary": "s", "body": "b", "basis": "observation", "reason": "r"}})
	if ok || !strings.Contains(text, `"save.incomplete"`) {
		t.Fatalf("%s reusing a move's request_id: %s", name, text)
	}
	if ok, text = srv.call(t, "memory_remember", remember("old-reader-save-00003", movedTo, "MOVED_MARKER new scope", recordID, second)); ok {
		t.Fatalf("%s correction in the new scope: %s", name, text)
	}
	sameFiles(t, name+" refused writes", before, files(t, root))
	// A correction in the old scope saves; its backup then never completes.
	if ok, text = srv.call(t, "memory_remember", remember("old-reader-save-00004", movedFrom, "MOVED_MARKER old scope", recordID, second)); !ok {
		t.Fatalf("%s correction in the old scope: %s", name, text)
	}
	third, _ := receipt(text)
	var status string
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		// The refused checkpoint reads as an unconfirmed push.
		out, _ := e.run("", old, "server", "status", "--config", config)
		if strings.Contains(out, `"state":"pending"`) && strings.Contains(out, `"remote":"unreachable"`) && strings.Contains(out, "push not confirmed; retrying") {
			status = out
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s backup state: %s", name, out)
		}
	}
	srv.stop()
	if !strings.Contains(status, `"ok":true`) {
		t.Fatalf("%s server status: %s", name, status)
	}
	t.Logf("%s: reads work in the old scope; memory_move not served; a reused move request_id is save.incomplete; a new-scope correction is refused; an old-scope correction saves and its backup never completes; server status works and shows it pending, remote unreachable: %s", name, strings.TrimSpace(status))
	// Back on this build, the record is still in Y, with that correction.
	got := e.ok("", current, "memory", "recall", "--binding", binding, "--scope-kind", "project", "--scope-id", "example-y")
	if b, _ := json.Marshal(got); !strings.Contains(string(b), third) || !strings.Contains(string(b), `"id":"example-y"`) {
		t.Fatalf("current recall after the old correction: %s", b)
	}

	if r := e.ok("", old, "memory", "inspect", "--binding", binding); r["healthy"] != true {
		t.Fatalf("%s inspect: %v", name, r)
	}
	worktree := e.git(root, "status", "--porcelain")
	if out, done := e.run("", old, "memory", "checkpoint", "--binding", binding); done || !strings.Contains(out, `ok":false`) {
		t.Fatalf("%s checkpoint: %s", name, out)
	}
	if e.git(root, "status", "--porcelain") != worktree {
		t.Fatalf("%s checkpoint changed the worktree", name)
	}
	menu := exec.CommandContext(ctx, old, "menu", "--plain")
	menu.Env = e.vars("MANDALORE_BINDING=" + binding)
	menu.Stdin = strings.NewReader("3\n7\n")
	if out, _ := menu.CombinedOutput(); !strings.Contains(string(out), "Cannot read local synchronization state") {
		t.Fatalf("%s menu status: %s", name, out)
	}
	preview := `{"binding_path":"` + binding + `","destination":"` + filepath.Join(t.TempDir(), "report") + `","selection":{"scopes":[{"kind":"project","id":"example-x"}]}}`
	if out, _ := e.run(preview, old, "export", "preview"); !strings.Contains(out, `ok":false`) {
		t.Fatalf("%s export preview: %s", name, out)
	}
	if out, _ := e.run("", old, "server", "migrate", "preflight", "--clone", root, "--format", "json"); strings.Contains(out, `ok":true`) {
		t.Fatalf("%s preflight read the move intent: %s", name, out)
	}
	t.Logf("%s: inspect works; checkpoint, menu status, export preview and cutover preflight refuse", name)

	for _, clone := range []string{behind, ahead} {
		cfg := loopbackConfig(t, filepath.Join(base, filepath.Base(clone)+"-config"), freeAddress(t), map[string]any{"slug": "primary", "root": clone, "signet_id": manifest.ID, "device_id": device, "remote": "file://" + remote})
		head, tree := e.git(clone, "rev-parse", "HEAD"), files(t, clone)
		if out, done := e.run("", old, "server", "reconcile", "--config", cfg, "--signet", "primary"); done {
			t.Fatalf("%s reconcile of %s: %s", name, filepath.Base(clone), out)
		}
		if e.git(clone, "rev-parse", "HEAD") != head {
			t.Fatalf("%s reconcile moved %s", name, filepath.Base(clone))
		}
		sameFiles(t, name+" reconcile", tree, files(t, clone))
	}
	head, tree := e.git(ahead, "rev-parse", "HEAD"), files(t, ahead)
	menu = exec.CommandContext(ctx, old, "menu", "--plain")
	menu.Env = e.vars("MANDALORE_BINDING=" + aheadBinding)
	menu.Stdin = strings.NewReader("4\n2\n7\n")
	if out, _ := menu.CombinedOutput(); !strings.Contains(string(out), "Remote candidate could not be safely reconciled") {
		t.Fatalf("%s menu sync: %s", name, out)
	}
	if e.git(ahead, "rev-parse", "HEAD") != head {
		t.Fatalf("%s menu sync lost the unpushed commit", name)
	}
	sameFiles(t, name+" menu sync", tree, files(t, ahead))
	t.Logf("%s: reconcile (behind and ahead) and menu sync refuse; heads and files kept", name)

	if format == "format2" {
		oldMigrate(e, old, base, root, config, remote)
	}
	oldCanon(e, old, base, manifest.ID, remote, func(srv served, foundling, head string) {
		ok, text := srv.call(t, "foundling_canon_recall", map[string]any{"foundling_id": foundling, "query": "synthetic moved record", "scope": movedFrom})
		if !ok || !strings.Contains(text, "MOVED_MARKER second") {
			t.Fatalf("%s canon recall in the old scope: %s", name, text)
		}
		t.Logf("%s canon refresh served commit %s with the record in its old scope", name, head[:12])
	})
}

// oldMigrate runs the old server migrate init on the remote with the move,
// and migrate verify on the server checkout. Both refuse late, at a
// checkpoint, after writing.
func oldMigrate(e env, old, base, root, config, remote string) {
	t, name := e.t, filepath.Base(old)
	initConfig := filepath.Join(base, "init-config", "server.json")
	if err := os.MkdirAll(filepath.Dir(initConfig), 0700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"listen": freeAddress(t), "public_url": "http://127.0.0.1", "state_dir": filepath.Join(base, "init-config", "state"), "owner": map[string]string{"name": "Synthetic owner"}, "auth": "loopback-trust", "local_scope": "read-write", "signets": []any{}})
	if err := os.WriteFile(initConfig, raw, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"server", "migrate", "init", "--config", initConfig, "--remote", "file://" + remote, "--slug", "fresh", "--root", filepath.Join(base, "fresh")}
	entries, _ := os.ReadDir(filepath.Join(root, "provenance/devices"))
	for _, d := range entries {
		if id, ok := strings.CutSuffix(d.Name(), ".json"); ok {
			args = append(args, "--device-lost", id, "--reason", "Synthetic device")
		}
	}
	out, done := e.run("", old, args...)
	if done || !strings.Contains(out, "cannot commit the server device enrollment") {
		t.Fatalf("%s migrate init: %s", name, out)
	}
	after, _ := os.ReadFile(initConfig)
	if string(after) != string(raw) {
		t.Fatalf("%s migrate init changed the configuration", name)
	}
	if _, err := os.Stat(filepath.Join(base, "fresh", "signet.json")); err != nil {
		t.Fatalf("%s migrate init left no checkout: %v", name, err)
	}
	t.Logf("%s migrate init: %s", name, strings.TrimSpace(out))

	out, done = e.run("", old, "server", "migrate", "verify", "--config", config, "--slug", "primary")
	if done || !strings.Contains(out, "withdrawn, uncommitted") {
		t.Fatalf("%s migrate verify: %s", name, out)
	}
	t.Logf("%s migrate verify: %s", name, strings.TrimSpace(out))
}
