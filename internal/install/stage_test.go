package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestArmorerContextIsLocalExactAndOwned(t *testing.T) {
	o := fixture(t)
	o.StateDir = filepath.Join(filepath.Dir(o.StateDir), "state with 'quotes'")
	p, err := Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	const name = "plugins/mandalore/skills/the-armorer/references/connection.json"
	public, err := packageFiles()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := public[name]; exists {
		t.Fatal("public package contains machine-local context")
	}
	files, receipt, err := bundle(p)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(files[name], &got); err != nil {
		t.Fatal("missing valid local Armorer context", err)
	}
	want := map[string]any{"schema_version": float64(1), "runtime": p.Runtime, "binding": p.Binding, "state_dir": p.StateDir, "native_home": p.NativeHome, "native_binary": p.NativeBinary, "connection_root": p.Root}
	if !reflect.DeepEqual(got, want) || receipt.Files[name] != hash(files[name]) {
		t.Fatal("context must match exact plan and ownership receipt", got)
	}
	if err := publishBundle(p); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.Root, name), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyTree(p.Root, receipt.Files, true); err == nil {
		t.Fatal("edited administrative context accepted for repair")
	}
}

func TestStagingRetainsExactBytesAndRefusesChangedOrRedirectedTargets(t *testing.T) {
	o := fixture(t)
	p, err := Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := stageRuntime(p); err != nil {
		t.Fatal(err)
	}
	if got, err := digest(p.Runtime); err != nil || got != p.BinarySHA256 {
		t.Fatal("wrong staged bytes", err)
	}
	if st, err := os.Stat(p.Runtime); err != nil || st.Mode().Perm() != 0700 {
		t.Fatal("runtime permissions", err)
	}
	if err := stageRuntime(p); err != nil {
		t.Fatal("idempotent stage", err)
	}
	if err := os.WriteFile(p.Runtime, []byte("preserve edited copy"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := stageRuntime(p); err == nil {
		t.Fatal("edited target replaced")
	}
	if got, _ := os.ReadFile(p.Runtime); string(got) != "preserve edited copy" {
		t.Fatal("edited copy lost")
	}
	o = fixture(t)
	p, err = Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(o.Binary, []byte("changed since preview"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := stageRuntime(p); err == nil {
		t.Fatal("changed input accepted")
	}
	if _, err := os.Lstat(p.Runtime); !os.IsNotExist(err) {
		t.Fatal("published changed artifact")
	}
	o = fixture(t)
	p, err = Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(o.StateDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(o.Binding), filepath.Join(o.StateDir, "runtimes")); err != nil {
		t.Fatal(err)
	}
	if err := stageRuntime(p); err == nil {
		t.Fatal("followed redirected runtime path")
	}
}

func TestBundlePinsDefaultsAndPreservesMissingOrEditedGenerations(t *testing.T) {
	o := fixture(t)
	p, err := Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := publishBundle(p); err != nil {
		t.Fatal(err)
	}
	r, err := loadReceipt(p.Root)
	if err != nil || r.Plan != p {
		t.Fatal("receipt mismatch", err)
	}
	if err := verifyTree(p.Root, r.Files, false); err != nil {
		t.Fatal(err)
	}
	if err := publishBundle(p); err != nil {
		t.Fatal("idempotent publish", err)
	}
	file := filepath.Join(p.Root, "plugins/mandalore/hooks/hooks.json")
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := verifyTree(p.Root, r.Files, false); err == nil {
		t.Fatal("missing resource accepted")
	}
	if err := verifyTree(p.Root, r.Files, true); err != nil {
		t.Fatal("missing-only recovery refused", err)
	}
	if err := publishBundle(p); err == nil {
		t.Fatal("in-place repair of partial generation")
	}
	if err := os.WriteFile(file, []byte("user edit"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyTree(p.Root, r.Files, true); err == nil {
		t.Fatal("edited generation accepted for replacement")
	}
	if got, _ := os.ReadFile(file); string(got) != "user edit" {
		t.Fatal("user edit lost")
	}
	o.Generation = "repair-1"
	repair, err := Prepare(o)
	if err != nil || repair.Root == p.Root {
		t.Fatal("repair did not choose a fresh root", err)
	}
	if err := publishBundle(repair); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(file); string(got) != "user edit" {
		t.Fatal("old generation changed")
	}
}

func TestReceiptCannotRedirectNativeCacheWithAnEditedVersion(t *testing.T) {
	p, err := Prepare(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := publishBundle(p); err != nil {
		t.Fatal(err)
	}
	r, err := loadReceipt(p.Root)
	if err != nil {
		t.Fatal(err)
	}
	r.Plan.Version = "../../escape"
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.Root, "connection.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadReceipt(p.Root); err == nil {
		t.Fatal("edited cache path accepted")
	}
}
