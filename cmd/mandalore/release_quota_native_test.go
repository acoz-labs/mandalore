package main

import (
	"bufio"
	"errors"
	"os"
	"testing"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/distribution"
)

// Native presentation driver: compile with go test -c, run in a real terminal
// with MANDALORE_QUOTA_NATIVE_SCENARIO set. The production menu renderer and
// keyboard handling are real; API results are synthetic, not installation or
// network evidence. This test-only seam is absent from the production binary.
func TestReleaseQuotaNativePresentation(t *testing.T) {
	scenario := os.Getenv("MANDALORE_QUOTA_NATIVE_SCENARIO")
	if scenario == "" {
		t.Skip("explicit synthetic native presentation only")
	}
	switch scenario {
	case "timing", "unknown", "ordinary", "partial", "default-no":
	default:
		t.Fatal("unknown presentation scenario")
	}
	m, _, _, o := releaseMenuFixture(t, "")
	m.in = bufio.NewReader(os.Stdin)
	m.out = os.Stdout
	if os.Getenv("MANDALORE_QUOTA_NATIVE_PLAIN") != "1" {
		m.tui = console.New(os.Stdin, os.Stdout)
	}
	m.block(console.Block{Title: "Synthetic quota presentation", Body: "Real menu rendering; simulated API results. No network, installation, memory or credentials are used."})
	original := m.releaseInvoke
	applies := 0
	m.releaseInvoke = func(name string, value any) api.Envelope {
		if name == "release_plan" && (scenario == "partial" || scenario == "default-no") {
			v := original(name, value)
			p := v.Result.(distribution.InstallPlan)
			p.Candidate = ""
			p.Prefix = "/synthetic/quota-prefix"
			p.Launcher = p.Prefix + "/bin/mandalore"
			p.Runtime = "/synthetic/new"
			if scenario == "partial" {
				p.Observed.LauncherTarget = "/synthetic/old"
			}
			p.Source.Kind = "github-release"
			p.Source.Published = &distribution.ReleaseView{URL: "https://github.com/acoz-labs/mandalore/releases/tag/v1.0.0", Manifest: p.Source.Manifest}
			return api.Success(p)
		}
		if name == "release_apply" {
			applies++
		}
		if scenario == "ordinary" {
			return api.Failure("release.failed", "release request was refused", false)
		}
		retry := distribution.ReleaseRetry{HTTPStatus: 429, Kind: "unspecified"}
		if scenario == "timing" {
			retry.Kind = "primary"
			retry.RetryAfterSeconds = 60
			retry.ResetAt = "2026-09-17T00:00:00Z"
		}
		quota := &distribution.ReleaseRateLimitError{Retry: retry}
		v := api.Failure("release.rate_limited", quota.Error(), scenario == "partial")
		v.Error.ReleaseRetry = &retry
		if scenario == "partial" {
			// Composition coverage only: production release reads occur before
			// activation. This does not assert a real post-activation quota call.
			v.Error.ReleaseResult = &distribution.InstallResult{Phase: "launcher-activated", Prefix: "/synthetic/quota-prefix", Launcher: "/synthetic/quota-prefix/bin/mandalore", Pending: "/synthetic/pending.json", PreviousRuntime: "/synthetic/old", Runtime: "/synthetic/new", Connections: "unchanged", DestinationChanged: true}
		}
		return v
	}
	err := m.installRelease(o)
	if scenario == "default-no" {
		if !errors.Is(err, console.ErrBack) || applies != 0 {
			t.Fatal("default-No did not preserve state", err, applies)
		}
		m.block(console.Block{Title: "Stopped", Body: "No apply was invoked."})
		return
	}
	if err == nil {
		t.Fatal("synthetic refusal unexpectedly succeeded")
	}
	m.block(console.Block{Title: "[FAIL] Needs attention", Body: err.Error(), Tone: console.Failure})
	want := 0
	if scenario == "partial" {
		want = 1
	}
	if applies != want || m.outputErr != nil {
		t.Fatal("unexpected operation count or output failure", applies, m.outputErr)
	}
}
