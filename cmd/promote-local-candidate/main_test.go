package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/distribution"
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
	m := distribution.ParsedManifest{Manifest: distribution.Manifest{SourceCommit: strings.Repeat("a", 40), Tag: "v1.2.3"}, SHA256: strings.Repeat("b", 64)}
	args := []string{"--directory", "retained", "--identity", m.Identity(), "--issue", "12", "--expected-actor", "Reviewer"}
	for _, scenario := range []string{"approved", "denied", "changed", "partial"} {
		t.Run(scenario, func(t *testing.T) {
			pub := &fakePublisher{result: distribution.PublicationReceipt{Phase: "publication-verified", Identity: m.Identity(), SourceCommit: m.Manifest.SourceCommit, URL: "https://github.com/acoz-labs/mandalore/releases/tag/v1.2.3", ReleaseID: 1, Assets: make([]distribution.ReleaseAsset, 8)}}
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
