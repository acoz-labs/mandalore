package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestReleaseInspectionIsCLIOnlyUnboundAndReadOnly(t *testing.T) {
	found := false
	for _, op := range Catalog() {
		if op.Name == "release_inspect" {
			found = true
			if !op.CLIOnly || op.RequiresBinding || !op.ReadOnly || !op.Network {
				t.Fatal("release inspection has wrong authority flags", op)
			}
		}
	}
	if !found {
		t.Fatal("release inspection missing from typed discovery")
	}
	raw, _ := json.Marshal(map[string]string{"candidate": t.TempDir()})
	out := New(nil, true).Call(context.Background(), "release_inspect", raw)
	if out.OK || out.Error.Code != "release.failed" || out.Error.WriteMayHaveOccurred || out.Error.InspectBeforeRetry {
		t.Fatal("local candidate failure has wrong effect report", out)
	}
	if strings.Contains(out.Error.Message, "binding") {
		t.Fatal("unbound release inspection required a memory binding")
	}
	for _, data := range []string{`{"version":"1.0.0","candidate":"/example"}`, `{"version":"../latest"}`, `{"unexpected":true}`} {
		out := New(nil, true).Call(context.Background(), "release_inspect", []byte(data))
		if out.OK || out.Error.Code != "input.invalid" {
			t.Fatal("invalid selection did not fail before network", out)
		}
	}
}
