package distribution

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io/fs"
	"testing"
	"testing/fstest"

	codexplugin "github.com/acoz-labs/mandalore/plugins/codex"
)

func TestPluginPackageDeterministicAndMatchesEmbeddedInput(t *testing.T) {
	before, err := codexplugin.Files.ReadFile("plugins/mandalore/.codex-plugin/plugin.json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := PreparePlugin(codexplugin.Files, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	q, err := PreparePlugin(codexplugin.Files, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(p.Archive, q.Archive) || p.SHA256 != q.SHA256 {
		t.Fatal("plugin package is not deterministic")
	}
	b, err := json.Marshal(p.Files)
	if err != nil {
		t.Fatal(err)
	}
	if Digest(b) != p.SHA256 {
		t.Fatal("package identity differs from installed embedded-map identity")
	}
	var manifest struct{ Name, Version string }
	if err := json.Unmarshal(p.Files["plugins/mandalore/.codex-plugin/plugin.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "mandalore" || manifest.Version != "1.0.0" {
		t.Fatalf("wrong stamped identity: %+v", manifest)
	}
	after, err := codexplugin.Files.ReadFile("plugins/mandalore/.codex-plugin/plugin.json")
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("packaging mutated its source")
	}
	if err := VerifyPlugin(p.Archive, "1.0.0", p.SHA256); err != nil {
		t.Fatal(err)
	}
	if err := VerifyPlugin(p.Archive, "1.0.1", p.SHA256); err == nil {
		t.Fatal("mismatched plugin version accepted")
	}
	if err := VerifyPlugin(p.Archive, "1.0.0", Digest([]byte("foreign"))); err == nil {
		t.Fatal("mismatched embedded identity accepted")
	}
	z, err := zip.NewReader(bytes.NewReader(p.Archive), int64(len(p.Archive)))
	if err != nil {
		t.Fatal(err)
	}
	previous := ""
	for _, entry := range z.File {
		if entry.Name <= previous {
			t.Fatal("archive ordering is not stable")
		}
		previous = entry.Name
		if entry.Modified.Year() != 1980 || entry.Method != zip.Store {
			t.Fatal("archive timestamp/compression not normalized")
		}
		if !entry.Mode().IsRegular() || (entry.Mode().Perm() != 0644 && entry.Mode().Perm() != 0755) {
			t.Fatal("archive mode not normalized")
		}
	}
}

func TestPluginPackagingRejectsRedirectedAndUnboundedInput(t *testing.T) {
	files := fstest.MapFS{}
	for _, root := range []string{".agents", "plugins/mandalore"} {
		if err := fs.WalkDir(codexplugin.Files, root, func(name string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			b, err := codexplugin.Files.ReadFile(name)
			files[name] = &fstest.MapFile{Data: b, Mode: 0644}
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []fs.FileMode{fs.ModeSymlink, fs.ModeNamedPipe, fs.ModeDevice} {
		files["plugins/mandalore/unsafe"] = &fstest.MapFile{Data: []byte("elsewhere"), Mode: mode}
		if _, err := PreparePlugin(files, "1.0.0"); err == nil {
			t.Fatalf("unsafe input mode accepted: %v", mode)
		}
	}
	files["plugins/mandalore/unsafe"] = &fstest.MapFile{Data: bytes.Repeat([]byte("x"), MaxPluginBytes+1), Mode: 0644}
	if _, err := PreparePlugin(files, "1.0.0"); err == nil {
		t.Fatal("oversized plugin accepted")
	}
	delete(files, "plugins/mandalore/unsafe")
	files["plugins/mandalore/.codex-plugin/plugin.json"].Data = []byte(`{"name":"foreign","version":"0.0.0"}`)
	if _, err := PreparePlugin(files, "1.0.0"); err == nil {
		t.Fatal("foreign plugin stamped as mandalore")
	}
}

func TestVerifyPluginRejectsUnsafeArchives(t *testing.T) {
	p, err := PreparePlugin(codexplugin.Files, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../escape", "/escape", "plugins/mandalore/../../escape", "plugins/mandalore/escape\\x", "plugins/mandalore/escape\x1b", ".agents/plugins/marketplace.json"} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			w := zip.NewWriter(&buf)
			for path, data := range p.Files {
				h := &zip.FileHeader{Name: path, Method: zip.Store}
				h.SetMode(pluginMode(path))
				f, err := w.CreateHeader(h)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := f.Write(data); err != nil {
					t.Fatal(err)
				}
			}
			h := &zip.FileHeader{Name: name, Method: zip.Store}
			h.SetMode(0644)
			f, err := w.CreateHeader(h)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.Write([]byte("unsafe")); err != nil {
				t.Fatal(err)
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			if err := VerifyPlugin(buf.Bytes(), "1.0.0", p.SHA256); err == nil {
				t.Fatal("unsafe/duplicate archive accepted")
			}
		})
	}
	if err := VerifyPlugin([]byte("not zip"), "1.0.0", p.SHA256); err == nil {
		t.Fatal("corrupt archive accepted")
	}
}
