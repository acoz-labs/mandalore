package main

import (
	"bufio"
	"context"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/server"
)

const defaultLaunchdLabel = "dev.acoz.mandalore.server"

var launchdLabel = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]{0,127}$`)

// launchdSupported and launchctl are replaced by tests; only macOS has launchd.
var launchdSupported = runtime.GOOS == "darwin"

var launchctl = func(ctx context.Context, args ...string) error {
	output, err := exec.CommandContext(ctx, "launchctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %s: %v: %s", args[0], err, strings.TrimSpace(string(output)))
	}
	return nil
}

// runLaunchd installs or removes the per-user launchd agent that runs
// `mandalore serve`. Both print what they will change and ask first; the plan
// and question go to errout, the result envelope to out.
func runLaunchd(ctx context.Context, install bool, args []string, input io.Reader, out, errout io.Writer) int {
	name := "server uninstall-launchd"
	usage := "Use server uninstall-launchd [--label LABEL]."
	if install {
		name, usage = "server install-launchd", "Use server install-launchd --config FILE [--label LABEL]."
	}
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	config := f.String("config", "", "Server configuration file")
	label := f.String("label", defaultLaunchdLabel, "launchd label")
	ok := f.Parse(args) == nil && f.NArg() == 0 && (*config != "") == install && launchdLabel.MatchString(*label)
	if !ok {
		return bad(out, usage)
	}
	if !launchdSupported {
		return emit(out, api.Failure("launchd.unsupported", "launchd is macOS only; on Linux run mandalore serve under systemd or the Docker image (docs/deploy/local.md).", false))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return emit(out, api.Failure("launchd.failed", "Cannot find the home directory.", false))
	}
	plist := filepath.Join(home, "Library", "LaunchAgents", *label+".plist")
	target := fmt.Sprintf("gui/%d", os.Getuid())
	if install {
		return installLaunchd(ctx, *config, *label, home, plist, target, input, out, errout)
	}
	if _, err := os.Lstat(plist); err != nil {
		return emit(out, api.Failure("launchd.not_installed", "No launchd agent "+*label+" is installed.", false))
	}
	fmt.Fprintf(errout, "Unload %s and remove %s.\n", *label, plist)
	if !confirmed(input, errout, "Unload and remove it?") {
		return emit(out, api.Failure("launchd.declined", "Nothing was unloaded or removed.", false))
	}
	if err := launchctl(ctx, "bootout", target+"/"+*label); err != nil && launchctl(ctx, "print", target+"/"+*label) == nil {
		return emit(out, api.Failure("launchd.failed", "The agent is still loaded and was kept: "+err.Error(), false))
	}
	if err := os.Remove(plist); err != nil {
		return emit(out, api.Failure("launchd.failed", "The agent is unloaded but its plist could not be removed.", true))
	}
	return emit(out, api.Success(map[string]any{"label": *label, "plist": plist, "removed": true,
		"notice": "Unloaded and removed. Logs stay in ~/Library/Logs/mandalore/."}))
}

func installLaunchd(ctx context.Context, config, label, home, plist, target string, input io.Reader, out, errout io.Writer) int {
	if _, err := server.LoadConfig(config); err != nil {
		return bad(out, err.Error())
	}
	config, err := filepath.Abs(config)
	if err != nil {
		return bad(out, "Cannot resolve the configuration path.")
	}
	executable, err := os.Executable()
	if err != nil {
		return emit(out, api.Failure("launchd.failed", "Cannot find this mandalore executable.", false))
	}
	if _, err := os.Lstat(plist); err == nil {
		return emit(out, api.Failure("launchd.exists", "A launchd agent "+label+" is already installed; run server uninstall-launchd first.", false))
	}
	logs := filepath.Join(home, "Library", "Logs", "mandalore")
	body := launchdPlist(label, []string{executable, "serve", "--config", config}, filepath.Join(logs, label+".log"))
	fmt.Fprintf(errout, "%s\nWrite this to %s and load it with launchctl bootstrap %s.\n", body, plist, target)
	if !confirmed(input, errout, "Write and load it?") {
		return emit(out, api.Failure("launchd.declined", "Nothing was written or loaded.", false))
	}
	if err := errors.Join(os.MkdirAll(filepath.Dir(plist), 0755), os.MkdirAll(logs, 0700)); err != nil {
		return emit(out, api.Failure("launchd.failed", "Cannot create the LaunchAgents or log directory.", false))
	}
	if err := os.WriteFile(plist, []byte(body), 0644); err != nil {
		return emit(out, api.Failure("launchd.failed", "Cannot write the plist.", false))
	}
	if err := launchctl(ctx, "bootstrap", target, plist); err != nil {
		_ = os.Remove(plist)
		return emit(out, api.Failure("launchd.failed", "The agent did not load and its plist was removed: "+err.Error(), false))
	}
	return emit(out, api.Success(map[string]any{"label": label, "plist": plist, "log": filepath.Join(logs, label+".log"), "loaded": true,
		"notice": "Loaded. launchd restarts it if it exits; launchctl print " + target + "/" + label + " shows its state."}))
}

// launchdPlist renders a KeepAlive agent that runs args and logs to log.
func launchdPlist(label string, args []string, log string) string {
	text := func(s string) string {
		var b strings.Builder
		_ = xml.EscapeText(&b, []byte(s))
		return b.String()
	}
	var program strings.Builder
	for _, arg := range args {
		program.WriteString("\t\t<string>" + text(arg) + "</string>\n")
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + text(label) + `</string>
	<key>ProgramArguments</key>
	<array>
` + program.String() + `	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>` + text(log) + `</string>
	<key>StandardErrorPath</key>
	<string>` + text(log) + `</string>
</dict>
</plist>
`
}

// confirmed asks a default-No question and reads one answer line.
func confirmed(input io.Reader, errout io.Writer, question string) bool {
	fmt.Fprint(errout, question+" [y/N] ")
	answer, _ := bufio.NewReader(input).ReadString('\n')
	return slices.Contains([]string{"y", "yes"}, strings.ToLower(strings.TrimSpace(answer)))
}
