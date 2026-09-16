package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPiBundleIsOwnedPinnedAndPreservesPublicPayload(t *testing.T) {
	p, err := PreparePi(PiOptions{Options: fixture(t), ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := stageRuntime(p.runtimePlan()); err != nil {
		t.Fatal(err)
	}
	if err := publishPiBundle(p); err != nil {
		t.Fatal(err)
	}
	if err := publishPiBundle(p); err != nil {
		t.Fatal("idempotent staging", err)
	}
	r, err := ownedPi(p.Root, p.StateDir, p.NativeHome, false)
	if err != nil {
		t.Fatal(err)
	}
	if r.Plan != p {
		t.Fatal("changed receipt")
	}
	raw, err := os.ReadFile(filepath.Join(p.Root, "package/connection.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	if config["harness"] != "pi" || config["read_only"] != true || config["binding_sha256"] != p.BindingSHA256 || config["runtime_sha256"] != p.BinarySHA256 || config["connection_root"] != p.Root {
		t.Fatal("unpinned connection context")
	}
	if err := os.Remove(filepath.Join(p.Root, "package/index.js")); err != nil {
		t.Fatal(err)
	}
	if _, err := ownedPi(p.Root, p.StateDir, p.NativeHome, false); err == nil {
		t.Fatal("missing package treated as healthy")
	}
	if _, err := ownedPi(p.Root, p.StateDir, p.NativeHome, true); err != nil {
		t.Fatal("missing-only repair refused", err)
	}
	if err := os.WriteFile(filepath.Join(p.Root, "package/foreign"), []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ownedPi(p.Root, p.StateDir, p.NativeHome, true); err == nil {
		t.Fatal("repair adopted foreign file")
	}
}

func TestPiPlanRecognizesOnlyIntactOwnedNativeRegistration(t *testing.T) {
	o := PiOptions{Options: fixture(t)}
	p, err := PreparePi(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := stageRuntime(p.runtimePlan()); err != nil {
		t.Fatal(err)
	}
	if err := publishPiBundle(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(p.NativeHome, 0700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"packages": []string{filepath.Join(p.Root, "package")}})
	if err := os.WriteFile(filepath.Join(p.NativeHome, "settings.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	next, err := PreparePi(o)
	if err != nil {
		t.Fatal(err)
	}
	if next.PreviousRoot != p.Root || next.PreviousReceiptSHA256 == "" {
		t.Fatal("previous ownership not pinned")
	}
	if err := os.WriteFile(filepath.Join(p.Root, "package/index.js"), []byte("edited"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := PreparePi(o); err == nil {
		t.Fatal("edited previous package adopted")
	}
}
