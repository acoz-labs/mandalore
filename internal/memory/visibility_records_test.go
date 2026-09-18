package memory

import (
	"testing"
	"time"
)

func TestRevisionVisibilityFormatBoundary(t *testing.T) {
	r := revision("revision-first")
	r.Supersedes = []string{}
	device := func(string) error { return nil }
	if err := validateRevisionGraph([]Revision{r}, "signet-test", 1, device, device); err != nil {
		t.Fatal(err)
	}
	refs := []string{}
	r.Version, r.VisibilityRefs = 2, &refs
	if err := validateRevisionGraph([]Revision{r}, "signet-test", 1, device, device); err == nil {
		t.Fatal("format1 accepted format2 content")
	}
	if err := validateRevisionGraph([]Revision{r}, "signet-test", 2, device, device); err != nil {
		t.Fatal(err)
	}
	r.VisibilityRefs = nil
	if err := validateRevisionGraph([]Revision{r}, "signet-test", 2, device, device); err == nil {
		t.Fatal("format2 revision omitted causal context")
	}
	r.Version, r.VisibilityRefs = 1, &refs
	if err := validateRevisionGraph([]Revision{r}, "signet-test", 2, device, device); err == nil {
		t.Fatal("legacy revision fabricated causal context")
	}
}

func TestVisibilityRecordProjection(t *testing.T) {
	r := revision("revision-first")
	e := VisibilityEvent{Version: 1, ID: "visibility-first", RecordID: r.RecordID, Action: "withdraw", Observed: []string{r.ID}, Parents: []string{}, Reason: "Withdraw test guidance", RecordedAt: r.RecordedAt, Authorship: r.Authorship}
	device := func(string) error { return nil }
	states, err := ResolveVisibility([]Revision{r}, []VisibilityEvent{e}, device)
	if err != nil || states[r.RecordID].State != "withdrawn" {
		t.Fatal(states, err)
	}
	if len(RecallableHeads([]Revision{r}, states, time.Now())) != 0 {
		t.Fatal("withdrawn content eligible for recall")
	}
	if len(RecallableHeads([]Revision{r}, nil, time.Now())) != 0 {
		t.Fatal("missing state became visible")
	}
	e.RecordID = "record-other"
	if _, err := ResolveVisibility([]Revision{r}, []VisibilityEvent{e}, device); err == nil {
		t.Fatal("accepted event against another record")
	}
	e.RecordID = r.RecordID
	r.Version = 2
	refs := []string{e.ID}
	r.VisibilityRefs = &refs
	if _, err := ResolveVisibility([]Revision{r}, []VisibilityEvent{e}, device); err == nil {
		t.Fatal("accepted cross-graph cycle")
	}
}
