package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/testfixture"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func failure(t *testing.T, text string) api.MemoryError {
	t.Helper()
	var v struct {
		OK    bool             `json:"ok"`
		Error *api.MemoryError `json:"error"`
	}
	if err := json.Unmarshal([]byte(text), &v); err != nil || v.OK || v.Error == nil {
		t.Fatalf("not a failure: %s", text)
	}
	return *v.Error
}

func signetFiles(t *testing.T, root string) int {
	t.Helper()
	n := 0
	for _, dir := range []string{"memory/records", "memory/sources", ".mandalore"} {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && d.Name() != ".gitkeep" && (dir != ".mandalore" || strings.HasPrefix(d.Name(), "operation-")) {
				n++
			}
			return nil
		})
	}
	return n
}

// #243: an over-limit field is the caller's to fix, not an ambiguous save.
func TestRememberInputLimitsOverMCP(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	session := connect(t, h, "/mcp", writeToken)
	for _, f := range []struct {
		field string
		limit int
	}{{"summary", 256}, {"body", 8192}, {"reason", 1024}} {
		args := rememberArgs("limit-request-"+f.field+"-01", "Use synthetic fixtures.")
		record := args["record"].(map[string]any)
		record[f.field] = strings.Repeat("x", f.limit+1)
		before := signetFiles(t, a.Root)
		r, text := call(t, session, "memory_remember", args)
		e := failure(t, text)
		want := f.field + " is " + strconv.Itoa(f.limit+1) + " bytes; the limit is " + strconv.Itoa(f.limit) + " UTF-8 bytes"
		if !r.IsError || e.Code != "input.invalid" || e.Retryable || e.WriteMayHaveOccurred || e.InspectBeforeRetry || !strings.Contains(e.Message, want) {
			t.Fatalf("%s over its limit: %s", f.field, text)
		}
		if signetFiles(t, a.Root) != before {
			t.Fatalf("%s over its limit wrote files", f.field)
		}
		// Nothing was stored, so the same request_id takes corrected input.
		record[f.field] = strings.Repeat("x", f.limit)
		_, text = call(t, session, "memory_remember", args)
		receipt(t, text)
	}
	// The issue's case: 363 characters of ASCII.
	args := rememberArgs("limit-request-issue-01", "Use synthetic fixtures.")
	args["record"].(map[string]any)["summary"] = strings.Repeat("s", 363)
	if e := failure(t, mustText(t, session, args)); e.Code != "input.invalid" || !strings.Contains(e.Message, "summary is 363 bytes; the limit is 256") {
		t.Fatal(e)
	}
	// 129 characters is under maxLength 256 but 258 bytes.
	args = rememberArgs("limit-request-multibyte", "Use synthetic fixtures.")
	args["record"].(map[string]any)["summary"] = strings.Repeat("é", 129)
	if e := failure(t, mustText(t, session, args)); e.Code != "input.invalid" || !strings.Contains(e.Message, "summary is 258 bytes") {
		t.Fatal(e)
	}
	args["record"].(map[string]any)["summary"] = strings.Repeat("é", 128)
	receipt(t, mustText(t, session, args))
	// Other pre-write validation, without echoing input text.
	args = rememberArgs("limit-request-kind-0001", "PRIVATE_BODY_CANARY")
	args["record"].(map[string]any)["kind"] = "unsupported"
	if text := mustText(t, session, args); failure(t, text).Code != "input.invalid" || strings.Contains(text, "PRIVATE_BODY_CANARY") {
		t.Fatal(text)
	}
	args = rememberArgs("limit-request-basis-001", "Use synthetic fixtures.")
	args["record"].(map[string]any)["basis"] = " "
	if e := failure(t, mustText(t, session, args)); e.Code != "input.invalid" || e.WriteMayHaveOccurred || !strings.Contains(e.Message, "basis") {
		t.Fatal(e)
	}
	journal := map[string]any{"request_id": "limit-journal-request-01", "entry": map[string]any{"kind": "outcome", "summary": strings.Repeat("j", 4097)}}
	_, text := call(t, session, "memory_journal_append", journal)
	if e := failure(t, text); e.Code != "input.invalid" || e.WriteMayHaveOccurred || !strings.Contains(e.Message, "summary is 4097 bytes; the limit is 4096") {
		t.Fatal(e)
	}
}

// A failure after the intent is written stays save.incomplete, and the same
// request_id completes it.
func TestRememberMidWriteFailureStaysIncomplete(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	session := connect(t, h, "/mcp", writeToken)
	sources := filepath.Join(a.Root, "memory/sources")
	if err := os.Chmod(sources, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sources, 0700) })
	args := rememberArgs("midwrite-request-0001", "Use synthetic fixtures.")
	e := failure(t, mustText(t, session, args))
	if e.Code != "save.incomplete" || !e.WriteMayHaveOccurred || !e.InspectBeforeRetry || e.Retryable {
		t.Fatal(e)
	}
	if err := os.Chmod(sources, 0700); err != nil {
		t.Fatal(err)
	}
	receipt(t, mustText(t, session, args))
}

func TestWriteSchemasStateByteLimits(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	tools, err := connect(t, h, "/mcp", writeToken).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]map[string]float64{
		"memory_remember":       {"record.summary": 256, "record.body": 8192, "record.reason": 1024},
		"memory_journal_append": {"entry.kind": 64, "entry.summary": 4096},
	}
	for _, tool := range tools.Tools {
		b, _ := json.Marshal(tool.InputSchema)
		var schema map[string]any
		if err := json.Unmarshal(b, &schema); err != nil {
			t.Fatal(err)
		}
		for path, limit := range want[tool.Name] {
			object, field, _ := strings.Cut(path, ".")
			p := schema["properties"].(map[string]any)[object].(map[string]any)["properties"].(map[string]any)[field].(map[string]any)
			if p["maxLength"] != limit || p["minLength"] != float64(1) || !strings.Contains(p["description"].(string), "UTF-8 bytes") || !strings.Contains(p["description"].(string), "maxLength counts characters") {
				t.Fatalf("%s %s: %v", tool.Name, path, p)
			}
		}
		delete(want, tool.Name)
	}
	if len(want) != 0 {
		t.Fatalf("tools missing: %v", want)
	}
}

func mustText(t *testing.T, session *sdk.ClientSession, args map[string]any) string {
	t.Helper()
	_, text := call(t, session, "memory_remember", args)
	return text
}
