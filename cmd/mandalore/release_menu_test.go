package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/distribution"
)

// Presentation fixtures simulate API results. Actual native CLI installation is
// tested separately; these fixtures do not claim executable verification.
func releaseMenuFixture(t *testing.T, script string) (*menu, *bytes.Buffer, *int, distribution.InstallOptions) {
	t.Helper()
	var out bytes.Buffer
	calls := 0
	prefix := filepath.Join(t.TempDir(), "prefix")
	o := distribution.InstallOptions{Prefix: prefix, Candidate: "/synthetic/candidate"}
	p := distribution.InstallPlan{InstallOptions: o, OS: "darwin", Arch: "arm64", Source: distribution.InstallSource{Kind: "local-candidate", Manifest: distribution.ParsedManifest{SHA256: strings.Repeat("a", 64), Manifest: distribution.Manifest{Version: "1.0.0", SourceCommit: strings.Repeat("b", 40)}}}, Binary: distribution.Asset{SHA256: strings.Repeat("d", 64)}, Launcher: filepath.Join(prefix, "bin", "mandalore"), Runtime: filepath.Join(prefix, "lib", "mandalore", "retained-runtime")}
	m := &menu{ctx: context.Background(), in: bufio.NewReader(strings.NewReader(script)), out: &out}
	m.releaseInvoke = func(name string, value any) api.Envelope {
		switch name {
		case "release_plan":
			return api.Success(p)
		case "release_apply":
			calls++
			return api.Success(distribution.InstallResult{Phase: "complete", Prefix: prefix, Launcher: p.Launcher, Runtime: p.Runtime, Connections: "unchanged", Installed: true, DestinationChanged: true})
		default:
			t.Fatalf("unexpected release operation %s", name)
			return api.Envelope{}
		}
	}
	return m, &out, &calls, o
}

func TestReleaseMenuDefaultNoEOFAndOutputFailureDoNotApply(t *testing.T) {
	for _, script := range []string{"\n", "", ":back\n", "2"} {
		m, out, calls, o := releaseMenuFixture(t, script)
		err := m.installRelease(o)
		if !(errors.Is(err, console.ErrBack) || errors.Is(err, io.EOF)) || *calls != 0 || !strings.Contains(out.String(), "Review CLI installation") {
			t.Fatal("cancel applied or omitted preview", err, *calls, out.String())
		}
	}
	m, _, calls, o := releaseMenuFixture(t, "2\n1\n")
	m.out = failedMenuWriter{}
	if err := m.installRelease(o); err == nil || *calls != 0 {
		t.Fatal("invisible preview applied", err, *calls)
	}
}

func TestReleaseMenuSuccessDoesNotImplyConnectionUpdate(t *testing.T) {
	for _, script := range []string{"2\n1\n", "2\n"} {
		m, out, calls, o := releaseMenuFixture(t, script)
		err := m.installRelease(o)
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatal(err, out.String())
		}
		if *calls != 1 || !strings.Contains(out.String(), "CLI installation verified") || !strings.Contains(out.String(), "Native connections remain unchanged") || strings.Contains(out.String(), "Native connection verified") {
			t.Fatal("wrong success or effects", *calls, out.String())
		}
	}
}

func TestReleaseMenuPartialFailureShowsReceipt(t *testing.T) {
	m, out, _, o := releaseMenuFixture(t, "2\n")
	original := m.releaseInvoke
	m.releaseInvoke = func(name string, value any) api.Envelope {
		if name != "release_apply" {
			return original(name, value)
		}
		r := api.Failure("release.failed", "synthetic receipt failure", true)
		r.Error.ReleaseResult = &distribution.InstallResult{Phase: "launcher-activated", Runtime: "/synthetic/new", PreviousRuntime: "/synthetic/old", Pending: "/synthetic/pending.json", Connections: "unchanged", DestinationChanged: true}
		return r
	}
	if err := m.installRelease(o); err == nil || !strings.Contains(out.String(), "launcher-activated") || !strings.Contains(out.String(), "/synthetic/pending.json") || !strings.Contains(out.String(), "release apply <") || strings.Contains(out.String(), "CLI installation verified") {
		t.Fatal("partial state was lost", err, out.String())
	}
}

func TestReleaseMenuQuotaAdvicePreservesReceiptAndExplicitRetry(t *testing.T) {
	for _, delay := range []int64{0, 60} {
		for _, partial := range []bool{false, true} {
			m, out, _, o := releaseMenuFixture(t, "2\n")
			original := m.releaseInvoke
			applies := 0
			m.releaseInvoke = func(name string, value any) api.Envelope {
				if name != "release_apply" {
					return original(name, value)
				}
				applies++
				quota := &distribution.ReleaseRateLimitError{Retry: distribution.ReleaseRetry{HTTPStatus: 429, Kind: "unspecified", RetryAfterSeconds: delay}}
				r := api.Failure("release.rate_limited", quota.Error(), partial)
				r.Error.ReleaseRetry = &quota.Retry
				r.Error.ReleaseResult = &distribution.InstallResult{Phase: "staging", Connections: "unchanged", DestinationChanged: partial}
				if partial {
					r.Error.ReleaseResult.Phase = "launcher-activated"
					r.Error.ReleaseResult.Pending = "/synthetic/pending.json"
				}
				return r
			}
			err := m.installRelease(o)
			if err == nil || applies != 1 || !strings.Contains(err.Error(), "No automatic retry") || !strings.Contains(out.String(), "CLI receipt") || strings.Contains(out.String(), "CLI installation verified") {
				t.Fatal("advice, receipt or explicit retry lost", err, out.String(), applies)
			}
			if delay == 0 && !strings.Contains(err.Error(), "Retry timing is unavailable") {
				t.Fatal("missing unknown timing", err)
			}
			if delay > 0 && !strings.Contains(err.Error(), "60 seconds") {
				t.Fatal("missing bounded delay", err)
			}
			if partial && (!strings.Contains(err.Error(), "Changes may have occurred") || !strings.Contains(out.String(), "/synthetic/pending.json")) {
				t.Fatal("partial effects lost", err, out.String())
			}
		}
	}
}

func TestReleaseRecoveryInstructionQuotesPendingPath(t *testing.T) {
	m, out, _, _ := releaseMenuFixture(t, "")
	m.releaseResult(distribution.InstallResult{Pending: "/synthetic/owner's tools/pending.json"})
	if !strings.Contains(out.String(), "'/synthetic/owner'\"'\"'s tools/pending.json'") || !strings.Contains(out.String(), "original Mandalore executable") {
		t.Fatal("unsafe or ambiguous recovery instruction", out.String())
	}
}

func TestRecoveryCommandCopyPreservesShellMeaning(t *testing.T) {
	for _, name := range []string{"owner's tools", "two  spaces", "$(false); `false` & café"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), strings.Repeat("long-", 20)+name)
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			pending := filepath.Join(dir, "pending.json")
			if err := os.WriteFile(pending, []byte("synthetic reviewed plan"), 0600); err != nil {
				t.Fatal(err)
			}
			m, out, _, _ := releaseMenuFixture(t, "")
			m.releaseResult(distribution.InstallResult{Pending: pending})
			var command string
			for _, line := range strings.Split(out.String(), "\n") {
				if strings.HasPrefix(line, "mandalore release apply < ") {
					command = line
				}
			}
			if command == "" {
				t.Fatal("missing intact unindented recovery command")
			}
			// This function shadows the real CLI: no recovery or network is run.
			script := "mandalore() { printf '%s\\n' \"$#\" \"$1\" \"$2\"; /bin/cat; };\n" + command
			got, err := exec.Command("/bin/sh", "-c", script).CombinedOutput()
			if err != nil || string(got) != "2\nrelease\napply\nsynthetic reviewed plan" {
				t.Fatalf("copied command changed meaning: %q %v", got, err)
			}
		})
	}
}

func TestReleaseInstallCommandReadOnlyAndBadFlags(t *testing.T) {
	for _, args := range [][]string{{"release", "install", "--read-only"}, {"release", "install", "--version", "../bad"}, {"release", "install", "--unknown"}} {
		out, code := cli(t, args, "")
		if code != 2 || out.OK || out.Error.WriteMayHaveOccurred {
			t.Fatal("invalid/read-only install was not refused", out, code)
		}
	}
}

func TestReleaseMenuUnavailableAndOfflineAreNotUpdates(t *testing.T) {
	for _, code := range []string{"release.unavailable", "release.failed"} {
		m, out, calls, o := releaseMenuFixture(t, "2\n2\n")
		m.releaseInvoke = func(name string, _ any) api.Envelope {
			if name != "release_plan" {
				t.Fatal("failed preview attempted mutation")
			}
			return api.Failure(code, "Selected source unavailable, incompatible or offline.", false)
		}
		if err := m.installRelease(o); err == nil || *calls != 0 || strings.Contains(out.String(), "CLI installation verified") {
			t.Fatal("failed discovery claimed update", err, out.String())
		}
		if code == "release.unavailable" && !strings.Contains(out.String(), "No published release available") {
			t.Fatal("empty release state omitted")
		}
	}
}
