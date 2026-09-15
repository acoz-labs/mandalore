package foundlings

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func retrievalFixture(t *testing.T) (*Manager, memory.FoundlingRegistration, string) {
	t.Helper()
	m, r, root := connectedFixture(t)
	if _, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, ""); err != nil {
		t.Fatal(err)
	}
	return m, r, root
}

func TestSearchAndReadReturnBoundedAttributedReferenceNotCurrentMemory(t *testing.T) {
	m, r, root := retrievalFixture(t)
	before, sourceBefore := fileTree(t, m.memory.Root()), fileTree(t, root)
	result, err := m.Search(context.Background(), SearchInput{FoundlingID: r.FoundlingID, Query: "Copper Finch", Limit: 5})
	if err != nil || len(result.Items) != 1 || result.MatchingCount != 1 {
		t.Fatal(result, err)
	}
	hit := result.Items[0]
	if !hit.Unreviewed || hit.Origin.FoundlingID != r.FoundlingID || hit.Origin.RegistrationRevisionID != r.ID || hit.Origin.RelativeLocator != "decisions/old.md" || hit.Origin.SourcePin != r.Pin || len(hit.Origin.ContentSHA256) != 64 || !strings.Contains(hit.Text, "Copper Finch") {
		t.Fatal(hit)
	}
	read, err := m.Read(context.Background(), ReadInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: hit.Origin.RelativeLocator, Limit: 4096})
	if err != nil || !read.Complete || read.Truncated || read.Origin != hit.Origin {
		t.Fatal(read, err)
	}
	empty, err := m.Search(context.Background(), SearchInput{FoundlingID: r.FoundlingID, Query: "no-match", Limit: 5})
	if err != nil || empty.MatchingCount != 0 || len(empty.Items) != 0 || empty.Notice == "" {
		t.Fatal(empty, err)
	}
	if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) || !reflect.DeepEqual(sourceBefore, fileTree(t, root)) {
		t.Fatal("retrieval wrote source or signet")
	}
	current, err := m.memory.Recall("Copper Finch", nil, 5, 4096)
	if err != nil || current.MatchingCount != 0 {
		t.Fatal("reference became current memory", current, err)
	}
}

func TestReadRejectsUnsafeMissingAndStaleSelections(t *testing.T) {
	m, r, root := retrievalFixture(t)
	for _, locator := range []string{"../knowledge.json", "/knowledge.json", "run.sh", "missing.md"} {
		if _, err := m.Read(context.Background(), ReadInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: locator, Limit: 100}); err == nil {
			t.Fatal("invalid locator read", locator)
		}
	}
	if _, err := m.Read(context.Background(), ReadInput{FoundlingID: r.FoundlingID, RegistrationID: "registration-stale", Locator: "knowledge.json", Limit: 100}); err == nil {
		t.Fatal("stale revision read")
	}
	writeFixture(t, root, "knowledge.json", `{"changed":true}`)
	if _, err := m.Search(context.Background(), SearchInput{FoundlingID: r.FoundlingID, Query: "changed", Limit: 5}); err == nil {
		t.Fatal("changed source retrieved")
	}
}

func TestExcerptBoundsUTF8AndSerializedSize(t *testing.T) {
	for _, data := range []string{strings.Repeat("é界🦉", 2000), strings.Repeat("\t", 10000), strings.Repeat("\x01", 10000)} {
		s := &snapshot{Documents: map[string][]byte{"long.md": []byte(data)}}
		c := &Connection{FoundlingID: "foundling-test", RegistrationID: "registration-test", Source: memory.FoundlingSource{Kind: "local", Locator: "source-test"}, Pin: memory.SourcePin{Algorithm: "sha256", Value: strings.Repeat("a", 64)}}
		e, err := excerpt(s, c, "long.md", 0, 8192)
		if err != nil || !utf8.ValidString(e.Text) || !e.Truncated || e.Complete || e.NextOffset == nil || *e.NextOffset != len(e.Text) {
			t.Fatal(e, err)
		}
		encoded, err := json.Marshal(e)
		if err != nil || len(encoded) > 32768 {
			t.Fatal("unbounded excerpt", len(encoded), err)
		}
	}
}

func TestPromotionAdaptsTextAndSeparatesOriginalFromIncorporation(t *testing.T) {
	m, r, root := retrievalFixture(t)
	read, err := m.Read(context.Background(), ReadInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: "decisions/old.md", Limit: 4096})
	if err != nil {
		t.Fatal(err)
	}
	before := fileTree(t, root)
	w := memory.Write{Kind: "decision", Summary: "Project naming", Body: "Current direction uses Silver Heron; Copper Finch was the historical name.", Basis: "import", Reason: "Adapt historical evidence to current user direction"}
	promoted, err := m.Promote(context.Background(), PromotionInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: read.Origin.RelativeLocator, ContentSHA256: read.Origin.ContentSHA256, OriginalAuthor: "Historical author", OriginalRecordedAt: "2024-01-01T00:00:00Z", Write: w})
	if err != nil {
		t.Fatal(err)
	}
	if promoted.Authorship.DeviceID != "device-test" || promoted.Authorship.Harness != "test" || promoted.Body != w.Body {
		t.Fatal(promoted)
	}
	data, err := os.ReadFile(filepath.Join(m.memory.Root(), "memory", "sources", promoted.Evidence.SourceRefs[0]+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var source memory.Source
	err = json.Unmarshal(data, &source)
	if err != nil || source.ExternalOrigin == nil || source.ExternalOrigin.OriginalAuthor != "Historical author" || source.ExternalOrigin.OriginalRecordedAt != "2024-01-01T00:00:00Z" || source.ExternalOrigin.ContentSHA256 != read.Origin.ContentSHA256 {
		t.Fatal(source, err)
	}
	if !reflect.DeepEqual(before, fileTree(t, root)) {
		t.Fatal("promotion changed reference")
	}
	journal, err := m.memory.Journal("", 10)
	if err != nil || len(journal) != 0 {
		t.Fatal("promotion fabricated journal", journal, err)
	}
}

func TestPromotionRefusesStaleHashAndDisconnectedReferenceWithoutWriting(t *testing.T) {
	for _, kind := range []string{"hash", "source changed", "disconnected", "supplied origin", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			m, r, root := retrievalFixture(t)
			read, err := m.Read(context.Background(), ReadInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: "knowledge.json", Limit: 4096})
			if err != nil {
				t.Fatal(err)
			}
			in := PromotionInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: "knowledge.json", ContentSHA256: read.Origin.ContentSHA256, Write: memory.Write{Kind: "fact", Summary: "Project name", Body: "Silver Heron", Basis: "import", Reason: "Retain confirmed historical fact"}}
			ctx := context.Background()
			switch kind {
			case "hash":
				in.ContentSHA256 = strings.Repeat("0", 64)
			case "source changed":
				writeFixture(t, root, "knowledge.json", `{"project":"Changed"}`)
			case "disconnected":
				_, err = m.memory.WriteFoundling(memory.FoundlingWrite{FoundlingID: r.FoundlingID, Name: r.Name, Description: r.Description, Source: r.Source, Pin: r.Pin, State: "disconnected", Supersedes: []string{r.ID}, Reason: "Disconnect"})
				if err != nil {
					t.Fatal(err)
				}
			case "supplied origin":
				in.Write.ExternalOrigin = &memory.ExternalOrigin{}
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			before := fileTree(t, m.memory.Root())
			if _, err := m.Promote(ctx, in); err == nil {
				t.Fatal("invalid promotion accepted")
			}
			if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) {
				t.Fatal("failed promotion wrote evidence")
			}
		})
	}
}

func TestSearchOrderingLimitsAndMultibyteReadContinuation(t *testing.T) {
	m, r, root := connectedFixture(t)
	for i := 0; i < 12; i++ {
		writeFixture(t, root, fmt.Sprintf("%02d.md", i), strings.Repeat("é界🦉 ", 1000)+"needle")
	}
	view, err := Observe(context.Background(), r.Source, root)
	if err != nil {
		t.Fatal(err)
	}
	r, err = m.memory.WriteFoundling(memory.FoundlingWrite{FoundlingID: r.FoundlingID, Name: r.Name, Description: r.Description, Source: r.Source, Pin: view.Pin, State: "active", Supersedes: []string{r.ID}, Reason: "Explicit synthetic repin"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, ""); err != nil {
		t.Fatal(err)
	}
	// Retain the previous explicit large page while smaller defaults are tested
	// separately. The byte budget may otherwise fit fewer than the count limit.
	budget, preview := 32768, 1024
	in := SearchInput{FoundlingID: r.FoundlingID, Query: "NEEDLE", Limit: 10, BudgetBytes: &budget, ExcerptBytes: &preview}
	a, err := m.Search(context.Background(), in)
	if err != nil || a.MatchingCount != 12 || len(a.Items) != 10 || !a.Truncated {
		t.Fatalf("large page: items=%d matches=%d truncated=%v err=%v", len(a.Items), a.MatchingCount, a.Truncated, err)
	}
	b, err := m.Search(context.Background(), in)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("search order unstable", err)
	}
	for i, e := range a.Items {
		if e.Origin.RelativeLocator != fmt.Sprintf("%02d.md", i) || !strings.Contains(e.Text, "needle") || !utf8.ValidString(e.Text) {
			t.Fatal(e)
		}
	}
	encoded, err := json.Marshal(a)
	if err != nil || len(encoded) > 32768 {
		t.Fatal("unbounded search", len(encoded), err)
	}
	part, err := m.Read(context.Background(), ReadInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: "00.md", Limit: 6})
	if err != nil || part.Text != "é界" || part.NextOffset == nil || *part.NextOffset != 5 {
		t.Fatal(part, err)
	}
	next, err := m.Read(context.Background(), ReadInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: "00.md", Offset: *part.NextOffset, Limit: 4})
	if err != nil || next.Text != "🦉" {
		t.Fatal(next, err)
	}
	for _, offset := range []int{-1, 1, 999999} {
		if _, err := m.Read(context.Background(), ReadInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: "00.md", Offset: offset, Limit: 10}); err == nil {
			t.Fatal("invalid byte offset accepted", offset)
		}
	}
	for _, query := range []string{"", strings.Repeat("word ", 17), strings.Repeat("x", 1025)} {
		if _, err := m.Search(context.Background(), SearchInput{FoundlingID: r.FoundlingID, Query: query, Limit: 5}); err == nil {
			t.Fatal("unbounded search query accepted")
		}
	}
}

func TestPromotionUnknownOriginalAndExplicitSupersessionSurviveDisconnect(t *testing.T) {
	m, r, _ := retrievalFixture(t)
	e, err := m.Read(context.Background(), ReadInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: "knowledge.json", Limit: 4096})
	if err != nil {
		t.Fatal(err)
	}
	in := PromotionInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: e.Origin.RelativeLocator, ContentSHA256: e.Origin.ContentSHA256, Write: memory.Write{Kind: "decision", Summary: "Project name", Body: "Silver Heron", Basis: "import", Reason: "Confirmed reference"}}
	first, err := m.Promote(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	in.Write.RecordID = first.RecordID
	in.Write.Supersedes = []string{first.ID}
	in.Write.Body = "Current name is Amber Lark; Silver Heron was previous."
	in.Write.Reason = "Adapt historical reference to the latest confirmed direction"
	second, err := m.Promote(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.memory.WriteFoundling(memory.FoundlingWrite{FoundlingID: r.FoundlingID, Name: r.Name, Description: r.Description, Source: r.Source, Pin: r.Pin, State: "disconnected", Supersedes: []string{r.ID}, Reason: "Disconnect reference, retain learned history"})
	if err != nil {
		t.Fatal(err)
	}
	history, err := m.memory.History(first.RecordID)
	if err != nil || len(history) != 2 {
		t.Fatal(history, err)
	}
	recall, err := m.memory.Recall("Project name", nil, 5, 8192)
	if err != nil || len(recall.Current) != 1 || recall.Current[0].ID != second.ID {
		t.Fatal(recall, err)
	}
	data, err := os.ReadFile(filepath.Join(m.memory.Root(), "memory", "sources", second.Evidence.SourceRefs[0]+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var source memory.Source
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	if source.ExternalOrigin == nil || source.ExternalOrigin.OriginalAuthor != "" || source.ExternalOrigin.OriginalRecordedAt != "" {
		t.Fatal("invented original provenance", source)
	}
	if err := m.memory.Validate(); err != nil {
		t.Fatal(err)
	}
}
