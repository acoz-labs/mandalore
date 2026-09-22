package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/distribution"
	"github.com/acoz-labs/mandalore/internal/install"
)

func runReleaseInstall(ctx context.Context, args []string, input io.Reader, out io.Writer) int {
	f := flag.NewFlagSet("release install", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var o distribution.InstallOptions
	m := &menu{ctx: ctx, in: bufio.NewReader(input), out: out}
	f.StringVar(&o.Prefix, "prefix", "", "User-owned CLI prefix; otherwise prompt")
	f.StringVar(&o.Version, "version", "", "Explicit published version; otherwise latest stable")
	f.StringVar(&o.Candidate, "candidate", "", "Explicit local candidate directory")
	f.StringVar(&o.Retained, "retained", "", "Retained manifest SHA-256")
	f.StringVar(&m.binding, "binding", "", "Optional selected connection binding; not used by CLI install")
	f.StringVar(&m.profile.StateDir, "state-dir", "", "Optional native installation state")
	f.StringVar(&m.profile.NativeHome, "native-home", "", "Optional native Codex profile")
	f.StringVar(&m.profile.NativeBinary, "native-binary", "", "Optional native Codex executable")
	plain := f.Bool("plain", false, "Use numbered line-oriented prompts")
	readOnly := f.Bool("read-only", false, "Refuse this mutating journey")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		if errors.Is(err, flag.ErrHelp) {
			_, err := io.WriteString(out, help)
			if err != nil {
				return 1
			}
			return 0
		}
		return bad(out, "Invalid release install flags or arguments; use --help.")
	}
	if *readOnly {
		return emit(out, api.Failure("operation.read_only", "Use release inspect or plan for a read-only task.", false))
	}
	m.piProfile = m.profile
	m.claudeProfile = m.profile
	check := o
	if check.Prefix == "" {
		check.Prefix = "/selection"
	}
	if err := check.Validate(); err != nil {
		return bad(out, "Select one valid release source and a user-owned prefix; use --help.")
	}
	if !*plain {
		m.tui = console.New(input, out)
		if m.tui != nil {
			m.tui = m.tui.WithCancellation(ctx)
		}
	}
	m.block(console.Block{Title: "Mandalore · Install or update CLI", Body: "Opening this journey changes nothing. Review the selected artifact and destination before applying. Native memory connections are a separate choice."})
	err := m.installRelease(o)
	if m.outputErr != nil {
		return 1
	}
	if ctx.Err() != nil {
		return 130
	}
	if errors.Is(err, console.ErrBack) || errors.Is(err, io.EOF) {
		m.block(console.Block{Title: "Stopped", Body: "No further actions were taken. Any completed installation remains in place."})
		if m.outputErr != nil {
			return 1
		}
		return 0
	}
	if err != nil {
		m.block(console.Block{Title: "[FAIL] Needs attention", Body: err.Error(), Tone: console.Failure})
		return 1
	}
	return 0
}

func (m *menu) releaseCall(name string, value any) api.Envelope {
	if m.releaseInvoke != nil {
		return m.releaseInvoke(name, value)
	}
	return m.call(name, value, false)
}

func defaultReleasePrefix() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local"), nil
}

func (m *menu) chooseRelease() error {
	n, err := m.selectItem("Choose CLI source", []string{"Latest published stable release", "A specific published version", "A verified local candidate directory", "A retained runtime by manifest SHA-256", "Back"}, 0)
	if err != nil {
		return err
	}
	if n == 4 {
		return console.ErrBack
	}
	o := distribution.InstallOptions{Prefix: m.releasePrefix}
	switch n {
	case 1:
		o.Version, err = m.input("Published version (without v)", "")
	case 2:
		o.Candidate, err = m.input("Local candidate directory", "")
	case 3:
		o.Retained, err = m.input("Retained manifest SHA-256", "")
	}
	if err != nil {
		return err
	}
	if (n == 1 && o.Version == "") || (n == 2 && o.Candidate == "") || (n == 3 && o.Retained == "") {
		return errors.New("the selected source requires a value; no release was selected")
	}
	return m.installRelease(o)
}

func (m *menu) installRelease(o distribution.InstallOptions) error {
	if o.Prefix == "" {
		def, err := defaultReleasePrefix()
		if err != nil {
			return err
		}
		o.Prefix, err = m.input("User-owned CLI installation prefix", def)
		if err != nil {
			return err
		}
	}
	m.releasePrefix = o.Prefix
	m.block(console.Block{Title: "Checking source and installation", Body: "Verifying the selected manifest and destination. Planning does not execute a binary or change this machine."})
	if m.outputErr != nil {
		return m.outputErr
	}
	v := m.releaseCall("release_plan", o)
	if !v.OK {
		if v.Error.Code == "release.unavailable" {
			m.block(console.Block{Title: "No published release available", Body: "No installation was performed. Try a published version when available, or explicitly select a trusted local candidate.", Tone: console.Warning})
		}
		return m.outcome("CLI installation preview", v)
	}
	p := v.Result.(distribution.InstallPlan)
	m.releasePreview(p)
	if err := m.confirm(); err != nil {
		return err
	}
	m.block(console.Block{Title: "Installing verified CLI", Body: "Rechecking the reviewed plan, retaining the runtime and activating only its owned launcher. Native connections remain unchanged."})
	if m.outputErr != nil {
		return m.outputErr
	}
	v = m.releaseCall("release_apply", p)
	if err := m.outcome("CLI installation verified", v); err != nil {
		return err
	}
	r := v.Result.(distribution.InstallResult)
	m.releaseResult(r)
	if !launcherOnPath(r.Launcher) {
		m.block(console.Block{Title: "Run the installed CLI", Body: "The launcher directory is not currently on PATH. No shell settings were changed; you can run this full path now.", Fields: []console.Field{{Label: "Command", Value: r.Launcher}}})
	}
	if m.outputErr != nil {
		return m.outputErr
	}
	return m.offerReleaseConnection(p, r)
}

func launcherOnPath(launcher string) bool {
	want := filepath.Dir(launcher)
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		resolved, err := filepath.EvalSymlinks(entry)
		if err == nil && resolved == want {
			return true
		}
	}
	return false
}

func (m *menu) releasePreview(p distribution.InstallPlan) {
	mf := p.Source.Manifest.Manifest
	source, trust := p.Candidate, "Local byte verification is not publisher authentication. Applying trusts this explicitly selected source."
	if p.Source.Kind == "github-release" {
		source = "Official published release"
		trust = "Official immutable release metadata checked. Apply still verifies downloaded bytes and executes the selected runtime's version probe."
		if p.Source.Published != nil {
			source = p.Source.Published.URL
			if p.Source.Published.Prerelease || strings.Contains(strings.Split(mf.Version, "+")[0], "-") {
				source += " (prerelease)"
			}
		}
	} else if p.Source.Kind == "retained" {
		source = "Retained manifest " + p.Retained
		trust = "Rollback verifies the retained compatible runtime; it does not rewrite memory or switch a native connection."
	}
	previous := p.Observed.LauncherTarget
	if previous == "" {
		previous = "Not installed in this prefix"
	}
	m.block(console.Block{Title: "Review CLI installation", Body: trust, Fields: []console.Field{{Label: "Version", Value: mf.Version}, {Label: "Source", Value: source}, {Label: "Platform", Value: p.OS + "/" + p.Arch}, {Label: "Prefix", Value: p.Prefix}, {Label: "Launcher", Value: p.Launcher}, {Label: "Previous runtime", Value: previous}, {Label: "Selected runtime", Value: p.Runtime}}})
	m.block(console.Block{Title: "Artifact identity and effects", Body: "Only this CLI installation changes. Native connections, memory, credentials and shell settings remain unchanged; previous runtimes remain retained.", Fields: []console.Field{{Label: "Source commit", Value: mf.SourceCommit}, {Label: "Manifest SHA256", Value: p.Source.Manifest.SHA256}, {Label: "Binary SHA256", Value: p.Binary.SHA256}, {Label: "Embedded plugin SHA256", Value: mf.PluginSHA256}, {Label: "Retained target reusable", Value: strconv.FormatBool(p.RuntimeRetained)}, {Label: "Protocol / signet schema", Value: "1 / read 1, write 1"}}})
}

func (m *menu) releaseResult(r distribution.InstallResult) {
	body := "Native connections remain unchanged. Retained runtimes and completed phases are preserved."
	previous, pending := r.PreviousRuntime, r.Pending
	if previous == "" {
		previous = "None"
	}
	if pending == "" {
		pending = "None"
	}
	if r.Pending != "" {
		body += " Activation is incomplete. Inspect the pending record and reapply only the same reviewed plan; do not remove or replace unfamiliar files."
	}
	m.block(console.Block{Title: "CLI receipt", Body: body, Fields: []console.Field{{Label: "Phase", Value: r.Phase}, {Label: "Installation complete", Value: strconv.FormatBool(r.Installed)}, {Label: "Already current", Value: strconv.FormatBool(r.AlreadyCurrent)}, {Label: "Launcher", Value: r.Launcher}, {Label: "Selected runtime", Value: r.Runtime}, {Label: "Previous runtime", Value: previous}, {Label: "Pending record", Value: pending}}})
	if r.Pending != "" {
		quoted := "'" + strings.ReplaceAll(r.Pending, "'", "'\"'\"'") + "'"
		m.block(console.Block{Title: "Recover the reviewed installation", Body: "Inspect the pending record and resolve the reported problem first. Then use the original Mandalore executable you launched, not an incomplete new launcher. This retries the stored plan and rechecks ownership and retained bytes; it does not create a fresh plan or change memory connections. The command below is one logical line; your terminal may visually wrap it. Copy the whole line without adding newlines.", Command: "mandalore release apply < " + quoted})
	}
}

func (m *menu) offerReleaseConnection(p distribution.InstallPlan, r distribution.InstallResult) error {
	n, err := m.selectItem("Update one native memory connection?", []string{"Keep native connections unchanged", "Preview one Codex connection with this runtime", "Preview one Pi connection with this runtime", "Preview one Claude Code connection with this runtime"}, 0)
	if err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	if m.binding == "" {
		m.binding, err = binding.DefaultPath()
		if err != nil {
			return err
		}
	}
	if n == 2 {
		return m.offerPiRelease(p, r)
	} else if n == 3 {
		return m.offerClaudeRelease(p, r)
	}
	if err := m.nativeProfile(); err != nil {
		return fmt.Errorf("CLI installation remains complete; connection preview unavailable: %w", err)
	}
	for _, item := range []struct {
		title string
		value *string
	}{{"Selected signet binding", &m.binding}, {"Selected native Codex profile", &m.profile.NativeHome}, {"Selected native installation state", &m.profile.StateDir}, {"Native Codex executable", &m.profile.NativeBinary}} {
		*item.value, err = m.input(item.title, *item.value)
		if err != nil {
			return err
		}
	}
	m.block(console.Block{Title: "Preparing selected connection preview", Body: "This executes the verified newly installed runtime to prepare its own embedded plugin. It does not yet activate a native connection."})
	if m.outputErr != nil {
		return m.outputErr
	}
	prepare := m.prepareSelectedConnection
	if prepare == nil {
		prepare = install.PrepareViaRuntime
	}
	transport, err := m.selectSessionTransport("codex", m.profile)
	if err != nil {
		return err
	}
	c, err := prepare(m.ctx, install.Options{SessionTransportVersion: transport, StateDir: m.profile.StateDir, NativeHome: m.profile.NativeHome, NativeBinary: m.profile.NativeBinary, Binary: r.Runtime, Binding: m.binding})
	if err != nil {
		return fmt.Errorf("CLI installation remains complete; connection preview failed: %w", err)
	}
	if c.Binary != r.Runtime || c.BinarySHA256 != p.Binary.SHA256 || c.PackageSHA256 != p.Source.Manifest.Manifest.PluginSHA256 || c.PackageVersion != p.Source.Manifest.Manifest.Version {
		return errors.New("CLI installation remains complete; connection preview does not match the installed runtime and embedded plugin")
	}
	m.connectionPreview(c)
	if err := m.confirm(); err != nil {
		return err
	}
	apply := m.applySelectedConnection
	if apply == nil {
		apply = install.ApplyViaRuntimeAcknowledged
	}
	native, err := apply(m.ctx, c, false)
	if err != nil && native.Phase == "deferred" {
		if handoffErr := m.confirmStoppedSessions(); handoffErr != nil {
			return handoffErr
		}
		native, err = apply(m.ctx, c, true)
	}
	if err != nil {
		if native.Phase != "" {
			m.connectionResult(native)
		}
		return fmt.Errorf("CLI installation remains complete; native connection needs inspection: %w", err)
	}
	m.block(console.Block{Title: "[PASS] Native connection verified", Body: "Start a fresh Codex session to load the updated plugin, hooks and memory tools.", Tone: console.Success})
	m.connectionResult(native)
	return m.outputErr
}
