package distribution

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPendingPlanCanBeRecoveredWithoutInventingAnotherPlan(t *testing.T) {
	p, err := PlanInstall(context.Background(), localInstallOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	r, err := installForTest(t, p, func(phase string) error {
		if phase == "pending-recorded" {
			return errors.New("synthetic interruption")
		}
		return nil
	})
	if err == nil || r.Pending == "" {
		t.Fatal("fixture did not reach pending state", r, err)
	}
	raw, err := os.ReadFile(r.Pending)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParsePendingInstallPlan(raw)
	if err != nil || !reflect.DeepEqual(got, p) {
		t.Fatal("lost exact original plan", err)
	}
	for _, kind := range []string{"hash", "format", "unknown", "invalid plan", "oversized"} {
		t.Run(kind, func(t *testing.T) {
			var record map[string]any
			if err := json.Unmarshal(raw, &record); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "hash":
				record["plan_sha256"] = "wrong"
			case "format":
				record["format_version"] = 2
			case "unknown":
				record["unexpected"] = true
			case "invalid plan":
				record["plan"] = map[string]any{"prefix": "/"}
			}
			changed, _ := json.Marshal(record)
			if kind == "oversized" {
				changed = append(changed, make([]byte, MaxPendingInstallBytes)...)
			}
			if _, err := ParsePendingInstallPlan(changed); err == nil {
				t.Fatal("invalid pending plan accepted")
			}
		})
	}
	if _, err := os.Stat(filepath.Join(cliState(p.Prefix), "pending.json")); err != nil {
		t.Fatal("read-only parsing changed pending state", err)
	}
}
