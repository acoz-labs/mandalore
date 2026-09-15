package binding

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func guardedFixture(t *testing.T) (string, Guard, Binding) {
	t.Helper()
	s := signet(t)
	path := filepath.Join(t.TempDir(), "binding.json")
	b, err := Bind(s.Root, path, "Test device", "Test actor")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return path, Guard{SHA256: hex.EncodeToString(digest[:]), SignetID: b.SignetID}, b
}

func TestGuardedBindingRetainsIdentityAndProvenance(t *testing.T) {
	path, guard, b := guardedFixture(t)
	t.Chdir(t.TempDir())
	s, err := OpenGuarded(path, "pi", guard)
	if err != nil {
		t.Fatal(err)
	}
	if s.ID() != b.SignetID {
		t.Fatal("wrong bound signet", s.ID())
	}
	r, err := s.Remember(memory.Write{Kind: "fact", Summary: "Synthetic guarded write", Body: "Copper Finch", Basis: "observation", Reason: "Guard test"})
	if err != nil || r.Authorship.DeviceID != b.DeviceID || r.Authorship.Actor != b.Actor || r.Authorship.Harness != "pi" {
		t.Fatal("guard changed provenance", r, err)
	}
	if _, err := Open(path, "codex"); err != nil {
		t.Fatal("legacy unguarded caller regressed", err)
	}
}

func TestGuardedBindingRejectsChangedBytesAndSignet(t *testing.T) {
	path, guard, b := guardedFixture(t)
	for _, mutate := range []func(*Binding){
		func(b *Binding) { b.Actor = "Different actor" },
		func(b *Binding) {
			_, _, other := guardedFixture(t)
			*b = other
		},
	} {
		changed := b
		mutate(&changed)
		raw, err := json.Marshal(changed)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		if s, err := OpenGuarded(path, "pi", guard); err == nil || s != nil {
			t.Fatal("changed binding silently accepted", err)
		}
	}
}

func TestGuardedBindingValidatesBothGuardsBeforeOpening(t *testing.T) {
	path, guard, _ := guardedFixture(t)
	for _, invalid := range []Guard{
		{SHA256: guard.SHA256},
		{SignetID: guard.SignetID},
		{SHA256: "not-a-digest", SignetID: guard.SignetID},
		{SHA256: strings.Repeat("0", 64), SignetID: guard.SignetID},
		{SHA256: guard.SHA256, SignetID: "signet-other"},
		{SHA256: guard.SHA256, SignetID: "invalid\nidentity"},
	} {
		if s, err := OpenGuarded(path, "pi", invalid); err == nil || s != nil {
			t.Fatal("invalid guard accepted", invalid)
		}
	}
	if _, err := OpenGuarded(path, "pi", Guard{}); err != nil {
		t.Fatal("empty optional guard is not compatible", err)
	}
}

func TestGuardedBindingRejectsChangedRootIdentity(t *testing.T) {
	path, guard, b := guardedFixture(t)
	other := signet(t)
	raw, err := os.ReadFile(filepath.Join(other.Root, "signet.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b.Root, "signet.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if s, err := OpenGuarded(path, "pi", guard); err == nil || s != nil {
		t.Fatal("binding guard masked replaced root", err)
	}
}
