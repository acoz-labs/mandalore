package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/distribution"
	codexplugin "github.com/acoz-labs/mandalore/plugins/codex"
)

type inspectFixture struct {
	calls       int
	t           distribution.CandidateTransport
	changeAfter int
}

func (f *inspectFixture) InspectCandidate(_ context.Context, s distribution.CandidateSelection) (distribution.CandidateTransport, error) {
	f.calls++
	v := f.t
	if f.changeAfter > 0 && f.calls >= f.changeAfter {
		v.ArchiveSHA256 = strings.Repeat("0", 64)
	}
	return v, nil
}

func cliFixture(t *testing.T) (string, string, *inspectFixture, string) {
	t.Helper()
	dir := t.TempDir()
	plugin, err := distribution.PreparePlugin(codexplugin.Files, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	m := distribution.Manifest{FormatVersion: 1, Product: "mandalore", Version: "1.0.0", Tag: "v1.0.0", SourceCommit: strings.Repeat("a", 40), GoVersion: "go1.26.4", ProtocolVersion: 1, SignetReadVersions: []int{1}, SignetWriteVersions: []int{1}, PluginSHA256: plugin.SHA256}
	files := map[string][]byte{}
	for _, osName := range []string{"darwin", "linux"} {
		for _, arch := range []string{"amd64", "arm64"} {
			name := "mandalore_1.0.0_" + osName + "_" + arch
			b := []byte("inert fixture " + name)
			files[name] = b
			m.Assets = append(m.Assets, distribution.Asset{Kind: "cli", Name: name, OS: osName, Arch: arch, Size: int64(len(b)), SHA256: distribution.Digest(b)})
		}
	}
	for _, a := range []struct {
		kind, name string
		b          []byte
	}{{"codex-plugin", "mandalore_1.0.0_codex.zip", plugin.Archive}, {"bootstrap", "install.sh", []byte("inert fixture")}} {
		files[a.name] = a.b
		m.Assets = append(m.Assets, distribution.Asset{Kind: a.kind, Name: a.name, Size: int64(len(a.b)), SHA256: distribution.Digest(a.b)})
	}
	files["manifest.json"], err = distribution.EncodeManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	files["SHA256SUMS"], err = distribution.Checksums(files["manifest.json"])
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for name, data := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	f := &inspectFixture{t: distribution.CandidateTransport{CandidateSelection: distribution.CandidateSelection{SourceCommit: m.SourceCommit, RunID: 1, ArtifactID: 2}, FormatVersion: 1, Repository: "acoz-labs/mandalore", RepositoryID: 3, DefaultBranch: "main", Workflow: distribution.CandidateWorkflow, WorkflowID: 4, RunAttempt: 1, ArtifactName: "mandalore-candidate-" + m.SourceCommit + "-1", ArchiveSize: int64(b.Len()), ArchiveSHA256: distribution.Digest(b.Bytes()), CreatedAt: "2026-09-14T10:00:00Z", ExpiresAt: "2026-12-13T10:00:00Z"}}
	receipt, archive := filepath.Join(dir, "receipt.json"), filepath.Join(dir, "artifact.zip")
	raw, _ := json.Marshal(f.t)
	if err := os.WriteFile(receipt, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	p, _ := distribution.ParseManifest(files["manifest.json"])
	return receipt, archive, f, p.Identity()
}

func TestVerifyRefreshesProvenanceAndBindsAcceptanceIdentity(t *testing.T) {
	receipt, archive, f, identity := cliFixture(t)
	var out bytes.Buffer
	err := run(context.Background(), []string{"verify", "--receipt", receipt, "--archive", archive, "--expected-identity", identity}, &out, f)
	if err != nil || f.calls != 2 || !strings.Contains(out.String(), identity) {
		t.Fatal("verification did not bind both metadata checks", err, f.calls, out.String())
	}
	for _, change := range []int{1, 2} {
		f.calls = 0
		f.changeAfter = change
		out.Reset()
		if err := run(context.Background(), []string{"verify", "--receipt", receipt, "--archive", archive}, &out, f); err == nil || out.Len() != 0 || f.calls != change {
			t.Fatal("changed provenance produced success", change, err)
		}
	}
}

func TestVerifyRejectsCorruptionWrongAcceptanceAndUnreviewedPaths(t *testing.T) {
	for _, mode := range []string{"corrupt", "identity", "receipt-symlink", "archive-symlink", "unknown-receipt-field"} {
		t.Run(mode, func(t *testing.T) {
			receipt, archive, f, identity := cliFixture(t)
			switch mode {
			case "corrupt":
				if err := os.WriteFile(archive, []byte("corrupt"), 0600); err != nil {
					t.Fatal(err)
				}
			case "identity":
				identity = "mandalore:other:sha256:other"
			case "receipt-symlink":
				link := receipt + "-link"
				if err := os.Symlink(receipt, link); err != nil {
					t.Fatal(err)
				}
				receipt = link
			case "archive-symlink":
				link := archive + "-link"
				if err := os.Symlink(archive, link); err != nil {
					t.Fatal(err)
				}
				archive = link
			case "unknown-receipt-field":
				raw, _ := json.Marshal(f.t)
				raw = append(raw[:len(raw)-1], []byte(`,"trusted":true}`)...)
				if err := os.WriteFile(receipt, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			if err := run(context.Background(), []string{"verify", "--receipt", receipt, "--archive", archive, "--expected-identity", identity}, &out, f); err == nil || out.Len() != 0 {
				t.Fatal("unsafe input accepted", err)
			}
		})
	}
}

func TestArgumentsRefuseBeforeNetworkOrInputAccess(t *testing.T) {
	for _, args := range [][]string{nil, {"unknown"}, {"inspect", "--source", "main", "--run-id", "1", "--artifact-id", "2"}, {"inspect", "--source", strings.Repeat("a", 40), "--run-id", "9007199254740993", "--artifact-id", "2"}, {"verify"}, {"verify", "--receipt", "missing", "--archive", "missing", "--source", "changed"}} {
		f := &inspectFixture{}
		var out bytes.Buffer
		if err := run(context.Background(), args, &out, f); err == nil || f.calls != 0 || out.Len() != 0 {
			t.Fatal("invalid request progressed", args, err, f.calls)
		}
	}
}
