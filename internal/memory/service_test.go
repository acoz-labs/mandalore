package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) *Service {
	t.Helper()
	s, err := Create(filepath.Join(t.TempDir(), "bank"), "Example", "device-test", "Test host")
	if err != nil {
		t.Fatal(err)
	}
	service, err := OpenService(s.Root, Authorship{DeviceID: "device-test", Actor: "Example user", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestRecallBudgetIncludesConflictsAndNeverClipsClaims(t *testing.T) {
	s := fixture(t)
	input := Write{Kind: "decision", Summary: "Project name", Body: "Copper Finch", Basis: "user-direction", Reason: "Selected"}
	first, err := s.Remember(input)
	if err != nil {
		t.Fatal(err)
	}
	input.RecordID, input.Supersedes = first.RecordID, []string{first.ID}
	for _, name := range []string{"Silver Heron", "Amber Lark"} {
		input.Body = name
		if _, err := s.Remember(input); err != nil {
			t.Fatal(err)
		}
	}
	long := strings.Repeat("A useful observation. ", 200)
	if _, err := s.Remember(Write{Kind: "fact", Summary: "Long observation", Body: long, Basis: "observation", Reason: "Observed"}); err != nil {
		t.Fatal(err)
	}
	for _, budget := range []int{1024, 2048, 8192} {
		p, err := s.Recall("", nil, 10, budget)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(p)
		if len(b) > budget || p.ConflictCount != 1 || len(p.Conflicts) != 1 || len(p.Conflicts[0].HeadIDs) != 2 || p.MatchingCount != 1 {
			t.Fatalf("budget/conflict contract: %d bytes, %+v", len(b), p)
		}
		for _, r := range p.Current {
			if r.Body != long {
				t.Fatal("claim clipped or conflicted guidance promoted")
			}
		}
		if budget == 1024 && !p.Truncated {
			t.Fatal("omitted claim not disclosed")
		}
	}
}

func TestRejectedWritesDoNotLeaveEvidenceOrRecords(t *testing.T) {
	s := fixture(t)
	input := Write{Kind: "unsupported", Summary: "Invalid", Body: "Must not be stored", Basis: "observation", Reason: "Test"}
	if _, err := s.Remember(input); err == nil {
		t.Fatal("invalid kind accepted")
	}
	input.Kind, input.RecordID, input.Supersedes = "fact", "record-missing", []string{"revision-missing"}
	if _, err := s.Remember(input); err == nil {
		t.Fatal("missing predecessor accepted")
	}
	for _, directory := range []string{"memory/sources", "memory/records"} {
		entries, err := os.ReadDir(filepath.Join(s.Root(), directory))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.Name() != ".gitkeep" {
				t.Fatalf("rejected write left %s/%s", directory, entry.Name())
			}
		}
	}
}

func TestRememberRecallCorrectAndJournal(t *testing.T) {
	s := fixture(t)
	input := Write{Kind: "decision", Summary: "Project name", Body: "The fictional project is Copper Finch.", Basis: "user-direction", Reason: "User selected the name."}
	first, err := s.Remember(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Scope.ID != s.ID() || first.Authorship.DeviceID != "device-test" || len(first.Evidence.SourceRefs) != 1 {
		t.Fatalf("missing server-authored metadata: %+v", first)
	}
	input.RecordID, input.Supersedes = first.RecordID, []string{first.ID}
	input.Body, input.Reason = "The fictional project is Silver Heron.", "User renamed the project."
	second, err := s.Remember(input)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := s.Recall("project", nil, 5, 4096)
	if err != nil || len(packet.Current) != 1 || packet.Current[0].ID != second.ID || len(packet.Conflicts) != 0 {
		t.Fatalf("current recall: %+v %v", packet, err)
	}
	history, err := s.History(first.RecordID)
	if err != nil || len(history) != 2 {
		t.Fatalf("history: %+v %v", history, err)
	}
	if _, err := s.AppendJournal("decision", "Renamed the project to Silver Heron."); err != nil {
		t.Fatal(err)
	}
	entries, err := s.Journal("renamed", 5)
	if err != nil || len(entries) != 1 {
		t.Fatalf("journal: %+v %v", entries, err)
	}
}

func TestMemoryScopesAndResponseBudget(t *testing.T) {
	s := fixture(t)
	work := Scope{Kind: "project", ID: "project-work"}
	if _, err := s.Remember(Write{Kind: "fact", Summary: "Work code", Body: "WORK-CANARY", Basis: "observation", Reason: "Observed", Scope: &work}); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 4; n++ {
		if _, err := s.Remember(Write{Kind: "fact", Summary: "Personal code", Body: "PERSONAL-CANARY", Basis: "observation", Reason: "Observed"}); err != nil {
			t.Fatal(err)
		}
	}
	packet, err := s.Recall("", nil, 10, 1024)
	if err != nil || !packet.Truncated || len(packet.Current) == 0 {
		t.Fatalf("unbounded packet: %+v %v", packet, err)
	}
	for _, r := range packet.Current {
		if r.Scope == work {
			t.Fatal("unselected scope leaked")
		}
	}
	packet, err = s.Recall("", &work, 10, 4096)
	if err != nil || len(packet.Current) != 1 || packet.Current[0].Body != "WORK-CANARY" {
		t.Fatalf("explicit scope: %+v %v", packet, err)
	}
}

func TestMemoryServiceRejectsUnboundAndLegacyStores(t *testing.T) {
	s := fixture(t)
	if _, err := OpenService(s.Root(), Authorship{DeviceID: "device-missing", Actor: "Example", Harness: "test"}); err == nil {
		t.Fatal("unknown device accepted")
	}
	legacy := filepath.Join(t.TempDir(), "legacy")
	if err := os.Mkdir(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "agent.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenService(legacy, Authorship{DeviceID: "device-test", Actor: "Example", Harness: "test"}); err == nil {
		t.Fatal("assistant repository silently used as bank")
	}
	if _, err := s.Remember(Write{Kind: "fact", Summary: "Incomplete"}); err == nil {
		t.Fatal("incomplete claim written")
	}
}
