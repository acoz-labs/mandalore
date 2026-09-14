package distribution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	codexplugin "github.com/acoz-labs/mandalore/plugins/codex"
	"golang.org/x/sys/unix"
)

func candidateFixture(t *testing.T) (string, Manifest, []byte) {
	t.Helper()
	dir := t.TempDir()
	m := fixtureManifest()
	p, err := PreparePlugin(codexplugin.Files, m.Version)
	if err != nil {
		t.Fatal(err)
	}
	m.PluginSHA256 = p.SHA256
	for i := range m.Assets {
		a := &m.Assets[i]
		data := []byte("synthetic payload for " + a.Name)
		if a.Kind == "codex-plugin" {
			data = p.Archive
		}
		a.Size, a.SHA256 = int64(len(data)), Digest(data)
		if err := os.WriteFile(filepath.Join(dir, a.Name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	b, err := EncodeManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	checksums, err := Checksums(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), checksums, 0600); err != nil {
		t.Fatal(err)
	}
	return dir, m, b
}

func TestCandidateVerifiesAllFilesWithoutExecuting(t *testing.T) {
	dir, m, b := candidateFixture(t)
	// These are inert synthetic payloads, intentionally not native executables.
	// Byte verification is not native acceptance or publisher authentication.
	p, err := VerifyDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.SHA256 != Digest(b) || p.Manifest.SourceCommit != m.SourceCommit {
		t.Fatal("verified wrong candidate")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 8 {
		t.Fatal("verification wrote to candidate")
	}
	got, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Split(strings.TrimSpace(string(got)), "\n")) != 7 {
		t.Fatal("checksum set does not cover manifest and all six payloads")
	}
}

func TestCandidateRejectsPartialChangedOrRedirectedFiles(t *testing.T) {
	for _, kind := range []string{"cli", "codex-plugin", "bootstrap", "manifest", "checksums"} {
		for _, mutation := range []string{"missing", "same-size-corruption", "symlink", "directory"} {
			t.Run(kind+"/"+mutation, func(t *testing.T) {
				dir, m, _ := candidateFixture(t)
				name := "manifest.json"
				if kind == "checksums" {
					name = "SHA256SUMS"
				}
				for _, a := range m.Assets {
					if a.Kind == kind {
						name = a.Name
						break
					}
				}
				path := filepath.Join(dir, name)
				original, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				switch mutation {
				case "same-size-corruption":
					original[0] ^= 1
					err = os.WriteFile(path, original, 0600)
				case "symlink":
					outside := filepath.Join(t.TempDir(), "outside")
					if err := os.WriteFile(outside, original, 0600); err != nil {
						t.Fatal(err)
					}
					err = os.Symlink(outside, path)
				case "directory":
					err = os.Mkdir(path, 0700)
				}
				if err != nil {
					t.Fatal(err)
				}
				if _, err := VerifyDirectory(dir); err == nil {
					t.Fatal("invalid candidate accepted")
				}
			})
		}
	}
	for _, name := range []string{"unexpected.txt", "receipt.json"} {
		t.Run(name, func(t *testing.T) {
			dir, _, _ := candidateFixture(t)
			if err := os.WriteFile(filepath.Join(dir, name), []byte("not a release asset"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyDirectory(dir); err == nil {
				t.Fatal("undeclared candidate content accepted")
			}
		})
	}
}

func TestCandidateRejectsIndividuallyHashedButMismatchedPlugin(t *testing.T) {
	dir, m, _ := candidateFixture(t)
	m.PluginSHA256 = Digest([]byte("different embedded package"))
	b, err := EncodeManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	sums, err := Checksums(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), sums, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyDirectory(dir); err == nil {
		t.Fatal("plugin archive passed despite different embedded content identity")
	}
}

func TestChecksumsBindRawManifestAndAreDeterministic(t *testing.T) {
	_, m, b := candidateFixture(t)
	want := Digest(b) + "  manifest.json\n"
	got, err := Checksums(b)
	if err != nil || !strings.Contains(string(got), want) {
		t.Fatal("manifest omitted from checksums")
	}
	for _, a := range m.Assets {
		if !strings.Contains(string(got), a.SHA256+"  "+a.Name+"\n") {
			t.Fatal("payload omitted from checksums")
		}
	}
	other, err := Checksums(append(b, '\n'))
	if err != nil || string(got) == string(other) {
		t.Fatal("checksum manifest does not bind exact bytes")
	}
}

func TestCandidateRefusesFIFOWithoutWaitingForAWriter(t *testing.T) {
	rootFIFO := filepath.Join(t.TempDir(), "not-a-directory")
	if err := unix.Mkfifo(rootFIFO, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyDirectory(rootFIFO); err == nil {
		t.Fatal("FIFO accepted as a candidate directory")
	}
	dir, m, _ := candidateFixture(t)
	asset := filepath.Join(dir, m.Assets[0].Name)
	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(asset, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyDirectory(dir); err == nil {
		t.Fatal("FIFO accepted as a candidate payload")
	}
}
