package binding

import (
	"encoding/json"
	"github.com/acoz-labs/mandalore/internal/memory"
	"os"
	"path/filepath"
	"testing"
)

func signet(t *testing.T) *memory.Store {
	t.Helper()
	s, err := memory.Create(filepath.Join(t.TempDir(), "signet"), "Example", "device-first", "Test device")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestBindAndOpenIndependentOfCWD(t *testing.T) {
	s := signet(t)
	p := filepath.Join(t.TempDir(), "binding.json")
	b, err := Bind(s.Root, p, "Second test device", "Example actor")
	if err != nil {
		t.Fatal(err)
	}
	if b.SignetID != s.Signet.ID || b.DeviceID == "device-first" {
		t.Fatal("missing independent enrollment", b)
	}
	t.Chdir(t.TempDir())
	service, err := Open(p, "test-harness")
	if err != nil || service.ID() != s.Signet.ID {
		t.Fatal("cwd affected binding", err)
	}
	r, err := service.Remember(memory.Write{Kind: "fact", Summary: "Example", Body: "Bound evidence", Basis: "observation", Reason: "Observed"})
	if err != nil || r.Authorship.DeviceID != b.DeviceID || r.Authorship.Harness != "test-harness" {
		t.Fatal(r, err)
	}
	if _, err := Bind(s.Root, p, "Duplicate", "Example actor"); err == nil {
		t.Fatal("overwrote binding")
	}
}

func TestBindingOutsideSignetAndIdentityPinned(t *testing.T) {
	s := signet(t)
	if _, err := Bind(s.Root, filepath.Join(s.Root, "binding.json"), "Test", "Example"); err == nil {
		t.Fatal("binding inside signet")
	}
	parent := t.TempDir()
	link := filepath.Join(parent, "link")
	if err := os.Symlink(s.Root, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Bind(s.Root, filepath.Join(link, "nested/binding.json"), "Test", "Example"); err == nil {
		t.Fatal("binding through symlink inside signet")
	}
	p := filepath.Join(t.TempDir(), "binding.json")
	if _, err := Bind(s.Root, p, "Test", "Example"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(s.Root, "signet.json"))
	if err != nil {
		t.Fatal(err)
	}
	other := signet(t)
	replacement, err := os.ReadFile(filepath.Join(other.Root, "signet.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, "signet.json"), replacement, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(p, "test"); err == nil {
		t.Fatal("silently rebound different signet")
	}
	if err := os.WriteFile(filepath.Join(s.Root, "signet.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestBindingReadRefusesInSignetAndMalformedFiles(t *testing.T) {
	s := signet(t)
	p := filepath.Join(t.TempDir(), "binding.json")
	b, err := Bind(s.Root, p, "Test", "Example")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(s.Root, "binding.json")
	if err := os.WriteFile(inside, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(inside, "test"); err == nil {
		t.Fatal("read accepted in-signet binding")
	}
	if err := os.Remove(inside); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{string(data) + `{}`, `{"schema_version":99}`, `{"schema_version":1,"schema_version":99}`} {
		if err := os.WriteFile(p, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(p, "test"); err == nil {
			t.Fatal("accepted malformed binding")
		}
	}
}

func TestRelativeEnvironmentBindingRefused(t *testing.T) {
	t.Setenv("MANDALORE_BINDING", "relative/binding.json")
	if _, err := DefaultPath(); err == nil {
		t.Fatal("environment binding depends on cwd")
	}
}

func TestIndependentBindingsNeverCrossSignets(t *testing.T) {
	first, second := signet(t), signet(t)
	base := t.TempDir()
	var ids []string
	for i, s := range []*memory.Store{first, second} {
		path := filepath.Join(base, []string{"first.json", "second.json"}[i])
		b, err := Bind(s.Root, path, "Test", "Example")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, b.DeviceID)
		service, err := Open(path, "test")
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if _, err := service.Remember(memory.Write{Kind: "fact", Summary: "First only", Body: "Copper Finch", Basis: "user-direction", Reason: "Confirmed"}); err != nil {
				t.Fatal(err)
			}
		} else {
			out, err := service.Recall("Copper", nil, 5, 8192)
			if err != nil || len(out.Current) != 0 {
				t.Fatal("cross-signet recall", out, err)
			}
		}
	}
	if ids[0] == ids[1] {
		t.Fatal("shared device enrollment")
	}
}
