package retention

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func fixture(t *testing.T) (Request, *memory.Service, memory.Revision, memory.JournalEntry) {
	t.Helper()
	s, err := memory.Create(filepath.Join(t.TempDir(), "bank"), "PRIVATE_NAME", "device-test", "PRIVATE_DEVICE")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "binding.json")
	if _, err := binding.Bind(s.Root, path, "PRIVATE_DEVICE", "PRIVATE_ACTOR"); err != nil {
		t.Fatal(err)
	}
	service, err := binding.Open(path, "test")
	if err != nil {
		t.Fatal(err)
	}
	r, err := service.Remember(memory.Write{Kind: "fact", Summary: "PRIVATE_SUMMARY", Body: "PRIVATE_BODY", Basis: "observation", Reason: "PRIVATE_REASON"})
	if err != nil {
		t.Fatal(err)
	}
	j, err := service.AppendJournal("test", "PRIVATE_JOURNAL")
	if err != nil {
		t.Fatal(err)
	}
	return Request{BindingPath: path, Selection: Selection{RecordIDs: []string{r.RecordID}}, Policy: Policy{ID: "policy-review", Visibility: "any"}}, service, r, j
}

func TestPreviewIsPinnedMetadataOnlyAndLeavesSourceUnchanged(t *testing.T) {
	in, s, r, j := fixture(t)
	before := map[string][]byte{}
	if err := filepath.WalkDir(s.Root(), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		before[path] = b
		return err
	}); err != nil {
		t.Fatal(err)
	}
	p, err := Preview(context.Background(), in)
	if err != nil || len(p.Records) != 1 || p.Records[0].RecordID != r.RecordID || !p.Records[0].Matched || len(p.Journals) != 0 || len(p.BindingSHA256) != 64 || len(p.SourceSHA256) != 64 || len(p.PolicySHA256) != 64 {
		t.Fatal(p, err)
	}
	b, _ := json.Marshal(p)
	if strings.Contains(string(b), "PRIVATE_") {
		t.Fatal("preview exposed content/authorship/name")
	}
	again, err := Preview(context.Background(), in)
	if err != nil || !reflect.DeepEqual(p, again) {
		t.Fatal("unstable read-only preview", err)
	}
	after := map[string][]byte{}
	if err := filepath.WalkDir(s.Root(), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		after[path] = b
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("preview wrote source")
	}
	in.Selection.JournalIDs = []string{j.ID}
	p, err = Preview(context.Background(), in)
	if err != nil || len(p.Journals) != 1 || p.Journals[0].ID != j.ID {
		t.Fatal("explicit journal selection lost", p, err)
	}
}

func TestAgePolicyUsesExplicitFieldAndStrictCutoff(t *testing.T) {
	in, _, r, j := fixture(t)
	in.Policy.Age = &AgePolicy{Timestamp: "recorded_at", Before: r.RecordedAt}
	p, err := Preview(context.Background(), in)
	if err != nil || p.Records[0].Matched {
		t.Fatal("inclusive cutoff or implicit age", p, err)
	}
	at, _ := time.Parse(time.RFC3339Nano, r.RecordedAt)
	in.Policy.Age.Before = at.Add(time.Nanosecond).Format(time.RFC3339Nano)
	p, err = Preview(context.Background(), in)
	if err != nil || !p.Records[0].Matched {
		t.Fatal(p, err)
	}
	in.Policy.Age.Timestamp = "last_verified_at"
	in.Selection.JournalIDs = []string{j.ID}
	p, err = Preview(context.Background(), in)
	if err != nil || p.Records[0].Matched || p.Records[0].Reason != "timestamp-missing" || p.Journals[0].Matched || p.Journals[0].Reason != "timestamp-not-applicable" {
		t.Fatal(p, err)
	}
}

func TestPreviewRejectsImplicitUnknownInvalidAndCanceledSelections(t *testing.T) {
	in, _, _, _ := fixture(t)
	for _, change := range []func(*Request){
		func(r *Request) { r.Selection = Selection{} },
		func(r *Request) { r.Policy = Policy{} },
		func(r *Request) { r.Selection.RecordIDs = []string{"record-missing"} },
		func(r *Request) { r.Policy.Age = &AgePolicy{Timestamp: "recorded_at", Before: "yesterday"} },
		func(r *Request) { r.Policy.Age = &AgePolicy{Timestamp: "unknown", Before: "2026-01-01T00:00:00Z"} },
	} {
		bad := in
		change(&bad)
		if p, err := Preview(context.Background(), bad); err == nil || p.SignetID != "" {
			t.Fatal("invalid preview returned data", p, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Preview(ctx, in); err != context.Canceled {
		t.Fatal(err)
	}
}

func TestPreviewRefusesOversizedResultRatherThanTruncating(t *testing.T) {
	in, service, r, _ := fixture(t)
	for i := 0; i < 400; i++ {
		next := r
		next.ID = fmt.Sprintf("revision-%s-%03d", strings.Repeat("a", 48), i)
		next.Supersedes = []string{r.ID}
		data, _ := json.Marshal(next)
		if err := os.WriteFile(filepath.Join(service.Root(), "memory/records", r.RecordID, next.ID+".json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if p, err := Preview(context.Background(), in); err == nil || len(p.Records) != 0 || p.SignetID != "" {
		t.Fatal("oversized preview returned partial plan", err)
	}
}

func TestPreviewPinsChangesAndRejectsSymlinkBinding(t *testing.T) {
	in, service, _, _ := fixture(t)
	p, err := Preview(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(in.BindingPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(in.BindingPath, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	q, err := Preview(context.Background(), in)
	if err != nil || p.BindingSHA256 == q.BindingSHA256 || p.SourceSHA256 != q.SourceSHA256 {
		t.Fatal("binding pin missing", err)
	}
	in.Policy.Visibility = "withheld"
	next, err := Preview(context.Background(), in)
	if err != nil || next.PolicySHA256 == q.PolicySHA256 || next.Records[0].Matched {
		t.Fatal("policy pin/filter missing", err)
	}
	if _, err := service.Remember(memory.Write{Kind: "fact", Summary: "Another record", Body: "PRIVATE_NEW_BODY", Basis: "observation", Reason: "Test"}); err != nil {
		t.Fatal(err)
	}
	changed, err := Preview(context.Background(), in)
	if err != nil || changed.SourceSHA256 == next.SourceSHA256 || changed.BindingSHA256 != next.BindingSHA256 || len(changed.Records) != 1 {
		t.Fatal("source pin/selection did not track fresh data", err)
	}
	alias := filepath.Join(t.TempDir(), "alias.json")
	if err := os.Symlink(in.BindingPath, alias); err != nil {
		t.Fatal(err)
	}
	in.BindingPath = alias
	if _, err := Preview(context.Background(), in); err == nil {
		t.Fatal("symlink binding accepted")
	}
}
