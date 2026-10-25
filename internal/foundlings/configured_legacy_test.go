package foundlings

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

// A configured Manager reads legacy references from configured roots and
// never needs or writes a clone-local connection.
func TestConfiguredLegacyRootsReplaceLocalConnections(t *testing.T) {
	local, r, root := connectedFixture(t)
	m := NewConfigured(local.memory, Config{Legacy: map[string]string{r.FoundlingID: root}})
	before, sourceBefore := fileTree(t, m.memory.Root()), fileTree(t, root)
	v, err := m.Inspect(context.Background(), r.FoundlingID)
	if err != nil || v.State != "available" || v.Connection.Root != root {
		t.Fatal(v, err)
	}
	result, err := m.Search(context.Background(), SearchInput{FoundlingID: r.FoundlingID, Query: "Copper Finch", Limit: 5})
	if err != nil || len(result.Items) != 1 || result.Items[0].Origin.RegistrationRevisionID != r.ID || result.Items[0].Origin.SourcePin != r.Pin {
		t.Fatal(result, err)
	}
	read, err := m.Read(context.Background(), ReadInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: "decisions/old.md", Limit: 4096})
	if err != nil || read.Origin != result.Items[0].Origin {
		t.Fatal(read, err)
	}
	if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) || !reflect.DeepEqual(sourceBefore, fileTree(t, root)) {
		t.Fatal("configured reads wrote the source or signet")
	}
	if _, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, ""); err == nil {
		t.Fatal("configured Manager wrote a local connection")
	}
	// The local Manager is unconnected; the configured one ignores that.
	if v, err := local.Inspect(context.Background(), r.FoundlingID); err != nil || v.State != "unconnected" {
		t.Fatal(v, err)
	}
	unconfigured := NewConfigured(local.memory, Config{})
	if _, err := unconfigured.Search(context.Background(), SearchInput{FoundlingID: r.FoundlingID, Query: "Copper Finch", Limit: 5}); !errors.Is(err, ErrConnection) {
		t.Fatal("unconfigured legacy readable", err)
	}
	if v, err := unconfigured.Inspect(context.Background(), r.FoundlingID); err != nil || v.State != "unconnected" {
		t.Fatal(v, err)
	}
	missing := NewConfigured(local.memory, Config{Legacy: map[string]string{r.FoundlingID: filepath.Join(t.TempDir(), "absent")}})
	if _, err := missing.Search(context.Background(), SearchInput{FoundlingID: r.FoundlingID, Query: "Copper Finch", Limit: 5}); !errors.Is(err, ErrUnavailable) {
		t.Fatal("missing root not unavailable", err)
	}

	author, err := m.memory.As(memory.Authorship{DeviceID: "device-test", Actor: "Example user", Harness: "client-kind"})
	if err != nil {
		t.Fatal(err)
	}
	promoter := NewConfigured(author, Config{Legacy: map[string]string{r.FoundlingID: root}})
	w := memory.Write{Kind: "decision", Summary: "Project naming", Body: "Current direction uses Silver Heron.", Basis: "import", Reason: "Adapt historical evidence"}
	promoted, err := promoter.Promote(context.Background(), PromotionInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: read.Origin.RelativeLocator, ContentSHA256: read.Origin.ContentSHA256, Write: w})
	if err != nil || promoted.Authorship.Harness != "client-kind" {
		t.Fatal(promoted, err)
	}
	if _, err := os.Stat(filepath.Join(m.memory.Root(), ".mandalore/foundlings")); !os.IsNotExist(err) {
		t.Fatal("configured Manager created connection state", err)
	}
}
