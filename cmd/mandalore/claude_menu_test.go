package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/install"
)

func claudeMenuFixture(t *testing.T, script string) (*menu, *bytes.Buffer, *int) {
	t.Helper()
	var out bytes.Buffer
	calls := 0
	m := &menu{ctx: context.Background(), in: bufio.NewReader(strings.NewReader(script)), out: &out,
		binding: "/synthetic/binding", binary: "/synthetic/selected-runtime",
		claudeProfile: install.Profile{StateDir: "/synthetic/state", NativeHome: "/synthetic/claude", NativeBinary: "/synthetic/claude-cli"}}
	m.prepareSelectedClaude = func(_ context.Context, o install.ClaudeOptions) (install.ClaudePlan, error) {
		if o.NativeHome != "/synthetic/claude" || o.Binding != m.binding || o.Binary != m.binary {
			t.Fatal("wrong selected connection", o)
		}
		return install.ClaudePlan{ClaudeOptions: o, Harness: "claude-code", SignetID: "signet-synthetic", Root: "/synthetic/retained", PackageVersion: "1.2.3"}, nil
	}
	m.applySelectedClaude = func(_ context.Context, p install.ClaudeApplyInput) (install.ClaudeResult, error) {
		calls++
		return install.ClaudeResult{Connection: p.Plan, Installed: true, Phase: "verified", RequiresFreshSession: true}, nil
	}
	return m, &out, &calls
}

func TestClaudeMenuConnectRequiresReviewedConsent(t *testing.T) {
	for _, consent := range []string{"\n", "", "2", ":back\n", "2\n"} {
		// Harness, runtime, binding/profile/state/native, read-only, confirmation.
		m, out, calls := claudeMenuFixture(t, "3\n\n\n\n\n\n2\n"+consent)
		apply := m.applySelectedClaude
		m.applySelectedClaude = func(ctx context.Context, p install.ClaudeApplyInput) (install.ClaudeResult, error) {
			if !p.Plan.ReadOnly {
				t.Fatal("configured read-only mode lost")
			}
			return apply(ctx, p)
		}
		err := m.connect()
		if !strings.Contains(out.String(), "Review Claude Code connection") || !strings.Contains(out.String(), "Read-only (enforced)") {
			t.Fatal("missing preview", err, out.String())
		}
		if consent == "2\n" {
			if err != nil || *calls != 1 || !strings.Contains(out.String(), "fresh Claude Code session") {
				t.Fatal(err, *calls, out.String())
			}
		} else if *calls != 0 || !(errors.Is(err, console.ErrBack) || errors.Is(err, io.EOF)) {
			t.Fatal("unreviewed apply", err, *calls)
		}
	}
}

func TestClaudeMenuUpdateRequiresStoppedSessions(t *testing.T) {
	for _, answer := range []string{"\n", "", "2\n"} {
		m, _, calls := claudeMenuFixture(t, "2\n"+answer)
		apply := m.applySelectedClaude
		m.applySelectedClaude = func(ctx context.Context, in install.ClaudeApplyInput) (install.ClaudeResult, error) {
			if !in.SessionsStopped {
				t.Fatal("update lost explicit stopped-session acknowledgement")
			}
			return apply(ctx, in)
		}
		err := m.applyClaudePlan(install.ClaudePlan{PreviousRoot: "/synthetic/old"})
		if answer == "2\n" {
			if err != nil || *calls != 1 {
				t.Fatal(err, *calls)
			}
		} else if *calls != 0 || err == nil {
			t.Fatal("update without stopped sessions", err, *calls)
		}
	}
}

func TestClaudeMenuInvisiblePreviewCannotExecute(t *testing.T) {
	m, _, calls := claudeMenuFixture(t, "2\n")
	m.out = failedMenuWriter{}
	if err := m.applyClaudePlan(install.ClaudePlan{}); err == nil || *calls != 0 {
		t.Fatal("invisible preview applied", err, *calls)
	}
}

func TestClaudeReleaseHandoffMatchesInstalledRuntime(t *testing.T) {
	for _, mismatch := range []bool{false, true} {
		m, _, _, o := releaseMenuFixture(t, "2\n4\n\n\n\n\n1\n2\n")
		m.claudeProfile = install.Profile{StateDir: "/synthetic/state", NativeHome: "/synthetic/claude", NativeBinary: "/synthetic/claude-cli"}
		applies := 0
		m.prepareSelectedClaude = func(_ context.Context, options install.ClaudeOptions) (install.ClaudePlan, error) {
			if !strings.HasPrefix(options.Binary, o.Prefix) {
				t.Fatal("selected runtime lost", options)
			}
			version := "1.0.0"
			if mismatch {
				version = "0.0.0-dev"
			}
			return install.ClaudePlan{ClaudeOptions: options, BinarySHA256: strings.Repeat("d", 64), PackageVersion: version}, nil
		}
		m.applySelectedClaude = func(_ context.Context, in install.ClaudeApplyInput) (install.ClaudeResult, error) {
			applies++
			return install.ClaudeResult{Connection: in.Plan, Installed: true, Phase: "verified"}, nil
		}
		err := m.installRelease(o)
		if mismatch {
			if err == nil || applies != 0 {
				t.Fatal("mismatched release applied", err, applies)
			}
		} else if err != nil || applies != 1 {
			t.Fatal(err, applies)
		}
	}
}
