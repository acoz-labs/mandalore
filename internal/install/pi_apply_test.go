package install

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakePi struct {
	calls        []string
	fail         string
	interference bool
}

func (f *fakePi) run(_ context.Context, o Options, args ...string) ([]byte, error) {
	f.calls = append(f.calls, args[0])
	if args[0] == "--version" {
		return []byte("0.85.1\n"), nil
	}
	if args[0] == f.fail {
		return nil, errors.New("synthetic native interruption")
	}
	s, err := inspectPiSettings(o.NativeHome)
	if err != nil {
		return nil, err
	}
	packages := []any{}
	for _, p := range s.Packages {
		if args[0] == "remove" && p.Path == args[1] {
			continue
		}
		packages = append(packages, p.Value)
	}
	if args[0] == "install" {
		relative, _ := filepath.Rel(o.NativeHome, args[1])
		packages = append(packages, relative)
	}
	s.Other["packages"] = packages
	if f.interference {
		s.Other["theme"] = "changed unexpectedly"
	}
	raw, _ := json.MarshalIndent(s.Other, "", "  ")
	if err := os.MkdirAll(o.NativeHome, 0700); err != nil {
		return nil, err
	}
	return nil, os.WriteFile(filepath.Join(o.NativeHome, "settings.json"), raw, 0600)
}
func noPiProbe(context.Context, PiPlan) error { return nil }

func TestPiApplyInstallsRetainsAndIsIdempotent(t *testing.T) {
	o := PiOptions{Options: fixture(t)}
	if err := os.Mkdir(o.NativeHome, 0700); err != nil {
		t.Fatal(err)
	}
	settings := []byte(`{"theme":"light","packages":[{"source":"../unrelated","skills":[]}]}`)
	if err := os.WriteFile(filepath.Join(o.NativeHome, "settings.json"), settings, 0600); err != nil {
		t.Fatal(err)
	}
	p, err := PreparePi(o)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakePi{}
	r, err := applyPi(context.Background(), p, f.run, noPiProbe)
	if err != nil || !r.Installed || r.Phase != "verified" || !r.RequiresFreshSession || r.Uncertain || r.Attempt == "" {
		t.Fatal(r, err)
	}
	if _, err := os.Stat(filepath.Join(r.Attempt, "verified.json")); err != nil {
		t.Fatal("phase receipt missing", err)
	}
	f.calls = nil
	r, err = applyPi(context.Background(), p, f.run, noPiProbe)
	if err != nil || !r.Installed || !r.AlreadyCurrent || len(f.calls) != 1 || f.calls[0] != "--version" {
		t.Fatal("idempotent inspection mutated native state", r, err, f.calls)
	}
}

func TestPiApplyInterruptedUpdateRetainsBothGenerations(t *testing.T) {
	o := PiOptions{Options: fixture(t)}
	p, err := PreparePi(o)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakePi{}
	if _, err := applyPi(context.Background(), p, f.run, noPiProbe); err != nil {
		t.Fatal(err)
	}
	o.Generation = "update-one"
	next, err := PreparePi(o)
	if err != nil {
		t.Fatal(err)
	}
	f.fail = "install"
	r, err := applyPi(context.Background(), next, f.run, noPiProbe)
	if err == nil || r.Installed || r.Phase != "install-started" || !r.Uncertain {
		t.Fatal("partial update was hidden", r, err)
	}
	for _, root := range []string{p.Root, next.Root} {
		if _, err := ownedPi(root, p.StateDir, p.NativeHome, false); err != nil {
			t.Fatal("generation lost", err)
		}
	}
	s, err := inspectPiSettings(p.NativeHome)
	if err != nil {
		t.Fatal(err)
	}
	if root, err := piSelectedRegistration(s, p.StateDir); err != nil || root != "" {
		t.Fatal("inactive state not preserved", root, err)
	}
	if _, err := os.Stat(filepath.Join(r.Attempt, "registration-removed.json")); err != nil {
		t.Fatal("removal evidence missing", err)
	}
}

func TestPiApplyRefusesStaleSettingsAndReportsNativeInterference(t *testing.T) {
	for _, kind := range []string{"stale", "interference", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			o := PiOptions{Options: fixture(t)}
			p, err := PreparePi(o)
			if err != nil {
				t.Fatal(err)
			}
			f := &fakePi{interference: kind == "interference"}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "cancelled" {
				cancel()
			}
			if kind == "stale" {
				if err := os.Mkdir(o.NativeHome, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(o.NativeHome, "settings.json"), []byte(`{"theme":"new"}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			r, err := applyPi(ctx, p, f.run, noPiProbe)
			if err == nil || r.Installed {
				t.Fatal("unsafe success", r, err)
			}
			if kind != "interference" && len(f.calls) != 0 {
				t.Fatal("refusal invoked native binary", f.calls)
			}
			if kind == "interference" && !r.Uncertain {
				t.Fatal("interference not reported")
			}
		})
	}
}
