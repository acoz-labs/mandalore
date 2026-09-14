package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationCLIAndTypedAdministrationSharePlanAndPublication(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source, output := filepath.Join(base, "old"), filepath.Join(base, "conversion")
	for _, dir := range []string{"memory/records", "memory/sources", "memory/events", "provenance/devices", "provenance/changes"} {
		if err := os.MkdirAll(filepath.Join(source, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "bank.json"), []byte(`{"schema_version":1,"id":"bank-example","name":"Example"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	args := []string{"migration", "preflight", "--source", source, "--output", output, "--device-label", "Migration laptop", "--actor", "Example"}
	if code := run(context.Background(), args, strings.NewReader(""), &out, &out); code != 0 {
		t.Fatal(code, out.String())
	}
	preview := append([]byte{}, out.Bytes()...)
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(preview, &envelope); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"migration", "apply"}, {"migration", "apply", "--writers-stopped", "--read-only"}} {
		out.Reset()
		if code := run(context.Background(), args, bytes.NewReader(preview), &out, &out); code == 0 {
			t.Fatal("unacknowledged/read-only apply succeeded")
		}
		if _, err := os.Lstat(output); !os.IsNotExist(err) {
			t.Fatal("refusal created output")
		}
	}
	out.Reset()
	if code := run(context.Background(), []string{"migration", "apply", "--writers-stopped"}, bytes.NewReader(preview), &out, &out); code != 0 || !strings.Contains(out.String(), `"published":true`) {
		t.Fatal(code, out.String())
	}
	out.Reset()
	input := []byte(`{"plan":` + string(envelope.Result) + `,"writers_stopped":true}`)
	if code := run(context.Background(), []string{"call", "migration_apply"}, bytes.NewReader(input), &out, &out); code == 0 || !strings.Contains(out.String(), `"migration_result"`) || !strings.Contains(out.String(), `"write_may_have_occurred":false`) {
		t.Fatal("typed retry must refuse without overwriting", code, out.String())
	}
}
