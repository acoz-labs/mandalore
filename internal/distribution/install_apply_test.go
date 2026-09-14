package distribution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// State-machine fixtures use inert payloads. Only this explicit test verifier
// bypasses native execution; ApplyInstall always uses the production verifier.
func inertInstallVerifier(context.Context, string, InstallPlan) error { return nil }

func installForTest(t *testing.T, p InstallPlan, after func(string) error) (InstallResult, error) {
	t.Helper()
	return applyInstall(context.Background(), p, NewReleaseClient(), inertInstallVerifier, after)
}

func TestInstallApplyRetainsActivatesAndReuses(t *testing.T) {
	o := localInstallOptions(t)
	p, err := PlanInstall(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	r, err := installForTest(t, p, nil)
	if err != nil || !r.Installed || r.Phase != "complete" || r.Connections != "unchanged" || !r.DestinationChanged {
		t.Fatal("installation failed", r, err)
	}
	got, err := os.Readlink(p.Launcher)
	if err != nil || got != p.Runtime {
		t.Fatal("launcher not activated", err)
	}
	installed, err := retainedManifest(p.Prefix, p.Source.Manifest.SHA256, p.OS, p.Arch)
	if err != nil || !reflect.DeepEqual(installed, p.Source.Manifest) {
		t.Fatal("retained bytes differ", err)
	}
	if _, err := os.Lstat(filepath.Join(cliState(p.Prefix), "pending.json")); !os.IsNotExist(err) {
		t.Fatal("successful activation retained pending state")
	}
	r, err = installForTest(t, p, nil)
	if err != nil || !r.Installed || r.DestinationChanged {
		t.Fatal("exact completed plan did not replay without writes", r, err)
	}
	p2, err := PlanInstall(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	r, err = installForTest(t, p2, nil)
	if err != nil || !r.Installed || r.DestinationChanged {
		t.Fatal("already-current selection rewrote installation", r, err)
	}
}

func nextInstallCandidate(t *testing.T) string {
	t.Helper()
	dir, m, _ := candidateFixture(t)
	m.SourceCommit = strings.Repeat("b", 40)
	for i := range m.Assets {
		a := &m.Assets[i]
		if a.Kind != "cli" {
			continue
		}
		b := []byte("another synthetic CLI: " + a.Name)
		a.Size, a.SHA256 = int64(len(b)), Digest(b)
		if err := os.WriteFile(filepath.Join(dir, a.Name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	b, err := EncodeManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	sums, err := Checksums(b)
	if err != nil {
		t.Fatal(err)
	}
	for name, b := range map[string][]byte{"manifest.json": b, "SHA256SUMS": sums} {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestInstallApplyUpdateAndCompatibleRetainedRollback(t *testing.T) {
	o := localInstallOptions(t)
	first, err := PlanInstall(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installForTest(t, first, nil); err != nil {
		t.Fatal(err)
	}
	o.Candidate = nextInstallCandidate(t)
	second, err := PlanInstall(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	r, err := installForTest(t, second, nil)
	if err != nil || r.PreviousRuntime != first.Runtime || !r.Installed {
		t.Fatal("update failed", r, err)
	}
	if _, err := retainedManifest(first.Prefix, first.Source.Manifest.SHA256, first.OS, first.Arch); err != nil {
		t.Fatal("update removed previous runtime", err)
	}
	rollback, err := PlanInstall(context.Background(), InstallOptions{Prefix: o.Prefix, Retained: first.Source.Manifest.SHA256})
	if err != nil {
		t.Fatal(err)
	}
	r, err = installForTest(t, rollback, nil)
	if err != nil || r.Runtime != first.Runtime || r.PreviousRuntime != second.Runtime {
		t.Fatal("compatible rollback failed", r, err)
	}
	if _, err := retainedManifest(second.Prefix, second.Source.Manifest.SHA256, second.OS, second.Arch); err != nil {
		t.Fatal("rollback removed newer runtime", err)
	}
}

func TestInstallApplyRefusesStalePlansBeforeDestinationWrites(t *testing.T) {
	for _, kind := range []string{"source", "prefix", "launcher", "plan", "probe"} {
		t.Run(kind, func(t *testing.T) {
			o := localInstallOptions(t)
			p, err := PlanInstall(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			probe := inertInstallVerifier
			switch kind {
			case "source":
				if err := os.WriteFile(filepath.Join(o.Candidate, p.Binary.Name), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "prefix":
				if err := os.Mkdir(o.Prefix, 0700); err != nil {
					t.Fatal(err)
				}
			case "launcher":
				if err := os.MkdirAll(filepath.Dir(p.Launcher), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p.Launcher, []byte("foreign"), 0600); err != nil {
					t.Fatal(err)
				}
			case "plan":
				p.Runtime = "/synthetic/foreign"
			case "probe":
				probe = func(context.Context, string, InstallPlan) error { return errors.New("fixture probe refused") }
			}
			r, err := applyInstall(context.Background(), p, NewReleaseClient(), probe, nil)
			if err == nil || r.DestinationChanged || r.Installed {
				t.Fatal("stale plan or bad runtime wrote destination", r, err)
			}
			if _, err := os.Lstat(cliState(p.Prefix)); !os.IsNotExist(err) {
				t.Fatal("refusal created managed state")
			}
		})
	}
}

func TestInstallApplyInterruptedActivationCanResumeExactPlan(t *testing.T) {
	for _, existing := range []bool{false, true} {
		for _, phase := range []string{"pending-recorded", "launcher-activated", "receipt-recorded"} {
			t.Run(fmt.Sprintf("existing=%v/%s", existing, phase), func(t *testing.T) {
				o := localInstallOptions(t)
				if existing {
					p, err := PlanInstall(context.Background(), o)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := installForTest(t, p, nil); err != nil {
						t.Fatal(err)
					}
					o.Candidate = nextInstallCandidate(t)
				}
				p, err := PlanInstall(context.Background(), o)
				if err != nil {
					t.Fatal(err)
				}
				r, err := installForTest(t, p, func(at string) error {
					if at == phase {
						return context.Canceled
					}
					return nil
				})
				if !errors.Is(err, context.Canceled) || !r.DestinationChanged || r.Pending == "" || r.Installed {
					t.Fatal("interruption lost partial-state evidence", r, err)
				}
				if _, err := PlanInstall(context.Background(), o); err == nil {
					t.Fatal("pending activation accepted as clean installation")
				}
				r, err = installForTest(t, p, nil)
				if err != nil || !r.Installed || r.Phase != "complete" {
					t.Fatal("matching interrupted activation did not recover", r, err)
				}
			})
		}
	}
}

func TestInstallApplyRecoveryRefusesDifferentPlanOrChangedState(t *testing.T) {
	for _, kind := range []string{"plan", "launcher", "receipt", "runtime", "pending"} {
		t.Run(kind, func(t *testing.T) {
			o := localInstallOptions(t)
			p, err := PlanInstall(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := installForTest(t, p, func(at string) error {
				if at == "pending-recorded" {
					return context.Canceled
				}
				return nil
			}); err == nil {
				t.Fatal("did not interrupt")
			}
			switch kind {
			case "plan":
				p.Source.Manifest.SHA256 = strings.Repeat("0", 64)
			case "launcher":
				if err := os.WriteFile(p.Launcher, []byte("foreign"), 0600); err != nil {
					t.Fatal(err)
				}
			case "receipt":
				if err := os.WriteFile(filepath.Join(cliState(p.Prefix), "receipt.json"), []byte("foreign"), 0600); err != nil {
					t.Fatal(err)
				}
			case "runtime":
				if err := os.WriteFile(p.Runtime, []byte("corrupted"), 0700); err != nil {
					t.Fatal(err)
				}
			case "pending":
				if err := os.WriteFile(filepath.Join(cliState(p.Prefix), "pending.json"), []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if r, err := installForTest(t, p, nil); err == nil || r.Installed {
				t.Fatal("conflicting recovery accepted", r, err)
			}
		})
	}
}

func TestInstallApplySerializesAndRevalidatesAfterStaging(t *testing.T) {
	o := localInstallOptions(t)
	p, err := PlanInstall(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installForTest(t, p, nil); err != nil {
		t.Fatal(err)
	}
	o.Candidate = nextInstallCandidate(t)
	p, err = PlanInstall(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	var secondErr error
	r, err := installForTest(t, p, func(at string) error {
		if at == "lock-acquired" {
			_, secondErr = installForTest(t, p, nil)
		}
		return nil
	})
	if err != nil || !r.Installed || secondErr == nil {
		t.Fatal("concurrent apply not serialized", r, err, secondErr)
	}

	o = localInstallOptions(t)
	p, err = PlanInstall(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	r, err = installForTest(t, p, func(at string) error {
		if at == "verified" {
			return os.Mkdir(o.Prefix, 0700)
		}
		return nil
	})
	if err == nil || r.DestinationChanged {
		t.Fatal("destination change during staging was not rejected", r, err)
	}
}

func TestInstallApplyProductionRejectsInertPayloadWithoutActivation(t *testing.T) {
	o := localInstallOptions(t)
	p, err := PlanInstall(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	r, err := ApplyInstall(context.Background(), p)
	if err == nil || r.DestinationChanged || r.Installed {
		t.Fatal("inert bytes passed production executable verification", r, err)
	}
	if _, err := os.Lstat(o.Prefix); !os.IsNotExist(err) {
		t.Fatal("probe refusal created prefix")
	}
	b, _ := json.Marshal(p)
	if _, err := ParseInstallPlan(b); err != nil {
		t.Fatal(err)
	}
}

func TestInstallApplyPublishedPayloadAndDownloadFailure(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(fmt.Sprintf("corrupt=%v", corrupt), func(t *testing.T) {
			f := newReleaseFixture(t)
			p, err := planInstall(context.Background(), InstallOptions{Prefix: filepath.Join(t.TempDir(), "prefix")}, f.client)
			if err != nil {
				t.Fatal(err)
			}
			if corrupt {
				for _, a := range f.assets {
					if a["name"] == p.Binary.Name {
						f.body[fmt.Sprintf("/repos/acoz-labs/mandalore/releases/assets/%d", a["id"])] = []byte("corrupt")
					}
				}
			}
			r, err := applyInstall(context.Background(), p, f.client, inertInstallVerifier, nil)
			if corrupt {
				if err == nil || r.DestinationChanged || r.Installed {
					t.Fatal("corrupt download activated", r, err)
				}
				if _, err := os.Lstat(p.Prefix); !os.IsNotExist(err) {
					t.Fatal("failed download wrote prefix")
				}
			} else if err != nil || !r.Installed {
				t.Fatal("verified published fixture failed", r, err)
			}
		})
	}
}

func TestInstallApplyPermissionFailuresPreserveRecovery(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission denial requires a non-root process")
	}
	for _, point := range []string{"pending-recorded", "launcher-activated"} {
		t.Run(point, func(t *testing.T) {
			o := localInstallOptions(t)
			first, err := PlanInstall(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := installForTest(t, first, nil); err != nil {
				t.Fatal(err)
			}
			old, err := os.ReadFile(filepath.Join(cliState(first.Prefix), "receipt.json"))
			if err != nil {
				t.Fatal(err)
			}
			o.Candidate = nextInstallCandidate(t)
			p, err := PlanInstall(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			path := cliState(p.Prefix)
			if point == "pending-recorded" {
				path = filepath.Dir(p.Launcher)
			}
			defer os.Chmod(path, 0700)
			r, err := installForTest(t, p, func(at string) error {
				if at == point {
					return os.Chmod(path, 0500)
				}
				return nil
			})
			if err == nil || !r.DestinationChanged || r.Pending == "" {
				t.Fatal("write failure lost recovery state", r, err)
			}
			if !sameInstallBytes(filepath.Join(cliState(p.Prefix), "receipt.json"), old) {
				t.Fatal("failed activation replaced previous receipt")
			}
			if err := os.Chmod(path, 0700); err != nil {
				t.Fatal(err)
			}
			if r, err := installForTest(t, p, nil); err != nil || !r.Installed {
				t.Fatal("permission failure could not recover", r, err)
			}
		})
	}
}

func TestRuntimeVersionVerificationBindsEveryIdentity(t *testing.T) {
	p, err := PlanInstall(context.Background(), localInstallOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	m := p.Source.Manifest.Manifest
	fields := map[string]any{"name": "mandalore", "version": m.Version, "source_commit": m.SourceCommit, "os": p.OS, "arch": p.Arch, "go_version": m.GoVersion, "protocol_version": 1, "codex_hook_protocol": 1, "signet_read_versions": []int{1}, "signet_write_versions": []int{1}, "plugin_version": m.Version, "plugin_sha256": m.PluginSHA256}
	raw, _ := json.Marshal(map[string]any{"protocol_version": 1, "ok": true, "result": fields})
	if err := verifyRuntimeVersion(raw, p); err != nil {
		t.Fatal("valid metadata refused", err)
	}
	for key := range fields {
		old := fields[key]
		fields[key] = "wrong"
		raw, _ := json.Marshal(map[string]any{"protocol_version": 1, "ok": true, "result": fields})
		if err := verifyRuntimeVersion(raw, p); err == nil {
			t.Fatal("wrong metadata accepted", key)
		}
		fields[key] = old
	}
	if err := verifyRuntimeVersion([]byte(`{"ok":true,"ok":true}`), p); err == nil {
		t.Fatal("duplicate runtime JSON accepted")
	}
}
