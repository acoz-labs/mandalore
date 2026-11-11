package exportreport

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

// #250: a report reads moves, selects a record by its current scope and
// reports the move as history.
func TestReportFollowsMoves(t *testing.T) {
	store, err := memory.Create(filepath.Join(t.TempDir(), "bank"), "Synthetic bank", "device-test", "Synthetic device")
	if err != nil {
		t.Fatal(err)
	}
	s, err := memory.OpenService(store.Root, memory.Authorship{DeviceID: "device-test", Actor: "Synthetic actor", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	from, to := memory.Scope{Kind: "project", ID: "example-x"}, memory.Scope{Kind: "project", ID: "example-y"}
	r, err := s.Remember(memory.Write{Kind: "fact", Scope: &from, Summary: "Misfiled", Body: "MOVED_MARKER", Basis: "observation", Reason: "Synthetic"})
	if err != nil {
		t.Fatal(err)
	}
	m, err := s.Move(memory.MoveWrite{RecordID: r.RecordID, Scope: to, ContentHeads: []string{r.ID}, MoveHeads: []string{}, Reason: "MOVE_REASON"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := ReadReportSnapshot(context.Background(), s.Root())
	if err != nil || len(snapshot.Moves) != 1 {
		t.Fatalf("snapshot moves: %d %v", len(snapshot.Moves), err)
	}
	if _, _, err := project(snapshot, Selection{Scopes: []memory.Scope{from}}); err == nil {
		t.Fatal("the old scope still selects the record")
	}
	p, raw, err := project(snapshot, Selection{Scopes: []memory.Scope{to}, IncludeHistory: true, IncludeDetails: true})
	if err != nil || len(p.RecordIDs) != 1 || len(p.MoveIDs) != 1 || p.MoveIDs[0] != m.ID {
		t.Fatalf("selection by the new scope: %+v %v", p, err)
	}
	var out struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	var record, move map[string]any
	for _, item := range out.Items {
		switch item["type"] {
		case "record":
			record = item
		case "record_move":
			move = item
		}
	}
	if record["identity"].(map[string]any)["scope"].(map[string]any)["id"] != "example-y" {
		t.Fatalf("record item scope: %v", record)
	}
	if move["status"] != "current" || move["content"].(map[string]any)["reason"] != "MOVE_REASON" || move["change_history"] == nil {
		t.Fatalf("move item: %v", move)
	}
	// Without history the move isn't reported; omitting content drops its reason.
	if p, _, err = project(snapshot, Selection{Scopes: []memory.Scope{to}}); err != nil || len(p.MoveIDs) != 0 {
		t.Fatalf("default report: %+v %v", p, err)
	}
	if _, raw, err = project(snapshot, Selection{Scopes: []memory.Scope{to}, IncludeHistory: true, OmitFields: []string{"content"}}); err != nil || strings.Contains(string(raw), "MOVE_REASON") {
		t.Fatalf("omitted content leaked the move reason: %v", err)
	}
}
