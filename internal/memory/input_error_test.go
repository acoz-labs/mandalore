package memory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validWrite() Write {
	return Write{Kind: "fact", Summary: "Synthetic convention", Body: "Use synthetic fixtures.", Basis: "user-direction", Reason: "Confirmed in a synthetic test"}
}

// written lists every revision, source and intent file, so a refused save can
// prove it wrote nothing.
func written(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	for _, dir := range []string{"memory/records", "memory/sources", ".mandalore"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() != ".gitkeep" && (dir != ".mandalore" || strings.HasPrefix(d.Name(), "operation-")) {
				out = append(out, path)
			}
			return nil
		})
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
	}
	return out
}

func formatTwoService(t *testing.T) *Service {
	t.Helper()
	s := visibilityStore(t, fixtureStore(t))
	service, err := OpenService(s.Root, Authorship{DeviceID: "device-laptop", Actor: "Example", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func refused(t *testing.T, s *Service, key string, in Write, target error, wants ...string) {
	t.Helper()
	before := len(written(t, s.Root()))
	_, err := s.RememberIdempotent(key, in)
	if !errors.Is(err, target) {
		t.Fatalf("%s: got %v, want %v", key, err, target)
	}
	for _, want := range wants {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: %q lacks %q", key, err, want)
		}
	}
	if after := written(t, s.Root()); len(after) != before {
		t.Fatalf("%s: refused save wrote %v", key, after)
	}
}

func TestRememberLimitsAreInputErrors(t *testing.T) {
	s := fixture(t)
	for _, f := range []struct {
		name  string
		limit int
		set   func(*Write, string)
	}{
		{"summary", MaxSummaryBytes, func(w *Write, v string) { w.Summary = v }},
		{"body", MaxBodyBytes, func(w *Write, v string) { w.Body = v }},
		{"reason", MaxReasonBytes, func(w *Write, v string) { w.Reason = v }},
	} {
		in := validWrite()
		f.set(&in, strings.Repeat("x", f.limit+1))
		refused(t, s, f.name+"-over", in, ErrInvalidInput, f.name+" is ", "the limit is ", "bytes")
		if _, err := s.RememberIdempotent(f.name+"-over", in); !strings.Contains(err.Error(), f.name) {
			t.Fatal(err)
		}
		// Nothing was stored, so the same key takes corrected input.
		f.set(&in, strings.Repeat("x", f.limit))
		if _, err := s.RememberIdempotent(f.name+"-over", in); err != nil {
			t.Fatalf("%s at the limit: %v", f.name, err)
		}
		for _, bad := range []string{"", "   ", "a\x00b"} {
			in := validWrite()
			f.set(&in, bad)
			refused(t, s, f.name+"-blank", in, ErrInvalidInput, f.name+" is required")
		}
	}
	in := validWrite()
	in.Summary = strings.Repeat("é", 129) // 258 bytes, 129 characters
	refused(t, s, "multibyte", in, ErrInvalidInput, "summary is 258 bytes; the limit is 256 UTF-8 bytes")
	in.Summary = strings.Repeat("é", 128)
	if _, err := s.RememberIdempotent("multibyte", in); err != nil {
		t.Fatal(err)
	}
}

func TestRememberValidationErrorsAreInputErrors(t *testing.T) {
	s := fixture(t)
	for name, change := range map[string]func(*Write){
		"blank basis":      func(w *Write) { w.Basis = " " },
		"bad basis":        func(w *Write) { w.Basis = "hearsay" },
		"bad kind":         func(w *Write) { w.Kind = "unsupported" },
		"bad sensitivity":  func(w *Write) { w.Sensitivity = "high" },
		"bad confidence":   func(w *Write) { w.Confidence = "certain" },
		"bad scope id":     func(w *Write) { w.Scope = &Scope{Kind: "project", ID: "Not An ID"} },
		"bad scope kind":   func(w *Write) { w.Scope = &Scope{Kind: "team", ID: "team-example"} },
		"record_id alone":  func(w *Write) { w.RecordID = "record-example" },
		"supersedes alone": func(w *Write) { w.Supersedes = []string{"revision-example"} },
		"33 supersedes": func(w *Write) {
			w.RecordID = "record-example"
			for i := range 33 {
				w.Supersedes = append(w.Supersedes, "revision-"+strings.Repeat("a", i+1))
			}
		},
	} {
		in := validWrite()
		change(&in)
		refused(t, s, name, in, ErrInvalidInput)
	}
	in := validWrite()
	in.Kind, in.Body = "unsupported", "PRIVATE_BODY_CANARY"
	if _, err := s.RememberIdempotent("canary", in); err == nil || !strings.Contains(err.Error(), "kind") || strings.Contains(err.Error(), "PRIVATE_BODY_CANARY") || strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("message must name the field only: %v", err)
	}
}

func TestJournalLimitsAreInputErrors(t *testing.T) {
	s := fixture(t)
	if _, err := s.AppendJournalIdempotent("journal-kind", strings.Repeat("k", 65), "Outcome"); !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "kind is 65 bytes; the limit is 64") {
		t.Fatal(err)
	}
	if _, err := s.AppendJournalIdempotent("journal-summary", "outcome", strings.Repeat("s", 4097)); !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "summary is 4097 bytes; the limit is 4096") {
		t.Fatal(err)
	}
	if _, err := s.AppendJournalIdempotent("journal-summary", "outcome", strings.Repeat("s", 4096)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendJournal(strings.Repeat("k", 65), "Outcome"); !errors.Is(err, ErrInvalidInput) {
		t.Fatal(err)
	}
}

// Corrections, per signet format: format 2 compares supersedes with the
// current heads before the graph check.
func TestCorrectionErrorsByFormat(t *testing.T) {
	for _, format := range []struct {
		name    string
		service func(*testing.T) *Service
		missing error
	}{
		{"format 1", fixture, ErrInvalidInput},
		{"format 2", formatTwoService, ErrStaleHeads},
	} {
		t.Run(format.name, func(t *testing.T) {
			s := format.service(t)
			first, err := s.RememberIdempotent("first", validWrite())
			if err != nil {
				t.Fatal(err)
			}
			other, err := s.RememberIdempotent("other", validWrite())
			if err != nil {
				t.Fatal(err)
			}
			correct := func(change func(*Write)) Write {
				in := validWrite()
				in.RecordID, in.Supersedes, in.Body = first.RecordID, []string{first.ID}, "Corrected"
				change(&in)
				return in
			}
			refused(t, s, "missing", correct(func(w *Write) { w.Supersedes = []string{"revision-missing"} }), format.missing)
			refused(t, s, "other record", correct(func(w *Write) { w.Supersedes = []string{other.ID} }), format.missing)
			refused(t, s, "unknown record", correct(func(w *Write) { w.RecordID = "record-unknown" }), format.missing)
			refused(t, s, "changed kind", correct(func(w *Write) { w.Kind = "decision" }), ErrInvalidInput, "memory_history")
			refused(t, s, "changed scope", correct(func(w *Write) { w.Scope = &Scope{Kind: "project", ID: "project-example"} }), ErrInvalidInput, "memory_history")
			if _, err := s.RememberIdempotent("correct", correct(func(*Write) {})); err != nil {
				t.Fatal(err)
			}
			// first is no longer a head: format 1 keeps the fork, format 2 refuses.
			stale := correct(func(w *Write) { w.Body = "Forked" })
			if format.missing == ErrStaleHeads {
				refused(t, s, "stale", stale, ErrStaleHeads)
			} else if _, err := s.RememberIdempotent("stale", stale); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBrokenStoreIsNotAnInputError(t *testing.T) {
	s := fixture(t)
	r, err := s.RememberIdempotent("first", validWrite())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(s.Root(), "memory/sources", r.Evidence.SourceRefs[0]+".json")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RememberIdempotent("second", validWrite()); err == nil || errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrStaleHeads) {
		t.Fatalf("a broken store was blamed on the caller: %v", err)
	}
}

// After the intent is written, a resumed save is never reported as invalid
// input or stale heads, even when the heads have since moved.
func TestResumedSaveIsNeverAnInputError(t *testing.T) {
	s := formatTwoService(t)
	first, err := s.RememberIdempotent("first", validWrite())
	if err != nil {
		t.Fatal(err)
	}
	in := validWrite()
	in.RecordID, in.Supersedes, in.Body = first.RecordID, []string{first.ID}, "Corrected"
	sources := filepath.Join(s.Root(), "memory/sources")
	if err := os.Chmod(sources, 0500); err != nil {
		t.Fatal(err)
	}
	_, err = s.RememberIdempotent("interrupted", in)
	if err := os.Chmod(sources, 0700); err != nil {
		t.Fatal(err)
	}
	if err == nil || errors.Is(err, ErrInvalidInput) {
		t.Fatalf("interrupted save: %v", err)
	}
	moved := in
	moved.Body = "Moved the heads"
	if _, err := s.RememberIdempotent("moved", moved); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RememberIdempotent("interrupted", in); err == nil || errors.Is(err, ErrStaleHeads) || errors.Is(err, ErrInvalidInput) {
		t.Fatalf("resumed save: %v", err)
	}
}
