package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPiDoctorAndRepairPreserveHistoryAndBinding(t *testing.T) {
	o := PiOptions{Options: fixture(t), ReadOnly: true}
	p, err := PreparePi(o)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakePi{}
	if _, err := applyPi(context.Background(), p, f.run, noPiProbe); err != nil {
		t.Fatal(err)
	}
	profile := Profile{StateDir: p.StateDir, NativeHome: p.NativeHome, NativeBinary: p.NativeBinary}
	if report := doctorPi(context.Background(), profile, f.run); !report.Healthy {
		t.Fatal(report)
	}
	if err := os.Remove(filepath.Join(p.Root, "package/index.js")); err != nil {
		t.Fatal(err)
	}
	if report := doctorPi(context.Background(), profile, f.run); report.Healthy {
		t.Fatal("missing file reported healthy")
	}
	next, err := PreparePiRepair(RepairInput{Root: p.Root})
	if err != nil {
		t.Fatal(err)
	}
	if next.Root == p.Root || next.RecoverFrom != p.Root || !next.ReadOnly || next.BindingSHA256 != p.BindingSHA256 {
		t.Fatal("repair changed identity or overwrote generation")
	}
	if _, err := applyPi(context.Background(), next, f.run, noPiProbe); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(p.Root, "package/index.js")); !os.IsNotExist(err) {
		t.Fatal("old generation rewritten")
	}
	if report := doctorPi(context.Background(), profile, f.run); !report.Healthy {
		t.Fatal(report)
	}
	before, err := os.ReadFile(p.Binding)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Binding, append(before, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := PreparePiRepair(RepairInput{Root: next.Root}); err == nil {
		t.Fatal("changed binding silently accepted by repair")
	}
}

func TestPiRepairRecoversAnInterruptedInactiveUpdate(t *testing.T) {
	o := PiOptions{Options: fixture(t)}
	p, err := PreparePi(o)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakePi{}
	if _, err := applyPi(context.Background(), p, f.run, noPiProbe); err != nil {
		t.Fatal(err)
	}
	o.Generation = "interrupted"
	next, err := PreparePi(o)
	if err != nil {
		t.Fatal(err)
	}
	f.fail = "install"
	if _, err := applyPi(context.Background(), next, f.run, noPiProbe); err == nil {
		t.Fatal("expected interruption")
	}
	repair, err := PreparePiRepair(RepairInput{Root: next.Root})
	if err != nil {
		t.Fatal(err)
	}
	f.fail = ""
	r, err := applyPi(context.Background(), repair, f.run, noPiProbe)
	if err != nil || !r.Installed {
		t.Fatal(r, err)
	}
}

func TestPiDoctorDoesNotCreateMissingProfile(t *testing.T) {
	o := fixture(t)
	f := &fakePi{}
	r := doctorPi(context.Background(), Profile{StateDir: o.StateDir, NativeHome: o.NativeHome, NativeBinary: o.NativeBinary}, f.run)
	if r.Healthy || len(f.calls) != 0 {
		t.Fatal("missing profile triggered native execution", r)
	}
	if _, err := os.Stat(o.NativeHome); !os.IsNotExist(err) {
		t.Fatal("doctor created missing profile")
	}
}
