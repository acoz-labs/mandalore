package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/testfixture"
)

// launchdRunningState is what launchctl print shows for an agent on its first,
// still-running start.
const launchdRunningState = "gui/501/dev.example = {\n\tstate = running\n\truns = 1\n\tpid = 4242\n\tendpoints = {\n\t\tstate = active\n\t}\n}\n"

// stubLaunchd pretends to be macOS with HOME in a temp dir, records launchctl
// calls, answers print with launchdRunningState and fails the subcommands
// named in fail.
func stubLaunchd(t *testing.T, fail ...string) (home string, calls *[][]string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	supported, run, settle := launchdSupported, launchctl, launchdSettle
	t.Cleanup(func() { launchdSupported, launchctl, launchdSettle = supported, run, settle })
	launchdSupported, launchdSettle = true, 0
	calls = &[][]string{}
	launchctl = func(_ context.Context, args ...string) (string, error) {
		*calls = append(*calls, args)
		if slices.Contains(fail, args[0]) {
			return "", errors.New("launchctl " + args[0] + ": exit status 5")
		}
		if args[0] == "print" {
			return launchdRunningState, nil
		}
		return "", nil
	}
	return home, calls
}

func launchdConfig(t *testing.T) string {
	t.Helper()
	s := testfixture.NewServerSignet(t, "launchd")
	raw, _ := json.Marshal(map[string]any{"listen": "127.0.0.1:8730", "public_url": "http://127.0.0.1:8730", "state_dir": filepath.Join(t.TempDir(), "state"), "owner": map[string]any{"name": "Example"},
		"auth": "loopback-trust", "local_scope": "read",
		"signets": []any{map[string]any{"slug": "primary", "root": s.Root, "signet_id": s.SignetID, "device_id": s.DeviceID, "remote": s.Remote}}})
	path := filepath.Join(t.TempDir(), "a&b", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func launchdCLI(t *testing.T, answer string, args ...string) (int, map[string]any, string) {
	t.Helper()
	var out, errout bytes.Buffer
	code := run(context.Background(), append([]string{"server"}, args...), strings.NewReader(answer), &out, &errout)
	var v map[string]any
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatalf("%v: not JSON: %s", args, out.String())
	}
	return code, v, errout.String()
}

func TestLaunchdInstallPrintsAsksThenLoads(t *testing.T) {
	home, calls := stubLaunchd(t)
	config := launchdConfig(t)
	plist := filepath.Join(home, "Library", "LaunchAgents", "dev.example.mandalore-test.plist")
	target := fmt.Sprintf("gui/%d", os.Getuid())

	for _, answer := range []string{"", "n\n", "no\n"} {
		code, v, shown := launchdCLI(t, answer, "install-launchd", "--config", config, "--label", "dev.example.mandalore-test")
		if code == 0 || v["error"].(map[string]any)["code"] != "launchd.declined" || !strings.Contains(shown, "<key>KeepAlive</key>") || !strings.HasSuffix(shown, "[y/N] ") {
			t.Fatalf("decline %q: %d %v %s", answer, code, v, shown)
		}
		if _, err := os.Lstat(plist); !os.IsNotExist(err) || len(*calls) != 0 {
			t.Fatal("a declined install wrote or loaded something", err, *calls)
		}
	}

	code, v, shown := launchdCLI(t, "y\n", "install-launchd", "--config", config, "--label", "dev.example.mandalore-test")
	if code != 0 || v["result"].(map[string]any)["running"] != true {
		t.Fatalf("install: %d %v", code, v)
	}
	written, err := os.ReadFile(plist)
	if err != nil || !strings.Contains(shown, string(written)) {
		t.Fatal("installed plist differs from the one shown", err)
	}
	if info, _ := os.Stat(plist); info.Mode().Perm() != 0644 {
		t.Fatal("plist mode", info.Mode())
	}
	if !slices.EqualFunc(*calls, [][]string{{"bootstrap", target, plist}, {"print", target + "/dev.example.mandalore-test"}}, slices.Equal) {
		t.Fatal("launchctl calls", *calls)
	}
	// The plist is well-formed XML and runs this executable against the
	// absolute config path, escaped.
	if err := xml.Unmarshal(written, new(struct{})); err != nil {
		t.Fatal("plist is not XML", err)
	}
	executable, _ := os.Executable()
	for _, want := range []string{"<string>dev.example.mandalore-test</string>", "<string>" + executable + "</string>", "<string>serve</string>",
		"a&amp;b", filepath.Join(home, "Library", "Logs", "mandalore", "dev.example.mandalore-test.log")} {
		if !strings.Contains(string(written), want) {
			t.Fatalf("plist lacks %q:\n%s", want, written)
		}
	}

	if code, v, _ := launchdCLI(t, "y\n", "install-launchd", "--config", config, "--label", "dev.example.mandalore-test"); code == 0 || v["error"].(map[string]any)["code"] != "launchd.exists" {
		t.Fatal("second install was not refused", v)
	}

	if code, v, _ := launchdCLI(t, "\n", "uninstall-launchd", "--label", "dev.example.mandalore-test"); code == 0 || v["error"].(map[string]any)["code"] != "launchd.declined" {
		t.Fatal("uninstall did not default to No", v)
	}
	if code, v, _ := launchdCLI(t, "yes\n", "uninstall-launchd", "--label", "dev.example.mandalore-test"); code != 0 || v["result"].(map[string]any)["removed"] != true {
		t.Fatal("uninstall", code, v)
	}
	if _, err := os.Lstat(plist); !os.IsNotExist(err) || !slices.Equal((*calls)[len(*calls)-1], []string{"bootout", target + "/dev.example.mandalore-test"}) {
		t.Fatal("uninstall left the plist or skipped bootout", err, *calls)
	}
	if code, v, _ := launchdCLI(t, "y\n", "uninstall-launchd", "--label", "dev.example.mandalore-test"); code == 0 || v["error"].(map[string]any)["code"] != "launchd.not_installed" {
		t.Fatal("uninstall of a missing agent", v)
	}
}

func TestLaunchdFailuresLeaveNoHalfInstall(t *testing.T) {
	home, _ := stubLaunchd(t, "bootstrap")
	config := launchdConfig(t)
	if code, v, _ := launchdCLI(t, "y\n", "install-launchd", "--config", config); code == 0 || v["error"].(map[string]any)["code"] != "launchd.failed" {
		t.Fatal("failed bootstrap reported success", v)
	}
	plist := filepath.Join(home, "Library", "LaunchAgents", defaultLaunchdLabel+".plist")
	if _, err := os.Lstat(plist); !os.IsNotExist(err) {
		t.Fatal("a plist that did not load was kept", err)
	}

	// bootout fails and print still finds the agent: keep the plist.
	_, calls := stubLaunchd(t, "bootout")
	plist = filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", defaultLaunchdLabel+".plist")
	if err := os.MkdirAll(filepath.Dir(plist), 0755); err != nil || os.WriteFile(plist, []byte("<plist/>"), 0644) != nil {
		t.Fatal(err)
	}
	if code, v, _ := launchdCLI(t, "y\n", "uninstall-launchd"); code == 0 || v["error"].(map[string]any)["code"] != "launchd.failed" {
		t.Fatal("uninstall of a still-loaded agent reported success", v)
	}
	if _, err := os.Lstat(plist); err != nil || len(*calls) != 2 || (*calls)[1][0] != "print" {
		t.Fatal("still-loaded agent's plist removed", err, *calls)
	}

	// bootout fails because the agent is not loaded: remove the plist.
	_, _ = stubLaunchd(t, "bootout", "print")
	plist = filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", defaultLaunchdLabel+".plist")
	if err := os.MkdirAll(filepath.Dir(plist), 0755); err != nil || os.WriteFile(plist, []byte("<plist/>"), 0644) != nil {
		t.Fatal(err)
	}
	if code, v, _ := launchdCLI(t, "y\n", "uninstall-launchd"); code != 0 {
		t.Fatal("uninstall of an unloaded agent", v)
	}
}

// A serve that exits on startup is loaded but not running: install says so,
// and keeps the plist so the log and uninstall-launchd still apply.
func TestLaunchdInstallReportsAServeThatIsNotRunning(t *testing.T) {
	for name, state := range map[string]string{
		"spawn scheduled":   "gui/501/dev.example = {\n\tstate = spawn scheduled\n\truns = 2\n\tlast exit code = 1\n}\n",
		"restarted":         "gui/501/dev.example = {\n\tstate = running\n\truns = 3\n}\n",
		"only nested state": "gui/501/dev.example = {\n\tstate = not running\n\truns = 1\n\tendpoints = {\n\t\tstate = running\n\t}\n}\n",
		"print fails":       "",
	} {
		t.Run(name, func(t *testing.T) {
			home, _ := stubLaunchd(t)
			stub := launchctl
			launchctl = func(ctx context.Context, args ...string) (string, error) {
				if args[0] == "print" {
					if state == "" {
						return "", errors.New("launchctl print: exit status 113")
					}
					return state, nil
				}
				return stub(ctx, args...)
			}
			code, v, _ := launchdCLI(t, "y\n", "install-launchd", "--config", launchdConfig(t))
			failure, _ := v["error"].(map[string]any)
			if code == 0 || failure["code"] != "launchd.not_running" || failure["write_may_have_occurred"] != true {
				t.Fatalf("not-running serve reported as installed: %d %v", code, v)
			}
			if _, err := os.Lstat(filepath.Join(home, "Library", "LaunchAgents", defaultLaunchdLabel+".plist")); err != nil {
				t.Fatal("loaded agent's plist removed", err)
			}
		})
	}
}

func TestLaunchdRefusesBadInputBeforeAsking(t *testing.T) {
	_, calls := stubLaunchd(t)
	config := launchdConfig(t)
	for _, args := range [][]string{
		{"install-launchd"},
		{"install-launchd", "--config", config, "--label", "../escape"},
		{"install-launchd", "--config", config, "--label", ""},
		{"install-launchd", "--config", config, "extra"},
		{"install-launchd", "--config", "/fixture/private/missing-config"},
		{"uninstall-launchd", "--config", config},
		{"uninstall-launchd", "--label", "a/b"},
	} {
		code, v, shown := launchdCLI(t, "y\n", args...)
		if code == 0 || v["error"].(map[string]any)["code"] != "input.invalid" || shown != "" || strings.Contains(fmt.Sprint(v), "/fixture/private/") {
			t.Fatalf("%v: %d %v %q", args, code, v, shown)
		}
	}
	launchdSupported = false
	if code, v, _ := launchdCLI(t, "y\n", "install-launchd", "--config", config); code == 0 || v["error"].(map[string]any)["code"] != "launchd.unsupported" {
		t.Fatal("install off macOS", v)
	}
	if len(*calls) != 0 {
		t.Fatal("launchctl ran", *calls)
	}
}
