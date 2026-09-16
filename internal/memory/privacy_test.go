package memory

import "testing"

func TestPrivacySensitivityLabelsDoNotFilterRecall(t *testing.T) {
	for _, label := range []string{"public", "private", "sensitive", "restricted"} {
		t.Run(label, func(t *testing.T) {
			s := fixture(t)
			r, err := s.Remember(Write{Kind: "fact", Summary: "Synthetic privacy example", Body: "SYNTHETIC-NOT-A-SECRET", Basis: "observation", Reason: "Fixture", Sensitivity: label})
			if err != nil || r.Sensitivity != label {
				t.Fatal("label was not preserved", err)
			}
			packet, err := s.Recall("", nil, 5, 8192)
			if err != nil || len(packet.Current) != 1 || packet.Current[0].ID != r.ID || packet.Current[0].Body != r.Body {
				t.Fatal("sensitivity metadata unexpectedly changed recall", err)
			}
		})
	}
}

func TestPrivacyCorrectionPreservesHistoryAndJournal(t *testing.T) {
	s := fixture(t)
	const original = "SYNTHETIC-ORIGINAL-NOT-A-SECRET"
	input := Write{Kind: "fact", Summary: "Example", Body: original, Basis: "observation", Reason: "Fixture", Sensitivity: "restricted"}
	first, err := s.Remember(input)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := s.AppendJournal("fixture", original)
	if err != nil {
		t.Fatal(err)
	}
	input.RecordID, input.Supersedes = first.RecordID, []string{first.ID}
	input.Body, input.Reason = "Replacement without the original marker.", "Synthetic correction"
	second, err := s.Remember(input)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := s.Recall("", nil, 5, 8192)
	if err != nil || len(packet.Current) != 1 || packet.Current[0].ID != second.ID || packet.Current[0].Body != second.Body {
		t.Fatal("correction did not update current recall", err)
	}
	history, err := s.History(first.RecordID)
	if err != nil || len(history) != 2 {
		t.Fatal("missing correction history", err)
	}
	found := false
	for _, r := range history {
		found = found || (r.ID == first.ID && r.Body == original)
	}
	if !found {
		t.Fatal("correction erased predecessor")
	}
	entries, err := s.Journal(original, 5)
	if err != nil || len(entries) != 1 || entries[0].ID != entry.ID || entries[0].Summary != original {
		t.Fatal("correction changed independent journal evidence", err)
	}
}
