package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/readiness"
)

func (m *menu) armorer() error {
	for {
		n, err := m.selectItem("The Armorer", []string{"Assess this machine · No execution or changes", "Inspect native connection · Runs the selected native program", "Back"}, 0)
		if err != nil {
			return err
		}
		switch n {
		case 0:
			err = m.assessMachine()
		case 1:
			err = m.doctor()
		case 2:
			return nil
		}
		if err != nil && !errors.Is(err, console.ErrBack) {
			return err
		}
	}
}

func (m *menu) assessMachine() error {
	harness, err := m.chooseHarness()
	if err != nil {
		return err
	}
	profile := m.profile
	if harness == "pi" {
		profile = m.piProfile
	} else if harness == "claude-code" {
		profile = m.claudeProfile
	}
	in := readiness.Input{Harness: harness, StateDir: profile.StateDir, NativeHome: profile.NativeHome, NativeBinary: profile.NativeBinary, Binding: m.binding, IncludePrompt: true}
	for {
		selected, selectionErr := readiness.ResolveSelection(in)
		if selectionErr != nil {
			m.block(console.Block{Title: "[WARN] Selection needs attention", Body: selectionErr.Error() + " Choose Edit selection to correct explicit paths or override invalid defaults.", Tone: console.Warning})
		} else {
			m.assessmentSelection(selected, harness)
		}
		n, err := m.selectItem("Assessment selection", []string{"Assess selected paths", "Edit selection", "Back"}, 0)
		if err != nil {
			return err
		}
		switch n {
		case 2:
			return nil
		case 1:
			if err := m.editAssessment(&in); err != nil && !errors.Is(err, console.ErrBack) {
				return err
			}
			continue
		}
		if selectionErr != nil {
			continue
		}
		v := m.call("connection_assess", in, false)
		if !v.OK {
			if m.ctx.Err() != nil {
				return m.ctx.Err()
			}
			if v.Error.Code == "input.invalid" {
				m.block(console.Block{Title: "[WARN] Selection needs attention", Body: v.Error.Message, Tone: console.Warning})
				continue
			}
			return m.outcome("Machine readiness", v)
		}
		r := v.Result.(readiness.Report)
		if err := m.assessmentReport(r); err != nil && !errors.Is(err, console.ErrBack) {
			return err
		}
	}
}

func selectedValue(p readiness.SelectedPath) string {
	if p.Path == "" {
		if p.Source == "unselected" {
			return "None selected (active registration is not inferred)"
		}
		return "Not selected · " + p.Source
	}
	return p.Path + " (" + p.Source + ")"
}

func (m *menu) assessmentSelection(s readiness.Selection, harness string) {
	m.block(console.Block{Title: "Selected paths · " + harness, Body: "Local metadata only. No programs, providers, memory content or changes. Retained generations are never selected automatically.", Fields: []console.Field{
		{Label: "Native program", Value: selectedValue(s.NativeBinary)}, {Label: "Native profile", Value: selectedValue(s.NativeHome)},
		{Label: "Installation", Value: selectedValue(s.StateDir)}, {Label: "Binding", Value: selectedValue(s.Binding)}, {Label: "Retained root", Value: selectedValue(s.ConnectionRoot)},
	}})
}

func (m *menu) editAssessment(in *readiness.Input) error {
	for {
		n, err := m.selectItem("Edit assessment selection", []string{"Native executable", "Native profile", "Installation state", "Signet binding", "Optional retained root", "Back"}, 5)
		if err != nil {
			return err
		}
		if n == 5 {
			return nil
		}
		values := []*string{&in.NativeBinary, &in.NativeHome, &in.StateDir, &in.Binding, &in.ConnectionRoot}
		def := *values[n]
		if s, err := readiness.ResolveSelection(*in); err == nil {
			def = []string{s.NativeBinary.Path, s.NativeHome.Path, s.StateDir.Path, s.Binding.Path, s.ConnectionRoot.Path}[n]
		}
		m.block(console.Block{Title: "Edit one path", Body: "Enter an absolute path. :default restores automatic selection; :none clears the optional retained root. :back leaves this field unchanged."})
		value, err := m.input([]string{"Native executable", "Native profile", "Installation state", "Signet binding", "Optional retained root"}[n], def)
		if errors.Is(err, console.ErrBack) {
			continue
		}
		if err != nil {
			return err
		}
		if value == ":default" || n == 4 && value == ":none" {
			value = ""
		}
		*values[n] = value
	}
}

func componentLabel(id string) string {
	labels := map[string]string{"memory-runtime": "Memory runtime", "git-sync": "Git for delivery", "codex": "Codex", "pi": "Pi", "claude-code": "Claude Code", "node": "Node for Pi", "native-profile": "Native profile", "installation-state": "Installation state", "binding": "Signet binding", "retained-runtime": "Retained runtime"}
	if label := labels[id]; label != "" {
		return label
	}
	return id
}

func (m *menu) assessmentSummary(r readiness.Report) {
	body := "Completed local assessment. No programs run or files changed. Completion is not an overall readiness verdict."
	if !r.Complete {
		body = "Partial local assessment. Some observations could not be completed. No programs run or files changed."
	}
	m.block(console.Block{Title: "The Armorer · Machine readiness", Body: body})
	m.block(console.Block{Title: "Next step", Body: r.NextAction.Summary})
	for _, block := range assessmentComponentBlocks(r.Components, console.ReportWidth(m.out)) {
		m.block(block)
	}
	m.block(console.Block{Title: "Still untested", Body: strings.Join(r.Untested, "; ")})
}

func assessmentComponentBlocks(components []readiness.Component, width int) []console.Block {
	blocks := []console.Block{{Title: "Observed components", Body: "Evidence is scenario-specific; historical does not mean broken. Details explains scope and limitations."}}
	for _, c := range components {
		if width < 60 {
			blocks = append(blocks, console.Block{Title: componentLabel(c.ID), Fields: []console.Field{{Label: "Support", Value: c.Support}, {Label: "Setup", Value: c.Setup}, {Label: "Evidence", Value: c.Evidence}}})
		} else {
			blocks[0].Fields = append(blocks[0].Fields, console.Field{Label: componentLabel(c.ID), Value: "Support: " + c.Support + "; Setup: " + c.Setup + "; Evidence: " + c.Evidence})
		}
	}
	return blocks
}

func (m *menu) assessmentReport(r readiness.Report) error {
	if err := m.ctx.Err(); err != nil {
		return err
	}
	m.assessmentSummary(r)
	for {
		n, err := m.selectItem("Assessment actions", []string{"Back", "Details", "Show agent follow-up prompt", "Run native checks"}, 0)
		if err != nil {
			return err
		}
		switch n {
		case 0:
			return nil
		case 1:
			m.assessmentDetails(r)
		case 2:
			m.block(console.Block{Title: "Agent follow-up prompt · Assessment-time snapshot", Body: r.Prompt})
		case 3:
			profile := install.Profile{StateDir: r.Selection.StateDir.Path, NativeHome: r.Selection.NativeHome.Path, NativeBinary: r.Selection.NativeBinary.Path}
			err := m.nativeInspection(r.Harness, profile)
			if errors.Is(err, console.ErrBack) {
				continue
			}
			if err != nil {
				if m.ctx.Err() != nil {
					return m.ctx.Err()
				}
				// EOF/cancel must not be converted into a successful native result.
				if errors.Is(err, errMenuInputLimit) || errors.Is(err, io.EOF) {
					return err
				}
				m.failed = true
				m.block(console.Block{Title: "[FAIL] Native inspection needs attention", Body: err.Error(), Tone: console.Failure})
			}
			m.block(console.Block{Title: "Assessment snapshot unchanged", Body: "Native inspection is separate. No automatic repair or retry was performed. Return to selection and reassess after any separately authorized changes."})
		}
	}
}

func (m *menu) assessmentDetails(r readiness.Report) {
	m.assessmentSelection(r.Selection, r.Harness)
	t := r.Toolkit
	m.block(console.Block{Title: "Executing toolkit and separate on-disk observation", Body: r.Notice, Fields: []console.Field{
		{Label: "Platform", Value: t.Platform.OS + "/" + t.Platform.Arch}, {Label: "Version", Value: t.Version}, {Label: "Source", Value: t.SourceCommit},
		{Label: "Disk path", Value: t.OnDisk.Path}, {Label: "Disk SHA256", Value: t.OnDisk.SHA256}, {Label: "Disk bytes", Value: fmt.Sprint(t.OnDisk.Size)},
		{Label: "Embedded package", Value: t.Package.Harness + " " + t.Package.Version}, {Label: "Package SHA256", Value: t.Package.SHA256},
	}})
	for _, c := range r.Components {
		m.block(console.Block{Title: componentLabel(c.ID), Fields: []console.Field{{Label: "Support", Value: c.Support}, {Label: "Setup", Value: c.Setup}, {Label: "Evidence", Value: c.Evidence}, {Label: "Finding", Value: c.Code}, {Label: "Observation complete", Value: fmt.Sprint(c.Complete)}}})
		if c.File != nil {
			m.jsonBlock("Measured on-disk file", c.File)
		}
		for _, match := range c.Matches {
			m.block(console.Block{Title: "Scenario match · " + match.EvidenceID, Fields: []console.Field{{Label: "State", Value: match.State}, {Label: "Reasons", Value: strings.Join(match.Reasons, ", ")}}})
		}
	}
	if r.Retained != nil {
		m.jsonBlock("Retained observation · Declarations are not loaded package proof", r.Retained)
	}
	m.block(console.Block{Title: "Declared support", Fields: []console.Field{{Label: "Targets", Value: fmt.Sprint(r.Declarations.Targets)}, {Label: "Signet read versions", Value: fmt.Sprint(r.Declarations.SignetReadVersions)}, {Label: "Signet write versions", Value: fmt.Sprint(r.Declarations.SignetWriteVersions)}, {Label: "Memory / Codex / Pi / Claude protocols", Value: fmt.Sprintf("%d / %d / %d / %d", r.Declarations.MemoryProtocol, r.Declarations.CodexHookProtocol, r.Declarations.PiHarnessProtocol, r.Declarations.ClaudeHookProtocol)}}})
	for _, d := range r.Declarations.Components {
		fields := []console.Field{{Label: "Native contract", Value: d.NativeContract}}
		for _, dep := range d.Dependencies {
			fields = append(fields, console.Field{Label: dep.ID, Value: dep.Constraint})
		}
		m.block(console.Block{Title: "Dependencies · " + componentLabel(d.ID), Fields: fields})
	}
	for _, e := range r.Declarations.Evidence {
		m.block(console.Block{Title: "Recorded evidence · " + e.ID, Body: "Recorded scenario only; not proof of this live connection.", Fields: []console.Field{{Label: "Component / scenario", Value: e.Component + " / " + e.Scenario}, {Label: "Platform", Value: e.Platform.OS + "/" + e.Platform.Arch}, {Label: "Level", Value: e.Level}, {Label: "Source commit", Value: e.SourceCommit}, {Label: "Manifest SHA256", Value: e.ManifestSHA256}}})
		m.jsonBlock("Recorded identities", e.Identities)
		for _, source := range e.Sources {
			m.block(console.Block{Title: "Immutable evidence source", Body: source})
		}
	}
}

func (m *menu) nativeInspection(harness string, profile install.Profile) error {
	if harness != "codex" && harness != "pi" && harness != "claude-code" {
		return errors.New("choose Codex, Pi or Claude Code before native inspection")
	}
	if profile.NativeBinary == "" {
		return errors.New("select a native executable before requesting native checks")
	}
	m.block(console.Block{Title: "Review native inspection", Body: "This runs the selected native program and may create native logs or cache files. It does not repair, synchronize, authenticate or prove live model context. The static assessment remains a separate snapshot.", Fields: []console.Field{{Label: "Harness", Value: harness}, {Label: "Program", Value: profile.NativeBinary}, {Label: "Profile", Value: profile.NativeHome}, {Label: "Installation", Value: profile.StateDir}}})
	n, err := m.selectItem("Run these native checks?", []string{"No · Return without running", "Yes · Run this selected native inspection"}, 0)
	if err != nil {
		return err
	}
	if n == 0 {
		return console.ErrBack
	}
	if err := m.ctx.Err(); err != nil {
		return err
	}
	if m.outputErr != nil {
		return m.outputErr
	}
	var v api.Envelope
	if harness == "pi" {
		m.piProfile = profile
	} else if harness == "claude-code" {
		m.claudeProfile = profile
	}
	if m.nativeInspect != nil {
		v = m.nativeInspect(harness, profile)
	} else {
		name := "connection_doctor"
		if harness == "pi" {
			name = "pi_connection_doctor"
		} else if harness == "claude-code" {
			name = "claude_code_connection_doctor"
		}
		v = m.call(name, profile, false)
	}
	if !v.OK {
		return m.outcome("The Armorer · Native inspection", v)
	}
	if harness == "pi" {
		m.piReport(v.Result.(install.PiReport))
	} else if harness == "claude-code" {
		m.claudeReport(v.Result.(install.ClaudeReport))
	} else {
		m.report(v.Result.(install.Report))
	}
	return m.outputErr
}
