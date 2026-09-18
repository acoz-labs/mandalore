package exportreport

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func previewFixture(t *testing.T) (Request, *memory.Service) {
	t.Helper()
	s := snapshotService(t)
	r, err := s.Remember(memory.Write{Kind: "fact", Summary: "PRIVATE_SUMMARY", Body: "PRIVATE_BODY", Basis: "observation", Reason: "PRIVATE_REASON"})
	if err != nil {
		t.Fatal(err)
	}
	b := filepath.Join(t.TempDir(), "config", "binding.json")
	if _, err := binding.Bind(s.Root(), b, "PRIVATE_DEVICE", "PRIVATE_ACTOR"); err != nil {
		t.Fatal(err)
	}
	return Request{BindingPath: b, Destination: filepath.Join(t.TempDir(), "report"), Selection: Selection{RecordIDs: []string{r.RecordID}}}, s
}

func TestPreviewIsDeterministicMetadataOnlyAndDoesNotWrite(t *testing.T) {
	in, _ := previewFixture(t)
	a, err := Preview(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Preview(context.Background(), in)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("unstable preview", err)
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "PRIVATE_") {
		t.Fatal("preview leaked source contents")
	}
	if a.ReportBytes < 1 || len(a.ReportSHA256) != 64 || len(a.SourceSHA256) != 64 || len(a.BindingSHA256) != 64 {
		t.Fatal("incomplete preview")
	}
	entries, err := os.ReadDir(filepath.Dir(in.Destination))
	if err != nil || len(entries) != 0 {
		t.Fatal("preview wrote destination", err)
	}
}

func TestPreviewRejectsOverlapsExistingAndUnresolvablePaths(t *testing.T) {
	for _, mode := range []string{"bank", "state", "binding", "existing", "symlink", "dangling", "missing-parent"} {
		t.Run(mode, func(t *testing.T) {
			in, s := previewFixture(t)
			switch mode {
			case "bank":
				in.Destination = filepath.Join(s.Root(), "export")
			case "state":
				in.Destination = filepath.Join(s.Root(), ".mandalore", "export")
			case "binding":
				in.Destination = filepath.Join(filepath.Dir(in.BindingPath), "export")
			case "existing":
				if err := os.Mkdir(in.Destination, 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(s.Root(), in.Destination); err != nil {
					t.Fatal(err)
				}
			case "dangling":
				if err := os.Symlink(in.Destination+"-absent", in.Destination); err != nil {
					t.Fatal(err)
				}
			case "missing-parent":
				in.Destination = filepath.Join(in.Destination, "report")
			}
			if _, err := Preview(context.Background(), in); err == nil {
				t.Fatal("unsafe destination accepted")
			}
		})
	}
}

func TestPreviewPinsSourceAndBindingChanges(t *testing.T) {
	in, s := previewFixture(t)
	a, err := Preview(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendJournal("test", "Unselected source change"); err != nil {
		t.Fatal(err)
	}
	b, err := Preview(context.Background(), in)
	if err != nil || a.SourceSHA256 == b.SourceSHA256 {
		t.Fatal("source change invisible", err)
	}
	data, err := os.ReadFile(in.BindingPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(in.BindingPath, append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Preview(context.Background(), in)
	if err != nil || b.BindingSHA256 == c.BindingSHA256 {
		t.Fatal("binding change invisible", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Preview(ctx, in); err != context.Canceled {
		t.Fatal("cancellation lost", err)
	}
}

func TestPreviewProtectsDisconnectedReferencesAndAncestorAliases(t *testing.T) {
	in, s := previewFixture(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "old.md"), []byte("historical reference"), 0600); err != nil {
		t.Fatal(err)
	}
	source := memory.FoundlingSource{Kind: "local", Locator: "source-synthetic-reference"}
	view, err := foundlings.Observe(context.Background(), source, root)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.WriteFoundling(memory.FoundlingWrite{Name: "Synthetic", Description: "Historical only", Source: source, Pin: view.Pin, State: "active", Reason: "Test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := foundlings.New(s).Connect(context.Background(), r.FoundlingID, r.ID, root, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.WriteFoundling(memory.FoundlingWrite{FoundlingID: r.FoundlingID, Name: r.Name, Description: r.Description, Source: r.Source, Pin: r.Pin, State: "disconnected", Supersedes: []string{r.ID}, Reason: "Test disconnected protection"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(context.Background(), in); err != nil {
		t.Fatal("unrelated destination refused", err)
	}
	for _, protected := range []string{root, s.Root(), filepath.Dir(in.BindingPath)} {
		alias := filepath.Join(t.TempDir(), "alias")
		if err := os.Symlink(protected, alias); err != nil {
			t.Fatal(err)
		}
		request := in
		request.Destination = filepath.Join(alias, "report")
		if _, err := Preview(context.Background(), request); err == nil {
			t.Fatal("alias into protected directory accepted")
		}
	}
	if err := os.Rename(root, root+"-moved"); err != nil {
		t.Fatal(err)
	}
	defer os.Rename(root+"-moved", root)
	if _, err := Preview(context.Background(), in); err == nil {
		t.Fatal("uninspectable reference root accepted")
	}
}
