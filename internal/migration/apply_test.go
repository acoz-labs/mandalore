package migration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestApplyPreservesHistoryOriginalBytesAndSeparateConversionOrigin(t *testing.T) {
	o := legacyFixture(t)
	before := tree(t, o.Source)
	p, err := Preflight(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Apply(context.Background(), ApplyInput{Plan: p, WritersStopped: true})
	if err != nil {
		t.Fatal(r, err)
	}
	if !r.Published || r.Phase != "published" || r.Staging != "" {
		t.Fatal(r)
	}
	if !reflect.DeepEqual(before, tree(t, o.Source)) {
		t.Fatal("source mutated")
	}
	for name, data := range before {
		if data == "directory" || strings.HasPrefix(name, ".git/") || strings.HasPrefix(name, ".my-friday/") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(o.Output, "original", name))
		if err != nil || string(b) != data {
			t.Fatalf("original snapshot differs at %s: %v", name, err)
		}
	}
	root := filepath.Join(o.Output, "signet")
	for _, name := range []string{"bank.json", ".git", "README.md", "provenance/changes", ".my-friday"} {
		if _, err := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("unexpected converted content: %s", name)
		}
	}
	s, err := memory.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	history, err := s.History("record-project")
	if err != nil || len(history) != 3 {
		t.Fatal(history, err)
	}
	for _, h := range history {
		encoded, err := json.Marshal(h)
		if err != nil || !strings.Contains(string(encoded), "9007199254740993123456789.123456789") {
			t.Fatal("retrieval rounded preserved extension", err)
		}
		if h.Scope != (memory.Scope{Kind: "signet", ID: "bank-example"}) || h.Authorship.Actor != "Original author" || h.Authorship.DeviceID != "device-original" || h.Authorship.Harness != "pi" || h.RecordedAt != "2026-09-06T12:00:00Z" {
			t.Fatal("original semantic identity changed", h)
		}
		b, err := os.ReadFile(filepath.Join(root, "memory/records/record-project", h.ID+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "9007199254740993123456789.123456789") || !strings.Contains(string(b), "Literal assistant and my-friday") {
			t.Fatal("lossy numeric or broad text conversion")
		}
	}
	packet, err := s.Recall(memory.Query{Scope: memory.Scope{Kind: "signet", ID: "bank-example"}}, time.Now())
	if err != nil || len(packet.Conflicts) != 1 || len(packet.Current) != 0 {
		t.Fatal(packet, err)
	}
	journal, err := s.Journal("", 100)
	if err != nil || len(journal) != 2 {
		t.Fatal(journal, err)
	}
	var conversion memory.JournalEntry
	for _, e := range journal {
		if e.Kind == "migration" {
			conversion = e
		}
	}
	if conversion.Authorship.DeviceID == "device-original" || conversion.Authorship.Actor != o.Actor || conversion.Authorship.Harness != "cli" || !strings.Contains(conversion.Summary, p.SourceSHA256) || strings.Contains(conversion.Summary, o.Source) {
		t.Fatal(conversion)
	}
	var receipt Receipt
	b, err := os.ReadFile(r.Receipt)
	if err != nil || json.Unmarshal(b, &receipt) != nil {
		t.Fatal(err)
	}
	if receipt.ConversionDevice.ID != conversion.Authorship.DeviceID || receipt.ConversionDevice.Label != o.DeviceLabel || receipt.Preserved.Revisions != 3 {
		t.Fatal(receipt)
	}
	newBinding, err := binding.Bind(root, filepath.Join(filepath.Dir(o.Source), "new-binding.json"), "New writer laptop", "Example")
	if err != nil || newBinding.DeviceID == conversion.Authorship.DeviceID || newBinding.DeviceID == "device-original" {
		t.Fatal(newBinding, err)
	}
	if !reflect.DeepEqual(before, tree(t, o.Source)) {
		t.Fatal("binding mutated old source")
	}
}

func TestApplyRefusesStalePlansAndRequiresWriterAcknowledgement(t *testing.T) {
	for _, test := range []string{"ack", "source", "binding", "plan", "output", "lock"} {
		t.Run(test, func(t *testing.T) {
			o := legacyFixture(t)
			if test == "binding" {
				o.Binding = filepath.Join(filepath.Dir(o.Source), "old-binding.json")
				put(t, filepath.Dir(o.Source), "old-binding.json", []byte(`{"schema_version":1,"bank_id":"bank-example","root":`+string(marshal(o.Source))+`,"device_id":"device-original","actor":"Original"}`))
			}
			p, err := Preflight(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			in := ApplyInput{Plan: p, WritersStopped: true}
			switch test {
			case "ack":
				in.WritersStopped = false
			case "source":
				put(t, o.Source, "README.md", []byte("changed"))
			case "binding":
				put(t, filepath.Dir(o.Source), "old-binding.json", []byte(`{}`))
			case "plan":
				in.Plan.Counts.Revisions++
			case "output":
				if err := os.Mkdir(o.Output, 0700); err != nil {
					t.Fatal(err)
				}
			case "lock":
				put(t, o.Source, ".my-friday/write.lock", nil)
			}
			before := tree(t, filepath.Dir(o.Source))
			r, err := Apply(context.Background(), in)
			if err == nil || r.Published {
				t.Fatal("stale plan accepted", r, err)
			}
			if !reflect.DeepEqual(before, tree(t, filepath.Dir(o.Source))) {
				t.Fatal("refusal changed files")
			}
		})
	}
}

func TestApplyPublicationFailureRetainsOwnedStageAndDoesNotOverwrite(t *testing.T) {
	o := legacyFixture(t)
	p, err := Preflight(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	before := tree(t, o.Source)
	r, err := apply(context.Background(), ApplyInput{Plan: p, WritersStopped: true}, func(from, to string) error {
		put(t, to, "sentinel", []byte("another creator won"))
		return publishDirectory(from, to)
	}, syncDirectory)
	if err == nil || r.Published || r.Staging == "" || r.Phase != "validated" {
		t.Fatal(r, err)
	}
	if b, _ := os.ReadFile(filepath.Join(o.Output, "sentinel")); string(b) != "another creator won" {
		t.Fatal("overwrote output")
	}
	if _, err := memory.Open(filepath.Join(r.Staging, "signet")); err != nil {
		t.Fatal("lost valid retained stage", err)
	}
	if !reflect.DeepEqual(before, tree(t, o.Source)) {
		t.Fatal("failed conversion changed source")
	}
}

func TestApplyReportsPublishedBundleWhenParentSyncFails(t *testing.T) {
	o := legacyFixture(t)
	p, err := Preflight(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	r, err := apply(context.Background(), ApplyInput{Plan: p, WritersStopped: true}, publishDirectory, func(dir string) error {
		if dir == filepath.Dir(o.Output) {
			return errors.New("injected parent sync failure")
		}
		return syncDirectory(dir)
	})
	if err == nil || !r.Published || r.Phase != "published" || r.Staging != "" {
		t.Fatal(r, err)
	}
	if _, err := memory.Open(filepath.Join(o.Output, "signet")); err != nil {
		t.Fatal(err)
	}
}

func TestApplyDetectsSourceChangeDuringStaging(t *testing.T) {
	o := legacyFixture(t)
	p, err := Preflight(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	changed := false
	r, err := apply(context.Background(), ApplyInput{Plan: p, WritersStopped: true}, publishDirectory, func(dir string) error {
		if !changed {
			changed = true
			put(t, o.Source, "README.md", []byte("concurrent source change"))
		}
		return syncDirectory(dir)
	})
	if err == nil || r.Published || r.Staging == "" {
		t.Fatal(r, err)
	}
	if _, err := os.Lstat(o.Output); !os.IsNotExist(err) {
		t.Fatal("published changed source")
	}
}

func TestPinnedPublicFixtureConvertsWithoutRewritingOtherScopes(t *testing.T) {
	root, err := filepath.Abs("testdata/bank-v1")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "conversion")
	p, err := Preflight(context.Background(), Options{Source: root, Output: out, Actor: "Example", DeviceLabel: "Migration laptop"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Counts.Revisions != 2 || p.Counts.SourceChanges != 1 || p.Counts.Devices != 2 {
		t.Fatal(p.Counts)
	}
	r, err := Apply(context.Background(), ApplyInput{Plan: p, WritersStopped: true})
	if err != nil {
		t.Fatal(r, err)
	}
	s, err := memory.Open(r.Signet)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := s.Recall(memory.Query{Scope: memory.Scope{Kind: "signet", ID: p.SourceID}}, time.Now())
	if err != nil || len(packet.Current) != 1 || !strings.Contains(packet.Current[0].Body, "Silver Heron") {
		t.Fatal(packet, err)
	}
	if _, err := os.Lstat(filepath.Join(out, "original/provenance/changes/change-readme.json")); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"project", "account", "task"} {
		t.Run(kind, func(t *testing.T) {
			o := legacyFixture(t)
			for _, id := range []string{"revision-root", "revision-left", "revision-right"} {
				name := "memory/records/record-project/" + id + ".json"
				b, err := os.ReadFile(filepath.Join(o.Source, name))
				if err != nil {
					t.Fatal(err)
				}
				put(t, o.Source, name, []byte(strings.Replace(string(b), `"kind":"assistant"`, `"kind":"`+kind+`"`, 1)))
			}
			p, err := Preflight(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			r, err := Apply(context.Background(), ApplyInput{Plan: p, WritersStopped: true})
			if err != nil {
				t.Fatal(r, err)
			}
			for _, id := range []string{"revision-root", "revision-left", "revision-right"} {
				name := "memory/records/record-project/" + id + ".json"
				a, _ := os.ReadFile(filepath.Join(o.Source, name))
				b, _ := os.ReadFile(filepath.Join(r.Signet, name))
				if string(a) != string(b) {
					t.Fatal("non-bank scope revision rewritten")
				}
			}
		})
	}
}
