package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/install"
)

func piMenuFixture(t *testing.T, script string) (*menu, *bytes.Buffer, *int) {
	t.Helper()
	var out bytes.Buffer
	calls := 0
	m := &menu{ctx: context.Background(), in: bufio.NewReader(strings.NewReader(script)), out: &out,
		binding: "/synthetic/binding", binary: "/synthetic/selected-runtime",
		piProfile: install.Profile{StateDir: "/synthetic/state", NativeHome: "/synthetic/pi", NativeBinary: "/synthetic/pi-cli"}}
	m.prepareSelectedPi = func(_ context.Context, o install.PiOptions) (install.PiPlan, error) {
		if o.NativeHome != "/synthetic/pi" || o.Binding != m.binding || o.Binary != m.binary {
			t.Fatal("wrong selected connection", o)
		}
		return install.PiPlan{PiOptions: o, Harness: "pi", SignetID: "signet-synthetic", Root: "/synthetic/retained", PackageVersion: "1.2.3"}, nil
	}
	m.applySelectedPi = func(_ context.Context, p install.PiPlan) (install.PiResult, error) {
		calls++
		return install.PiResult{Connection: p, Installed: true, Phase: "verified", RequiresFreshSession: true}, nil
	}
	return m, &out, &calls
}

func TestPiMenuConnectRequiresReviewedConsent(t *testing.T) {
	for _, consent := range []string{"\n", "", "2", ":back\n", "2\n"} {
		// Harness, runtime, binding/profile/state/native, read-only, confirmation.
		m, out, calls := piMenuFixture(t, "2\n\n\n\n\n\n2\n"+consent)
		apply := m.applySelectedPi
		m.applySelectedPi = func(ctx context.Context, p install.PiPlan) (install.PiResult, error) {
			if !p.ReadOnly {
				t.Fatal("configured read-only mode lost")
			}
			return apply(ctx, p)
		}
		err := m.connect()
		if !strings.Contains(out.String(), "Review Pi connection") || !strings.Contains(out.String(), "Read-only (enforced)") {
			t.Fatal("missing preview", err, out.String())
		}
		if consent == "2\n" {
			if err != nil || *calls != 1 || !strings.Contains(out.String(), "fresh Pi session") {
				t.Fatal(err, *calls, out.String())
			}
		} else if *calls != 0 || !(errors.Is(err, console.ErrBack) || errors.Is(err, io.EOF)) {
			t.Fatal("unreviewed apply", err, *calls)
		}
	}
}

func TestPiMenuBackAndInvisiblePreviewCannotExecute(t *testing.T) {
	for _, script := range []string{"3\n", ":back\n", ""} {
		m, _, calls := piMenuFixture(t, script)
		m.prepareSelectedPi = func(context.Context, install.PiOptions) (install.PiPlan, error) {
			t.Fatal("back executed runtime")
			return install.PiPlan{}, nil
		}
		err := m.connect()
		if *calls != 0 || !(errors.Is(err, console.ErrBack) || errors.Is(err, io.EOF)) {
			t.Fatal(err)
		}
	}
	m, _, calls := piMenuFixture(t, "2\n\n\n\n\n\n1\n2\n")
	m.out = failedMenuWriter{}
	m.prepareSelectedPi = func(context.Context, install.PiOptions) (install.PiPlan, error) {
		t.Fatal("invisible preview executed runtime")
		return install.PiPlan{}, nil
	}
	if err := m.connect(); err == nil || *calls != 0 {
		t.Fatal(err)
	}
}

func TestPiMenuPartialReceiptAndDoctorFailureAreVisible(t *testing.T) {
	m, out, _ := piMenuFixture(t, "2\n\n\n\n\n\n1\n2\n")
	m.applySelectedPi = func(_ context.Context, p install.PiPlan) (install.PiResult, error) {
		return install.PiResult{Connection: p, Phase: "install-started", Attempt: "/synthetic/attempt", Uncertain: true}, errors.New("synthetic cancellation")
	}
	if err := m.connect(); err == nil || !strings.Contains(out.String(), "install-started") || !strings.Contains(out.String(), "/synthetic/attempt") || strings.Contains(out.String(), "[PASS] Pi connection verified") {
		t.Fatal(err, out.String())
	}
	v := api.Failure("connection.failed", "synthetic inspection failure", false)
	v.Error.PiConnectionReport = &install.PiReport{Checks: []install.Check{{Name: "native-registration", Status: "fail", Detail: "No connection"}}}
	if err := m.outcome("Pi inspection", v); err == nil || !strings.Contains(out.String(), "[FAIL] native-registration") {
		t.Fatal(err, out.String())
	}
}

func TestPiMenuReleaseSelectsInstalledRuntimeAndSeparateConsent(t *testing.T) {
	for _, consent := range []string{"\n", "2\n"} {
		m, out, _, o := releaseMenuFixture(t, "2\n3\n\n\n\n\n1\n"+consent)
		m.piProfile = install.Profile{StateDir: "/synthetic/state", NativeHome: "/synthetic/pi", NativeBinary: "/synthetic/pi-cli"}
		applies := 0
		m.prepareSelectedPi = func(_ context.Context, options install.PiOptions) (install.PiPlan, error) {
			if !strings.HasPrefix(options.Binary, o.Prefix) || options.NativeHome != m.piProfile.NativeHome {
				t.Fatal(options)
			}
			return install.PiPlan{PiOptions: options, BinarySHA256: strings.Repeat("d", 64), PackageSHA256: strings.Repeat("e", 64), PackageVersion: "1.0.0"}, nil
		}
		m.applySelectedPi = func(_ context.Context, p install.PiPlan) (install.PiResult, error) {
			applies++
			return install.PiResult{Connection: p, Installed: true, RequiresFreshSession: true, Phase: "verified"}, nil
		}
		err := m.installRelease(o)
		if consent == "\n" {
			if applies != 0 || !errors.Is(err, console.ErrBack) {
				t.Fatal(err, applies)
			}
		} else if err != nil || applies != 1 || !strings.Contains(out.String(), "[PASS] Pi connection verified") {
			t.Fatal(err, applies, out.String())
		}
	}
}

func TestPiMenuReleaseRefusesIdentityMismatchAndKeepsPartialEvidence(t *testing.T) {
	for _, kind := range []string{"runtime", "version", "partial"} {
		t.Run(kind, func(t *testing.T) {
			m, out, _, o := releaseMenuFixture(t, "2\n3\n\n\n\n\n1\n2\n")
			m.piProfile = install.Profile{StateDir: "/synthetic/state", NativeHome: "/synthetic/pi", NativeBinary: "/synthetic/pi-cli"}
			m.prepareSelectedPi = func(_ context.Context, options install.PiOptions) (install.PiPlan, error) {
				p := install.PiPlan{PiOptions: options, BinarySHA256: strings.Repeat("d", 64), PackageVersion: "1.0.0"}
				if kind == "runtime" {
					p.BinarySHA256 = strings.Repeat("f", 64)
				}
				if kind == "version" {
					p.PackageVersion = "0.0.0-dev"
				}
				return p, nil
			}
			m.applySelectedPi = func(_ context.Context, p install.PiPlan) (install.PiResult, error) {
				if kind != "partial" {
					t.Fatal("mismatched plan applied")
				}
				return install.PiResult{Connection: p, Phase: "registration-removed", Uncertain: true}, errors.New("synthetic interruption")
			}
			err := m.installRelease(o)
			if err == nil || !strings.Contains(err.Error(), "CLI installation remains complete") || !strings.Contains(out.String(), "CLI installation verified") || strings.Contains(out.String(), "[PASS] Pi connection verified") {
				t.Fatal(err, out.String())
			}
			if kind == "partial" && !strings.Contains(out.String(), "registration-removed") {
				t.Fatal("lost partial receipt", out.String())
			}
		})
	}
}

func TestPiMenuInspectionUsesSelectedProfileWithoutCreatingIt(t *testing.T) {
	dir := t.TempDir()
	profile := install.Profile{StateDir: dir + "/state", NativeHome: dir + "/pi", NativeBinary: dir + "/pi-cli"}
	m, out, _ := piMenuFixture(t, "2\n\n\n\n")
	m.piProfile = profile
	// Resolving Codex earlier must not redirect the Pi journey.
	m.profile = install.Profile{StateDir: "/synthetic/codex-state", NativeHome: "/synthetic/codex", NativeBinary: "/synthetic/codex-cli"}
	if err := m.doctor(); err == nil || !strings.Contains(out.String(), "[FAIL] native-profile") || !strings.Contains(strings.Join(strings.Fields(out.String()), ""), profile.NativeHome) {
		t.Fatal(err, out.String())
	}
	if m.piProfile != profile {
		t.Fatal("Pi defaults inherited Codex")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatal("inspection created state", err, entries)
	}
}
