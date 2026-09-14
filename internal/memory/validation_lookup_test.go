package memory

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDeviceValidationMemoReusesOnlySuccessfulChecks(t *testing.T) {
	calls := map[string]int{}
	failure := errors.New("invalid device")
	invalid := map[string]bool{"device-invalid": true}
	check := func(id string) error {
		calls[id]++
		if invalid[id] {
			return failure
		}
		return nil
	}
	lookup := memoizeDeviceValidation(check)
	for range 3 {
		if err := lookup("device-first"); err != nil {
			t.Fatal(err)
		}
	}
	if calls["device-first"] != 1 {
		t.Fatalf("same validation reread a successful device %d times", calls["device-first"])
	}
	if err := lookup("device-second"); err != nil || calls["device-second"] != 1 {
		t.Fatal("different device bypassed validation", err, calls)
	}
	for range 2 {
		if err := lookup("device-invalid"); !errors.Is(err, failure) {
			t.Fatal("failure hidden", err)
		}
	}
	if calls["device-invalid"] != 2 {
		t.Fatal("failed checks cached", calls)
	}
	invalid["device-first"] = true
	nextOperation := memoizeDeviceValidation(check)
	if err := nextOperation("device-first"); !errors.Is(err, failure) || calls["device-first"] != 2 {
		t.Fatal("successful check leaked into another operation", err, calls)
	}
}

func TestGraphDeviceValidationIsFreshOnEveryRead(t *testing.T) {
	for _, change := range []string{"invalid-device", "unknown-source-device"} {
		t.Run(change, func(t *testing.T) {
			s := fixture(t)
			r, err := s.Remember(Write{Kind: "fact", Summary: "Synthetic evidence", Body: "A confirmed fixture fact.", Basis: "observation", Reason: "Validation freshness test"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Recall("", nil, 5, 4096); err != nil {
				t.Fatal(err)
			}
			var path string
			var data []byte
			if change == "invalid-device" {
				path = filepath.Join(s.Root(), "provenance/devices", s.author.DeviceID+".json")
				data, err = json.Marshal(Device{Version: 99, ID: s.author.DeviceID, Label: "Changed fixture device"})
			} else {
				path = filepath.Join(s.Root(), "memory/sources", r.Evidence.SourceRefs[0]+".json")
				var source Source
				if err := readJSON(path, &source); err != nil {
					t.Fatal(err)
				}
				source.DeviceID = "device-not-registered"
				data, err = json.Marshal(source)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			before := snapshot(t, s.Root())
			if _, err := s.Recall("", nil, 5, 4096); err == nil {
				t.Fatal("recall reused an earlier device/source validation")
			}
			if _, err := s.Scopes(); err == nil {
				t.Fatal("scope inventory hid changed invalid provenance")
			}
			if !reflect.DeepEqual(before, snapshot(t, s.Root())) {
				t.Fatal("validation repaired data")
			}
		})
	}
}
