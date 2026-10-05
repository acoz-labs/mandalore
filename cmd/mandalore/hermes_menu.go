package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/distribution"
	"github.com/acoz-labs/mandalore/internal/install"
)

func (m *menu) hermesInputs(withBinding bool) error {
	if err := connectionHarnessDefaults(&m.hermesProfile, "hermes"); err != nil {
		return err
	}
	inputs := []struct {
		title string
		value *string
	}{}
	if withBinding {
		inputs = append(inputs, struct {
			title string
			value *string
		}{"Selected signet binding", &m.binding})
	}
	inputs = append(inputs, []struct {
		title string
		value *string
	}{
		{"Selected native Hermes profile", &m.hermesProfile.NativeHome},
		{"Selected native installation state", &m.hermesProfile.StateDir},
		{"Native Hermes executable", &m.hermesProfile.NativeBinary},
	}...)
	for _, item := range inputs {
		value, err := m.input(item.title, *item.value)
		if err != nil {
			return err
		}
		*item.value = value
	}
	return nil
}

func (m *menu) prepareHermes(binary string) (install.HermesPlan, error) {
	if err := m.hermesInputs(true); err != nil {
		return install.HermesPlan{}, err
	}
	mode, readOnly, exists, err := install.ExistingSessionMode("hermes", m.hermesProfile)
	if err != nil {
		return install.HermesPlan{}, err
	}
	def := 0
	if exists && readOnly {
		def = 1
	} else if exists && mode == 0 {
		def = 2
	}
	n, err := m.selectItem("Hermes memory access", []string{"Enabled session · Automatic refresh and delivery", "Read-only · Enforce no memory writes or synchronization", "Legacy writable · Model-directed delivery"}, def)
	if err != nil {
		return install.HermesPlan{}, err
	}
	m.block(console.Block{Title: "Preparing Hermes connection preview", Body: "This executes the selected trusted Mandalore runtime to read its own package and prepare a plan. It does not yet register Hermes or change memory. A file hash is not publisher authentication."})
	if m.outputErr != nil {
		return install.HermesPlan{}, m.outputErr
	}
	prepare := m.prepareSelectedHermes
	if prepare == nil {
		prepare = install.PrepareHermesViaRuntime
	}
	transport := 0
	if n == 0 {
		transport = 1
	}
	return prepare(m.ctx, install.HermesOptions{Options: install.Options{SessionTransportVersion: transport, Binary: binary, Binding: m.binding, StateDir: m.hermesProfile.StateDir, NativeHome: m.hermesProfile.NativeHome, NativeBinary: m.hermesProfile.NativeBinary}, ReadOnly: n == 1})
}

func (m *menu) connectHermes() error {
	binary, err := m.input("Trusted local Mandalore runtime (not a download)", m.binary)
	if err != nil {
		return err
	}
	p, err := m.prepareHermes(binary)
	if err != nil {
		return err
	}
	return m.applyHermesPlan(p)
}

func (m *menu) applyHermesPlan(p install.HermesPlan) error {
	m.hermesPreview(p)
	if err := m.confirm(); err != nil {
		return err
	}
	apply := m.applySelectedHermes
	if apply == nil {
		apply = install.ApplyHermesViaRuntime
	}
	r, err := apply(m.ctx, p)
	if r.Phase != "" {
		m.hermesResult(r)
	}
	if err != nil {
		return fmt.Errorf("Hermes connection needs inspection; no automatic retry or rollback: %w", err)
	}
	m.binary = p.Binary
	m.block(console.Block{Title: "[PASS] Hermes connection verified", Body: "Start a fresh Hermes session or use native reload to load the memory tools and skills. Installation does not prove authentication or active model context.", Tone: console.Success})
	return m.outputErr
}

func (m *menu) hermesPreview(p install.HermesPlan) {
	mode := "Legacy learning enabled · Model-directed delivery"
	if p.SessionTransportVersion == 1 {
		mode = "Enabled session · Automatic refresh before turns and delivery after saves"
	}
	previous := p.PreviousRoot
	if previous == "" {
		previous = "None"
	}
	if p.ReadOnly {
		mode = "Read-only (enforced)"
	}
	m.block(console.Block{Title: "Review Hermes connection", Body: "Apply runs the selected binaries and changes only the proven owned Hermes registration. Updates remove the previous registration before installing the new one; an interruption can leave memory disconnected. Retain old generations and phase receipts. No signet edits, authentication setup or synchronization.", Fields: []console.Field{
		{Label: "Harness", Value: "Hermes"}, {Label: "Signet ID", Value: p.SignetID}, {Label: "Binding", Value: p.Binding}, {Label: "Memory access", Value: mode},
		{Label: "Native profile", Value: p.NativeHome}, {Label: "Native binary", Value: p.NativeBinary}, {Label: "Native SHA256", Value: p.NativeSHA256},
		{Label: "Selected runtime", Value: p.Binary}, {Label: "Runtime SHA256", Value: p.BinarySHA256}, {Label: "Hermesnned runtime", Value: p.Runtime},
		{Label: "Installation state", Value: p.StateDir}, {Label: "Managed generation", Value: p.Root}, {Label: "Previous generation", Value: previous},
		{Label: "Package version", Value: p.PackageVersion}, {Label: "Hermes package SHA256", Value: p.PackageSHA256},
	}})
}

func (m *menu) hermesResult(r install.HermesResult) {
	previous := r.Connection.PreviousRoot
	if previous == "" {
		previous = "None"
	}
	m.block(console.Block{Title: "Hermes connection receipt", Body: r.Notice, Fields: []console.Field{
		{Label: "Completed phase", Value: r.Phase}, {Label: "Target", Value: r.Connection.Root}, {Label: "Previous generation", Value: previous},
		{Label: "Attempt receipt", Value: r.Attempt}, {Label: "Native effects uncertain", Value: strconv.FormatBool(r.Uncertain)},
		{Label: "Installed", Value: strconv.FormatBool(r.Installed)}, {Label: "Fresh session required", Value: strconv.FormatBool(r.RequiresFreshSession)},
	}})
}

func (m *menu) hermesReport(r install.HermesReport) {
	// Same check rendering as Codex; Hermes ownership and results stay typed separately.
	m.block(console.Block{Title: "The Armorer · Hermes", Fields: []console.Field{{Label: "Native profile", Value: m.hermesProfile.NativeHome}}})
	m.report(install.Report{Checks: r.Checks, Notice: r.Notice})
	if r.Connection != nil {
		p := r.Connection
		m.block(console.Block{Title: "Retained Hermes connection", Fields: []console.Field{{Label: "Root", Value: p.Root}, {Label: "Signet ID", Value: p.SignetID}, {Label: "Binding", Value: p.Binding}, {Label: "Runtime", Value: p.Runtime}}})
	}
}

func (m *menu) doctorHermes() error {
	if err := m.hermesInputs(false); err != nil {
		return err
	}
	return m.nativeInspection("hermes", m.hermesProfile)
}

func (m *menu) repairHermes() error {
	root, err := m.input("Retained Hermes connection root (shown by The Armorer or its receipt)", "")
	if err != nil {
		return err
	}
	m.block(console.Block{Title: "Preparing owned Hermes recovery", Body: "Inspect the retained ownership receipt and execute its verified runtime to preview repair. Preserve the signet binding and memory access mode. Nothing is registered until you confirm the plan."})
	if m.outputErr != nil {
		return m.outputErr
	}
	p, err := install.PrepareHermesRepairViaRuntime(m.ctx, install.RepairInput{Root: root, NativeBinary: m.hermesProfile.NativeBinary})
	if err != nil {
		return err
	}
	return m.applyHermesPlan(p)
}

func (m *menu) offerHermesRelease(p distribution.InstallPlan, r distribution.InstallResult) error {
	c, err := m.prepareHermes(r.Runtime)
	if err != nil {
		return fmt.Errorf("CLI installation remains complete; Hermes preview unavailable: %w", err)
	}
	// The format-1 manifest's plugin hash names Codex, not Hermes. The selected
	// verified binary owns Hermes package identity; check its independently stamped version.
	if c.Binary != r.Runtime || c.BinarySHA256 != p.Binary.SHA256 || c.PackageVersion != p.Source.Manifest.Manifest.Version {
		return errors.New("CLI installation remains complete; Hermes preview does not match the installed runtime")
	}
	if err := m.applyHermesPlan(c); err != nil {
		return fmt.Errorf("CLI installation remains complete; %w", err)
	}
	return nil
}
