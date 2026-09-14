package memory

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHistoryPreservesExactExtensionNumberFromStorage(t *testing.T) {
	s := fixtureStore(t)
	r := revision("revision-precise")
	r.Extensions = map[string]any{"example.precise": json.Number("9007199254740993123456789.123456789")}
	if err := s.Put(r); err != nil {
		t.Fatal(err)
	}
	history, err := s.History(r.RecordID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(history)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "9007199254740993123456789.123456789") {
		t.Fatal("history rounded stored extension number", string(b))
	}
}
