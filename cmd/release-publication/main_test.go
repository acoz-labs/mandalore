package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/distribution"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

type fakeVerifier struct {
	calls    int
	fail     bool
	receipt  distribution.PublicationReceipt
	manifest distribution.ParsedManifest
}

func (f *fakeVerifier) VerifyPublication(_ context.Context, version, identity string) (distribution.PublicationReceipt, error) {
	f.calls++
	if f.fail {
		return f.receipt, errors.New("verification refused")
	}
	return f.receipt, nil
}

func (f *fakeVerifier) Inspect(context.Context, string) (distribution.ReleaseView, error) {
	return distribution.ReleaseView{Manifest: f.manifest}, nil
}

func TestSelectionIsLocalAndExplicit(t *testing.T) {
	identity := "mandalore:" + strings.Repeat("a", 40) + ":sha256:" + strings.Repeat("b", 64)
	f := &fakeVerifier{}
	var out bytes.Buffer
	if err := run(context.Background(), []string{"selection", "--version", "1.0.0", "--identity", identity}, &out, f); err != nil || f.calls != 0 || !strings.Contains(out.String(), `"tag":"v1.0.0"`) {
		t.Fatal("invalid selection", err, out.String())
	}
	for _, args := range [][]string{{}, {"publish"}, {"verify"}, {"selection", "--version", "latest", "--identity", identity}, {"verify", "--version", "1.0.0", "--identity", "unverified"}, {"verify", "--version", "1.0.0", "--identity", identity, "extra"}} {
		out.Reset()
		if err := run(context.Background(), args, &out, f); err == nil || f.calls != 0 || out.Len() != 0 {
			t.Fatal("invalid input caused effects or output", args, err)
		}
	}
}

func TestVerifyEmitsOnlyFreshCompleteSuccess(t *testing.T) {
	m, assets := testfixture.Release20(t)
	sha, identity := m.Manifest.SourceCommit, m.Identity()
	other := m
	other.SHA256 = strings.Repeat("b", 64)
	for _, kind := range []string{"success", "error", "phase", "identity", "source", "url", "pending", "assets", "id", "missing", "extra", "digest", "manifest"} {
		t.Run(kind, func(t *testing.T) {
			f := &fakeVerifier{manifest: m, receipt: distribution.PublicationReceipt{Identity: identity, SourceCommit: sha, Phase: "publication-verified", ReleaseID: 42, URL: "https://github.com/acoz-labs/mandalore/releases/tag/v2.0.0", Assets: append([]distribution.ReleaseAsset{}, assets...)}}
			switch kind {
			case "error":
				f.fail = true
			case "phase":
				f.receipt.Phase = "published"
			case "identity":
				f.receipt.Identity = "wrong"
			case "source":
				f.receipt.SourceCommit = "wrong"
			case "url":
				f.receipt.URL = "https://example.invalid"
			case "pending":
				f.receipt.PendingOperation = "publish-draft"
			case "assets":
				f.receipt.Assets = nil
			case "id":
				f.receipt.ReleaseID = 0
			case "missing":
				f.receipt.Assets = f.receipt.Assets[1:]
			case "extra":
				f.receipt.Assets = append(f.receipt.Assets, distribution.ReleaseAsset{Name: "mandalore_2.0.0_codex.zip", Size: 30, SHA256: strings.Repeat("f", 64)})
			case "digest":
				f.receipt.Assets[1].SHA256 = strings.Repeat("f", 64)
			case "manifest":
				f.manifest = other
			}
			var out bytes.Buffer
			err := run(context.Background(), []string{"verify", "--version", "2.0.0", "--identity", identity}, &out, f)
			if f.calls != 1 {
				t.Fatal("verification was not invoked")
			}
			if kind == "success" {
				if err != nil || out.Len() == 0 {
					t.Fatal(err)
				}
			} else if err == nil || out.Len() != 0 {
				t.Fatal("unverified receipt emitted", err, out.String())
			}
		})
	}
}
