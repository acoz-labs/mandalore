package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/distribution"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

type fakePublisher struct {
	calls  int
	result distribution.PublicationReceipt
}

func (p *fakePublisher) Publish(context.Context, string, string) (distribution.PublicationReceipt, error) {
	p.calls++
	return p.result, nil
}

func TestLocalPublicationRequiresIndependentAuthorityAndUnchangedBytes(t *testing.T) {
	m, assets := testfixture.Release20(t)
	args := []string{"--directory", "retained", "--identity", m.Identity(), "--issue", "12", "--expected-actor", "Reviewer"}
	for _, scenario := range []string{"approved", "denied", "changed", "partial", "missing", "extra", "duplicate", "digest"} {
		t.Run(scenario, func(t *testing.T) {
			published := append([]distribution.ReleaseAsset{}, assets...)
			switch scenario {
			case "missing":
				published = published[:len(published)-1]
			case "extra":
				published = append(published, distribution.ReleaseAsset{Name: "mandalore_2.0.0_codex.zip", Size: 30, SHA256: strings.Repeat("f", 64)})
			case "duplicate":
				published[len(published)-1] = published[0]
			case "digest":
				published[2].SHA256 = strings.Repeat("f", 64)
			}
			pub := &fakePublisher{result: distribution.PublicationReceipt{Phase: "publication-verified", Identity: m.Identity(), SourceCommit: m.Manifest.SourceCommit, URL: "https://github.com/acoz-labs/mandalore/releases/tag/v2.0.0", ReleaseID: 1, Assets: published}}
			calls := 0
			s := services{publisher: pub, verify: func(string) (distribution.ParsedManifest, error) {
				calls++
				copy := m
				if scenario == "changed" && calls > 1 {
					copy.SHA256 = strings.Repeat("c", 64)
				}
				return copy, nil
			}, authorize: func(_ context.Context, candidate distribution.ParsedManifest, issue int, actor string) error {
				if candidate.Identity() != m.Identity() || issue != 12 || actor != "Reviewer" {
					t.Fatal("authority binding changed")
				}
				if scenario == "denied" {
					return errors.New("rejected")
				}
				return nil
			}}
			if scenario == "partial" {
				pub.result.PendingOperation = "upload"
			}
			_, err := run(context.Background(), args, s)
			if (err == nil) != (scenario == "approved") {
				t.Fatalf("unexpected result: %v", err)
			}
			if (scenario == "denied" || scenario == "changed") && pub.calls != 0 {
				t.Fatal("published without matching authority/bytes")
			}
		})
	}
}
