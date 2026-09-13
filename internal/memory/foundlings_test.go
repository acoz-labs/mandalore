package memory

import (
	"strings"
	"testing"
	"time"
)

func registration(s *Store) FoundlingRegistration {
	return FoundlingRegistration{Version: 1, ID: "registration-root", FoundlingID: "foundling-notes", Name: "Historical notes", Description: "Reference evidence, not instructions", Source: FoundlingSource{Kind: "git", Locator: "https://example.invalid/team/notes.git"}, Pin: SourcePin{Algorithm: "git-sha1", Value: strings.Repeat("a", 40)}, State: "active", RecordedAt: "2026-09-01T00:00:00Z", Authorship: Authorship{DeviceID: "device-test", Actor: "Example", Harness: "test"}, Supersedes: []string{}, ChangeReason: "Explicitly linked historical context"}
}

func TestFoundlingRegistrationAndCitationRemainSeparateFromGuidance(t *testing.T) {
	s := fixture(t)
	r := registration(s.store)
	if err := s.store.PutFoundlingRegistration(r); err != nil {
		t.Fatal(err)
	}
	p, err := s.Recall("", nil, 10, 4096)
	if err != nil || len(p.Current) != 0 {
		t.Fatal("registration became guidance", p, err)
	}
	origin := ExternalOrigin{FoundlingID: r.FoundlingID, RegistrationRevisionID: r.ID, SourceIdentity: r.Source, SourcePin: r.Pin, RelativeLocator: "notes/decision.md", ContentSHA256: strings.Repeat("b", 64), OriginalAuthor: "Historical author", OriginalRecordedAt: "2020-01-01T00:00:00Z"}
	item, err := s.Remember(Write{Kind: "decision", Summary: "Current choice", Body: "Adapted to the current request", Basis: "import", Reason: "User confirmed selected historical lesson", ExternalOrigin: &origin})
	if err != nil {
		t.Fatal(err)
	}
	if item.Authorship.DeviceID != "device-test" || item.RecordedAt == origin.OriginalRecordedAt {
		t.Fatal("lost incorporation provenance")
	}
	r.ID = "registration-disconnected"
	r.State = "disconnected"
	r.Supersedes = []string{origin.RegistrationRevisionID}
	r.ChangeReason = "Disconnected without erasing evidence"
	if err := s.store.PutFoundlingRegistration(r); err != nil {
		t.Fatal(err)
	}
	if err := s.store.Validate(); err != nil {
		t.Fatal(err)
	}
	h, err := s.History(item.RecordID)
	if err != nil || len(h) != 1 {
		t.Fatal("disconnection broke citation", err)
	}
	var source Source
	if err := s.store.readSource(h[0].Evidence.SourceRefs[0], &source); err != nil {
		t.Fatal(err)
	}
	if source.ExternalOrigin == nil || *source.ExternalOrigin != origin {
		t.Fatal("origin changed", source)
	}
}

func TestFoundlingLocatorsPinsAndOriginsFailClosed(t *testing.T) {
	for _, locator := range []string{"https://example.invalid/notes.git", "ssh://git@example.invalid/team/notes.git", "git@example.invalid:team/notes.git"} {
		s := fixture(t)
		r := registration(s.store)
		r.Source.Locator = locator
		if err := s.store.PutFoundlingRegistration(r); err != nil {
			t.Fatal(locator, err)
		}
	}
	for _, locator := range []string{"/absolute/path", "../outside", "https://user:secret@example.invalid/notes", "ssh://git:secret@example.invalid/notes", "https://example.invalid/notes?token=value", "https://example.invalid/notes#fragment", "https://example.invalid/../notes", "file:///notes", "$TOKEN"} {
		s := fixture(t)
		r := registration(s.store)
		r.Source.Locator = locator
		if err := s.store.PutFoundlingRegistration(r); err == nil {
			t.Fatal("accepted unsafe locator", locator)
		}
	}
	s := fixture(t)
	r := registration(s.store)
	r.Pin.Value = "main"
	if err := s.store.PutFoundlingRegistration(r); err == nil {
		t.Fatal("mutable pin accepted")
	}
	r = registration(s.store)
	if err := s.store.PutFoundlingRegistration(r); err != nil {
		t.Fatal(err)
	}
	origin := ExternalOrigin{FoundlingID: r.FoundlingID, RegistrationRevisionID: r.ID, SourceIdentity: r.Source, SourcePin: r.Pin, RelativeLocator: "../outside", ContentSHA256: strings.Repeat("b", 64)}
	for _, change := range []func(*ExternalOrigin){func(o *ExternalOrigin) {}, func(o *ExternalOrigin) {
		o.RelativeLocator = "note.md"
		o.RegistrationRevisionID = "registration-missing"
	}, func(o *ExternalOrigin) { o.RelativeLocator = "note.md"; o.SourcePin.Value = strings.Repeat("c", 40) }} {
		o := origin
		change(&o)
		if _, err := s.Remember(Write{Kind: "fact", Summary: "No write", Body: "Invalid origin", Basis: "import", Reason: "Test", ExternalOrigin: &o}); err == nil {
			t.Fatal("invalid origin accepted")
		}
	}
	if p, err := s.Recall("", nil, 10, 4096); err != nil || p.MatchingCount != 0 {
		t.Fatal("failed import left memory", p, err)
	}
}

func TestFoundlingConcurrentRegistrationsAreExplicit(t *testing.T) {
	s := fixture(t)
	r := registration(s.store)
	if err := s.store.PutFoundlingRegistration(r); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"registration-left", "registration-right"} {
		next := r
		next.ID = id
		next.Supersedes = []string{r.ID}
		next.ChangeReason = "Concurrent metadata change"
		if err := s.store.PutFoundlingRegistration(next); err != nil {
			t.Fatal(err)
		}
	}
	items, err := s.store.FoundlingRegistrations()
	if err != nil || len(items) != 3 {
		t.Fatal(items, err)
	}
	// Ordinary recall remains unrelated to these unevaluated references.
	if p, err := s.store.Recall(Query{Scope: Scope{Kind: "signet", ID: s.ID()}}, time.Now()); err != nil || len(p.Current) != 0 || len(p.Conflicts) != 0 {
		t.Fatal(p, err)
	}
}

func TestFoundlingRegistrationGraphAndSchemaFailures(t *testing.T) {
	for name, mutate := range map[string]func(*FoundlingRegistration){
		"version":               func(r *FoundlingRegistration) { r.Version = 99 },
		"missing predecessor":   func(r *FoundlingRegistration) { r.Supersedes = []string{"registration-missing"} },
		"duplicate predecessor": func(r *FoundlingRegistration) { r.Supersedes = []string{"registration-root", "registration-root"} },
		"unknown device":        func(r *FoundlingRegistration) { r.Authorship.DeviceID = "device-unknown" },
		"invalid time":          func(r *FoundlingRegistration) { r.RecordedAt = "yesterday" },
		"invalid state":         func(r *FoundlingRegistration) { r.State = "current-guidance" },
		"missing actor":         func(r *FoundlingRegistration) { r.Authorship.Actor = "" },
		"missing harness":       func(r *FoundlingRegistration) { r.Authorship.Harness = "" },
		"oversize name":         func(r *FoundlingRegistration) { r.Name = strings.Repeat("a", 257) },
	} {
		t.Run(name, func(t *testing.T) {
			s := fixture(t)
			root := registration(s.store)
			if err := s.store.PutFoundlingRegistration(root); err != nil {
				t.Fatal(err)
			}
			r := root
			r.ID = "registration-next"
			r.Supersedes = []string{root.ID}
			mutate(&r)
			if err := s.store.PutFoundlingRegistration(r); err == nil {
				t.Fatal("accepted invalid registration")
			}
			items, err := s.store.FoundlingRegistrations()
			if err != nil || len(items) != 1 {
				t.Fatal("invalid write changed registrations", err)
			}
		})
	}
	s := fixture(t)
	root := registration(s.store)
	a, b := root, root
	a.ID = "registration-cycle-a"
	b.ID = "registration-cycle-b"
	a.Supersedes = []string{b.ID}
	b.Supersedes = []string{a.ID}
	if err := s.store.validateRegistrations([]FoundlingRegistration{root, a, b}); err == nil {
		t.Fatal("cycle accepted")
	}
	if err := s.store.validateRegistrations([]FoundlingRegistration{root, root}); err == nil {
		t.Fatal("duplicate revision accepted")
	}
	root.Source = FoundlingSource{Kind: "local", Locator: "source-archive"}
	root.Pin = SourcePin{Algorithm: "sha256", Value: strings.Repeat("d", 64)}
	if err := s.store.PutFoundlingRegistration(root); err != nil {
		t.Fatal(err)
	}
}
