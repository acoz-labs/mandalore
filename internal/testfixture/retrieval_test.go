package testfixture

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCorpusCountsAndCurrentHeads(t *testing.T) {
	for _, c := range []Corpus{{Records: 20, Depth: 1}, {Records: 20, Depth: 2}, {Records: 20, Depth: 3, ConflictEvery: 17}, {Records: 20, Depth: 1, ConflictEvery: 17}} {
		t.Run(c.Name(), func(t *testing.T) {
			f := New(t, c)
			expectedRevisions, expectedConflicts := c.Records*c.Depth, 0
			if c.ConflictEvery > 0 {
				expectedConflicts = (c.Records-1)/c.ConflictEvery + 1
				extra := 1
				if c.Depth == 1 {
					extra = 2
				}
				expectedRevisions += expectedConflicts * extra
			}
			if f.Revisions != expectedRevisions || f.Conflicts != expectedConflicts || f.Records != c.Records || f.Files <= 2*f.Revisions || f.Bytes <= int64(f.Revisions*100) {
				t.Fatalf("bad fixture counts: %+v", f)
			}
			p, err := f.Service.Recall(f.Query, &f.Scope, 5, 8192)
			if err != nil || len(p.Current) != 1 || p.Current[0].ID != f.ExpectedID {
				t.Fatal("narrow fixture result", p, err)
			}
			scopes, err := f.Service.Scopes()
			if err != nil || len(scopes) != 10 {
				t.Fatal("scope counts", scopes, err)
			}
			records, conflicts, current := 0, 0, 0
			for _, scope := range scopes {
				records += scope.RecordCount
				p, err := f.Service.Recall("", &scope.Scope, 50, 32768)
				if err != nil || p.Truncated {
					t.Fatal(p, err)
				}
				conflicts += p.ConflictCount
				current += p.MatchingCount
				data, err := json.Marshal(p)
				if err != nil || len(data) > 32768 {
					t.Fatal("payload", len(data), err)
				}
			}
			if records != c.Records || conflicts != f.Conflicts || current+conflicts != c.Records {
				t.Fatal("heads/records accounting", records, conflicts, current)
			}
		})
	}
}

func TestNearestRankMeasurementPercentiles(t *testing.T) {
	values := make([]time.Duration, 20)
	for i := range values {
		values[i] = time.Duration(20-i) * time.Millisecond
	}
	if percentile(values, .5) != 10*time.Millisecond || percentile(values, .95) != 19*time.Millisecond {
		t.Fatal("invalid percentile accounting")
	}
}
