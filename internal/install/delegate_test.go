package install

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func delegatedFixture(t *testing.T) (Options, Plan) {
	t.Helper()
	o := fixture(t)
	o.Binary = filepath.Join(filepath.Dir(o.Binding), "selected new runtime")
	script := "#!/bin/sh\ncase \"$*\" in\n'call connection_plan --read-only') /bin/cat \"$DELEGATE_PLAN_FIXTURE\";;\n'call connection_apply') /bin/cat \"$DELEGATE_RESULT_FIXTURE\"; exit \"${DELEGATE_EXIT:-0}\";;\n*) exit 99;;\nesac\n"
	if err := os.WriteFile(o.Binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	p, err := Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate an actually different embedded package, not the parent's package.
	p.PackageSHA256 = hash([]byte("new embedded package"))
	p.PackageVersion = "1.1.0"
	key := planKey(p)
	p.Root = filepath.Join(o.StateDir, "connections", key)
	p.Version = "1.1.0+codex." + key
	planFile := filepath.Join(filepath.Dir(o.Binding), "reply-plan.json")
	resultFile := filepath.Join(filepath.Dir(o.Binding), "reply-result.json")
	t.Setenv("DELEGATE_PLAN_FIXTURE", planFile)
	t.Setenv("DELEGATE_RESULT_FIXTURE", resultFile)
	writeDelegateReply(t, planFile, map[string]any{"protocol_version": 1, "ok": true, "result": p})
	writeDelegateReply(t, resultFile, map[string]any{"protocol_version": 1, "ok": true, "result": Result{Connection: p, Installed: true, RequiresFreshSession: true, Phase: "verified"}})
	return o, p
}

func writeDelegateReply(t *testing.T, path string, v any) {
	t.Helper()
	b, _ := json.Marshal(v)
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestDelegatedConnectionUsesSelectedRuntimePackage(t *testing.T) {
	o, want := delegatedFixture(t)
	p, err := PrepareViaRuntime(context.Background(), o)
	if err != nil || p.PackageSHA256 != want.PackageSHA256 || p.Root != want.Root {
		t.Fatal("did not prepare with selected runtime", p, err)
	}
	parent, err := Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	if p.PackageSHA256 == parent.PackageSHA256 {
		t.Fatal("fixture failed to distinguish old embedded package")
	}
	r, err := ApplyViaRuntime(context.Background(), p)
	if err != nil || !r.Installed || r.Connection.PackageSHA256 != want.PackageSHA256 {
		t.Fatal("delegated apply used wrong runtime", r, err)
	}
	if _, err := os.Lstat(o.StateDir); !os.IsNotExist(err) {
		t.Fatal("parent attempted native apply instead of delegation")
	}
}

func TestDelegatedConnectionRefusesChangedOrUnboundReply(t *testing.T) {
	for _, field := range []string{"source", "binding", "profile", "digest", "package", "root", "truncated", "duplicate"} {
		t.Run(field, func(t *testing.T) {
			o, p := delegatedFixture(t)
			bad := p
			switch field {
			case "source":
				bad.Binary = "/another/runtime"
			case "binding":
				bad.Binding = "/another/binding"
			case "profile":
				bad.NativeHome = "/another/profile"
			case "digest":
				bad.BinarySHA256 = strings.Repeat("0", 64)
			case "package":
				bad.PackageSHA256 = "invalid"
			case "root":
				bad.Root = "/unrelated/root"
			}
			path := os.Getenv("DELEGATE_PLAN_FIXTURE")
			writeDelegateReply(t, path, map[string]any{"protocol_version": 1, "ok": true, "result": bad})
			if field == "truncated" {
				if err := os.WriteFile(path, []byte(`{"ok":true}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if field == "duplicate" {
				if err := os.WriteFile(path, []byte(`{"ok":true,"ok":true}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := PrepareViaRuntime(context.Background(), o); err == nil {
				t.Fatal("unbound reply accepted")
			}
		})
	}
}

func TestDelegatedApplyPreservesPartialReceiptAndRefusesStalePlan(t *testing.T) {
	o, p := delegatedFixture(t)
	partial := Result{Connection: p, Phase: "registration-removed", PreviousRoot: "/synthetic/previous", RequiresFreshSession: true}
	t.Setenv("DELEGATE_EXIT", "1")
	writeDelegateReply(t, os.Getenv("DELEGATE_RESULT_FIXTURE"), map[string]any{"protocol_version": 1, "ok": false, "error": map[string]any{"connection_result": partial, "code": "connection.failed"}})
	r, err := ApplyViaRuntime(context.Background(), p)
	if err == nil || r.Phase != partial.Phase || r.PreviousRoot != partial.PreviousRoot {
		t.Fatal("partial native outcome lost", r, err)
	}
	if err := os.WriteFile(o.Binary, []byte("#!/bin/sh\nexit 99\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyViaRuntime(context.Background(), p); err == nil {
		t.Fatal("changed selected runtime accepted")
	}
}
