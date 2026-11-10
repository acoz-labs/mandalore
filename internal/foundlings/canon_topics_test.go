package foundlings

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

// #246: a signet with names and scoped journal entries can be a canon source.
func TestCanonServesSourceWithTopicFiles(t *testing.T) {
	m, source, reg, rev, root := canonFixture(t)
	topic := memory.Scope{Kind: "project", ID: "atlas-launch"}
	label, err := source.NameScope(memory.ScopeLabelWrite{Scope: topic, Name: "Atlas launch", Description: "Synthetic launch topic.", Reason: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.AppendJournal("outcome", "Atlas checklist drafted", &topic); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "Name a topic")
	receipt := refreshTest(t, m)
	if !strings.HasPrefix(gitTest(t, root, "rev-parse", "HEAD"), receipt.Pin.Value) {
		t.Fatal("refresh did not advance to the topic commit")
	}
	if got := recallTest(t, m, reg.FoundlingID); len(got.Memory.Current) != 1 || got.Memory.Current[0].ID != rev.ID {
		t.Fatalf("records not served: %+v", got.Memory)
	}
	status, err := m.CanonStatus(context.Background())
	if err != nil || status[0].State != "available" {
		t.Fatal(status, err)
	}
	cache, _ := m.canonRoot(false)
	snapshot := filepath.Join(cache, "commit-"+receipt.Pin.Value+"-"+receipt.Digest)
	if _, err := os.Stat(filepath.Join(snapshot, "memory/scopes/project/atlas-launch/labels", label.ID+".json")); err != nil {
		t.Fatal("verified snapshot lacks the label:", err)
	}
	// A malformed label makes the snapshot invalid, like any bad evidence.
	bad := filepath.Join(root, "memory/scopes/project/atlas-launch/labels", memory.NewID("label")+".json")
	if err := os.WriteFile(bad, []byte(`{"schema_version":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "Malformed label")
	r, err := m.Refresh(context.Background())
	if err != nil || r[0].State != "unavailable" {
		t.Fatal(r, err)
	}
}
