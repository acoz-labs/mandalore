package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func delegatedPiFixture(t *testing.T) (PiOptions, PiPlan) {
	t.Helper()
	o := PiOptions{Options: fixture(t), ReadOnly: true}
	o.Binary = filepath.Join(filepath.Dir(o.Binding), "selected Pi runtime")
	script := "#!/bin/sh\ncase \"$*\" in\n'call pi_connection_plan --read-only'|'call pi_connection_repair_plan --read-only') /bin/cat \"$PI_DELEGATE_PLAN\"; exit \"${PI_DELEGATE_PLAN_EXIT:-0}\";;\n'call pi_connection_apply') /bin/cat \"$PI_DELEGATE_RESULT\"; exit \"${PI_DELEGATE_EXIT:-0}\";;\n*) exit 99;;\nesac\n"
	if err := os.WriteFile(o.Binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	p, err := PreparePi(o)
	if err != nil {
		t.Fatal(err)
	}
	p.PackageSHA256 = hash([]byte("different selected Pi package"))
	p.PackageVersion = "1.2.3"
	p.Root = filepath.Join(p.StateDir, "pi", "connections", piPlanKey(p))
	plan := filepath.Join(filepath.Dir(o.Binding), "pi-plan-reply.json")
	result := filepath.Join(filepath.Dir(o.Binding), "pi-result-reply.json")
	t.Setenv("PI_DELEGATE_PLAN", plan)
	t.Setenv("PI_DELEGATE_RESULT", result)
	writeDelegateReply(t, plan, map[string]any{"protocol_version": 1, "ok": true, "result": p})
	writeDelegateReply(t, result, map[string]any{"protocol_version": 1, "ok": true, "result": PiResult{Connection: p, Installed: true, RequiresFreshSession: true, Phase: "verified"}})
	return o, p
}

func TestDelegatedPiRepairUsesOwnedRuntimeAndPreservesMode(t *testing.T) {
	for _, change := range []string{"", "mode", "binding", "package", "same-generation", "changed-binding"} {
		t.Run(change, func(t *testing.T) {
			o, _ := delegatedPiFixture(t)
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
			if err := os.Remove(filepath.Join(p.Root, "package/index.js")); err != nil {
				t.Fatal(err)
			}
			next, err := PreparePiRepair(RepairInput{Root: p.Root})
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "mode":
				next.ReadOnly = false
			case "binding":
				next.Binding = "/another/binding"
			case "package":
				next.PackageSHA256 = hash([]byte("different"))
			case "same-generation":
				next.Generation = p.Generation
			case "changed-binding":
				data, _ := os.ReadFile(p.Binding)
				if err := os.WriteFile(p.Binding, append(data, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
			}
			writeDelegateReply(t, os.Getenv("PI_DELEGATE_PLAN"), map[string]any{"protocol_version": 1, "ok": true, "result": next})
			got, err := PreparePiRepairViaRuntime(context.Background(), RepairInput{Root: p.Root})
			if change == "" {
				if err != nil || got != next || got.Binary != p.Runtime || !got.ReadOnly {
					t.Fatal(got, err)
				}
			} else if err == nil {
				t.Fatal("unsafe repair accepted", change)
			}
			if _, err := os.Stat(filepath.Join(p.Root, "package/index.js")); !os.IsNotExist(err) {
				t.Fatal("preview changed old files")
			}
			if _, err := os.Stat(next.Root); !os.IsNotExist(err) {
				t.Fatal("preview staged generation")
			}
		})
	}
}

func TestPiDelegateSignalFixture(t *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-2] != "pi-delegate-signal-fixture" {
		return
	}
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	child := exec.Command("/bin/sleep", "60")
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if child.Start() != nil {
		os.Exit(2)
	}
	if os.WriteFile(os.Args[len(os.Args)-1], []byte(fmt.Sprintf("%d\n", child.Process.Pid)), 0600) != nil {
		_ = syscall.Kill(-child.Process.Pid, syscall.SIGKILL)
		_ = child.Wait()
		os.Exit(2)
	}
	select {
	case <-interrupt:
		_ = syscall.Kill(-child.Process.Pid, syscall.SIGKILL)
		_ = child.Wait()
		_, _ = fmt.Fprint(os.Stdout, `{"partial":true,"child_reaped":true}`)
		os.Exit(1)
	case <-time.After(10 * time.Second):
		_ = syscall.Kill(-child.Process.Pid, syscall.SIGKILL)
		_ = child.Wait()
		os.Exit(3)
	}
}

func TestPiDelegateCancellationLetsRuntimeReapAndReturnEvidence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	marker := filepath.Join(t.TempDir(), "ready")
	type outcome struct {
		raw []byte
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		raw, err := executePiDelegate(ctx, os.Args[0], "", nil, "-test.run=^TestPiDelegateSignalFixture$", "--", "pi-delegate-signal-fixture", marker)
		done <- outcome{raw, err}
	}()
	pid := 0
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(marker); err == nil && strings.HasSuffix(string(raw), "\n") {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(raw)))
			if pid > 0 {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pid < 1 {
		t.Fatal("owned child readiness not observed")
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatal("child was not live at cancellation", err)
	}
	cancel()
	select {
	case got := <-done:
		if !errors.Is(got.err, context.Canceled) || string(got.raw) != `{"partial":true,"child_reaped":true}` {
			t.Fatal("partial receipt lost during cancellation", string(got.raw), got.err)
		}
		if err := syscall.Kill(pid, 0); err != syscall.ESRCH {
			t.Fatal("owned native child not reaped", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("delegated cancellation was not bounded")
	}
}

func TestDelegatedPiUsesSelectedPackageAndKeepsPartialReceipt(t *testing.T) {
	o, want := delegatedPiFixture(t)
	p, err := PreparePiViaRuntime(context.Background(), o)
	if err != nil || p != want {
		t.Fatal("selected package not used", err)
	}
	r, err := ApplyPiViaRuntime(context.Background(), p)
	if err != nil || !r.Installed {
		t.Fatal(r, err)
	}
	t.Setenv("PI_DELEGATE_EXIT", "1")
	partial := PiResult{Connection: p, Phase: "install-started", Uncertain: true, RequiresFreshSession: true}
	writeDelegateReply(t, os.Getenv("PI_DELEGATE_RESULT"), map[string]any{"protocol_version": 1, "ok": false, "error": map[string]any{"pi_connection_result": partial}})
	r, err = ApplyPiViaRuntime(context.Background(), p)
	if err == nil || r.Phase != partial.Phase || !r.Uncertain {
		t.Fatal("partial evidence lost", r, err)
	}
	if _, err := os.Stat(o.StateDir); !os.IsNotExist(err) {
		t.Fatal("parent performed native installation")
	}
}

func TestDelegatedPiRefusesUnboundPlanAndChangedInputs(t *testing.T) {
	for _, kind := range []string{"binding", "signet", "read-only", "profile", "package", "root", "settings", "source"} {
		t.Run(kind, func(t *testing.T) {
			o, p := delegatedPiFixture(t)
			switch kind {
			case "binding":
				p.Binding = "/another/binding"
			case "signet":
				p.SignetID = "signet-other"
			case "read-only":
				p.ReadOnly = false
			case "profile":
				p.NativeHome = "/another/profile"
			case "package":
				p.PackageSHA256 = "invalid"
			case "root":
				p.Root = "/another/root"
			case "settings":
				p.SettingsExists = true
				p.SettingsSHA256 = hash([]byte("changed"))
			case "source":
				if err := os.WriteFile(o.Binary, []byte("#!/bin/sh\nexit 99\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			writeDelegateReply(t, os.Getenv("PI_DELEGATE_PLAN"), map[string]any{"protocol_version": 1, "ok": true, "result": p})
			if _, err := PreparePiViaRuntime(context.Background(), o); err == nil {
				t.Fatal("unbound delegated plan accepted")
			}
		})
	}
}

func TestPiPreviewFailurePreservesOnlyBoundedTypedDiagnostics(t *testing.T) {
	valid := `{"protocol_version":1,"ok":false,"error":{"code":"connection.failed","message":"foreign Pi registration; preserve it for explicit review"}}`
	for _, tc := range []struct {
		name, raw string
		want      bool
	}{
		{"typed", valid, true},
		{"invalid", "raw child output", false},
		{"duplicate", strings.Replace(valid, `"ok":false`, `"ok":true,"ok":false`, 1), false},
		{"protocol", strings.Replace(valid, `"protocol_version":1`, `"protocol_version":2`, 1), false},
		{"success", strings.Replace(valid, `"ok":false`, `"ok":true`, 1), false},
		{"missing-ok", strings.Replace(valid, `"ok":false,`, "", 1), false},
		{"terminal-control", strings.Replace(valid, "foreign Pi", `\u001b[31mforeign Pi`, 1), false},
		{"oversized-message", strings.Replace(valid, "foreign Pi", strings.Repeat("x", 2049), 1), false},
		{"missing-code", strings.Replace(valid, `"code":"connection.failed",`, "", 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fallback := errors.New("process failed; raw output suppressed")
			got := piPreviewFailure([]byte(tc.raw), fallback)
			if tc.want {
				if got == fallback || !strings.Contains(got.Error(), "foreign Pi registration") {
					t.Fatal("useful structured diagnostic lost", got)
				}
			} else if got != fallback {
				t.Fatal("unusable child output exposed", got)
			}
		})
	}
	if got := piPreviewFailure([]byte(valid), context.Canceled); !errors.Is(got, context.Canceled) {
		t.Fatal("cancellation replaced by child diagnostic", got)
	}
	if got := piPreviewFailure([]byte(valid), context.DeadlineExceeded); !errors.Is(got, context.DeadlineExceeded) {
		t.Fatal("deadline replaced by child diagnostic", got)
	}
}

func TestDelegatedPiReportsTypedPreviewRejectionWithoutEffects(t *testing.T) {
	for _, exit := range []string{"0", "2"} {
		t.Run(exit, func(t *testing.T) {
			o, _ := delegatedPiFixture(t)
			t.Setenv("PI_DELEGATE_PLAN_EXIT", exit)
			writeDelegateReply(t, os.Getenv("PI_DELEGATE_PLAN"), map[string]any{
				"protocol_version": 1, "ok": false,
				"error": map[string]any{"code": "input.invalid", "message": "foreign Pi registration; preserve it for explicit review"},
			})
			if _, err := PreparePiViaRuntime(context.Background(), o); err == nil || !strings.Contains(err.Error(), "foreign Pi registration") {
				t.Fatal("preview rejection reason lost", err)
			}
			if _, err := os.Stat(o.StateDir); !os.IsNotExist(err) {
				t.Fatal("rejected preview changed installation state")
			}
		})
	}
}
