package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/distribution"
	"github.com/acoz-labs/mandalore/internal/install"
)

func (m *menu) chooseHarness() (string, error) {
	def := 0
	if m.harness == "pi" {
		def = 1
	}
	n, err := m.selectItem("Choose native harness", []string{"Codex", "Pi", "Back"}, def)
	if err != nil {
		return "", err
	}
	if n == 2 {
		return "", console.ErrBack
	}
	m.harness = []string{"codex", "pi"}[n]
	return m.harness, nil
}

func (m *menu) piInputs(withBinding bool) error {
	if err := connectionHarnessDefaults(&m.piProfile, "pi"); err != nil {
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
		{"Selected native Pi profile", &m.piProfile.NativeHome},
		{"Selected native installation state", &m.piProfile.StateDir},
		{"Native Pi executable", &m.piProfile.NativeBinary},
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

func (m *menu) preparePi(binary string) (install.PiPlan, error) {
	if err := m.piInputs(true); err != nil {
		return install.PiPlan{}, err
	}
	n, err := m.selectItem("Pi memory access", []string{"Learning enabled · Allow authorized memory changes", "Read-only · Enforce no memory writes or synchronization"}, 0)
	if err != nil {
		return install.PiPlan{}, err
	}
	m.block(console.Block{Title: "Preparing Pi connection preview", Body: "This executes the selected trusted Mandalore runtime to read its own package and prepare a plan. It does not yet register Pi or change memory. A file hash is not publisher authentication."})
	if m.outputErr != nil {
		return install.PiPlan{}, m.outputErr
	}
	prepare := m.prepareSelectedPi
	if prepare == nil {
		prepare = install.PreparePiViaRuntime
	}
	return prepare(m.ctx, install.PiOptions{Options: install.Options{Binary: binary, Binding: m.binding, StateDir: m.piProfile.StateDir, NativeHome: m.piProfile.NativeHome, NativeBinary: m.piProfile.NativeBinary}, ReadOnly: n == 1})
}

func (m *menu) connectPi() error {
	binary, err := m.input("Trusted local Mandalore runtime (not a download)", m.binary)
	if err != nil {
		return err
	}
	p, err := m.preparePi(binary)
	if err != nil {
		return err
	}
	return m.applyPiPlan(p)
}

func (m *menu) applyPiPlan(p install.PiPlan) error {
	m.piPreview(p)
	if err := m.confirm(); err != nil {
		return err
	}
	apply := m.applySelectedPi
	if apply == nil {
		apply = install.ApplyPiViaRuntime
	}
	r, err := apply(m.ctx, p)
	if r.Phase != "" {
		m.piResult(r)
	}
	if err != nil {
		return fmt.Errorf("Pi connection needs inspection; no automatic retry or rollback: %w", err)
	}
	m.binary = p.Binary
	m.block(console.Block{Title: "[PASS] Pi connection verified", Body: "Start a fresh Pi session or use native reload to load the memory tools and skills. Installation does not prove authentication or active model context.", Tone: console.Success})
	return m.outputErr
}

func (m *menu) piPreview(p install.PiPlan) {
	mode := "Learning enabled"
	if p.ReadOnly {
		mode = "Read-only (enforced)"
	}
	m.block(console.Block{Title: "Review Pi connection", Body: "Apply runs the selected binaries and changes only the proven owned Pi registration. Updates remove the previous registration before installing the new one; an interruption can leave memory disconnected. Retain old generations and phase receipts. No signet edits, authentication setup or synchronization.", Fields: []console.Field{
		{Label: "Harness", Value: "Pi"}, {Label: "Signet ID", Value: p.SignetID}, {Label: "Binding", Value: p.Binding}, {Label: "Memory access", Value: mode},
		{Label: "Native profile", Value: p.NativeHome}, {Label: "Native binary", Value: p.NativeBinary}, {Label: "Native SHA256", Value: p.NativeSHA256},
		{Label: "Selected runtime", Value: p.Binary}, {Label: "Runtime SHA256", Value: p.BinarySHA256}, {Label: "Pinned runtime", Value: p.Runtime},
		{Label: "Installation state", Value: p.StateDir}, {Label: "Managed generation", Value: p.Root}, {Label: "Previous generation", Value: p.PreviousRoot},
		{Label: "Package version", Value: p.PackageVersion}, {Label: "Pi package SHA256", Value: p.PackageSHA256},
	}})
}

func (m *menu) piResult(r install.PiResult) {
	m.block(console.Block{Title: "Pi connection receipt", Body: r.Notice, Fields: []console.Field{
		{Label: "Completed phase", Value: r.Phase}, {Label: "Target", Value: r.Connection.Root}, {Label: "Previous generation", Value: r.Connection.PreviousRoot},
		{Label: "Attempt receipt", Value: r.Attempt}, {Label: "Native effects uncertain", Value: strconv.FormatBool(r.Uncertain)},
		{Label: "Installed", Value: strconv.FormatBool(r.Installed)}, {Label: "Fresh session required", Value: strconv.FormatBool(r.RequiresFreshSession)},
	}})
}

func (m *menu) piReport(r install.PiReport) {
	// Same check rendering as Codex; Pi ownership and results stay typed separately.
	m.block(console.Block{Title: "The Armorer · Pi", Fields: []console.Field{{Label: "Native profile", Value: m.piProfile.NativeHome}}})
	m.report(install.Report{Checks: r.Checks, Notice: r.Notice})
	if r.Connection != nil {
		p := r.Connection
		m.block(console.Block{Title: "Retained Pi connection", Fields: []console.Field{{Label: "Root", Value: p.Root}, {Label: "Signet ID", Value: p.SignetID}, {Label: "Binding", Value: p.Binding}, {Label: "Runtime", Value: p.Runtime}}})
	}
}

func (m *menu) doctorPi() error {
	if err := m.piInputs(false); err != nil {
		return err
	}
	v := m.call("pi_connection_doctor", m.piProfile, false)
	if !v.OK {
		return m.outcome("The Armorer · Pi", v)
	}
	m.piReport(v.Result.(install.PiReport))
	return m.outputErr
}

func (m *menu) repairPi() error {
	root, err := m.input("Retained Pi connection root (shown by The Armorer or its receipt)", "")
	if err != nil {
		return err
	}
	m.block(console.Block{Title: "Preparing owned Pi recovery", Body: "Inspect the retained ownership receipt and execute its verified runtime to preview repair. Preserve the signet binding and memory access mode. Nothing is registered until you confirm the plan."})
	if m.outputErr != nil {
		return m.outputErr
	}
	p, err := install.PreparePiRepairViaRuntime(m.ctx, install.RepairInput{Root: root, NativeBinary: m.piProfile.NativeBinary})
	if err != nil {
		return err
	}
	return m.applyPiPlan(p)
}

func (m *menu) offerPiRelease(p distribution.InstallPlan, r distribution.InstallResult) error {
	c, err := m.preparePi(r.Runtime)
	if err != nil {
		return fmt.Errorf("CLI installation remains complete; Pi preview unavailable: %w", err)
	}
	// The format-1 manifest's plugin hash names Codex, not Pi. The selected
	// verified binary owns Pi package identity; check its independently stamped version.
	if c.Binary != r.Runtime || c.BinarySHA256 != p.Binary.SHA256 || c.PackageVersion != p.Source.Manifest.Manifest.Version {
		return errors.New("CLI installation remains complete; Pi preview does not match the installed runtime")
	}
	if err := m.applyPiPlan(c); err != nil {
		return fmt.Errorf("CLI installation remains complete; %w", err)
	}
	return nil
}
