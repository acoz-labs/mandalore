package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func inlineCall(t *testing.T, a *API, name string, input any) Envelope {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	out := a.Call(context.Background(), name, raw)
	if !out.OK {
		t.Fatalf("%s: %+v", name, out.Error)
	}
	return out
}

func inlineInventory(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	out := map[string][32]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".mandalore" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		out[rel] = sha256.Sum256(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestCorrectionValidationPreservesOriginalRecord(t *testing.T) {
	for _, name := range []string{"memory_remember"} {
		t.Run(name, func(t *testing.T) {
			a := fixture(t)
			original := a.Call(context.Background(), "memory_remember", []byte(`{"kind":"fact","summary":"Preview color","body":"Amber","basis":"user-direction","reason":"Confirmed"}`))
			if !original.OK {
				t.Fatal(original)
			}
			receipt := original.Result.(Receipt)
			call := func(record map[string]any) Envelope {
				raw, _ := json.Marshal(record)
				return a.Call(context.Background(), name, raw)
			}
			record := map[string]any{"kind": "fact", "summary": "Preview color", "body": "Cobalt", "basis": "user-direction", "reason": "User correction", "supersedes": []string{receipt.RecordID}}
			before := inlineInventory(t, a.service.Root())
			for _, withRecordID := range []bool{false, true} {
				if withRecordID {
					record["record_id"] = receipt.RecordID
				}
				result := call(record)
				if result.OK || result.Error.WriteMayHaveOccurred {
					t.Fatal("invalid correction published", result)
				}
				if !reflect.DeepEqual(before, inlineInventory(t, a.service.Root())) {
					t.Fatal("validation changed original signet")
				}
			}
			// Use the original record identity plus its actual revision identity; no
			// duplicate or withdrawal is needed to recover from the invalid request.
			record["supersedes"] = []string{receipt.ID}
			corrected := call(record)
			if !corrected.OK {
				t.Fatal(corrected)
			}
			saved := corrected.Result.(Receipt)
			if saved.RecordID != receipt.RecordID || saved.ID == receipt.ID || !saved.DurableLocally {
				t.Fatal("correction changed identity", saved)
			}
			history := a.Call(context.Background(), "memory_history", []byte(`{"record_id":"`+receipt.RecordID+`"}`))
			if !history.OK {
				t.Fatal(history)
			}
			encoded, _ := json.Marshal(history.Result)
			if !bytes.Contains(encoded, []byte(receipt.ID)) || !bytes.Contains(encoded, []byte(saved.ID)) {
				t.Fatal("correction lost revision history", string(encoded))
			}
		})
	}
}
