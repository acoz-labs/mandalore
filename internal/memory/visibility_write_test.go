package memory

import (
	"reflect"
	"testing"
)

func TestContentWriteCapturesVisibilityAndChecksHeads(t *testing.T) {
	r := revision("revision-next", "revision-current")
	states := map[string]VisibilityState{r.RecordID: {State: "withdrawn", ContentHeads: []string{"revision-current"}, VisibilityHeads: []string{"visibility-withdraw"}}}
	if err := prepareContentWrite(&r, 2, states); err != nil {
		t.Fatal(err)
	}
	if r.Version != 2 || r.VisibilityRefs == nil || !reflect.DeepEqual(*r.VisibilityRefs, []string{"visibility-withdraw"}) {
		t.Fatal(r)
	}
	stale := revision("revision-stale", "revision-old")
	if err := prepareContentWrite(&stale, 2, states); err == nil {
		t.Fatal("accepted stale local content heads")
	}
	stale = revision("revision-stale", "revision-current")
	stale.Version = 2
	refs := []string{"visibility-old"}
	stale.VisibilityRefs = &refs
	if err := prepareContentWrite(&stale, 2, states); err == nil {
		t.Fatal("overwrote stale visibility acknowledgement")
	}
	fresh := revision("revision-fresh")
	if err := prepareContentWrite(&fresh, 2, nil); err != nil {
		t.Fatal(err)
	}
	if fresh.VisibilityRefs == nil || *fresh.VisibilityRefs == nil || len(*fresh.VisibilityRefs) != 0 {
		t.Fatal("new format requires explicit empty context")
	}
	legacy := revision("revision-legacy")
	if err := prepareContentWrite(&legacy, 1, nil); err != nil || legacy.Version != 1 || legacy.VisibilityRefs != nil {
		t.Fatal(legacy, err)
	}
}
