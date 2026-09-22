package claudecode

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func fixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	root, path := filepath.Join(dir, "signet"), filepath.Join(dir, "binding.json")
	if _, err := memory.Create(root, "Synthetic", "device-bootstrap", "Fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := binding.Bind(root, path, "Fixture", "Test user"); err != nil {
		t.Fatal(err)
	}
	s, err := binding.Open(path, "test")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []memory.Write{
		{Kind: "preference", Summary: "Answer style", Body: "Prefer concise answers.", Basis: "user-direction", Reason: "User requested"},
		{Kind: "fact", Summary: "Project name", Body: "PRIVATE_PROJECT_CANARY", Scope: &memory.Scope{Kind: "project", ID: "copper-finch"}, Basis: "user-direction", Reason: "User confirmed"},
	} {
		if _, err := s.Remember(w); err != nil {
			t.Fatal(err)
		}
	}
	return root, path
}

func snapshot(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	result := map[string][32]byte{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[path] = sha256.Sum256(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func invoke(t *testing.T, path, event string) map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := Run(path, testGuard(t, path), strings.NewReader(event), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() > 9501 {
		t.Fatalf("unbounded hook output: %d", out.Len())
	}
	var v map[string]any
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestHooksAreReadOnlyScopedAndSupportNativeEvolution(t *testing.T) {
	root, path := fixture(t)
	before := snapshot(t, filepath.Dir(root))
	start := invoke(t, path, `{"hook_event_name":"SessionStart","source":"compact","transcript_path":"/not-to-be-read","future_field":{"a":true}}`)
	context := start["hookSpecificOutput"].(map[string]any)
	if context["hookEventName"] != "SessionStart" || !strings.Contains(context["additionalContext"].(string), "this-is-the-way") {
		t.Fatal(start)
	}
	for _, prompt := range []string{"concise answers", "this is the way", "Read-only: do not save or sync"} {
		b, _ := json.Marshal(map[string]string{"hook_event_name": "UserPromptSubmit", "prompt": prompt})
		v := invoke(t, path, string(b))
		encoded, _ := json.Marshal(v)
		if strings.Contains(string(encoded), "PRIVATE_PROJECT_CANARY") {
			t.Fatal("unrelated project evidence leaked")
		}
		if !strings.Contains(string(encoded), "copper-finch") {
			t.Fatal("scope routing missing")
		}
		if prompt == "concise answers" && !strings.Contains(string(encoded), "Prefer concise answers") {
			t.Fatal("bank-wide recall missing")
		}
	}
	if !reflect.DeepEqual(before, snapshot(t, filepath.Dir(root))) {
		t.Fatal("hook changed files")
	}
}

func TestMalformedEventsWarnWithoutEchoOrBindingAccess(t *testing.T) {
	for _, event := range []string{
		`{"hook_event_name":"SessionStart"} trailing SECRET_CANARY`,
		`{"hook_event_name":"SessionStart","hook_event_name":"Stop"}`,
		`{"hook_event_name":"UserPromptSubmit","prompt":false}`,
		`{"hook_event_name":"UserPromptSubmit","prompt":null}`,
		`{"hook_event_name":"UserPromptSubmit","prompt":"a","prompt":"b"}`,
		`[]`, `null`, `{`, strings.Repeat("SECRET_CANARY", 6000),
	} {
		v := invoke(t, "/does-not-exist", event)
		b, _ := json.Marshal(v)
		if v["systemMessage"] == nil || strings.Contains(string(b), "SECRET_CANARY") || v["hookSpecificOutput"] != nil {
			t.Fatalf("unsafe response: %s", b)
		}
	}
	if v := invoke(t, "/does-not-exist", `{"hook_event_name":"Stop"}`); len(v) != 0 {
		t.Fatal(v)
	}
}

func TestUnavailableBindingWarnsAndLongUnicodeFitsBudget(t *testing.T) {
	v := invoke(t, "/does-not-exist", `{"hook_event_name":"SessionStart"}`)
	if v["systemMessage"] == nil {
		t.Fatal(v)
	}
	_, path := fixture(t)
	e, _ := json.Marshal(map[string]string{"hook_event_name": "UserPromptSubmit", "prompt": strings.Repeat("界", 4000)})
	if v := invoke(t, path, string(e)); v["hookSpecificOutput"] == nil {
		t.Fatal(v)
	}
}

func TestBoundIdentityMismatchProducesWarningWithoutEvidence(t *testing.T) {
	_, path := fixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var b binding.Binding
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	// Root and authorship remain valid, so the identity pin itself must reject.
	b.SignetID = "signet-different"
	data, err = json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, filepath.Dir(path))
	v := invoke(t, path, `{"hook_event_name":"UserPromptSubmit","prompt":"answers"}`)
	if v["systemMessage"] == nil || v["hookSpecificOutput"] != nil {
		t.Fatal("foreign signet accepted", v)
	}
	if !reflect.DeepEqual(before, snapshot(t, filepath.Dir(path))) {
		t.Fatal("warning changed files")
	}
}

func testGuard(t *testing.T, path string) binding.Guard {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		return binding.Guard{SHA256: strings.Repeat("0", 64), SignetID: "invalid"}
	}
	var b binding.Binding
	_ = json.Unmarshal(raw, &b)
	h := sha256.Sum256(raw)
	return binding.Guard{SHA256: hex.EncodeToString(h[:]), SignetID: b.SignetID}
}

func TestHookRejectsBindingRetargetAndMissingGuard(t *testing.T) {
	root, path := fixture(t)
	guard := testGuard(t, path)
	before := snapshot(t, root)
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(raw, ' '), 0600); err != nil {
		t.Fatal(err)
	}
	for _, g := range []binding.Guard{guard, {}} {
		var out bytes.Buffer
		if err := Run(path, g, strings.NewReader(`{"hook_event_name":"UserPromptSubmit","prompt":"Answer style"}`), &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "systemMessage") || strings.Contains(out.String(), "additionalContext") {
			t.Fatal("changed selection leaked evidence", out.String())
		}
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("hook modified signet")
	}
}
