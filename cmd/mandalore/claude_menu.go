package main

import (
	"errors"
	"fmt"
	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/distribution"
	"github.com/acoz-labs/mandalore/internal/install"
	"strconv"
)

func (m *menu) claudeInputs(withBinding bool) error {
	if err := connectionHarnessDefaults(&m.claudeProfile, "claude-code"); err != nil {
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
		{"Selected native Claude Code profile", &m.claudeProfile.NativeHome},
		{"Selected native installation state", &m.claudeProfile.StateDir},
		{"Native Claude Code executable", &m.claudeProfile.NativeBinary},
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

func (m *menu) prepareClaude(binary string) (install.ClaudePlan, error) {
	if err := m.claudeInputs(true); err != nil {
		return install.ClaudePlan{}, err
	}
	n, err := m.selectItem("Claude Code memory access", []string{"Learning enabled · Allow authorized memory changes", "Read-only · Enforce no memory writes or synchronization"}, 0)
	if err != nil {
		return install.ClaudePlan{}, err
	}
	m.block(console.Block{Title: "Preparing Claude Code connection preview", Body: "This executes the selected trusted Mandalore runtime to read its own package and prepare a plan. It does not yet register Claude or change memory. A file hash is not publisher authentication."})
	if m.outputErr != nil {
		return install.ClaudePlan{}, m.outputErr
	}
	prepare := m.prepareSelectedClaude
	if prepare == nil {
		prepare = install.PrepareClaudeViaRuntime
	}
	return prepare(m.ctx, install.ClaudeOptions{Options: install.Options{Binary: binary, Binding: m.binding, StateDir: m.claudeProfile.StateDir, NativeHome: m.claudeProfile.NativeHome, NativeBinary: m.claudeProfile.NativeBinary}, ReadOnly: n == 1})
}

func (m *menu) connectClaude() error {
	binary, err := m.input("Trusted local Mandalore runtime (not a download)", m.binary)
	if err != nil {
		return err
	}
	p, err := m.prepareClaude(binary)
	if err != nil {
		return err
	}
	return m.applyClaudePlan(p)
}

func (m *menu) applyClaudePlan(p install.ClaudePlan) error {
	m.claudePreview(p)
	if err := m.confirm(); err != nil {
		return err
	}
	apply := m.applySelectedClaude
	if apply == nil {
		apply = install.ApplyClaudeViaRuntime
	}
	stopped := false
	if p.PreviousRoot != "" {
		n, err := m.selectItem("Have all affected Claude Code sessions exited?", []string{"No · Defer plugin update", "Yes · Sessions exited; apply this update"}, 0)
		if err != nil {
			return err
		}
		if n == 0 {
			return console.ErrBack
		}
		stopped = true
	}
	r, err := apply(m.ctx, install.ClaudeApplyInput{Plan: p, SessionsStopped: stopped})
	if r.Phase != "" {
		m.claudeResult(r)
	}
	if err != nil {
		return fmt.Errorf("Claude Code connection needs inspection; no automatic retry or rollback: %w", err)
	}
	m.binary = p.Binary
	m.block(console.Block{Title: "[PASS] Claude Code connection verified", Body: "Start a fresh Claude Code session to load the memory tools and skills. Installation does not prove authentication or active model context.", Tone: console.Success})
	return m.outputErr
}

func (m *menu) claudePreview(p install.ClaudePlan) {
	mode := "Learning enabled"
	previous := p.PreviousRoot
	if previous == "" {
		previous = "None"
	}
	if p.ReadOnly {
		mode = "Read-only (enforced)"
	}
	m.block(console.Block{Title: "Review Claude Code connection", Body: "Apply runs the selected binaries and changes only the proven owned Claude Code registration. Updates remove the previous registration before installing the new one; an interruption can leave memory disconnected. Retain old generations and phase receipts. No signet edits, authentication setup or synchronization.", Fields: []console.Field{
		{Label: "Harness", Value: "Claude Code"}, {Label: "Signet ID", Value: p.SignetID}, {Label: "Binding", Value: p.Binding}, {Label: "Memory access", Value: mode},
		{Label: "Native profile", Value: p.NativeHome}, {Label: "Native binary", Value: p.NativeBinary}, {Label: "Native SHA256", Value: p.NativeSHA256},
		{Label: "Selected runtime", Value: p.Binary}, {Label: "Runtime SHA256", Value: p.BinarySHA256}, {Label: "Pinned runtime", Value: p.Runtime},
		{Label: "Installation state", Value: p.StateDir}, {Label: "Managed generation", Value: p.Root}, {Label: "Previous generation", Value: previous},
		{Label: "Package version", Value: p.PackageVersion}, {Label: "Claude package SHA256", Value: p.PackageSHA256},
	}})
}

func (m *menu) claudeResult(r install.ClaudeResult) {
	previous := r.Connection.PreviousRoot
	if previous == "" {
		previous = "None"
	}
	m.block(console.Block{Title: "Claude Code connection receipt", Body: r.Notice, Fields: []console.Field{
		{Label: "Completed phase", Value: r.Phase}, {Label: "Target", Value: r.Connection.Root}, {Label: "Previous generation", Value: previous},
		{Label: "Attempt receipt", Value: r.Attempt}, {Label: "Native effects uncertain", Value: strconv.FormatBool(r.Uncertain)},
		{Label: "Installed", Value: strconv.FormatBool(r.Installed)}, {Label: "Fresh session required", Value: strconv.FormatBool(r.RequiresFreshSession)},
	}})
}

func (m *menu) claudeReport(r install.ClaudeReport) {
	// Same check rendering as Codex; Claude ownership and results stay typed separately.
	m.block(console.Block{Title: "The Armorer · Claude", Fields: []console.Field{{Label: "Native profile", Value: m.claudeProfile.NativeHome}}})
	m.report(install.Report{Checks: r.Checks, Notice: r.Notice})
	if r.Connection != nil {
		p := r.Connection
		m.block(console.Block{Title: "Retained Claude Code connection", Fields: []console.Field{{Label: "Root", Value: p.Root}, {Label: "Signet ID", Value: p.SignetID}, {Label: "Binding", Value: p.Binding}, {Label: "Runtime", Value: p.Runtime}}})
	}
}

func (m *menu) doctorClaude() error {
	if err := m.claudeInputs(false); err != nil {
		return err
	}
	return m.nativeInspection("claude-code", m.claudeProfile)
}

func (m *menu) repairClaude() error {
	root, err := m.input("Retained Claude Code connection root (shown by The Armorer or its receipt)", "")
	if err != nil {
		return err
	}
	m.block(console.Block{Title: "Preparing owned Claude recovery", Body: "Inspect the retained ownership receipt and execute its verified runtime to preview repair. Preserve the signet binding and memory access mode. Nothing is registered until you confirm the plan."})
	if m.outputErr != nil {
		return m.outputErr
	}
	p, err := install.PrepareClaudeRepairViaRuntime(m.ctx, install.RepairInput{Root: root, NativeBinary: m.claudeProfile.NativeBinary})
	if err != nil {
		return err
	}
	return m.applyClaudePlan(p)
}

func (m *menu) offerClaudeRelease(p distribution.InstallPlan, r distribution.InstallResult) error {
	c, err := m.prepareClaude(r.Runtime)
	if err != nil {
		return fmt.Errorf("CLI installation remains complete; Claude preview unavailable: %w", err)
	}
	// The format-1 manifest's plugin hash names Codex, not Claude Code. The selected
	// verified binary owns Claude package identity; check its independently stamped version.
	if c.Binary != r.Runtime || c.BinarySHA256 != p.Binary.SHA256 || c.PackageVersion != p.Source.Manifest.Manifest.Version {
		return errors.New("CLI installation remains complete; Claude preview does not match the installed runtime")
	}
	if err := m.applyClaudePlan(c); err != nil {
		return fmt.Errorf("CLI installation remains complete; %w", err)
	}
	return nil
}
