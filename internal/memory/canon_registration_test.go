package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonRegistrationVersionAndLegacyPreservation(t *testing.T) {
	s := fixture(t)
	legacy, err := s.WriteFoundling(foundlingWrite())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.Root(), "foundlings/registrations", legacy.FoundlingID, legacy.ID+".json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	explicit := foundlingWrite()
	explicit.Mode = "legacy"
	explicitLegacy, err := s.WriteFoundling(explicit)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := os.ReadFile(filepath.Join(s.Root(), "foundlings/registrations", explicitLegacy.FoundlingID, explicitLegacy.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"mode"`) || explicitLegacy.Version != 1 {
		t.Fatal("v1 legacy encoding incompatible", string(encoded))
	}

	input := foundlingWrite()
	input.Mode = "canon"
	input.Branch = "main"
	input.SourceSignetID = "signet-work"
	input.Source = FoundlingSource{Kind: "git", Locator: "https://example.invalid/work.git"}
	input.Pin = SourcePin{Algorithm: "git-sha1", Value: strings.Repeat("a", 40)}
	canon, err := s.WriteFoundling(input)
	if err != nil {
		t.Fatal(err)
	}
	if canon.Version != 2 || canon.EffectiveMode() != "canon" {
		t.Fatal(canon)
	}
	view, err := s.Foundling(canon.FoundlingID)
	if err != nil || view.Mode != "canon" || view.SourceSignetID != input.SourceSignetID || view.Branch != "main" {
		t.Fatal(view, err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("legacy rewritten")
	}
	view, err = s.Foundling(legacy.FoundlingID)
	if err != nil || view.Mode != "legacy" || legacy.Version != 1 {
		t.Fatal(view, err)
	}
	for _, branch := range []string{"", "-option", "main~1", "../main", "main:other", "a/.hidden", "main.lock", "a//b", "main@{1}", "a\\b"} {
		input.Branch = branch
		if _, err := s.WriteFoundling(input); err == nil {
			t.Fatalf("accepted branch %q", branch)
		}
	}
	input.Branch = "main"
	input.SourceSignetID = ""
	if _, err := s.WriteFoundling(input); err == nil {
		t.Fatal("missing identity accepted")
	}
	bad := canon
	bad.Version = 1
	bad.ID = NewID("registration")
	bad.FoundlingID = NewID("foundling")
	if err := s.store.PutFoundlingRegistration(bad); err == nil {
		t.Fatal("canon accepted as legacy v1")
	}
}

func TestCanonCitationTracksFetchedCommit(t *testing.T) {
	s := fixture(t)
	r := registration(s.store)
	r.Version = 2
	r.Mode = "canon"
	r.Branch = "main"
	r.SourceSignetID = "signet-work"
	if err := s.store.PutFoundlingRegistration(r); err != nil {
		t.Fatal(err)
	}
	origin := ExternalOrigin{FoundlingID: r.FoundlingID, RegistrationRevisionID: r.ID, SourceIdentity: r.Source, SourcePin: SourcePin{Algorithm: "git-sha1", Value: strings.Repeat("b", 40)}, RelativeLocator: "memory/records/record-work/revision-work.json", ContentSHA256: strings.Repeat("c", 64)}
	if err := s.store.validateOrigin(&origin); err != nil {
		t.Fatal(err)
	}
	origin.SourcePin.Value = "moving-head"
	if err := s.store.validateOrigin(&origin); err == nil {
		t.Fatal("nonimmutable citation accepted")
	}
}
