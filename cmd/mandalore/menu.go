package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/install"
)

// This adapter owns conversation flow only. Every durable operation delegates
// to the same API that non-interactive callers use.
type menu struct {
	ctx       context.Context
	in        *bufio.Reader
	out       io.Writer
	tui       *console.Console
	binding   string
	profile   install.Profile
	binary    string
	failed    bool
	outputErr error
}

var errMenuInputLimit = errors.New("answer exceeds 4096 bytes; menu stopped without interpreting remaining input")

func runMenu(ctx context.Context, args []string, input io.Reader, out io.Writer) int {
	f := flag.NewFlagSet("menu", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	plain := f.Bool("plain", false, "Numbered plain-text prompts")
	m := &menu{ctx: ctx, in: bufio.NewReader(input), out: out}
	f.StringVar(&m.binding, "binding", "", "Selected local binding")
	f.StringVar(&m.profile.StateDir, "state-dir", "", "Installation directory")
	f.StringVar(&m.profile.NativeHome, "native-home", "", "Native Codex profile")
	f.StringVar(&m.profile.NativeBinary, "native-binary", "", "Native Codex executable")
	f.StringVar(&m.binary, "binary", "", "Trusted local runtime artifact")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = io.WriteString(out, help)
			return 0
		}
		return bad(out, "Invalid menu options; use --help.")
	}
	if !*plain {
		m.tui = console.New(input, out)
		if m.tui != nil {
			m.tui = m.tui.WithCancellation(ctx)
		}
	}
	if m.binding == "" {
		var err error
		m.binding, err = binding.DefaultPath()
		if err != nil {
			return bad(out, "Cannot select binding default; supply --binding with an absolute path.")
		}
	}
	if m.binary == "" {
		m.binary, _ = os.Executable()
	}
	m.block(console.Block{Title: "Mandalore", Body: "Memory across time and space. Opening this menu changes nothing. A signet is your private memory bank."})
	choices := []string{"Signet · Create a new local memory bank", "Signet · Connect an existing local clone", "Signet · Inspect selected memory and sync status", "Signet · Synchronize with its configured remote", "Codex · Connect or update from a local artifact", "Codex · Doctor (read-only structural checks)", "Codex · Repair from a retained connection", "Foundlings · Manage historical references", "Exit"}
	for {
		if m.outputErr != nil {
			return 1
		}
		n, err := m.selectItem("What would you like to do?", choices, 8)
		if err == nil && n == 8 {
			break
		}
		if err == nil {
			switch n {
			case 0:
				err = m.setup(true)
			case 1:
				err = m.setup(false)
			case 2:
				err = m.inspect()
			case 3:
				err = m.sync()
			case 4:
				err = m.connect()
			case 5:
				err = m.doctor()
			case 6:
				err = m.repair()
			case 7:
				err = m.foundlings()
			}
		}
		if m.ctx.Err() != nil {
			return 130
		}
		if m.outputErr != nil {
			return 1
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if errors.Is(err, console.ErrBack) {
			continue
		}
		if err != nil {
			m.failed = true
			m.block(console.Block{Title: "[FAIL] Needs attention", Body: err.Error(), Tone: console.Failure})
			if errors.Is(err, errMenuInputLimit) {
				return 1
			}
		}
	}
	if m.ctx.Err() != nil {
		return 130
	}
	if m.failed {
		return 1
	}
	return 0
}

func (m *menu) block(b console.Block) {
	if m.outputErr != nil {
		return
	}
	if m.tui != nil {
		m.outputErr = m.tui.Block(b)
		return
	}
	_, m.outputErr = io.WriteString(m.out, console.RenderBlock(b, console.Theme{}, console.ReportWidth(m.out)))
}

func (m *menu) promptMark() {
	if m.outputErr == nil {
		_, m.outputErr = io.WriteString(m.out, "> ")
	}
}

// Require a complete newline-terminated answer. In particular, EOF after "2"
// is not approval. Keep one reader across every journey so buffered input is
// neither lost nor unexpectedly consumed by a second scanner.
func (m *menu) line() (string, error) {
	if err := m.ctx.Err(); err != nil {
		return "", err
	}
	if m.outputErr != nil {
		return "", m.outputErr
	}
	raw, err := m.in.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) {
		return "", errMenuInputLimit
	}
	if err != nil {
		return "", err
	}
	line := string(raw)
	if err := m.ctx.Err(); err != nil {
		return "", err
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	for _, r := range line {
		if unicode.IsControl(r) {
			return "", errors.New("answer contains terminal controls; enter plain text")
		}
	}
	line = strings.TrimSpace(line)
	if line == ":back" {
		return "", console.ErrBack
	}
	return line, nil
}

func (m *menu) selectItem(title string, choices []string, def int) (int, error) {
	if err := m.ctx.Err(); err != nil {
		return 0, err
	}
	if m.tui != nil {
		return m.tui.Select(title, choices, def)
	}
	for {
		m.block(console.Block{Title: title, Choices: choices, Body: fmt.Sprintf("Enter a number (default %d); :back cancels; EOF exits.", def+1)})
		m.promptMark()
		line, err := m.line()
		if err != nil {
			return 0, err
		}
		if line == "" {
			return def, nil
		}
		n, err := strconv.Atoi(line)
		if err == nil && n > 0 && n <= len(choices) {
			return n - 1, nil
		}
		m.block(console.Block{Title: "[WARN] Choose a listed number", Tone: console.Warning})
	}
}

func (m *menu) input(title, def string) (string, error) {
	if err := m.ctx.Err(); err != nil {
		return "", err
	}
	if m.tui != nil {
		value, err := m.tui.Input(title, def)
		if value == ":back" && err == nil {
			err = console.ErrBack
		}
		return value, err
	}
	m.block(console.Block{Title: title, Fields: []console.Field{{Label: "Default", Value: def}}, Body: ":back cancels; EOF exits."})
	m.promptMark()
	value, err := m.line()
	if err == nil && value == "" {
		value = def
	}
	return value, err
}

func (m *menu) confirm() error {
	if m.outputErr != nil {
		return m.outputErr
	}
	n, err := m.selectItem("Apply these changes?", []string{"No · Return without applying", "Yes · Apply this reviewed plan"}, 0)
	if err != nil {
		return err
	}
	if n == 0 {
		return console.ErrBack
	}
	return m.ctx.Err()
}

func (m *menu) call(name string, value any, bound bool) api.Envelope {
	if value == nil {
		value = struct{}{}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return api.Failure("input.invalid", "Cannot encode operation input.", false)
	}
	a := api.New(nil, false)
	if bound {
		s, err := binding.Open(m.binding, "menu")
		if err != nil {
			return api.Failure("binding.invalid", "Cannot open the selected binding; inspect it or connect a signet first.", false)
		}
		a = api.New(s, false)
	}
	return a.Call(m.ctx, name, raw)
}

func (m *menu) outcome(title string, v api.Envelope) error {
	if !v.OK {
		if v.Error.FoundlingResult != nil {
			m.foundlingReceipt(*v.Error.FoundlingResult)
		}
		if v.Error.ConnectionReport != nil {
			m.report(*v.Error.ConnectionReport)
		}
		if v.Error.ConnectionResult != nil {
			m.connectionResult(*v.Error.ConnectionResult)
		}
		if v.Error.SyncStatus != nil {
			m.jsonBlock("Synchronization receipt", v.Error.SyncStatus)
		}
		message := v.Error.Message
		if v.Error.WriteMayHaveOccurred {
			message += " Changes may have occurred; inspect the retained paths and receipt before retrying. No automatic rollback was performed."
		}
		return errors.New(message)
	}
	m.block(console.Block{Title: "[PASS] " + title, Tone: console.Success})
	return nil
}

func (m *menu) jsonBlock(title string, value any) {
	// Small status objects are flattened into labeled fields, never dumped as
	// an unbounded raw log. Complete machine-readable receipts remain on the CLI.
	raw, _ := json.Marshal(value)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	var notice string
	_ = json.Unmarshal(fields["notice"], &notice)
	delete(fields, "notice")
	var ordered []console.Field
	var nested []string
	// json.Marshal sorts keys, but maps do not: derive display order separately.
	for _, key := range sortedMenuKeys(fields) {
		var object map[string]json.RawMessage
		if json.Unmarshal(fields[key], &object) == nil && len(object) > 0 {
			nested = append(nested, key)
			continue
		}
		value := string(fields[key])
		var text string
		if json.Unmarshal(fields[key], &text) == nil {
			value = text
		}
		ordered = append(ordered, console.Field{Label: strings.ReplaceAll(key, "_", " "), Value: value})
	}
	m.block(console.Block{Title: title, Body: notice, Fields: ordered})
	for _, key := range nested {
		m.jsonBlock(title+" · "+strings.ReplaceAll(key, "_", " "), fields[key])
	}
}

func sortedMenuKeys(fields map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (m *menu) setup(create bool) error {
	var root, name, label, actor, path string
	inputs := []struct {
		title, def string
		dest       *string
	}{{"Local signet directory (absolute path)", "", &root}}
	if create {
		inputs = append(inputs, struct {
			title, def string
			dest       *string
		}{"Signet name", "My signet", &name})
	}
	inputs = append(inputs, []struct {
		title, def string
		dest       *string
	}{
		{"Machine label (stored in memory; choose deliberately)", "", &label},
		{"Writer attribution (stored in memory)", "", &actor},
		{"New machine-local binding file", m.binding, &path},
	}...)
	for _, item := range inputs {
		value, err := m.input(item.title, item.def)
		if err != nil {
			return err
		}
		if value == "" {
			return errors.New("all setup fields are required")
		}
		*item.dest = value
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(path) {
		return errors.New("signet and binding paths must be absolute")
	}
	if err := binding.ValidateBindingDestination(root, path); err != nil {
		return err
	}
	if len(label) > 256 || len(actor) > 256 {
		return errors.New("machine label and writer must fit within 256 bytes")
	}
	body := "Create a local memory bank, enroll this device, write the binding outside Git, then initialize standalone Git. These are separate durable steps. No remote, credentials or native plugin will be configured."
	if !create {
		body = "Connect an already-cloned Mandalore signet and enroll a new device. Preserve the bank identity. No fetch, push, Git initialization or native plugin installation. Predecessor banks must be migrated separately."
	}
	m.block(console.Block{Title: "Review local setup", Body: body, Fields: []console.Field{{Label: "Signet", Value: root}, {Label: "Name", Value: name}, {Label: "Machine", Value: label}, {Label: "Writer", Value: actor}, {Label: "Binding", Value: path}}})
	if err := m.confirm(); err != nil {
		return err
	}
	if create {
		v := m.call("signet_create", api.CreateInput{Repository: root, Name: name, DeviceLabel: label}, false)
		if err := m.outcome("Signet created locally", v); err != nil {
			return err
		}
	}
	v := m.call("signet_bind", api.BindInput{Repository: root, Binding: path, DeviceLabel: label, Actor: actor}, false)
	if err := m.outcome("Device binding created", v); err != nil {
		if create {
			return fmt.Errorf("Partial setup: signet created at %s, but binding did not complete. Existing files were retained. %w", root, err)
		}
		return err
	}
	m.binding = path
	if create {
		v = m.call("memory_git_init", nil, true)
		if err := m.outcome("Local Git initialized", v); err != nil {
			return fmt.Errorf("Partial setup: signet and binding are ready, but Git initialization did not complete. Inspect the signet and retry memory git-init with the displayed binding. %w", err)
		}
	}
	title := "[PASS] Local signet connected"
	if create {
		title = "[PASS] Local signet ready"
	}
	body = "Selected for this menu. Connect Codex when ready. No synchronization was performed; inspect its existing Git configuration before synchronizing."
	if create {
		body = "Remote synchronization is not configured. Keep this private bank backed up; configure its native Git origin separately. Next: connect Codex. No agent was launched."
	}
	m.block(console.Block{Title: title, Body: body, Tone: console.Success, Fields: []console.Field{{Label: "Signet", Value: root}, {Label: "Binding", Value: path}}})
	return nil
}

func (m *menu) inspect() error {
	m.block(console.Block{Title: "Selected memory", Fields: []console.Field{{Label: "Binding", Value: m.binding}}, Body: "Read-only local inspection; remote freshness is not tested."})
	for _, name := range []string{"memory_inspect", "memory_sync_status"} {
		v := m.call(name, nil, true)
		title := "Signet structure"
		if name == "memory_sync_status" {
			title = "Local synchronization status"
		}
		if err := m.outcome(title, v); err != nil {
			return err
		}
		m.jsonBlock(title, v.Result)
	}
	return nil
}

func (m *menu) sync() error {
	if err := m.inspect(); err != nil {
		return err
	}
	m.block(console.Block{Title: "Review synchronization", Body: "Checkpoint local changes, fetch, validate, integrate and push through this signet's configured Git origin. Uses existing Git authentication. Pending/offline or conflicting results are not successful delivery. No force push.", Fields: []console.Field{{Label: "Binding", Value: m.binding}}})
	if err := m.confirm(); err != nil {
		return err
	}
	v := m.call("memory_sync", nil, true)
	if err := m.outcome("Synchronization command completed (inspect delivery state below)", v); err != nil {
		return err
	}
	m.jsonBlock("Delivery status", v.Result)
	return nil
}

func (m *menu) nativeProfile() error {
	// Resolve defaults only when a native journey is selected; lack of Codex
	// must not prevent memory-only setup or even opening/exiting the menu.
	return connectionDefaults(&m.profile)
}

func (m *menu) connect() error {
	if err := m.nativeProfile(); err != nil {
		return err
	}
	binary, err := m.input("Trusted local Mandalore runtime (not a download)", m.binary)
	if err != nil {
		return err
	}
	v := m.call("connection_plan", install.Options{StateDir: m.profile.StateDir, NativeHome: m.profile.NativeHome, NativeBinary: m.profile.NativeBinary, Binary: binary, Binding: m.binding}, false)
	if !v.OK {
		return m.outcome("Connection preview", v)
	}
	return m.applyPlan(v.Result.(install.Plan))
}

func (m *menu) applyPlan(p install.Plan) error {
	m.block(console.Block{Title: "Review Codex connection", Body: "Apply executes the selected trusted binaries and manages this native plugin registration. A hash identifies bytes; it does not prove publisher trust. Retain old source/runtime copies; native cache may be replaced. No signet edits or authentication setup. Review hooks and start a fresh native session afterward.", Fields: []console.Field{
		{Label: "Signet ID", Value: p.SignetID}, {Label: "Binding", Value: p.Binding}, {Label: "Selected runtime", Value: p.Binary}, {Label: "Runtime SHA256", Value: p.BinarySHA256}, {Label: "Pinned runtime", Value: p.Runtime}, {Label: "Native binary", Value: p.NativeBinary}, {Label: "Native SHA256", Value: p.NativeSHA256}, {Label: "Native profile", Value: p.NativeHome}, {Label: "Installation state", Value: p.StateDir}, {Label: "Managed package", Value: p.Root}, {Label: "Package version", Value: p.Version}, {Label: "Embedded package SHA256", Value: p.PackageSHA256},
	}})
	if err := m.confirm(); err != nil {
		return err
	}
	v := m.call("connection_apply", p, false)
	if err := m.outcome("Native connection verified", v); err != nil {
		return err
	}
	m.connectionResult(v.Result.(install.Result))
	m.binary = p.Binary
	return nil
}

func (m *menu) connectionResult(r install.Result) {
	m.block(console.Block{Title: "Connection receipt", Body: r.Notice, Fields: []console.Field{{Label: "Completed phase", Value: r.Phase}, {Label: "Target", Value: r.Connection.Root}, {Label: "Previous source", Value: r.PreviousRoot}, {Label: "Fresh session required", Value: strconv.FormatBool(r.RequiresFreshSession)}}})
}

func (m *menu) report(r install.Report) {
	m.block(console.Block{Title: "Connection doctor", Body: r.Notice})
	for _, check := range r.Checks {
		tone := console.Warning
		if check.Status == "pass" {
			tone = console.Success
		}
		if check.Status == "fail" {
			tone = console.Failure
		}
		m.block(console.Block{Title: "[" + strings.ToUpper(check.Status) + "] " + check.Name, Body: check.Detail, Tone: tone})
	}
	if r.Connection != nil {
		m.block(console.Block{Title: "Retained connection", Fields: []console.Field{{Label: "Root", Value: r.Connection.Root}, {Label: "Binding", Value: r.Connection.Binding}}})
	}
}

func (m *menu) doctor() error {
	if err := m.nativeProfile(); err != nil {
		return err
	}
	v := m.call("connection_doctor", m.profile, false)
	if !v.OK {
		return m.outcome("Doctor", v)
	}
	m.report(v.Result.(install.Report))
	return nil
}

func (m *menu) repair() error {
	root, err := m.input("Retained managed connection root (shown by doctor)", "")
	if err != nil {
		return err
	}
	v := m.call("connection_repair_plan", install.RepairInput{Root: root, NativeBinary: m.profile.NativeBinary}, false)
	if !v.OK {
		return m.outcome("Repair preview", v)
	}
	return m.applyPlan(v.Result.(install.Plan))
}
