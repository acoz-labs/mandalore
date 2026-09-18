package memory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreparedUpgradeStopsNormalWriters(t *testing.T) {
	s := fixtureStore(t)
	u := UpgradeRecord{Version: 1, ID: "upgrade-test", SignetID: s.Signet.ID, From: 1, To: 2, OriginalManifestSHA256: strings.Repeat("a", 64), PortableSHA256: strings.Repeat("b", 64), BaseHead: strings.Repeat("c", 40), RecordedAt: "2026-09-18T12:00:00Z", Authorship: revision("revision-template").Authorship}
	dir := filepath.Join(s.Root, "provenance/upgrades")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := writeNewJSON(filepath.Join(dir, u.ID+".json"), u); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); !errors.Is(err, ErrUpgradePending) {
		t.Fatalf("expected pending state, got %v", err)
	}
	called := false
	if err := s.WithExclusiveLock(func() error { called = true; return nil }); !errors.Is(err, ErrUpgradePending) || called {
		t.Fatalf("ordinary callback ran through pending upgrade: %v %v", called, err)
	}
	if _, err := s.RecordEvent("test", "Must not be written", u.Authorship); !errors.Is(err, ErrUpgradePending) {
		t.Fatal("journal write was not blocked", err)
	}
	if err := s.WithFormatUpgradeLock(func() error { called = true; return nil }); err != nil || !called {
		t.Fatal("explicit recovery cannot inspect valid pending state", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	called = false
	if err := s.WithFormatUpgradeLock(func() error { called = true; return nil }); err == nil || called {
		t.Fatal("recovery bypassed malformed evidence")
	}
}
