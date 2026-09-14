package distribution

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func localInstallOptions(t *testing.T) InstallOptions {
	t.Helper()
	candidate, _, _ := candidateFixture(t)
	return InstallOptions{Candidate: candidate, Prefix: filepath.Join(t.TempDir(), "prefix")}
}

func TestInstallPlanAbsentPrefixIsReadOnly(t *testing.T) {
	o := localInstallOptions(t)
	p, err := PlanInstall(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if p.FormatVersion != 1 || p.OS != runtime.GOOS || p.Arch != runtime.GOARCH || p.Source.Kind != "local-candidate" || p.Observed.ReceiptSHA256 != "" || p.Observed.LauncherTarget != "" {
		t.Fatal("incorrect plan", p)
	}
	if p.Launcher != filepath.Join(p.Prefix, "bin", "mandalore") || !strings.Contains(p.Runtime, p.Source.Manifest.SHA256) || p.Binary.Kind != "cli" || p.Binary.OS != runtime.GOOS || !strings.Contains(p.Notice, "unchanged") {
		t.Fatal("plan omitted exact paths, platform, identity or effects")
	}
	if _, err := os.Lstat(o.Prefix); !os.IsNotExist(err) {
		t.Fatal("preview wrote prefix", err)
	}
	if _, err := VerifyDirectory(o.Candidate); err != nil {
		t.Fatal("preview changed source", err)
	}
	// Strict round trip binds the complete observed plan, not a selected version.
	b, _ := json.Marshal(p)
	got, err := ParseInstallPlan(b)
	if err != nil || !reflect.DeepEqual(got, p) {
		t.Fatal("plan does not round trip", err)
	}
	for _, b := range [][]byte{append(b, []byte(` {}`)...), []byte(`{"format_version":1,"format_version":1}`), []byte(`{"extra":true}`), []byte(strings.Repeat(" ", MaxInstallPlanBytes+1))} {
		if _, err := ParseInstallPlan(b); err == nil {
			t.Fatal("invalid serialized plan accepted")
		}
	}
}

// Fixture construction is not successful installation evidence. The installer
// writer must separately pass actual activation and recovery tests.
func fixtureOwnedInstall(t *testing.T, o InstallOptions) InstallPlan {
	t.Helper()
	p, err := PlanInstall(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(p.Runtime)
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	for from, to := range map[string]string{"manifest.json": "manifest.json", p.Binary.Name: "mandalore"} {
		b, err := os.ReadFile(filepath.Join(o.Candidate, from))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, to), b, 0700); err != nil {
			t.Fatal(err)
		}
	}
	r := cliReceipt{FormatVersion: 1, Product: "mandalore", Prefix: p.Prefix, Current: p.Source.Manifest.SHA256}
	b, _ := json.Marshal(r)
	if err := os.WriteFile(filepath.Join(p.Prefix, "lib", "mandalore", "receipt.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p.Launcher), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(p.Runtime, p.Launcher); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInstallPlanOwnedAndRetainedSelection(t *testing.T) {
	o := localInstallOptions(t)
	installed := fixtureOwnedInstall(t, o)
	p, err := PlanInstall(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if p.Observed.ReceiptSHA256 == "" || p.Observed.LauncherTarget != installed.Runtime || p.Observed.Current != installed.Source.Manifest.SHA256 || !p.RuntimeRetained {
		t.Fatal("owned installation not recognized")
	}
	retained, err := PlanInstall(context.Background(), InstallOptions{Prefix: o.Prefix, Retained: installed.Source.Manifest.SHA256})
	if err != nil || retained.Source.Kind != "retained" || retained.Runtime != installed.Runtime || !reflect.DeepEqual(retained.Source.Manifest, p.Source.Manifest) {
		t.Fatal("retained compatible selection failed", err)
	}
	// Source and destination identity are re-observed; same options are not a
	// promise that the preview remains current after another installer acts.
	before := p.Observed.ReceiptSHA256
	path := filepath.Join(p.Prefix, "lib", "mandalore", "receipt.json")
	b, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	p, err = PlanInstall(context.Background(), o)
	if err != nil || p.Observed.ReceiptSHA256 == before {
		t.Fatal("receipt byte change invisible", err)
	}
}

func TestInstallPlanRefusesForeignOrPartialDestinations(t *testing.T) {
	for _, kind := range []string{"regular-launcher", "foreign-symlink", "dangling-launcher", "bin-link", "lib-link", "state-link", "unknown-state", "receipt-without-launcher", "launcher-without-receipt", "edited-runtime", "edited-manifest", "receipt-link", "receipt-fifo", "retained-link", "receipt-extra", "receipt-prefix", "receipt-format", "pending", "writable-prefix"} {
		t.Run(kind, func(t *testing.T) {
			o := localInstallOptions(t)
			p := fixtureOwnedInstall(t, o)
			state := filepath.Join(p.Prefix, "lib", "mandalore")
			receipt := filepath.Join(state, "receipt.json")
			remove := func(path string) {
				t.Helper()
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			put := func(path, text string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			link := func(target, path string) {
				t.Helper()
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "regular-launcher":
				remove(p.Launcher)
				put(p.Launcher, "user-owned command")
			case "foreign-symlink":
				remove(p.Launcher)
				link("/usr/bin/true", p.Launcher)
			case "dangling-launcher":
				remove(p.Launcher)
				link("/synthetic/missing", p.Launcher)
			case "bin-link", "lib-link", "state-link", "retained-link":
				path := filepath.Dir(p.Runtime)
				if kind == "bin-link" {
					path = filepath.Join(p.Prefix, "bin")
				}
				if kind == "lib-link" {
					path = filepath.Join(p.Prefix, "lib")
				}
				if kind == "state-link" {
					path = state
				}
				moved := filepath.Join(t.TempDir(), "moved")
				if err := os.Rename(path, moved); err != nil {
					t.Fatal(err)
				}
				link(moved, path)
			case "unknown-state":
				remove(receipt)
				remove(p.Launcher)
			case "receipt-without-launcher":
				remove(p.Launcher)
			case "launcher-without-receipt":
				remove(receipt)
			case "edited-runtime":
				put(p.Runtime, "corrupted runtime")
			case "edited-manifest":
				put(filepath.Join(filepath.Dir(p.Runtime), "manifest.json"), "{}")
			case "receipt-link":
				remove(receipt)
				link(filepath.Join(o.Candidate, "manifest.json"), receipt)
			case "receipt-fifo":
				remove(receipt)
				if err := unix.Mkfifo(receipt, 0600); err != nil {
					t.Fatal(err)
				}
			case "receipt-extra":
				b, _ := os.ReadFile(receipt)
				put(receipt, strings.TrimSuffix(string(b), "}")+`,"unexpected":1}`)
			case "receipt-prefix":
				b, _ := os.ReadFile(receipt)
				var r cliReceipt
				_ = json.Unmarshal(b, &r)
				r.Prefix = "/another-prefix"
				b, _ = json.Marshal(r)
				put(receipt, string(b))
			case "receipt-format":
				b, _ := os.ReadFile(receipt)
				var r cliReceipt
				_ = json.Unmarshal(b, &r)
				r.FormatVersion = 2
				b, _ = json.Marshal(r)
				put(receipt, string(b))
			case "pending":
				put(filepath.Join(state, "pending.json"), "{}")
			case "writable-prefix":
				if err := os.Chmod(p.Prefix, 0777); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := PlanInstall(context.Background(), o); err == nil {
				t.Fatal("foreign, corrupted or partial installation accepted")
			}
		})
	}
}

func TestInstallPlanRejectsInvalidSelectionAndCancellation(t *testing.T) {
	o := localInstallOptions(t)
	for _, mutate := range []func(*InstallOptions){
		func(o *InstallOptions) { o.Version = "../latest" },
		func(o *InstallOptions) { o.Version = "1.0.0" },
		func(o *InstallOptions) { o.Retained = strings.Repeat("a", 64) },
		func(o *InstallOptions) { o.Candidate = ""; o.Retained = "../outside" },
		func(o *InstallOptions) { o.Prefix = "" },
		func(o *InstallOptions) { o.Prefix = "/" },
		func(o *InstallOptions) { o.Prefix = "/example\npath" },
	} {
		bad := o
		mutate(&bad)
		if _, err := PlanInstall(context.Background(), bad); err == nil {
			t.Fatal("invalid plan options accepted", bad)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := PlanInstall(ctx, o); err != context.Canceled {
		t.Fatal("cancellation ignored", err)
	}
}

func TestInstallPlanPublishedPinsSelectionWithoutDownloadingExecutable(t *testing.T) {
	f := newReleaseFixture(t)
	o := InstallOptions{Prefix: filepath.Join(t.TempDir(), "prefix")}
	p, err := planInstall(context.Background(), o, f.client)
	if err != nil {
		t.Fatal(err)
	}
	if p.Version != "1.0.0" || p.Source.Kind != "github-release" || p.Source.Published.ID != 42 || !reflect.DeepEqual(p.Source.Published.Manifest, p.Source.Manifest) {
		t.Fatal("published plan did not pin latest to exact release")
	}
	for _, a := range f.assets {
		if a["name"] != "manifest.json" && a["name"] != "SHA256SUMS" {
			for _, request := range f.requests {
				if strings.HasSuffix(request, fmt.Sprintf("/assets/%d", a["id"])) {
					t.Fatal("preview downloaded a payload")
				}
			}
		}
	}
	if _, err := os.Lstat(o.Prefix); !os.IsNotExist(err) {
		t.Fatal("published preview created destination")
	}
	b, _ := json.Marshal(p)
	if _, err := ParseInstallPlan(b); err != nil {
		t.Fatal("published plan does not parse", err)
	}
}

func TestInstallPlanRejectsChangedEffectsAndIdentities(t *testing.T) {
	p, err := PlanInstall(context.Background(), localInstallOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*InstallPlan){
		func(p *InstallPlan) { p.FormatVersion = 2 },
		func(p *InstallPlan) { p.Source.Manifest.Manifest.ProtocolVersion = 2 },
		func(p *InstallPlan) { p.Source.Manifest.SHA256 = "../escape" },
		func(p *InstallPlan) { p.Source.Kind = "unknown" },
		func(p *InstallPlan) { p.Source.Kind = "github-release" },
		func(p *InstallPlan) { p.Source.Kind = "retained" },
		func(p *InstallPlan) { p.Launcher = "/user-owned-command" },
		func(p *InstallPlan) { p.Runtime = "/user-owned-runtime" },
		func(p *InstallPlan) { p.OS = "windows" },
		func(p *InstallPlan) { p.Arch = "386" },
		func(p *InstallPlan) { p.Binary.SHA256 = strings.Repeat("0", 64) },
		func(p *InstallPlan) { p.Notice = "also switch all memory connections" },
	} {
		b, _ := json.Marshal(p)
		var bad InstallPlan
		_ = json.Unmarshal(b, &bad)
		mutate(&bad)
		b, _ = json.Marshal(bad)
		if _, err := ParseInstallPlan(b); err == nil {
			t.Fatal("changed plan effects or identity accepted")
		}
	}
}
