package install

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

// Exercise the actual embedded Python adapter against the actual staged CLI.
// The native activation seam is synthetic; native Hermes acceptance is separate.
func TestHermesAdapterWithRetainedEngine(t *testing.T) {
	o := HermesOptions{Options: fixture(t)}
	o.SessionTransportVersion = 1
	o.Binary = filepath.Join(filepath.Dir(o.Binding), "built-runtime")
	build := exec.Command("go", "build", "-o", o.Binary, "../../cmd/mandalore")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build runtime: %v: %s", err, output)
	}
	p, err := PrepareHermes(o)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeHermes{}
	if _, err := applyHermes(context.Background(), p, f.run, probeHermesRuntime); err != nil {
		t.Fatal(err)
	}
	ro := o
	ro.ReadOnly, ro.SessionTransportVersion = true, 0
	ro.NativeHome = filepath.Join(filepath.Dir(o.Binding), "readonly-profile")
	ro.StateDir = filepath.Join(filepath.Dir(o.Binding), "readonly-state")
	readonly, err := PrepareHermes(ro)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := applyHermes(context.Background(), readonly, f.run, probeHermesRuntime); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", "-B", "-c", `import importlib.util,json,sys
from pathlib import Path
root=Path(sys.argv[1]);home=Path(sys.argv[2])
spec=importlib.util.spec_from_file_location("retained_mandalore",root/"__init__.py",submodule_search_locations=[str(root)])
module=importlib.util.module_from_spec(spec);sys.modules[spec.name]=module;spec.loader.exec_module(module)
conn=module.Connection(root,home)
packet=conn.context("Copper Finch",{"kind":"startup","session_id":"synthetic-session","event_key":"one"})
assert "Mandalore" in packet["context"] and "synchronization" in packet
saved=conn.call("memory_remember",{"kind":"fact","summary":"Synthetic retained engine","body":"Copper Finch uses the synthetic workspace.","basis":"user-direction","reason":"Synthetic integration test"})
assert saved["ok"] and saved["result"]["durable_locally"],saved
assert "session_sync" in saved,saved
recalled=conn.call("memory_recall",{"query":"Copper Finch"})
assert recalled["ok"] and "Copper Finch" in json.dumps(recalled),recalled
readonly=module.Connection(Path(sys.argv[3]),Path(sys.argv[4]))
denied=readonly.call("memory_remember",{"kind":"fact","summary":"Must not save","body":"Never saved","basis":"user-direction","reason":"Denied synthetic test"})
assert not denied["ok"] and denied["error"]["code"]=="operation.read_only",denied
assert not denied["error"]["write_may_have_occurred"]
assert readonly.call("memory_recall",{"query":"Copper Finch"})["ok"]
# The engine rejects a replaced binding before writes or synchronization.
binding=Path(readonly.config["binding"])
original=binding.read_bytes()
binding.write_bytes(b"{}")
denied=readonly.call("memory_recall",{})
assert not denied["ok"] and denied["error"]["code"]=="binding.invalid",denied
binding.write_bytes(original)
# Pins are checked again on every call, not just registration.
runtime=Path(conn.config["runtime"])
runtime.write_bytes(b"changed")
try:
 conn.call("memory_recall",{})
except module.TransportError as e:
 assert not e.envelope["error"]["write_may_have_occurred"]
else:
 raise AssertionError("changed runtime was executed")
print("actual retained engine, context, save receipts, recall, read-only and pin revalidation passed")`, filepath.Join(p.Root, "package"), p.NativeHome, filepath.Join(readonly.Root, "package"), readonly.NativeHome)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("retained adapter: %v: %s", err, output)
	}
}
