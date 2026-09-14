package distribution

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sourceTar(t *testing.T, headers []*tar.Header) []byte {
	t.Helper()
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	for _, h := range headers {
		if err := w.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			if _, err := w.Write(bytes.Repeat([]byte("x"), int(h.Size))); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestSourceExportRejectsLinksTraversalAndSpecialFiles(t *testing.T) {
	for _, h := range []*tar.Header{
		{Name: "../escape", Mode: 0644, Size: 1}, {Name: "/escape", Mode: 0644, Size: 1},
		{Name: "file\\name", Mode: 0644, Size: 1}, {Name: ".git/config", Mode: 0644, Size: 1},
		{Name: "file", Typeflag: tar.TypeSymlink, Linkname: "outside"},
		{Name: "file", Typeflag: tar.TypeLink, Linkname: "outside"}, {Name: "file", Typeflag: tar.TypeFifo},
	} {
		t.Run(h.Name+string(h.Typeflag), func(t *testing.T) {
			if err := unpackSource(sourceTar(t, []*tar.Header{h}), t.TempDir()); err == nil {
				t.Fatal("unsafe source export accepted")
			}
		})
	}
	h := &tar.Header{Name: "source.txt", Mode: 0644, Size: 2}
	if err := unpackSource(sourceTar(t, []*tar.Header{h, h}), t.TempDir()); err == nil {
		t.Fatal("duplicate export entry accepted")
	}
	dir := t.TempDir()
	if err := unpackSource(sourceTar(t, []*tar.Header{h}), dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "source.txt"))
	if err != nil || string(b) != "xx" {
		t.Fatal("valid source export failed")
	}
}

func TestBuildCommandsAreBoundedAndCancellationKillsGroup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := runBuildTool(ctx, "", nil, 1024, "/bin/sh", "-c", "sleep 5 & wait"); err == nil {
		t.Fatal("cancelled process succeeded")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("cancelled child group was not bounded")
	}
	if _, err := runBuildTool(context.Background(), "", nil, 8, "/bin/sh", "-c", "printf 123456789"); err == nil {
		t.Fatal("excessive output accepted")
	}
}

func TestBuildRejectsDirtySourceAndExistingOutput(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-qm", "fixture"}} {
		cmd := exec.Command(git, args...)
		cmd.Dir = source
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git fixture: %v %s", err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "uncommitted"), []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "candidate")
	if _, err := Build(context.Background(), BuildOptions{Source: source, Output: output}); err == nil || !strings.Contains(err.Error(), "clean") {
		t.Fatalf("dirty source did not fail before build: %v", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("dirty-source denial created a candidate")
	}
	if err := os.Mkdir(output, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(context.Background(), BuildOptions{Source: source, Output: output}); err == nil {
		t.Fatal("existing output was accepted")
	}
}
