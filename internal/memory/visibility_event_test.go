package memory

import (
	"errors"
	"testing"
)

func TestVisibilityEventMetadata(t *testing.T) {
	valid := func() VisibilityEvent {
		return VisibilityEvent{Version: 1, ID: "visibility-test", RecordID: "record-test", Action: "withdraw", Observed: []string{"revision-test"}, Parents: []string{}, Reason: "No longer use this guidance", RecordedAt: "2026-09-18T12:00:00Z", Authorship: Authorship{DeviceID: "device-test", Actor: "Example", Harness: "test"}}
	}
	device := func(id string) error {
		if id != "device-test" {
			return errors.New("unknown device")
		}
		return nil
	}
	if err := validateVisibilityEvent(valid(), device); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*VisibilityEvent){
		"version":        func(e *VisibilityEvent) { e.Version = 2 },
		"id":             func(e *VisibilityEvent) { e.ID = "../bad" },
		"record":         func(e *VisibilityEvent) { e.RecordID = "" },
		"action":         func(e *VisibilityEvent) { e.Action = "delete" },
		"reason":         func(e *VisibilityEvent) { e.Reason = " " },
		"date":           func(e *VisibilityEvent) { e.RecordedAt = "yesterday" },
		"device":         func(e *VisibilityEvent) { e.Authorship.DeviceID = "device-other" },
		"observed nil":   func(e *VisibilityEvent) { e.Observed = nil },
		"observed empty": func(e *VisibilityEvent) { e.Observed = []string{} },
		"parent nil":     func(e *VisibilityEvent) { e.Parents = nil },
		"duplicate":      func(e *VisibilityEvent) { e.Observed = []string{"revision-test", "revision-test"} },
		"bad parent":     func(e *VisibilityEvent) { e.Parents = []string{"../bad"} },
		"self parent":    func(e *VisibilityEvent) { e.Parents = []string{e.ID} },
	} {
		t.Run(name, func(t *testing.T) {
			e := valid()
			mutate(&e)
			if err := validateVisibilityEvent(e, device); err == nil {
				t.Fatal("accepted invalid event")
			}
		})
	}
}
