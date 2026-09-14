package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/distribution"
	"github.com/acoz-labs/mandalore/internal/install"
)

// Presentation fixtures simulate API results. Actual native CLI installation is
// tested separately; these fixtures do not claim executable verification.
func releaseMenuFixture(t *testing.T, script string) (*menu, *bytes.Buffer, *int, distribution.InstallOptions) {
	t.Helper()
	var out bytes.Buffer
	calls := 0
	prefix := filepath.Join(t.TempDir(), "prefix")
	o := distribution.InstallOptions{Prefix: prefix, Candidate: "/synthetic/candidate"}
	p := distribution.InstallPlan{InstallOptions: o, OS: "darwin", Arch: "arm64", Source: distribution.InstallSource{Kind: "local-candidate", Manifest: distribution.ParsedManifest{SHA256: strings.Repeat("a", 64), Manifest: distribution.Manifest{Version: "1.0.0", SourceCommit: strings.Repeat("b", 40), PluginSHA256: strings.Repeat("c", 64)}}}, Binary: distribution.Asset{SHA256: strings.Repeat("d", 64)}, Launcher: filepath.Join(prefix, "bin", "mandalore"), Runtime: filepath.Join(prefix, "lib", "mandalore", "retained-runtime")}
	m := &menu{ctx: context.Background(), in: bufio.NewReader(strings.NewReader(script)), out: &out, binding: "/synthetic/binding", profile: install.Profile{StateDir: "/synthetic/state", NativeHome: "/synthetic/native", NativeBinary: "/synthetic/codex"}}
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

func TestReleaseRecoveryInstructionQuotesPendingPath(t *testing.T) {
	m, out, _, _ := releaseMenuFixture(t, "")
	m.releaseResult(distribution.InstallResult{Pending: "/synthetic/owner's tools/pending.json"})
	if !strings.Contains(out.String(), "'/synthetic/owner'\"'\"'s tools/pending.json'") || !strings.Contains(out.String(), "original Mandalore executable") {
		t.Fatal("unsafe or ambiguous recovery instruction", out.String())
	}
}

func TestReleaseMenuConnectionGetsNewRuntimeAndSeparateConsent(t *testing.T) {
	for _, consent := range []string{"\n", "2\n"} {
		m, out, _, o := releaseMenuFixture(t, "2\n2\n\n\n\n\n"+consent)
		applies := 0
		m.prepareSelectedConnection = func(_ context.Context, options install.Options) (install.Plan, error) {
			if !strings.HasPrefix(options.Binary, o.Prefix) || options.Binding != m.binding {
				t.Fatal("handoff did not select installed runtime and explicit binding")
			}
			return install.Plan{Options: options, BinarySHA256: strings.Repeat("d", 64), PackageSHA256: strings.Repeat("c", 64), PackageVersion: "1.0.0"}, nil
		}
		m.applySelectedConnection = func(_ context.Context, p install.Plan) (install.Result, error) {
			applies++
			return install.Result{Connection: p, Installed: true, RequiresFreshSession: true, Phase: "verified"}, nil
		}
		err := m.installRelease(o)
		if consent == "\n" {
			if applies != 0 || (!errors.Is(err, console.ErrBack) && err != nil) {
				t.Fatal("native update applied without second consent", err, applies)
			}
		} else if err != nil || applies != 1 || !strings.Contains(out.String(), "Native connection verified") {
			t.Fatal("selected native update failed", err, applies, out.String())
		}
	}
}

func TestReleaseMenuRefusesOldPackageAndKeepsCLISuccessVisible(t *testing.T) {
	m, out, _, o := releaseMenuFixture(t, "2\n2\n\n\n\n\n2\n")
	m.prepareSelectedConnection = func(_ context.Context, options install.Options) (install.Plan, error) {
		return install.Plan{Options: options, BinarySHA256: strings.Repeat("d", 64), PackageSHA256: strings.Repeat("e", 64), PackageVersion: "1.0.0"}, nil
	}
	m.applySelectedConnection = func(context.Context, install.Plan) (install.Result, error) {
		t.Fatal("old package applied")
		return install.Result{}, nil
	}
	if err := m.installRelease(o); err == nil || !strings.Contains(out.String(), "CLI installation verified") {
		t.Fatal("package mismatch or prior success lost", err, out.String())
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

func TestReleaseMenuNativePartialRetainsCLISuccessAndPhase(t *testing.T) {
	m, out, _, o := releaseMenuFixture(t, "2\n2\n\n\n\n\n2\n")
	m.prepareSelectedConnection = func(_ context.Context, options install.Options) (install.Plan, error) {
		return install.Plan{Options: options, BinarySHA256: strings.Repeat("d", 64), PackageSHA256: strings.Repeat("c", 64), PackageVersion: "1.0.0"}, nil
	}
	m.applySelectedConnection = func(_ context.Context, p install.Plan) (install.Result, error) {
		return install.Result{Connection: p, Phase: "registration-removed", PreviousRoot: "/synthetic/previous"}, errors.New("synthetic native failure")
	}
	err := m.installRelease(o)
	if err == nil || !strings.Contains(err.Error(), "CLI installation remains complete") || !strings.Contains(out.String(), "registration-removed") || !strings.Contains(out.String(), "CLI installation verified") || strings.Contains(out.String(), "[PASS] Native connection verified") {
		t.Fatal("native failure obscured CLI success", err, out.String())
	}
}
