package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/distribution"
	codexplugin "github.com/acoz-labs/mandalore/plugins/codex"
)

type inspectFixture struct {
	calls       int
	changeAfter int
	t           distribution.CandidateTransport
	order       *[]string
}

func (f *inspectFixture) InspectCandidate(_ context.Context, s distribution.CandidateSelection) (distribution.CandidateTransport, error) {
	f.calls++
	if f.order != nil {
		*f.order = append(*f.order, "inspect")
	}
	if s != f.t.CandidateSelection {
		return f.t, errors.New("wrong selection")
	}
	r := f.t
	if f.changeAfter > 0 && f.calls >= f.changeAfter {
		r.ArchiveSHA256 = strings.Repeat("f", 64)
	}
	return r, nil
}

func promoteFixture(t *testing.T) (string, string, *inspectFixture, string) {
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

type publishFixture struct {
	calls     int
	fail      bool
	badResult bool
	identity  string
	order     *[]string
}

func (f *publishFixture) Publish(_ context.Context, path, identity string) (distribution.PublicationReceipt, error) {
	f.calls++
	*f.order = append(*f.order, "publish")
	p, err := distribution.VerifyDirectory(path)
	if err != nil {
		return distribution.PublicationReceipt{}, err
	}
	if identity != f.identity || p.Identity() != identity {
		return distribution.PublicationReceipt{}, errors.New("publisher received wrong bytes")
	}
	r := distribution.PublicationReceipt{Identity: identity, SourceCommit: p.Manifest.SourceCommit, Phase: "publication-verified", ReleaseID: 42, URL: "https://github.com/acoz-labs/mandalore/releases/tag/" + p.Manifest.Tag, Assets: make([]distribution.ReleaseAsset, 8)}
	if f.fail {
		r.Phase = "assets-verified"
		r.PendingOperation = "publish-draft"
		return r, errors.New("uncertain publication")
	}
	if f.badResult {
		r.Phase = "published"
	}
	return r, nil
}

func TestPromotionRevalidatesBeforeStagingAfterStagingAndAfterAuthorization(t *testing.T) {
	receipt, archive, i, identity := promoteFixture(t)
	order := []string{}
	i.order = &order
	p := &publishFixture{identity: identity, order: &order}
	authorize := func(_ context.Context, m distribution.ParsedManifest) error {
		order = append(order, "authorize")
		if m.Identity() != identity {
			return errors.New("wrong authorized bytes")
		}
		return nil
	}
	r, err := run(context.Background(), []string{"--receipt", receipt, "--archive", archive, "--identity", identity, "--staging-parent", t.TempDir()}, services{i, p, authorize})
	if err != nil || r.Phase != "publication-verified" || r.Publication == nil || r.StagingDirectory == "" || i.calls != 3 || p.calls != 1 || strings.Join(order, ",") != "inspect,inspect,authorize,inspect,publish" {
		t.Fatal(r, err, order)
	}
	if _, err := distribution.VerifyDirectory(r.StagingDirectory); err != nil {
		t.Fatal("staged bytes were not retained", err)
	}
}

func TestPromotionNeverPublishesUnverifiedOrUnauthorizedCandidate(t *testing.T) {
	for _, kind := range []string{"first provenance", "staged provenance", "authorized provenance", "identity", "corrupt archive", "authority", "cancelled authority", "missing archive", "archive link"} {
		t.Run(kind, func(t *testing.T) {
			receipt, archive, i, identity := promoteFixture(t)
			order := []string{}
			i.order = &order
			p := &publishFixture{identity: identity, order: &order}
			parent := t.TempDir()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "first provenance":
				i.changeAfter = 1
			case "staged provenance":
				i.changeAfter = 2
			case "authorized provenance":
				i.changeAfter = 3
			case "identity":
				identity += "x"
			case "corrupt archive":
				b, _ := os.ReadFile(archive)
				b[0] ^= 1
				if err := os.WriteFile(archive, b, 0600); err != nil {
					t.Fatal(err)
				}
			case "missing archive":
				archive = filepath.Join(t.TempDir(), "missing")
			case "archive link":
				link := filepath.Join(t.TempDir(), "link")
				if err := os.Symlink(archive, link); err != nil {
					t.Fatal(err)
				}
				archive = link
			}
			auth := func(context.Context, distribution.ParsedManifest) error {
				order = append(order, "authorize")
				if kind == "cancelled authority" {
					cancel()
				}
				if kind == "authority" {
					return errors.New("declined")
				}
				return nil
			}
			r, err := run(ctx, []string{"--receipt", receipt, "--archive", archive, "--identity", identity, "--staging-parent", parent}, services{i, p, auth})
			if err == nil || p.calls != 0 || r.Phase == "publication-verified" {
				t.Fatal("invalid candidate reached publisher", r, err, order)
			}
			if strings.Contains(kind, "provenance") && kind != "first provenance" || kind == "authority" || kind == "cancelled authority" {
				if r.StagingDirectory == "" {
					t.Fatal("lost retained staging effects", r)
				}
			}
		})
	}
}

func TestPromotionRetainsUncertainPublicationAndRefusesIncompleteSuccess(t *testing.T) {
	for _, kind := range []string{"uncertain", "incomplete"} {
		t.Run(kind, func(t *testing.T) {
			receipt, archive, i, identity := promoteFixture(t)
			order := []string{}
			p := &publishFixture{identity: identity, order: &order, fail: kind == "uncertain", badResult: kind == "incomplete"}
			r, err := run(context.Background(), []string{"--receipt", receipt, "--archive", archive, "--identity", identity, "--staging-parent", t.TempDir()}, services{i, p, func(context.Context, distribution.ParsedManifest) error { return nil }})
			if err == nil || r.Phase == "publication-verified" || r.Publication == nil || r.StagingDirectory == "" || p.calls != 1 {
				t.Fatal("partial publication lost or claimed complete", r, err)
			}
			if kind == "uncertain" && r.Publication.PendingOperation != "publish-draft" {
				t.Fatal("uncertain write lost", r)
			}
		})
	}
}

func TestAuthorizeInvokesBothBoundGuardsWithoutPolicyCredential(t *testing.T) {
	for _, kind := range []string{"success", "gate refusal", "preflight refusal", "cancelled", "missing token"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			if err := os.Mkdir("bin", 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GITHUB_REPOSITORY", "acoz-labs/mandalore")
			t.Setenv("GH_TOKEN", "synthetic-token")
			t.Setenv("RELEASE_POLICY_READ_TOKEN", "must-not-reach-guards")
			t.Setenv("GH_HOST", "example.invalid")
			t.Setenv("GH_DEBUG", "api")
			t.Setenv("RELEASE_SHA", "untrusted")
			t.Setenv("RELEASE_VERSION", "untrusted")
			t.Setenv("RELEASE_ARTIFACT", "untrusted")
			script := "#!/bin/sh\nset -eu\ntest -z \"${RELEASE_POLICY_READ_TOKEN:-}\"\ntest -z \"$GH_DEBUG\"\ntest \"$GH_HOST\" = github.com\ntest \"$GITHUB_SERVER_URL\" = https://github.com\nprintf '%s\\n' \"$0 $*|$RELEASE_SHA|$RELEASE_VERSION|$RELEASE_ARTIFACT|$STAGING_REQUIRED\" >> calls\n"
			for _, name := range []string{"release-gate", "finalize-release"} {
				body := script
				if kind == "gate refusal" && name == "release-gate" || kind == "preflight refusal" && name == "finalize-release" {
					body += "exit 1\n"
				}
				if kind == "cancelled" {
					body += "exec sleep 30\n"
				}
				if err := os.WriteFile(filepath.Join("bin", name), []byte(body), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "missing token" {
				t.Setenv("GH_TOKEN", "")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if kind == "cancelled" {
				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
			}
			m := distribution.ParsedManifest{Manifest: distribution.Manifest{SourceCommit: strings.Repeat("a", 40), Version: "1.0.0"}, SHA256: strings.Repeat("b", 64)}
			started := time.Now()
			err := authorize(ctx, m)
			log, _ := os.ReadFile("calls")
			if kind == "success" {
				if err != nil || !strings.Contains(string(log), "bin/release-gate --require-acceptance "+m.Manifest.SourceCommit) || !strings.Contains(string(log), "bin/finalize-release artifact --preflight") || strings.Count(string(log), m.Identity()) != 2 {
					t.Fatal(err, string(log))
				}
			} else if err == nil {
				t.Fatal("refusal accepted", kind)
			}
			if (kind == "gate refusal" || kind == "cancelled") && strings.Contains(string(log), "finalize-release") {
				t.Fatal("second guard ran after first guard failed")
			}
			if kind == "cancelled" && time.Since(started) > 3*time.Second {
				t.Fatal("guard cancellation did not stop its process group")
			}
			if kind == "missing token" && len(log) != 0 {
				t.Fatal("guard borrowed credentials")
			}
		})
	}
}
