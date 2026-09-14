package migration

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEmptyPortableDirectoriesArePreservedAndFingerprintChanges(t *testing.T) {
	o := legacyFixture(t)
	p, err := Preflight(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(o.Source, "memory/records/record-empty"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), ApplyInput{Plan: p, WritersStopped: true}); err == nil {
		t.Fatal("changed directory inventory accepted")
	}
	p, err = Preflight(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Apply(context.Background(), ApplyInput{Plan: p, WritersStopped: true})
	if err != nil {
		t.Fatal(r, err)
	}
	st, err := os.Stat(filepath.Join(o.Output, "original/memory/records/record-empty"))
	if err != nil || !st.IsDir() {
		t.Fatal("original empty directory was lost", err)
	}
}

func TestPreflightRejectsTraversalNullUnknownAndUnsupportedScopes(t *testing.T) {
	for _, test := range []string{"scope", "null", "unknown", "duplicate", "path", "overlap", "relative", "local symlink"} {
		t.Run(test, func(t *testing.T) {
			o := legacyFixture(t)
			source := o.Source
			name := "memory/records/record-project/revision-root.json"
			data, err := os.ReadFile(filepath.Join(source, name))
			if err != nil {
				t.Fatal(err)
			}
			switch test {
			case "scope":
				data = []byte(strings.Replace(string(data), `"kind":"assistant"`, `"kind":"signet"`, 1))
			case "null":
				data = []byte(strings.Replace(string(data), `"actor":"Original author"`, `"actor":null`, 1))
			case "unknown":
				data = []byte(strings.Replace(string(data), `"schema_version":1`, `"schema_version":1,"unexpected":"private-canary"`, 1))
			case "duplicate":
				data = []byte(strings.Replace(string(data), `"actor":"Original author"`, `"actor":"Original author","actor":"second"`, 1))
			case "path":
				data = []byte(strings.Replace(string(data), `"id":"revision-root"`, `"id":"../outside"`, 1))
			case "overlap":
				o.Output = filepath.Join(o.Source, "output")
			case "relative":
				o.Source = "relative"
			case "local symlink":
				if err := os.Rename(filepath.Join(source, ".my-friday"), filepath.Join(filepath.Dir(source), "local")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(filepath.Dir(source), "local"), filepath.Join(source, ".my-friday")); err != nil {
					t.Fatal(err)
				}
			}
			put(t, source, name, data)
			before := tree(t, filepath.Dir(source))
			if _, err := Preflight(context.Background(), o); err == nil {
				t.Fatal("invalid input accepted")
			}
			if !reflect.DeepEqual(before, tree(t, filepath.Dir(source))) {
				t.Fatal("invalid preflight wrote state")
			}
		})
	}
}

func TestPortableTotalSizeLimit(t *testing.T) {
	o := legacyFixture(t)
	for _, dir := range []string{"memory/records/record-one", "memory/records/record-two", "memory/records/record-three", "memory/records/record-four", "memory/records/record-five", "memory/records/record-six", "memory/records/record-seven", "memory/records/record-eight", "memory/records/record-nine", "memory/records/record-ten", "memory/records/record-eleven", "memory/records/record-twelve", "memory/records/record-thirteen", "memory/records/record-fourteen", "memory/records/record-fifteen", "memory/records/record-sixteen"} {
		put(t, o.Source, dir+"/.gitkeep", []byte(strings.Repeat("x", MaxFileBytes)))
	}
	if _, err := Preflight(context.Background(), o); err == nil || !strings.Contains(err.Error(), "64 MiB") {
		t.Fatal("total size limit not enforced", err)
	}
}
