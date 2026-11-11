package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestOldReaderOnTopicSignet runs a 2.1.2 binary (MANDALORE_OLD_BINARY: the
// release asset, or bin/check-old-reader's build of 79b378e) against signets
// that this build has written names, scoped journal entries and their intents
// into, on format 1 and format 2. It records what the old binary actually does.
func TestOldReaderOnTopicSignet(t *testing.T) {
	old := os.Getenv("MANDALORE_OLD_BINARY")
	if old == "" {
		t.Skip("MANDALORE_OLD_BINARY is unset; run bin/check-old-reader")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	current := filepath.Join(t.TempDir(), "mandalore")
	if out, err := exec.CommandContext(ctx, "go", "build", "-o", current, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	for _, format := range []string{"format1", "format2"} {
		t.Run(format, func(t *testing.T) { oldReader(ctx, t, old, current, format) })
	}
}

// env is an isolated process environment: a temporary HOME, so no binary
// reads or writes the real user's configuration, and no global Git config.
type env struct {
	t    *testing.T
	ctx  context.Context
	home string
}

func (e env) vars(extra ...string) []string {
	out := []string{}
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "HOME=") && !strings.HasPrefix(v, "GIT_") && !strings.HasPrefix(v, "MANDALORE_") && !strings.HasPrefix(v, "XDG_") {
			out = append(out, v)
		}
	}
	return append(append(out, "HOME="+e.home, "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0"), extra...)
}

// run returns a command's stdout and whether it exited 0.
func (e env) run(stdin, binary string, args ...string) (string, bool) {
	e.t.Helper()
	cmd := exec.CommandContext(e.ctx, binary, args...)
	cmd.Env = e.vars()
	cmd.Stdin = strings.NewReader(stdin)
	var out, errout bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errout
	err := cmd.Run()
	return out.String() + errout.String(), err == nil
}

func (e env) ok(stdin, binary string, args ...string) map[string]any {
	e.t.Helper()
	out, _ := e.run(stdin, binary, args...)
	var v struct {
		OK     bool           `json:"ok"`
		Result map[string]any `json:"result"`
	}
	if json.Unmarshal([]byte(out), &v) != nil || !v.OK {
		e.t.Fatalf("%s %v: %s", filepath.Base(binary), args, out)
	}
	return v.Result
}

func (e env) git(dir string, args ...string) string {
	e.t.Helper()
	cmd := exec.CommandContext(e.ctx, "git", append([]string{"-C", dir, "-c", "user.name=Synthetic", "-c", "user.email=synthetic@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Env = e.vars("GIT_CONFIG_GLOBAL=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		e.t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// files maps every portable and intent file to its bytes.
func files(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			if rel == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(rel, ".mandalore/") && !strings.HasPrefix(rel, ".mandalore/operation-") {
			return nil
		}
		b, err := os.ReadFile(path)
		out[rel] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func sameFiles(t *testing.T, what string, before, after map[string]string) {
	t.Helper()
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("%s changed %s", what, k)
		}
	}
	for k := range after {
		if _, ok := before[k]; !ok {
			t.Fatalf("%s added %s", what, k)
		}
	}
}

type served struct {
	session *sdk.ClientSession
	stop    func()
}

// serve starts binary on config and connects to it; stop interrupts that
// exact process and waits for it.
func (e env) serve(binary, config, address string) served {
	e.t.Helper()
	cmd := exec.CommandContext(e.ctx, binary, "serve", "--config", config)
	cmd.Env = e.vars()
	var logs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		e.t.Fatal(err)
	}
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		_ = cmd.Process.Signal(os.Interrupt)
		_ = cmd.Wait()
	}
	e.t.Cleanup(stop)
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if r, err := http.Get("http://" + address + "/healthz"); err == nil {
			r.Body.Close()
			if r.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			e.t.Fatalf("%s did not become healthy: %s", filepath.Base(binary), logs.String())
		}
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "old-reader-check", Version: "1"}, nil).Connect(e.ctx, &sdk.StreamableClientTransport{Endpoint: "http://" + address + "/mcp", MaxRetries: -1, DisableStandaloneSSE: true}, nil)
	if err != nil {
		e.t.Fatal(err)
	}
	return served{session: session, stop: func() { _ = session.Close(); stop() }}
}

func (s served) call(t *testing.T, name string, args any) (bool, string) {
	t.Helper()
	r, err := s.session.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	var text strings.Builder
	for _, c := range r.Content {
		if tc, ok := c.(*sdk.TextContent); ok {
			text.WriteString(tc.Text)
		}
	}
	return !r.IsError, text.String()
}

func freeAddress(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

// loopbackConfig writes a loopback-trust, read-write server config.
func loopbackConfig(t *testing.T, dir, address string, signet map[string]any) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "server.json")
	raw, _ := json.Marshal(map[string]any{"listen": address, "public_url": "http://" + address, "state_dir": filepath.Join(dir, "state"), "owner": map[string]string{"name": "Synthetic owner"}, "auth": "loopback-trust", "local_scope": "read-write", "canon_refresh_minutes": 1, "signets": []any{signet}})
	if err := os.WriteFile(config, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return config
}

func oldReader(ctx context.Context, t *testing.T, old, current, format string) {
	e := env{t: t, ctx: ctx, home: t.TempDir()}
	base := t.TempDir()
	root := filepath.Join(base, "server")
	// Format 2 is reached the way a real signet reaches it: the released format
	// 1 fixture upgraded by 2.1.2 below. (The committed format 2 fixture's
	// upgrade record names Git history that isn't committed with it.)
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
	// Two 2.1.2 checkouts of the same remote: one behind it, one with an
	// unpushed local commit.
	behind, ahead := filepath.Join(base, "behind"), filepath.Join(base, "ahead")
	e.git(base, "clone", "-q", remote, behind)
	e.git(base, "clone", "-q", remote, ahead)
	aheadBinding := filepath.Join(base, "ahead-binding.json")
	e.ok("", old, "signet", "bind", "--repository", ahead, "--binding", aheadBinding, "--device-label", "Synthetic old clone", "--actor", "Synthetic actor")
	e.ok(`{"kind":"fact","summary":"Local only","body":"Written by 2.1.2 and not pushed","basis":"observation","reason":"Synthetic"}`, old, "memory", "remember", "--binding", aheadBinding)
	e.ok("", old, "memory", "checkpoint", "--binding", aheadBinding)
	unpushed := e.git(ahead, "rev-parse", "HEAD")
	if e.git(ahead, "rev-list", "--count", "origin/main..HEAD") != "1" {
		t.Fatal("fixture clone has no unpushed commit")
	}

	// This build names a topic and journals in it, then backs up to the remote.
	address := freeAddress(t)
	config := loopbackConfig(t, base, address, map[string]any{"slug": "primary", "root": root, "signet_id": manifest.ID, "device_id": device, "remote": "file://" + remote})
	topic := map[string]any{"kind": "project", "id": "copper-finch"}
	srv := e.serve(current, config, address)
	if ok, text := srv.call(t, "memory_name_scope", map[string]any{"request_id": "old-reader-name-0001", "label": map[string]any{"scope": topic, "name": "Copper Finch", "description": "Synthetic topic for the old-reader check.", "reason": "Synthetic"}}); !ok {
		t.Fatal(text)
	}
	if ok, text := srv.call(t, "memory_journal_append", map[string]any{"request_id": "old-reader-journal-01", "entry": map[string]any{"kind": "outcome", "summary": "SCOPED_ENTRY_MARKER", "scope": topic}}); !ok {
		t.Fatal(text)
	}
	if ok, text := srv.call(t, "memory_prime", map[string]any{"scope": topic}); !ok || !strings.Contains(text, "SCOPED_ENTRY_MARKER") {
		t.Fatal(text)
	}
	for deadline := time.Now().Add(15 * time.Second); !strings.Contains(e.git(base, "--git-dir="+remote, "ls-tree", "-r", "--name-only", "main"), "memory/scopes/"); time.Sleep(100 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("topic files never reached the remote")
		}
	}
	srv.stop()

	// The 2.1.2 server reads and serves memory without the names.
	before := files(t, root)
	srv = e.serve(old, config, address)
	list, err := srv.session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range list.Tools {
		if tool.Name == "memory_prime" || tool.Name == "memory_name_scope" {
			t.Fatalf("2.1.2 lists %s", tool.Name)
		}
	}
	reads := []struct {
		name   string
		args   map[string]any
		want   string
		absent string
	}{
		{"memory_recall", map[string]any{"scope": topic}, `"conflict_count":1`, "Synthetic topic for the old-reader check"},
		{"memory_scopes", map[string]any{"limit": 50}, "copper-finch", "Synthetic topic for the old-reader check"},
		{"memory_journal", map[string]any{"limit": 50}, "Synthetic journal entry", "SCOPED_ENTRY_MARKER"},
		{"memory_context", map[string]any{"event": "start"}, "copper-finch", "Synthetic topic for the old-reader check"},
		{"memory_context", map[string]any{"event": "prompt", "prompt": "What is the Copper Finch project codename?"}, "", "Synthetic topic for the old-reader check"},
	}
	for _, r := range reads {
		ok, text := srv.call(t, r.name, r.args)
		if !ok || !strings.Contains(text, r.want) || strings.Contains(text, r.absent) {
			t.Fatalf("2.1.2 %s: %s", r.name, text)
		}
	}
	ok, text := srv.call(t, "memory_journal_append", map[string]any{"request_id": "old-reader-stray-scope", "entry": map[string]any{"kind": "outcome", "summary": "s", "scope": topic}})
	if ok || !strings.Contains(text, `"input.invalid"`) {
		t.Fatalf("2.1.2 entry.scope: %s", text)
	}
	ok, text = srv.call(t, "memory_journal_append", map[string]any{"request_id": "old-reader-journal-01", "entry": map[string]any{"kind": "outcome", "summary": "SCOPED_ENTRY_MARKER"}})
	if ok || !strings.Contains(text, `"save.incomplete"`) {
		t.Fatalf("2.1.2 retry of a scoped request: %s", text)
	}
	srv.stop()
	sameFiles(t, "2.1.2 server", before, files(t, root))
	t.Logf("2.1.2 server: reads work without names; entry.scope is input.invalid; reused scoped request_id is save.incomplete; nothing written")

	// Inspect works; checkpoint refuses the new paths and changes nothing.
	if r := e.ok("", old, "memory", "inspect", "--binding", binding); r["healthy"] != true {
		t.Fatalf("2.1.2 inspect: %v", r)
	}
	status := e.git(root, "status", "--porcelain")
	out, done := e.run("", old, "memory", "checkpoint", "--binding", binding)
	if done || !strings.Contains(out, "ok\":false") {
		t.Fatalf("2.1.2 checkpoint: %s", out)
	}
	if e.git(root, "status", "--porcelain") != status {
		t.Fatal("2.1.2 checkpoint changed the worktree")
	}
	t.Logf("2.1.2 checkpoint: %s", strings.TrimSpace(out))

	// Export refuses; this build exports the same selection.
	preview := `{"binding_path":"` + binding + `","destination":"` + filepath.Join(t.TempDir(), "report") + `","selection":{"scopes":[{"kind":"project","id":"copper-finch"}]}}`
	if out, _ = e.run(preview, old, "export", "preview"); !strings.Contains(out, "ok\":false") {
		t.Fatalf("2.1.2 export preview: %s", out)
	}
	t.Logf("2.1.2 export preview: %s", strings.TrimSpace(out))
	e.ok(preview, current, "export", "preview")

	// Cutover preflight: 2.1.2 can't decode the new intents; this build can.
	if out, _ = e.run("", old, "server", "migrate", "preflight", "--clone", root, "--format", "json"); strings.Contains(out, "ok\":true") {
		t.Fatalf("2.1.2 preflight read the new intents: %s", out)
	}
	t.Logf("2.1.2 preflight: %s", strings.TrimSpace(out))
	if r := e.ok("", current, "server", "migrate", "preflight", "--clone", root, "--format", "json"); len(r["pending_intents"].([]any)) != 0 {
		t.Fatalf("current preflight: %v", r)
	}

	// Reconcile: the behind clone refuses the new paths; the ahead clone
	// refuses and keeps its unpushed commit. Neither changes its files.
	for _, clone := range []string{behind, ahead} {
		cfg := loopbackConfig(t, filepath.Join(base, filepath.Base(clone)+"-config"), freeAddress(t), map[string]any{"slug": "primary", "root": clone, "signet_id": manifest.ID, "device_id": device, "remote": "file://" + remote})
		head, status, tree := e.git(clone, "rev-parse", "HEAD"), e.git(clone, "status", "--porcelain"), files(t, clone)
		out, done := e.run("", old, "server", "reconcile", "--config", cfg, "--signet", "primary")
		if done {
			t.Fatalf("2.1.2 reconcile of %s: %s", filepath.Base(clone), out)
		}
		if e.git(clone, "rev-parse", "HEAD") != head || e.git(clone, "status", "--porcelain") != status {
			t.Fatalf("2.1.2 reconcile moved %s", filepath.Base(clone))
		}
		sameFiles(t, "2.1.2 reconcile", tree, files(t, clone))
		t.Logf("2.1.2 reconcile (%s): %s", filepath.Base(clone), strings.TrimSpace(out))
	}
	// The menu's sync on the ahead clone merges rather than fast-forwarding,
	// so it reaches the candidate path check.
	head, status, tree := e.git(ahead, "rev-parse", "HEAD"), e.git(ahead, "status", "--porcelain"), files(t, ahead)
	menu := exec.CommandContext(ctx, old, "menu", "--plain")
	menu.Env = e.vars("MANDALORE_BINDING=" + aheadBinding)
	menu.Stdin = strings.NewReader("4\n2\n7\n")
	menuOut, _ := menu.CombinedOutput()
	if !strings.Contains(string(menuOut), "Remote candidate could not be safely reconciled") || !strings.Contains(string(menuOut), "delivered:          false") {
		t.Fatalf("2.1.2 menu sync: %s", menuOut)
	}
	if e.git(ahead, "rev-parse", "HEAD") != head || head != unpushed || e.git(ahead, "status", "--porcelain") != status {
		t.Fatal("2.1.2 menu sync lost the unpushed commit or changed the index")
	}
	sameFiles(t, "2.1.2 menu sync", tree, files(t, ahead))
	t.Logf("2.1.2 menu sync kept the unpushed commit %s and refused at validate: Remote candidate could not be safely reconciled", unpushed[:12])

	oldCanon(e, old, base, manifest.ID, remote, func(srv served, foundling, head string) {
		ok, text := srv.call(t, "foundling_canon_recall", map[string]any{"foundling_id": foundling, "query": "codename", "scope": map[string]any{"kind": "project", "id": "copper-finch"}})
		if !ok || !strings.Contains(text, `"conflict_count":1`) || strings.Contains(text, "Synthetic topic for the old-reader check") {
			t.Fatalf("2.1.2 canon recall: %s", text)
		}
		t.Logf("2.1.2 canon refresh served commit %s with its records and without names", head[:12])
	})
}

// oldCanon serves a signet's remote over local HTTPS, has the old server
// refresh it as a canon source, and runs check against that server.
func oldCanon(e env, old, base, sourceID, remote string, check func(srv served, foundling, head string)) {
	t := e.t
	exec_, err := exec.Command("git", "--exec-path").Output()
	if err != nil {
		t.Fatal(err)
	}
	backend := &cgi.Handler{Path: filepath.Join(strings.TrimSpace(string(exec_)), "git-http-backend"), Env: []string{"GIT_PROJECT_ROOT=" + filepath.Dir(remote), "GIT_HTTP_EXPORT_ALL=1"}}
	https := httptest.NewTLSServer(backend)
	t.Cleanup(https.Close)
	locator := "https://canon.example.com/source.git"
	gitconfig := "[http]\n\tsslVerify = false\n[url \"" + https.URL + "/" + filepath.Base(remote) + "\"]\n\tinsteadOf = " + locator + "\n"
	if err := os.WriteFile(filepath.Join(e.home, ".gitconfig"), []byte(gitconfig), 0600); err != nil {
		t.Fatal(err)
	}
	primary := filepath.Join(base, "primary")
	e.ok("", old, "signet", "create", "--repository", primary, "--name", "Synthetic primary", "--device-label", "Synthetic primary")
	binding := filepath.Join(base, "primary-binding.json")
	e.ok("", old, "signet", "bind", "--repository", primary, "--binding", binding, "--device-label", "Synthetic primary CLI", "--actor", "Synthetic actor")
	head := e.git(base, "--git-dir="+remote, "rev-parse", "main")
	reg := e.ok(`{"mode":"canon","branch":"main","source_signet_id":"`+sourceID+`","name":"Topic canon","description":"Synthetic canon source with topic files","source":{"kind":"git","locator":"`+locator+`"},"pin":{"algorithm":"git-sha1","value":"`+head+`"},"reason":"Synthetic"}`, old, "foundling", "register", "--binding", binding)
	registration, _ := reg["registration"].(map[string]any)
	foundling, _ := registration["foundling_id"].(string)
	if foundling == "" {
		t.Fatalf("registration: %v", reg)
	}
	e.ok("", old, "memory", "git-init", "--binding", binding)
	primaryRemote := filepath.Join(base, "primary.git")
	e.git(base, "init", "-q", "--bare", "--initial-branch=main", primaryRemote)
	var manifest struct {
		ID string `json:"id"`
	}
	raw, _ := os.ReadFile(filepath.Join(primary, "signet.json"))
	_ = json.Unmarshal(raw, &manifest)
	devices, _ := os.ReadDir(filepath.Join(primary, "provenance/devices"))
	device := strings.TrimSuffix(devices[len(devices)-1].Name(), ".json")
	address := freeAddress(t)
	config := loopbackConfig(t, filepath.Join(base, "canon-config"), address, map[string]any{"slug": "primary", "root": primary, "signet_id": manifest.ID, "device_id": device, "remote": "file://" + primaryRemote, "canon": []string{foundling}})
	if out, done := e.run("", old, "server", "canon", "refresh", "--config", config); !done || !strings.Contains(out, `"ok":true`) {
		t.Fatalf("2.1.2 canon refresh: %s", out)
	}
	srv := e.serve(old, config, address)
	for deadline := time.Now().Add(20 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		_, text := srv.call(t, "foundling_status", map[string]any{})
		var v struct {
			Result struct {
				Items []struct {
					State  string `json:"state"`
					Reason string `json:"reason"`
					Commit string `json:"commit"`
				} `json:"items"`
			} `json:"result"`
		}
		_ = json.Unmarshal([]byte(text), &v)
		if len(v.Result.Items) == 1 && (v.Result.Items[0].State == "available" || v.Result.Items[0].Reason != "") {
			if v.Result.Items[0].State != "available" || v.Result.Items[0].Commit != head {
				t.Fatalf("2.1.2 canon refresh did not serve the topic commit: %s", text)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("2.1.2 canon never refreshed: %s", text)
		}
	}
	check(srv, foundling, head)
	srv.stop()
}
