package install

import (
	"os"
	"path/filepath"
	"testing"
)

func piTestDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestPiInventoryIsReadOnlyAndResolvesNativeRelativeSources(t *testing.T) {
	home := filepath.Join(piTestDir(t), "profile")
	missing, err := inspectPiSettings(home)
	if err != nil || missing.Exists || len(missing.Packages) != 0 {
		t.Fatal(missing, err)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatal("read created profile")
	}
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"theme":"light","packages":["../connection/package",{"source":"../other","skills":[],"extensions":["*.js"]},"npm:unrelated@1.0.0"]}`)
	path := filepath.Join(home, "settings.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := inspectPiSettings(home)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Exists || got.SHA256 != hash(raw) || len(got.Packages) != 3 {
		t.Fatal(got)
	}
	if got.Packages[0].Path != filepath.Join(filepath.Dir(home), "connection/package") || got.Packages[0].Filtered || !got.Packages[1].Filtered || got.Packages[2].Path != "" {
		t.Fatal(got.Packages)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(raw) {
		t.Fatal("inspection changed settings", err)
	}
}

func TestPiInventoryRefusesMalformedAmbiguousAndRedirectedSettings(t *testing.T) {
	for _, raw := range []string{
		`{"packages":null}`, `{"packages":"wrong"}`, `{"packages":[2]}`,
		`{"packages":[{}]}`, `{"packages":[{"source":"../other","skills":true}]}`,
		`{"packages":[],"packages":[]}`, `{"packages":["../same","../same"]}`,
		`{"packages":["\u0000bad"]}`, `{"packages":[" "]}`,
	} {
		home := piTestDir(t)
		if err := os.WriteFile(filepath.Join(home, "settings.json"), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := inspectPiSettings(home); err == nil {
			t.Fatal("invalid native settings accepted", raw)
		}
	}
	home := piTestDir(t)
	other := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(other, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, filepath.Join(home, "settings.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectPiSettings(home); err == nil {
		t.Fatal("redirected settings accepted")
	}
}

func TestPiInventoryPreservationComparisonIncludesFilteredPackages(t *testing.T) {
	home := piTestDir(t)
	path := filepath.Join(home, "settings.json")
	read := func(raw string) piSettings {
		t.Helper()
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := inspectPiSettings(home)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	a := read(`{"theme":"light","packages":[{"source":"../other","skills":[]},"../old/package"]}`)
	b := read(`{"packages":[{"skills":[],"source":"../other"},"../new/package"],"theme":"light"}`)
	old, new := filepath.Join(filepath.Dir(home), "old/package"), filepath.Join(filepath.Dir(home), "new/package")
	if !piSettingsPreserved(a, b, old, new) {
		t.Fatal("native formatting or selected replacement rejected")
	}
	c := read(`{"theme":"dark","packages":[{"source":"../other","skills":[]},"../new/package"]}`)
	if piSettingsPreserved(a, c, old, new) {
		t.Fatal("unrelated setting change missed")
	}
	d := read(`{"theme":"light","packages":[{"source":"../other","skills":["*"]},"../new/package"]}`)
	if piSettingsPreserved(a, d, old, new) {
		t.Fatal("unrelated resource filter change missed")
	}
}
