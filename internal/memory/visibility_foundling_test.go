package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWithdrawnOriginCannotBeRepromotedByNewRegistration(t *testing.T) {
	s := visibilityStore(t, fixtureStore(t))
	service, err := OpenService(s.Root, revision("revision-template").Authorship)
	if err != nil {
		t.Fatal(err)
	}
	registration, err := service.WriteFoundling(foundlingWrite())
	if err != nil {
		t.Fatal(err)
	}
	origin := &ExternalOrigin{FoundlingID: registration.FoundlingID, RegistrationRevisionID: registration.ID, SourceIdentity: registration.Source, SourcePin: registration.Pin, RelativeLocator: "notes.md", ContentSHA256: strings.Repeat("d", 64)}
	input := Write{Kind: "fact", Summary: "Synthetic historical claim", Body: "A sourced synthetic claim.", Basis: "import", Reason: "Reviewed reference", ExternalOrigin: origin}
	first, err := service.RememberFromFoundling(input, func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	e := VisibilityEvent{Version: 1, ID: "visibility-withdraw", RecordID: first.RecordID, Action: "withdraw", Parents: []string{}, Observed: []string{first.ID}, Reason: "Withdraw this source-derived claim", RecordedAt: first.RecordedAt, Authorship: first.Authorship}
	dir := filepath.Join(s.Root, "memory/visibility", first.RecordID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := writeNewJSON(filepath.Join(dir, e.ID+".json"), e); err != nil {
		t.Fatal(err)
	}
	called := false
	verify := func() error { called = true; return nil }
	if _, err := service.RememberFromFoundling(input, verify); err == nil || called {
		t.Fatal("fresh ID bypassed origin withdrawal")
	}
	input.RecordID, input.Supersedes = first.RecordID, []string{first.ID}
	if _, err := service.RememberFromFoundling(input, verify); err == nil || called {
		t.Fatal("promoted into a withdrawn target")
	}
	input.RecordID, input.Supersedes = "", nil
	relinked, err := service.WriteFoundling(foundlingWrite())
	if err != nil {
		t.Fatal(err)
	}
	origin.FoundlingID, origin.RegistrationRevisionID = relinked.FoundlingID, relinked.ID
	if _, err := service.RememberFromFoundling(input, verify); err == nil || called {
		t.Fatal("new registration bypassed exact source identity")
	}
	records, err := s.revisions()
	if err != nil || len(records) != 1 {
		t.Fatal("rejected promotion changed memory", records, err)
	}
	origin.ContentSHA256 = strings.Repeat("e", 64)
	if _, err := service.RememberFromFoundling(input, verify); err != nil || !called {
		t.Fatal("blocked distinct source content as a phrase ban", err)
	}
}
