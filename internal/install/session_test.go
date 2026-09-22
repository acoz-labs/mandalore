package install

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSessionPlansPreserveLegacyAndPinExplicitAuthority(t *testing.T) {
	for _, harness := range []string{"codex", "pi", "claude-code"} {
		t.Run(harness, func(t *testing.T) {
			o := fixture(t)
			build := func() (map[string][]byte, map[string]string, string) {
				t.Helper()
				switch harness {
				case "codex":
					p, e := Prepare(o)
					if e != nil {
						t.Fatal(e)
					}
					files, r, e := bundle(p)
					if e != nil {
						t.Fatal(e)
					}
					return files, r.Files, "plugins/mandalore/"
				case "pi":
					p, e := PreparePi(PiOptions{Options: o})
					if e != nil {
						t.Fatal(e)
					}
					files, r, e := piBundle(p)
					if e != nil {
						t.Fatal(e)
					}
					return files, r.Files, "package/"
				default:
					p, e := PrepareClaude(ClaudeOptions{Options: o})
					if e != nil {
						t.Fatal(e)
					}
					files, r, e := claudeBundle(p)
					if e != nil {
						t.Fatal(e)
					}
					return files, r.Files, "package/"
				}
			}
			legacy, _, prefix := build()
			if _, ok := legacy[prefix+"session-policy.json"]; ok {
				t.Fatal("legacy connection gained transport authority")
			}
			o.SessionTransportVersion = 1
			files, receipt, prefix := build()
			raw := files[prefix+"session-policy.json"]
			if len(raw) == 0 || receipt[prefix+"session-policy.json"] != hash(raw) {
				t.Fatal("policy is not receipt-pinned")
			}
			var policy map[string]any
			if err := json.Unmarshal(raw, &policy); err != nil {
				t.Fatal(err)
			}
			if policy["binding"] != o.Binding || policy["mode"] != "enabled-session" || policy["runtime_sha256"] == "" {
				t.Fatal(policy)
			}
			bridge := string(files[prefix+"scripts/connection.sh"])
			if harness != "pi" && (!strings.Contains(bridge, "--session-policy-sha256") || !strings.Contains(bridge, hash(raw))) {
				t.Fatal("policy not pinned in native entry", bridge)
			}
		})
	}
}

func TestReadOnlyPlansCannotAuthorizeSessionTransport(t *testing.T) {
	o := fixture(t)
	o.SessionTransportVersion = 1
	if _, err := PreparePi(PiOptions{Options: o, ReadOnly: true}); err == nil {
		t.Fatal("Pi read-only upgraded")
	}
	if _, err := PrepareClaude(ClaudeOptions{Options: o, ReadOnly: true}); err == nil {
		t.Fatal("Claude read-only upgraded")
	}
	o.SessionTransportVersion = 2
	if _, err := Prepare(o); err == nil {
		t.Fatal("unsupported Codex policy accepted")
	}
	if _, err := PreparePi(PiOptions{Options: o}); err == nil {
		t.Fatal("unsupported Pi policy accepted")
	}
	if _, err := PrepareClaude(ClaudeOptions{Options: o}); err == nil {
		t.Fatal("unsupported Claude policy accepted")
	}
}

func TestSessionReceiptRejectsPolicyHashSubstitution(t *testing.T) {
	o := fixture(t)
	o.SessionTransportVersion = 1
	p, err := PrepareClaude(ClaudeOptions{Options: o})
	if err != nil {
		t.Fatal(err)
	}
	_, r, err := claudeBundle(p)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(r)
	if _, err := decodeClaudeReceipt(p.Root, raw); err != nil {
		t.Fatal(err)
	}
	r.Files["package/session-policy.json"] = strings.Repeat("a", 64)
	raw, _ = json.Marshal(r)
	if _, err := decodeClaudeReceipt(p.Root, raw); err == nil {
		t.Fatal("edited policy granted transport")
	}
}

func TestExistingSessionModePreservesReadOnlyAndLegacy(t *testing.T) {
	for _, harness := range []string{"pi", "claude-code"} {
		for _, readOnly := range []bool{false, true} {
			o := fixture(t)
			profile := Profile{StateDir: o.StateDir, NativeHome: o.NativeHome, NativeBinary: o.NativeBinary}
			if _, _, exists, err := ExistingSessionMode(harness, profile); err != nil || exists {
				t.Fatal("new connection misclassified", err, exists)
			}
			if harness == "pi" {
				p, err := PreparePi(PiOptions{Options: o, ReadOnly: readOnly})
				if err != nil {
					t.Fatal(err)
				}
				f := &fakePi{}
				if _, err := applyPi(context.Background(), p, f.run, noPiProbe); err != nil {
					t.Fatal(err)
				}
			} else {
				p, err := PrepareClaude(ClaudeOptions{Options: o, ReadOnly: readOnly})
				if err != nil {
					t.Fatal(err)
				}
				f := &fakeClaude{}
				if _, err := applyClaude(context.Background(), ClaudeApplyInput{Plan: p}, f.run, noClaudeProbe); err != nil {
					t.Fatal(err)
				}
			}
			mode, ro, exists, err := ExistingSessionMode(harness, profile)
			if err != nil || !exists || mode != 0 || ro != readOnly {
				t.Fatal("existing authority lost", mode, ro, exists, err)
			}
		}
	}
}
