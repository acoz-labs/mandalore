package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func assertInventoryOnly(t *testing.T, calls []string) {
	t.Helper()
	for _, call := range calls {
		if call != "plugin marketplace list --json" && call != "plugin list --json" {
			t.Fatalf("unexpected native mutation: %s", call)
		}
	}
}

func TestUpgradeDefaultsToDeferredAndPreservesOldCache(t *testing.T) {
	p, err := Prepare(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeNative{plan: p, clearCacheOnRemove: true}
	if _, err := apply(context.Background(), p, f.run, noProbe); err != nil {
		t.Fatal(err)
	}
	o := p.Options
	o.Generation = "next"
	next, err := Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	f.plan, f.calls = next, nil
	result, err := apply(context.Background(), next, f.run, noProbe)
	if err == nil || result.Installed || result.Phase != "deferred" {
		t.Fatalf("replacement was not deferred: %+v %v", result, err)
	}
	assertInventoryOnly(t, f.calls)
	if f.root != p.Root || f.pluginRoot != p.Root {
		t.Fatal("old registration changed")
	}
	if err := verifyCache(p, false); err != nil {
		t.Fatal("old cache changed", err)
	}
	if _, err := os.Stat(next.Root); !os.IsNotExist(err) {
		t.Fatal("denied replacement staged a new bundle", err)
	}
}

func TestVerifiedReplayDoesNotReinstall(t *testing.T) {
	p, err := Prepare(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeNative{plan: p}
	if _, err := apply(context.Background(), p, f.run, noProbe); err != nil {
		t.Fatal(err)
	}
	f.calls = nil
	result, err := apply(context.Background(), p, f.run, noProbe)
	if err != nil || !result.Installed {
		t.Fatal(result, err)
	}
	assertInventoryOnly(t, f.calls)
}

func TestStoppedSessionAcknowledgementAllowsReplacement(t *testing.T) {
	p, err := Prepare(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeNative{plan: p, clearCacheOnRemove: true}
	if _, err := apply(context.Background(), p, f.run, noProbe); err != nil {
		t.Fatal(err)
	}
	o := p.Options
	o.Generation = "acknowledged"
	next, err := Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	f.plan = next
	result, err := applyAcknowledged(context.Background(), next, true, f.run, noProbe)
	if err != nil || !result.Installed || f.root != next.Root {
		t.Fatal(result, err)
	}
	if err := verifyCache(next, false); err != nil {
		t.Fatal(err)
	}
	// The assertion is not retained permission for another replacement.
	o.Generation = "later"
	later, err := Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	f.plan, f.calls = later, nil
	result, err = apply(context.Background(), later, f.run, noProbe)
	if err == nil || result.Phase != "deferred" {
		t.Fatal(result, err)
	}
	assertInventoryOnly(t, f.calls)
}

func TestMissingCacheIsNotVerifiedReplay(t *testing.T) {
	p, err := Prepare(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeNative{plan: p}
	if _, err := apply(context.Background(), p, f.run, noProbe); err != nil {
		t.Fatal(err)
	}
	// Remove one generated dependency from this test's disposable cache only.
	if err := os.Remove(filepath.Join(cacheRoot(p), "scripts", "connection.sh")); err != nil {
		t.Fatal(err)
	}
	f.calls = nil
	result, err := apply(context.Background(), p, f.run, noProbe)
	if err == nil || result.Installed || result.Phase != "deferred" {
		t.Fatal(result, err)
	}
	assertInventoryOnly(t, f.calls)
}
