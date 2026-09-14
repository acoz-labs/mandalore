package foundlings

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func connectedFixture(t *testing.T) (*Manager, memory.FoundlingRegistration, string) {
	t.Helper()
	store, err := memory.Create(filepath.Join(t.TempDir(), "signet"), "Example", "device-test", "Synthetic machine")
	if err != nil {
		t.Fatal(err)
	}
	service, err := memory.OpenService(store.Root, memory.Authorship{DeviceID: "device-test", Actor: "Example user", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	source, root := localFixture(t)
	view, err := Observe(context.Background(), source, root)
	if err != nil {
		t.Fatal(err)
	}
	r, err := service.WriteFoundling(memory.FoundlingWrite{Name: "Historical notes", Description: "Reference only", Source: source, Pin: view.Pin, State: "active", Reason: "Selected synthetic notes"})
	if err != nil {
		t.Fatal(err)
	}
	return New(service), r, root
}

func TestConnectionIsExplicitLocalAndReadOnlyToInspect(t *testing.T) {
	m, r, root := connectedFixture(t)
	before := fileTree(t, m.memory.Root())
	v, err := m.Inspect(context.Background(), r.FoundlingID)
	if err != nil || v.State != "unconnected" {
		t.Fatal(v, err)
	}
	if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) {
		t.Fatal("inspection created local state")
	}
	sourceBefore := fileTree(t, root)
	result, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, "")
	if err != nil || !result.Connected || !result.Durable || result.Connection.SignetID != m.memory.ID() || result.Connection.Root != root {
		t.Fatal(result, err)
	}
	before = fileTree(t, m.memory.Root())
	v, err = m.Inspect(context.Background(), r.FoundlingID)
	if err != nil || v.State != "available" || v.Connection.ID != result.Connection.ID {
		t.Fatal(v, err)
	}
	if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) || !reflect.DeepEqual(sourceBefore, fileTree(t, root)) {
		t.Fatal("inspection changed signet or source")
	}
	if _, err := m.memory.Foundling("../invalid"); err == nil {
		t.Fatal("invalid lookup accepted")
	}
	if _, err := m.memory.Foundling("foundling-missing"); err == nil {
		t.Fatal("missing lookup accepted")
	}
}

func TestReconnectRequiresExpectedIdentityAndPreservesUnknownFiles(t *testing.T) {
	m, r, root := connectedFixture(t)
	first, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, "")
	if err != nil {
		t.Fatal(err)
	}
	_, replacement := localFixture(t)
	for _, expected := range []string{"", "connection-wrong"} {
		before := fileTree(t, m.memory.Root())
		if _, err := m.Connect(context.Background(), r.FoundlingID, r.ID, replacement, expected); err == nil {
			t.Fatal("replaced connection without matching expectation")
		}
		if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) {
			t.Fatal("rejected replacement changed local state")
		}
	}
	second, err := m.Connect(context.Background(), r.FoundlingID, r.ID, replacement, first.Connection.ID)
	if err != nil || !second.Connected || second.Connection.ID == first.Connection.ID {
		t.Fatal(second, err)
	}
	path := filepath.Join(m.memory.Root(), ".mandalore", "foundlings", r.FoundlingID+".json")
	writeFixture(t, filepath.Dir(path), filepath.Base(path), `{"unknown":"preserve me"}`)
	before := fileTree(t, m.memory.Root())
	if _, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, second.Connection.ID); err == nil {
		t.Fatal("replaced unknown connection")
	}
	if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) {
		t.Fatal("unknown state lost")
	}
	v, err := m.Inspect(context.Background(), r.FoundlingID)
	if err != nil || v.State != "invalid_connection" {
		t.Fatal(v, err)
	}
}

func TestConnectionWithholdsChangedMissingDisconnectedAndConflictedSources(t *testing.T) {
	for _, state := range []string{"changed", "unavailable", "disconnected", "conflicted", "stale registration"} {
		t.Run(state, func(t *testing.T) {
			m, r, root := connectedFixture(t)
			bound, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, "")
			if err != nil {
				t.Fatal(err)
			}
			want := state
			switch state {
			case "changed":
				writeFixture(t, root, "knowledge.json", `{"changed":true}`)
			case "unavailable":
				if err := os.Rename(root, root+"-moved"); err != nil {
					t.Fatal(err)
				}
				defer os.Rename(root+"-moved", root)
			default:
				w := memory.FoundlingWrite{FoundlingID: r.FoundlingID, Name: r.Name, Description: r.Description, Source: r.Source, Pin: r.Pin, State: "active", Supersedes: []string{r.ID}, Reason: "Explicit update"}
				if state == "disconnected" {
					w.State = "disconnected"
				}
				if _, err := m.memory.WriteFoundling(w); err != nil {
					t.Fatal(err)
				}
				if state == "conflicted" {
					if _, err := m.memory.WriteFoundling(w); err != nil {
						t.Fatal(err)
					}
				}
				if state == "stale registration" {
					want = "changed"
				}
			}
			before := fileTree(t, m.memory.Root())
			v, err := m.Inspect(context.Background(), r.FoundlingID)
			if err != nil || v.State != want {
				t.Fatal(v, err)
			}
			if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) {
				t.Fatal("inspect repaired source or connection")
			}
			if _, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, bound.Connection.ID); err == nil {
				t.Fatal("invalid or stale source connected")
			}
		})
	}
}

func TestConnectionRejectsSignetOverlapRedirectionAndWrongOwner(t *testing.T) {
	m, r, root := connectedFixture(t)
	for _, path := range []string{m.memory.Root(), filepath.Dir(m.memory.Root()), filepath.Join(m.memory.Root(), "memory")} {
		if _, err := m.Connect(context.Background(), r.FoundlingID, r.ID, path, ""); err == nil {
			t.Fatal("overlapping source accepted")
		}
	}
	bound, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, "")
	if err != nil {
		t.Fatal(err)
	}
	other, otherRegistration, _ := connectedFixture(t)
	encoded, err := json.Marshal(bound.Connection)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, other.memory.Root(), ".mandalore/foundlings/"+otherRegistration.FoundlingID+".json", string(encoded))
	v, err := other.Inspect(context.Background(), otherRegistration.FoundlingID)
	if err != nil || v.State != "invalid_connection" {
		t.Fatal("other signet connection accepted", v, err)
	}
	local := filepath.Join(m.memory.Root(), ".mandalore", "foundlings")
	if err := os.Rename(local, local+"-saved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(local+"-saved", local); err != nil {
		t.Fatal(err)
	}
	v, err = m.Inspect(context.Background(), r.FoundlingID)
	if err != nil || v.State != "invalid_connection" {
		t.Fatal("redirected local state accepted", v, err)
	}
}

func TestConnectionReconnectsMissingPathAndSameRequestKeepsIdentity(t *testing.T) {
	m, r, root := connectedFixture(t)
	first, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, root+"-moved"); err != nil {
		t.Fatal(err)
	}
	defer os.Rename(root+"-moved", root)
	second, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root+"-moved", first.Connection.ID)
	if err != nil || !second.Durable || first.Connection.ID == second.Connection.ID {
		t.Fatal(second, err)
	}
	repeated, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root+"-moved", second.Connection.ID)
	if err != nil || repeated.Connection != second.Connection {
		t.Fatal("same configuration changed identity", repeated, err)
	}
	v, err := m.Inspect(context.Background(), r.FoundlingID)
	if err != nil || v.State != "available" {
		t.Fatal(v, err)
	}
}

func TestConnectionTamperingCancellationAndAbsentLocalDirectory(t *testing.T) {
	m, r, root := connectedFixture(t)
	before := fileTree(t, m.memory.Root())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Connect(ctx, r.FoundlingID, r.ID, root, ""); err == nil {
		t.Fatal("cancelled connection accepted")
	}
	if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) {
		t.Fatal("cancelled operation wrote state")
	}
	bound, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, "")
	if err != nil {
		t.Fatal(err)
	}
	modified := bound.Connection
	modified.Root = filepath.Dir(root)
	encoded, err := json.Marshal(modified)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, m.memory.Root(), connectionPath(r.FoundlingID), string(encoded))
	v, err := m.Inspect(context.Background(), r.FoundlingID)
	if err != nil || v.State != "invalid_connection" {
		t.Fatal("edited metadata retained old connection identity", v, err)
	}
	local := filepath.Join(m.memory.Root(), ".mandalore")
	if err := os.Rename(local, local+"-saved"); err != nil {
		t.Fatal(err)
	}
	defer os.Rename(local+"-saved", local)
	before = fileTree(t, m.memory.Root())
	v, err = m.Inspect(context.Background(), r.FoundlingID)
	if err != nil || v.State != "unconnected" {
		t.Fatal(v, err)
	}
	if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) {
		t.Fatal("read recreated missing local directory")
	}
}

func TestPublishedConnectionReportsDurabilityFailureWithoutRollback(t *testing.T) {
	m, r, root := connectedFixture(t)
	sourceBefore := fileTree(t, root)
	result, err := m.connect(context.Background(), r.FoundlingID, r.ID, root, "", func(*os.Root, string) error { return errors.New("injected directory sync failure") })
	if err == nil || !result.Connected || result.Durable || result.Connection.ID == "" {
		t.Fatal("partial publication hidden", result, err)
	}
	v, err := m.Inspect(context.Background(), r.FoundlingID)
	if err != nil || v.State != "available" || v.Connection.ID != result.Connection.ID {
		t.Fatal("published connection lost", v, err)
	}
	entries, err := os.ReadDir(filepath.Join(m.memory.Root(), ".mandalore", "foundlings"))
	if err != nil || len(entries) != 1 || entries[0].Name() != r.FoundlingID+".json" {
		t.Fatal("temporary publication file retained", entries, err)
	}
	if !reflect.DeepEqual(sourceBefore, fileTree(t, root)) {
		t.Fatal("failure changed source")
	}
	if _, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, ""); err == nil {
		t.Fatal("blind retry replaced published connection")
	}
	if recovered, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, result.Connection.ID); err != nil || !recovered.Durable {
		t.Fatal("explicit recovery failed", recovered, err)
	}
}
