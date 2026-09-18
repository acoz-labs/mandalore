package formatupgrade

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

func fixture(t *testing.T) (Request, *memory.Service) {
	t.Helper()
	s, err := memory.Create(filepath.Join(t.TempDir(), "bank"), "PRIVATE_NAME", "device-test", "Synthetic device")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "binding.json")
	if _, err := binding.Bind(s.Root, p, "Synthetic device", "PRIVATE_ACTOR"); err != nil {
		t.Fatal(err)
	}
	a, err := binding.Open(p, "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Remember(memory.Write{Kind: "fact", Summary: "PRIVATE_SUMMARY", Body: "PRIVATE_BODY", Basis: "observation", Reason: "PRIVATE_REASON"}); err != nil {
		t.Fatal(err)
	}
	sy, err := signetsync.Open(s.Root, s.Signet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sy.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return Request{BindingPath: p}, a
}

func TestPreviewPinsBindingAndCheckpointWithoutContent(t *testing.T) {
	in, service := fixture(t)
	p, err := Preview(context.Background(), in)
	if err != nil || p.Version != 1 || p.FromVersion != 1 || p.ToVersion != 2 || p.Source.SignetID != service.ID() || len(p.BindingSHA256) != 64 {
		t.Fatal(p, err)
	}
	q, err := Preview(context.Background(), in)
	if err != nil || !reflect.DeepEqual(p, q) {
		t.Fatal("unstable preview", err)
	}
	b, err := json.Marshal(p)
	if err != nil || strings.Contains(string(b), "PRIVATE_") {
		t.Fatal("preview exposed content or authorship", err)
	}
	if _, err := os.Stat(filepath.Join(service.Root(), "provenance/upgrades")); !os.IsNotExist(err) {
		t.Fatal("preview prepared an upgrade", err)
	}
	if _, err := os.Stat(filepath.Join(service.Root(), ".mandalore/format-upgrade")); !os.IsNotExist(err) {
		t.Fatal("preview created recovery state", err)
	}
	raw, err := os.ReadFile(in.BindingPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(in.BindingPath, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	changed, err := Preview(context.Background(), in)
	if err != nil || changed.BindingSHA256 == p.BindingSHA256 || changed.Source != p.Source {
		t.Fatal("binding pin was lost", err)
	}
}

func TestPreviewRequiresExplicitBindingAndCleanSource(t *testing.T) {
	in, service := fixture(t)
	t.Setenv("MANDALORE_BINDING", in.BindingPath)
	if _, err := Preview(context.Background(), Request{}); err == nil {
		t.Fatal("implicit environment selection")
	}
	alias := filepath.Join(t.TempDir(), "alias.json")
	if err := os.Symlink(in.BindingPath, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(context.Background(), Request{BindingPath: alias}); err == nil {
		t.Fatal("symlink binding accepted")
	}
	if _, err := service.AppendJournal("test", "Uncheckpointed evidence"); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(context.Background(), in); err == nil {
		t.Fatal("preview implicitly checkpointed dirty source")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Preview(ctx, in); err != context.Canceled {
		t.Fatal("cancellation lost", err)
	}
}
