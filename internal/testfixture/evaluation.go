package testfixture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

// Evaluation is a checked-in synthetic selection set: records to load and
// cases, each a question with the one record that answers it.
type Evaluation struct {
	Wrappers struct {
		Tune    []string `json:"tune"`
		HeldOut []string `json:"held_out"`
	} `json:"wrappers"`
	NoMemory struct {
		Tune    []string `json:"tune"`
		HeldOut []string `json:"held_out"`
	} `json:"no_memory"`
	Session []string `json:"session"`
	Cases   []struct {
		Name       string `json:"name"`
		Expect     string `json:"expect"`
		Question   string `json:"question"`
		Paraphrase string `json:"paraphrase"`
	} `json:"cases"`
	Records []struct {
		Record  string              `json:"record"`
		Kind    string              `json:"kind"`
		Scope   memory.Scope        `json:"scope"`
		History []EvaluationVersion `json:"history"`
		EvaluationVersion
	} `json:"records"`
	// Scopes maps each record ID to its scope in the loaded signet.
	Scopes map[string]memory.Scope `json:"-"`
}

// EvaluationVersion is one revision's content.
type EvaluationVersion struct {
	Summary    string `json:"summary"`
	Body       string `json:"body"`
	RecordedAt string `json:"recorded_at"`
}

// LoadEvaluation writes the set at path into a new signet in a test directory
// and returns a service over it. A record's earlier revisions precede it in a
// single supersession chain; a signet-kind scope is the new signet.
func LoadEvaluation(t testing.TB, path string) (*memory.Service, Evaluation) {
	t.Helper()
	var e Evaluation
	data, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(data, &e)
	}
	if err != nil {
		t.Fatal(err)
	}
	store, err := memory.Create(filepath.Join(t.TempDir(), "signet"), "Synthetic selection evaluation", "device-evaluation", "Synthetic evaluation host")
	if err != nil {
		t.Fatal(err)
	}
	author := memory.Authorship{DeviceID: "device-evaluation", Actor: "Synthetic evaluator", Harness: "test"}
	e.Scopes = map[string]memory.Scope{}
	for _, r := range e.Records {
		scope := r.Scope
		if scope.Kind == "signet" {
			scope.ID = store.Signet.ID
		}
		e.Scopes[r.Record] = scope
		parent := ""
		for i, v := range append(r.History, r.EvaluationVersion) {
			rev := memory.Revision{Version: 1, ID: fmt.Sprintf("revision-%s-%d", r.Record[len("record-"):], i), RecordID: r.Record, Kind: r.Kind, Scope: scope, Summary: v.Summary, Body: v.Body, Sensitivity: "private", Volatility: "stable", RecordedAt: v.RecordedAt, EffectiveFrom: v.RecordedAt, Authorship: author, Evidence: memory.Evidence{Basis: "observation", Confidence: "high"}, Supersedes: []string{}, ChangeReason: "Synthetic evaluation revision"}
			if parent != "" {
				rev.Supersedes = []string{parent}
			}
			writeRevision(t, store.Root, rev)
			parent = rev.ID
		}
	}
	if err := store.Validate(); err != nil {
		t.Fatal("invalid evaluation set", err)
	}
	service, err := memory.OpenService(store.Root, author)
	if err != nil {
		t.Fatal(err)
	}
	return service, e
}
