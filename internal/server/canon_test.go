package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// canonSource is a synthetic canon signet C published to its bare file://
// remote. Only the test publishes to that remote; cleanup checks that the
// server never moved its refs.
type canonSource struct {
	testfixture.ServerSignet
	memory *memory.Service
	refs   string
}

func newCanonSource(t *testing.T) *canonSource {
	t.Helper()
	c := &canonSource{ServerSignet: testfixture.NewServerSignet(t, "canon")}
	var err error
	if c.memory, err = memory.OpenService(c.Root, memory.Authorship{DeviceID: c.DeviceID, Actor: "Synthetic Canon Owner", Harness: "test"}); err != nil {
		t.Fatal(err)
	}
	c.publish(t, "Canon convention: synthetic fixtures only.")
	t.Cleanup(func() {
		if now := refsOf(t, c.RemoteDir); now != c.refs {
			t.Errorf("canon remote refs changed by the server:\n%s\nwant\n%s", now, c.refs)
		}
	})
	return c
}

func refsOf(t *testing.T, bare string) string {
	t.Helper()
	return testfixture.Git(t, bare, "for-each-ref", "--format=%(refname) %(objectname)")
}

// publish saves one record in C, commits and pushes it, and returns the commit.
func (c *canonSource) publish(t *testing.T, body string) string {
	t.Helper()
	if _, err := c.memory.Remember(memory.Write{Kind: "fact", Summary: "Canon convention", Body: body, Basis: "user-direction", Reason: "Synthetic canon fixture"}); err != nil {
		t.Fatal(err)
	}
	testfixture.Git(t, c.Root, "add", "-A")
	testfixture.Git(t, c.Root, "commit", "-m", "Publish canon")
	testfixture.Git(t, c.Root, "push", c.Remote, "main")
	c.refs = refsOf(t, c.RemoteDir)
	return testfixture.Git(t, c.Root, "rev-parse", "HEAD")
}

// registerCanon registers C as a canon reference in P and routes its portable
// locator to C's file:// remote.
func registerCanon(t *testing.T, p testfixture.ServerSignet, c *canonSource) string {
	t.Helper()
	primary, err := memory.OpenService(p.Root, memory.Authorship{DeviceID: p.DeviceID, Actor: "Synthetic Owner", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	head := testfixture.Git(t, c.Root, "rev-parse", "HEAD")
	reg, err := primary.WriteFoundling(memory.FoundlingWrite{Name: "Synthetic canon", Description: "Reference signet", Mode: "canon", Branch: "main", SourceSignetID: c.SignetID,
		Source: memory.FoundlingSource{Kind: "git", Locator: "https://git.example.com/synthetic/canon.git"}, Pin: memory.SourcePin{Algorithm: "git-sha1", Value: head}, State: "active", Reason: "Synthetic canon registration"})
	if err != nil {
		t.Fatal(err)
	}
	saved := canonTransport
	canonTransport = func(string) string { return c.Remote }
	t.Cleanup(func() { canonTransport = saved })
	return reg.FoundlingID
}

func canonConfig(t *testing.T) (Config, testfixture.ServerSignet, *canonSource, string) {
	t.Helper()
	p := testfixture.NewServerSignet(t, "primary")
	c := newCanonSource(t)
	id := registerCanon(t, p, c)
	config := testConfig(t, p)
	config.Signets[0].Canon = []string{id}
	return config, p, c, id
}

func startCanonServer(t *testing.T, c Config, every time.Duration) (*Server, *httptest.Server) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t2 := fastTiming
	t2.canon = every
	s, err := newServer(c, &syncBuffer{}, t2)
	if err != nil {
		t.Fatal(err)
	}
	s.Start(ctx)
	h := httptest.NewServer(s.Handler())
	t.Cleanup(h.Close)
	return s, h
}

type canonRecall struct {
	OK     bool `json:"ok"`
	Result struct {
		Reference foundlings.CanonReceipt `json:"reference"`
		Memory    memory.RecallPacket     `json:"memory"`
	} `json:"result"`
}

func recallCanon(t *testing.T, session *sdk.ClientSession, id string) canonRecall {
	t.Helper()
	_, text := call(t, session, "foundling_canon_recall", map[string]any{"foundling_id": id, "query": "canon convention"})
	var v canonRecall
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		t.Fatal(text, err)
	}
	return v
}

func recalls(v canonRecall, body string) bool {
	for _, r := range v.Result.Memory.Current {
		if r.Body == body {
			return true
		}
	}
	return false
}

// Acceptance 1: C registered as canon in P is recallable through P's
// foundling_canon_recall, carrying C's commit, for read and write clients.
func TestCanonIsRecallableThroughPrimary(t *testing.T) {
	config, _, c, id := canonConfig(t)
	_, url := startCanonServer(t, config, time.Hour)
	head := testfixture.Git(t, c.Root, "rev-parse", "HEAD")
	for _, token := range []string{readToken, writeToken} {
		session := connect(t, url, "/mcp", token)
		var got canonRecall
		eventually(t, 10*time.Second, "startup refresh serves canon", func() bool {
			got = recallCanon(t, session, id)
			return got.OK
		})
		ref := got.Result.Reference
		if ref.State != "available" || ref.Commit != head || ref.ObservedAt == "" || ref.AgeSeconds == nil || !recalls(got, "Canon convention: synthetic fixtures only.") {
			t.Fatalf("canon recall: %+v", got)
		}
		// session_id is optional and ignored.
		_, text := call(t, session, "foundling_canon_recall", map[string]any{"session_id": "any-1x-session", "foundling_id": id, "query": "canon"})
		if !strings.Contains(text, `"ok":true`) {
			t.Fatal(text)
		}
	}
}

// Acceptance 2: a new record pushed to C is served within one refresh
// interval, or at once after `server canon refresh`.
func TestCanonRefreshesOnIntervalAndOnDemand(t *testing.T) {
	t.Run("interval", func(t *testing.T) {
		config, _, c, id := canonConfig(t)
		_, url := startCanonServer(t, config, 200*time.Millisecond)
		session := connect(t, url, "/mcp", readToken)
		eventually(t, 10*time.Second, "startup refresh", func() bool { return recallCanon(t, session, id).OK })
		head := c.publish(t, "Canon convention: refreshed on the interval.")
		eventually(t, 10*time.Second, "interval refresh serves the new record", func() bool {
			got := recallCanon(t, session, id)
			return got.Result.Reference.Commit == head && recalls(got, "Canon convention: refreshed on the interval.")
		})
	})
	t.Run("on demand", func(t *testing.T) {
		config, _, c, id := canonConfig(t)
		_, url := startCanonServer(t, config, time.Hour)
		session := connect(t, url, "/mcp", readToken)
		eventually(t, 10*time.Second, "startup refresh", func() bool { return recallCanon(t, session, id).OK })
		head := c.publish(t, "Canon convention: refreshed on demand.")
		if got := recallCanon(t, session, id); got.Result.Reference.Commit == head {
			t.Fatal("read fetched")
		}
		refreshed, err := RefreshCanon(context.Background(), config, "a")
		if err != nil || len(refreshed) != 1 || refreshed[0].Canon[0].State != "available" || refreshed[0].Canon[0].Commit != head {
			t.Fatalf("refresh: %+v %v", refreshed, err)
		}
		if got := recallCanon(t, session, id); got.Result.Reference.Commit != head || !recalls(got, "Canon convention: refreshed on demand.") {
			t.Fatalf("not served after refresh: %+v", got)
		}
		if _, err := RefreshCanon(context.Background(), config, "missing"); err == nil {
			t.Fatal("unknown signet refreshed")
		}
	})
}

// Acceptance 3: with C's remote unavailable, P keeps serving the last
// snapshot, reported stale with its age and error in reads and server status.
func TestCanonUnavailableRemoteServesStaleSnapshot(t *testing.T) {
	config, _, c, id := canonConfig(t)
	_, url := startCanonServer(t, config, time.Hour)
	session := connect(t, url, "/mcp", readToken)
	eventually(t, 10*time.Second, "startup refresh", func() bool { return recallCanon(t, session, id).OK })
	head := testfixture.Git(t, c.Root, "rev-parse", "HEAD")
	moved := c.RemoteDir + ".offline"
	if err := os.Rename(c.RemoteDir, moved); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Rename(moved, c.RemoteDir); err != nil {
			t.Fatal(err)
		}
	}()
	refreshed, err := RefreshCanon(context.Background(), config, "")
	if err != nil || refreshed[0].Canon[0].State != "stale" || refreshed[0].Canon[0].Reason != "transport_unavailable" {
		t.Fatalf("refresh: %+v %v", refreshed, err)
	}
	got := recallCanon(t, session, id)
	if !got.OK || got.Result.Reference.State != "stale" || got.Result.Reference.Commit != head || got.Result.Reference.AgeSeconds == nil || !recalls(got, "Canon convention: synthetic fixtures only.") {
		t.Fatalf("stale recall: %+v", got)
	}
	statuses, err := Status(context.Background(), config)
	if err != nil || len(statuses[0].Canon) != 1 {
		t.Fatalf("status: %+v %v", statuses, err)
	}
	if st := statuses[0].Canon[0]; st.FoundlingID != id || st.State != "stale" || st.Commit != head || st.AgeSeconds == nil || st.Reason != "transport_unavailable" || st.CheckedAt == "" {
		t.Fatalf("canon status: %+v", st)
	}
}

// Acceptance 5: a read client is refused promotion; a read-write client's
// promotion records its device.
func TestCanonPromotionNeedsWriteScopeAndRecordsTheClient(t *testing.T) {
	config, p, _, id := canonConfig(t)
	s, url := startCanonServer(t, config, time.Hour)
	reader, writer := connect(t, url, "/mcp", readToken), connect(t, url, "/mcp", writeToken)
	var got canonRecall
	eventually(t, 10*time.Second, "startup refresh", func() bool { got = recallCanon(t, reader, id); return got.OK })
	source := got.Result.Memory.Current[0]
	_, text := call(t, reader, "foundling_canon_heads", map[string]any{"foundling_id": id, "record_id": source.RecordID})
	var heads struct {
		Result struct {
			Reference foundlings.CanonReceipt `json:"reference"`
			Heads     []struct {
				ContentSHA256 string `json:"content_sha256"`
			} `json:"heads"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(text), &heads); err != nil || len(heads.Result.Heads) != 1 {
		t.Fatal(text, err)
	}
	promote := map[string]any{"foundling_id": id, "registration_revision_id": heads.Result.Reference.RegistrationID, "record_id": source.RecordID, "revision_id": source.ID, "content_sha256": heads.Result.Heads[0].ContentSHA256,
		"write": map[string]any{"kind": "fact", "summary": "Adopted canon convention", "body": "Adopted: synthetic fixtures only.", "basis": "import", "reason": "Explicit synthetic promotion"}}
	if r, err := reader.CallTool(context.Background(), &sdk.CallToolParams{Name: "foundling_canon_promote", Arguments: promote}); err == nil && !r.IsError {
		t.Fatal("read client promoted")
	}
	raw, _ := json.Marshal(promote)
	if res := s.signets["/mcp"].call(context.Background(), Client{ID: "r1", Kind: "claude.ai", Scope: ScopeRead}, "foundling_canon_promote", raw); !res.IsError {
		t.Fatal("read client promotion accepted by call path")
	}
	_, text = call(t, writer, "foundling_canon_promote", promote)
	var promoted struct {
		OK     bool `json:"ok"`
		Result struct {
			ID       string `json:"id"`
			RecordID string `json:"record_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(text), &promoted); err != nil || !promoted.OK {
		t.Fatal(text, err)
	}
	var devices map[string]string
	if err := readState(filepath.Join(config.StateDir, "signets", "a", "devices.json"), &devices); err != nil || devices["c1"] == "" {
		t.Fatalf("c1 not enrolled: %v %v", devices, err)
	}
	var r memory.Revision
	readJSON(t, filepath.Join(p.Root, "memory/records", promoted.Result.RecordID, promoted.Result.ID+".json"), &r)
	if want := (memory.Authorship{DeviceID: devices["c1"], Actor: "Synthetic Owner", Harness: "claude-code"}); r.Authorship != want || r.Body != "Adopted: synthetic fixtures only." {
		t.Fatalf("promotion authorship %+v", r.Authorship)
	}
}

// Acceptance 6: legacy foundlings read from the configured root, and their
// promotion records the client too.
func TestLegacyFoundlingsReadFromConfiguredRoots(t *testing.T) {
	p := testfixture.NewServerSignet(t, "primary")
	history := t.TempDir()
	if err := os.MkdirAll(filepath.Join(history, "decisions"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(history, "decisions", "old.md"), []byte("The historical project name was Copper Finch.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	source := memory.FoundlingSource{Kind: "local", Locator: "synthetic-history"}
	view, err := foundlings.Observe(context.Background(), source, history)
	if err != nil {
		t.Fatal(err)
	}
	primary, err := memory.OpenService(p.Root, memory.Authorship{DeviceID: p.DeviceID, Actor: "Synthetic Owner", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	reg, err := primary.WriteFoundling(memory.FoundlingWrite{Name: "Synthetic history", Description: "Pinned notes", Source: source, Pin: view.Pin, State: "active", Reason: "Synthetic legacy registration"})
	if err != nil {
		t.Fatal(err)
	}
	config := testConfig(t, p)
	config.Signets[0].Legacy = map[string]string{reg.FoundlingID: history}
	_, url := startCanonServer(t, config, time.Hour)
	reader, writer := connect(t, url, "/mcp", readToken), connect(t, url, "/mcp", writeToken)
	_, text := call(t, reader, "foundling_search", map[string]any{"foundling_id": reg.FoundlingID, "query": "Copper Finch"})
	var found struct {
		OK     bool `json:"ok"`
		Result struct {
			Items []foundlings.Excerpt `json:"items"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(text), &found); err != nil || !found.OK || len(found.Result.Items) != 1 {
		t.Fatal(text, err)
	}
	origin := found.Result.Items[0].Origin
	_, text = call(t, writer, "foundling_promote", map[string]any{"foundling_id": reg.FoundlingID, "registration_revision_id": reg.ID, "relative_locator": origin.RelativeLocator, "content_sha256": origin.ContentSHA256,
		"write": map[string]any{"kind": "fact", "summary": "Project naming", "body": "Copper Finch was the historical name.", "basis": "import", "reason": "Adapt historical evidence"}})
	if !strings.Contains(text, `"ok":true`) {
		t.Fatal(text)
	}
	if _, err := os.Stat(filepath.Join(p.Root, ".mandalore/foundlings")); !os.IsNotExist(err) {
		t.Fatal("server wrote a local connection", err)
	}
}

// Foundling tools appear only for the references a signet configures.
func TestFoundlingToolsFollowConfiguration(t *testing.T) {
	names := func(url *httptest.Server, token string) map[string]bool {
		list, err := connect(t, url, "/mcp", token).ListTools(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]bool{}
		for _, tool := range list.Tools {
			out[tool.Name] = true
		}
		return out
	}
	_, plain := startCanonServer(t, testConfig(t, testfixture.NewServerSignet(t, "plain")), time.Hour)
	for name := range names(plain, writeToken) {
		if strings.HasPrefix(name, "foundling_") {
			t.Fatalf("unconfigured signet offers %s", name)
		}
	}
	config, _, _, _ := canonConfig(t)
	_, url := startCanonServer(t, config, time.Hour)
	read, write := names(url, readToken), names(url, writeToken)
	for _, name := range []string{"foundling_list", "foundling_status", "foundling_canon_scopes", "foundling_canon_recall", "foundling_canon_heads"} {
		if !read[name] || !write[name] {
			t.Fatalf("canon read tool %s missing", name)
		}
	}
	if read["foundling_canon_promote"] || !write["foundling_canon_promote"] || write["foundling_search"] || write["foundling_promote"] || write["foundling_refresh"] {
		t.Fatalf("unexpected foundling tools: %v", write)
	}
}
