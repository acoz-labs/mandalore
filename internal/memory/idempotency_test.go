package memory

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestRememberIdempotentCrashRecovery(t *testing.T) {
	for _, stage := range []string{"intent", "source", "published"} {
		t.Run(stage, func(t *testing.T) {
			s := fixture(t)
			input := Write{Kind: "fact", Summary: "Durable fact", Body: "A useful observation", Basis: "observation", Reason: "Observed"}
			first, err := s.RememberIdempotent("principal/request", input)
			if err != nil {
				t.Fatal(err)
			}
			revisionPath := filepath.Join(s.Root(), "memory/records", first.RecordID, first.ID+".json")
			if stage != "published" {
				if err := os.Remove(revisionPath); err != nil {
					t.Fatal(err)
				}
			}
			if stage == "intent" {
				if err := os.Remove(filepath.Join(s.Root(), "memory/sources", first.Evidence.SourceRefs[0]+".json")); err != nil {
					t.Fatal(err)
				}
			}
			reopened, err := OpenService(s.Root(), s.author)
			if err != nil {
				t.Fatal(err)
			}
			second, err := reopened.RememberIdempotent("principal/request", input)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("retry changed result")
			}
			records, err := s.store.revisions()
			if err != nil || len(records) != 1 {
				t.Fatalf("duplicate or lost publication: %v %v", records, err)
			}
			input.Body = "Different payload"
			if _, err := reopened.RememberIdempotent("principal/request", input); err == nil {
				t.Fatal("allowed key reuse with different payload")
			}
			if err := s.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJournalIdempotentCrashRecoveryAndOperationBinding(t *testing.T) {
	s := fixture(t)
	first, err := s.AppendJournalIdempotent("principal/request", "outcome", "Completed useful work")
	if err != nil {
		t.Fatal(err)
	}
	at, _ := time.Parse(time.RFC3339Nano, first.RecordedAt)
	path := filepath.Join(s.Root(), "memory/events", at.UTC().Format("2006/01"), first.ID+".json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenService(s.Root(), s.author)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		second, err := reopened.AppendJournalIdempotent("principal/request", "outcome", "Completed useful work")
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatalf("retry mismatch: %v %v", second, err)
		}
	}
	if _, err := s.AppendJournalIdempotent("principal/request", "outcome", "Changed"); err == nil {
		t.Fatal("payload collision accepted")
	}
	if _, err := s.RememberIdempotent("principal/request", Write{}); err == nil {
		t.Fatal("operation collision accepted")
	}
	other := *s
	other.author.Actor = "Other identity"
	if _, err := other.AppendJournalIdempotent("principal/request", "outcome", "Completed useful work"); err == nil {
		t.Fatal("authorship collision accepted")
	}
	entries, err := s.Journal("", 100)
	if err != nil || len(entries) != 1 {
		t.Fatalf("duplicate journal: %v %v", entries, err)
	}
}
