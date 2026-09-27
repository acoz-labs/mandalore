package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchInspectionValidatesEachNativeConnectionWithoutExecution(t *testing.T) {
	for _, h := range []string{"codex", "pi", "claude-code"} {
		t.Run(h, func(t *testing.T) {
			o := fixture(t)
			var root string
			switch h {
			case "codex":
				p, e := Prepare(o)
				if e != nil {
					t.Fatal(e)
				}
				f := &fakeNative{plan: p}
				if _, e = apply(context.Background(), p, f.run, noProbe); e != nil {
					t.Fatal(e)
				}
				root = p.Root
			case "pi":
				p, e := PreparePi(PiOptions{Options: o, ReadOnly: true})
				if e != nil {
					t.Fatal(e)
				}
				f := &fakePi{}
				if _, e = applyPi(context.Background(), p, f.run, noPiProbe); e != nil {
					t.Fatal(e)
				}
				root = p.Root
			case "claude-code":
				p, e := PrepareClaude(ClaudeOptions{Options: o, ReadOnly: true})
				if e != nil {
					t.Fatal(e)
				}
				f := &fakeClaude{}
				if _, e = applyClaude(context.Background(), ClaudeApplyInput{Plan: p}, f.run, noClaudeProbe); e != nil {
					t.Fatal(e)
				}
				root = p.Root
			}
			s := ReceiptSelection{Harness: h, Root: root, StateDir: o.StateDir, NativeHome: o.NativeHome, NativeBinary: o.NativeBinary, Binding: o.Binding}
			// Fixture executable exits 99: success proves static inspection never ran it.
			c, e := InspectLaunch(s)
			if e != nil {
				t.Fatal(e)
			}
			if c.SignetID == "" || c.BindingSHA256 == "" || c.ReadOnly != (h != "codex") {
				t.Fatal(c)
			}
			foreign := s
			foreign.Binding = filepath.Join(filepath.Dir(o.Binding), "foreign.json")
			if _, e = InspectLaunch(foreign); e == nil {
				t.Fatal("foreign binding accepted")
			}
			raw, _ := os.ReadFile(o.Binding)
			if e = os.WriteFile(o.Binding, append(raw, '\n'), 0600); e != nil {
				t.Fatal(e)
			}
			if _, e = InspectLaunch(s); e == nil {
				t.Fatal("changed binding accepted")
			}
			if e = os.WriteFile(o.Binding, raw, 0600); e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(o.NativeBinary, []byte("#!/bin/sh\nexit 88\n"), 0700); e != nil {
				t.Fatal(e)
			}
			if _, e = InspectLaunch(s); e == nil {
				t.Fatal("changed executable accepted")
			}
		})
	}
}
