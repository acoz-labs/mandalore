package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/console"
	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func (m *menu) foundlings() error {
	m.block(console.Block{Title: "Foundlings", Body: "Historical references, not current guidance. Source paths stay local to this clone. Nothing is imported, fetched or executed automatically.", Fields: []console.Field{{Label: "Selected binding", Value: m.binding}}})
	for {
		n, err := m.selectItem("Manage historical references", []string{"List references and local availability", "Register a reference", "Connect an existing reference on this machine", "Inspect, search or update a reference pin", "Disconnect a reference", "Back"}, 5)
		if err != nil {
			return err
		}
		switch n {
		case 0:
			err = m.listFoundlings()
		case 1:
			err = m.registerFoundling()
		case 2:
			err = m.connectFoundling()
		case 3:
			err = m.inspectFoundling()
		case 4:
			err = m.disconnectFoundling()
		case 5:
			return nil
		}
		if errors.Is(err, console.ErrBack) {
			continue
		}
		if err != nil {
			return err
		}
	}
}

func (m *menu) foundlingPage(offset int) (memory.Page[memory.FoundlingSummary], error) {
	v := m.call("foundling_list", api.PageInput{Offset: offset}, true)
	if !v.OK {
		return memory.Page[memory.FoundlingSummary]{}, m.outcome("Reference list", v)
	}
	return v.Result.(memory.Page[memory.FoundlingSummary]), nil
}

func (m *menu) pickFoundling(title string) (memory.FoundlingSummary, error) {
	offset := 0
	for {
		page, err := m.foundlingPage(offset)
		if err != nil {
			return memory.FoundlingSummary{}, err
		}
		if len(page.Items) == 0 {
			m.block(console.Block{Title: "No registered references", Body: "Register a source first. Empty registrations do not establish that historical knowledge is absent."})
			return memory.FoundlingSummary{}, console.ErrBack
		}
		choices := []string{}
		for _, r := range page.Items {
			name := r.Name
			if name == "" {
				name = "Conflicting registration"
			}
			choices = append(choices, fmt.Sprintf("%s · registration %s · %s", name, r.State, r.FoundlingID))
		}
		next := -1
		if page.NextOffset != nil {
			next = len(choices)
			choices = append(choices, "Next page")
		}
		back := len(choices)
		choices = append(choices, "Back")
		n, err := m.selectItem(title, choices, back)
		if err != nil {
			return memory.FoundlingSummary{}, err
		}
		if n == back {
			return memory.FoundlingSummary{}, console.ErrBack
		}
		if n == next {
			offset = *page.NextOffset
			continue
		}
		return page.Items[n], nil
	}
}

func (m *menu) foundlingInspection(id string) (foundlings.Inspection, error) {
	v := m.call("foundling_inspect", api.FoundlingSelector{FoundlingID: id}, true)
	if !v.OK {
		return foundlings.Inspection{}, m.outcome("Reference inspection", v)
	}
	return v.Result.(foundlings.Inspection), nil
}

func (m *menu) showFoundling(v foundlings.Inspection) {
	r := v.Registration
	fields := []console.Field{{Label: "Reference", Value: r.Name}, {Label: "Stable ID", Value: r.FoundlingID}, {Label: "State", Value: v.State}, {Label: "Registration heads", Value: strconv.Itoa(r.HeadCount)}}
	if r.Source != nil {
		fields = append(fields, console.Field{Label: "Portable source", Value: r.Source.Kind + " · " + r.Source.Locator})
	}
	if r.Pin != nil {
		fields = append(fields, console.Field{Label: "Portable pin", Value: r.Pin.Algorithm + ":" + r.Pin.Value})
	}
	if v.Connection != nil {
		fields = append(fields, console.Field{Label: "Local-only path", Value: v.Connection.Root})
	}
	advice := map[string]string{"available": "Verified local reference text is available; it is not current guidance.", "unconnected": "Connect an existing local directory on this machine. No path is inferred or fetched.", "unavailable": "The local source is missing or unsupported. Inspect it or explicitly connect a new path.", "changed": "The registration or source changed. Inspect before explicitly reconnecting or updating its pin.", "invalid_connection": "Preserve the invalid local configuration and inspect it through the CLI; no automatic repair.", "disconnected": "Reference disconnected. Original files and previously promoted knowledge remain intact.", "conflicted": "Registration heads conflict. Inspect foundling history and explicitly reconcile them through the typed CLI."}
	tone := console.Warning
	if v.State == "available" {
		tone = console.Success
	}
	m.block(console.Block{Title: "Reference · " + v.State, Body: advice[v.State], Fields: fields, Tone: tone})
}

func (m *menu) listFoundlings() error {
	offset := 0
	for {
		page, err := m.foundlingPage(offset)
		if err != nil {
			return err
		}
		if len(page.Items) == 0 {
			m.block(console.Block{Title: "No registered references", Body: "Register a historical source to begin."})
			return nil
		}
		for _, r := range page.Items {
			m.block(console.Block{Title: "Checking reference", Body: r.Name})
			v, err := m.foundlingInspection(r.FoundlingID)
			if err != nil {
				return err
			}
			m.showFoundling(v)
		}
		if page.NextOffset == nil {
			return nil
		}
		n, err := m.selectItem("More references", []string{"Next page", "Back"}, 1)
		if err != nil {
			return err
		}
		if n == 1 {
			return nil
		}
		offset = *page.NextOffset
	}
}

func (m *menu) showSourcePreview(title string, v foundlings.Observation) {
	m.block(console.Block{Title: title, Body: "Reference-only text. Registration stores identity/pin, not a source snapshot. This path is local-only; no fetch or source execution.", Fields: []console.Field{{Label: "Portable source", Value: v.Source.Kind + " · " + v.Source.Locator}, {Label: "Portable pin", Value: v.Pin.Algorithm + ":" + v.Pin.Value}, {Label: "Local-only path", Value: v.Root}, {Label: "Eligible files", Value: strconv.Itoa(v.Files)}, {Label: "Eligible bytes", Value: strconv.Itoa(v.Bytes)}, {Label: "Excluded entries", Value: strconv.Itoa(v.Excluded)}}})
}

func (m *menu) previewFoundling(source memory.FoundlingSource, root string) (foundlings.Observation, error) {
	v := m.call("foundling_preview", api.FoundlingPreviewInput{Source: source, Root: root}, true)
	if !v.OK {
		return foundlings.Observation{}, m.outcome("Source preview", v)
	}
	return v.Result.(foundlings.Observation), nil
}

func (m *menu) registerFoundling() error {
	name, err := m.input("Reference display name", "")
	if err != nil {
		return err
	}
	description, err := m.input("What is this historical reference?", "")
	if err != nil {
		return err
	}
	kind, err := m.selectItem("Source kind", []string{"Local text directory", "Existing standalone Git checkout", "Back"}, 0)
	if err != nil {
		return err
	}
	if kind == 2 {
		return console.ErrBack
	}
	source := memory.FoundlingSource{Kind: "local"}
	defaultID := memory.NewID("source")
	prompt := "Portable source ID (not a path)"
	if kind == 1 {
		source.Kind = "git"
		defaultID = ""
		prompt = "Portable Git origin (credential-free HTTPS/SSH)"
	}
	source.Locator, err = m.input(prompt, defaultID)
	if err != nil {
		return err
	}
	root, err := m.input("Existing source directory on this machine (absolute path)", "")
	if err != nil {
		return err
	}
	reason, err := m.input("Why link this reference?", "Linked historical reference")
	if err != nil {
		return err
	}
	if name == "" || description == "" || reason == "" {
		return errors.New("name, description and reason are required")
	}
	preview, err := m.previewFoundling(source, root)
	if err != nil {
		return err
	}
	m.showSourcePreview("Review foundling registration", preview)
	m.block(console.Block{Title: "Changes to apply", Body: "Save one portable registration, then connect this local path. No knowledge is promoted, journaled or synchronized. If connection fails, the saved registration is retained.", Fields: []console.Field{{Label: "Name", Value: name}, {Label: "Description", Value: description}, {Label: "Reason", Value: reason}}})
	if err := m.confirm(); err != nil {
		return err
	}
	v := m.call("foundling_register", api.FoundlingRegisterInput{Name: name, Description: description, Source: source, Pin: preview.Pin, Root: preview.Root, Reason: reason}, true)
	if err := m.outcome("Foundling registered and connected", v); err != nil {
		return err
	}
	m.foundlingReceipt(v.Result.(api.FoundlingMutationResult))
	return nil
}

func (m *menu) foundlingReceipt(r api.FoundlingMutationResult) {
	if r.Registration != nil {
		m.block(console.Block{Title: "Registration saved", Fields: []console.Field{{Label: "Foundling ID", Value: r.Registration.FoundlingID}, {Label: "Revision", Value: r.Registration.ID}, {Label: "Registration state", Value: r.Registration.State}}, Body: "Portable history retained. Inspect this ID before retrying; do not register a duplicate."})
	}
	if r.Connection != nil {
		title := "Connection not completed"
		body := "Registration and connection are separate steps. Inspect the saved reference and local configuration before reconnecting."
		if r.Connection.Connected {
			title = "Local connection published"
			body = "Connection is local to this clone; source files and current knowledge were not changed."
		}
		m.block(console.Block{Title: title, Body: body, Fields: []console.Field{{Label: "Durable", Value: strconv.FormatBool(r.Connection.Durable)}, {Label: "Connection ID", Value: r.Connection.Connection.ID}, {Label: "Local-only path", Value: r.Connection.Connection.Root}}})
	}
}

func activeFoundling(r memory.FoundlingSummary) error {
	if r.State != "active" || len(r.HeadIDs) != 1 {
		return errors.New("select one active registration; inspect disconnected/conflicting history through the CLI before making changes")
	}
	return nil
}

func (m *menu) connectFoundling() error {
	r, err := m.pickFoundling("Connect which reference?")
	if err != nil {
		return err
	}
	if err := activeFoundling(r); err != nil {
		return err
	}
	v, err := m.foundlingInspection(r.FoundlingID)
	if err != nil {
		return err
	}
	m.showFoundling(v)
	if v.State == "invalid_connection" {
		return errors.New("existing local connection is invalid; preserve it and inspect through the CLI")
	}
	def, expected := "", ""
	if v.Connection != nil {
		def, expected = v.Connection.Root, v.Connection.ID
	}
	root, err := m.input("Existing local source directory (absolute path)", def)
	if err != nil {
		return err
	}
	preview, err := m.previewFoundling(*r.Source, root)
	if err != nil {
		return err
	}
	if preview.Pin != *r.Pin {
		return errors.New("selected source differs from the registered pin; use Inspect to review an explicit pin update, not an implicit reconnect")
	}
	m.showSourcePreview("Review local connection", preview)
	m.block(console.Block{Title: "Changes to apply", Body: "Connect only this clone-local path to the existing registration. No portable registration, source, knowledge or remote changes."})
	if err := m.confirm(); err != nil {
		return err
	}
	result := m.call("foundling_connect", api.FoundlingConnectInput{FoundlingID: r.FoundlingID, RegistrationID: r.HeadIDs[0], Root: preview.Root, ExpectedConnectionID: expected}, true)
	if err := m.outcome("Local reference connected", result); err != nil {
		return err
	}
	c := result.Result.(foundlings.ConnectResult)
	m.foundlingReceipt(api.FoundlingMutationResult{Phase: "complete", Connection: &c})
	return nil
}

func (m *menu) inspectFoundling() error {
	r, err := m.pickFoundling("Inspect which reference?")
	if err != nil {
		return err
	}
	for {
		v, err := m.foundlingInspection(r.FoundlingID)
		if err != nil {
			return err
		}
		m.showFoundling(v)
		r = v.Registration
		n, err := m.selectItem("Reference actions", []string{"Search reference text", "Read a relative document", "Review a source pin update", "Back"}, 3)
		if err != nil {
			return err
		}
		if n == 3 {
			return nil
		}
		if n == 2 {
			return m.repinFoundling(v)
		}
		if v.State != "available" {
			return errors.New("reference is not available; connect or reconcile it before reading")
		}
		if n == 0 {
			query, err := m.input("Search terms", "")
			if err != nil {
				return err
			}
			result := m.call("foundling_search", api.FoundlingSearchInput{FoundlingID: r.FoundlingID, Query: query}, true)
			if !result.OK {
				return m.outcome("Reference search", result)
			}
			p := result.Result.(foundlings.SearchResult)
			m.block(console.Block{Title: "Reference search", Body: p.Notice, Fields: []console.Field{{Label: "Matching documents", Value: strconv.Itoa(p.MatchingCount)}, {Label: "Results omitted", Value: strconv.FormatBool(p.Truncated)}}})
			for _, e := range p.Items {
				m.showReferenceExcerpt(e)
			}
		} else {
			locator, err := m.input("Exact relative document locator", "")
			if err != nil {
				return err
			}
			offsetText, err := m.input("UTF-8 byte offset", "0")
			if err != nil {
				return err
			}
			offset, err := strconv.Atoi(offsetText)
			if err != nil {
				return errors.New("byte offset must be an integer")
			}
			result := m.call("foundling_read", api.FoundlingReadInput{FoundlingID: r.FoundlingID, RegistrationID: r.HeadIDs[0], Locator: locator, Offset: offset}, true)
			if !result.OK {
				return m.outcome("Reference read", result)
			}
			m.showReferenceExcerpt(result.Result.(foundlings.Excerpt))
		}
	}
}

func (m *menu) showReferenceExcerpt(e foundlings.Excerpt) {
	fields := []console.Field{{Label: "Relative locator", Value: e.Origin.RelativeLocator}, {Label: "Complete document", Value: strconv.FormatBool(e.Complete)}, {Label: "Offset / total bytes", Value: fmt.Sprintf("%d / %d", e.Offset, e.TotalBytes)}, {Label: "File SHA256", Value: e.Origin.ContentSHA256}}
	if e.NextOffset != nil {
		fields = append(fields, console.Field{Label: "Next byte offset", Value: strconv.Itoa(*e.NextOffset)})
	}
	m.block(console.Block{Title: "Unreviewed reference · not current guidance", Body: e.Text, Fields: fields, Tone: console.Warning})
}

func (m *menu) repinFoundling(v foundlings.Inspection) error {
	r := v.Registration
	if err := activeFoundling(r); err != nil {
		return err
	}
	if v.State == "invalid_connection" {
		return errors.New("preserve and inspect invalid local configuration before updating the registration")
	}
	root, expected := "", ""
	if v.Connection != nil {
		root, expected = v.Connection.Root, v.Connection.ID
	}
	root, err := m.input("Existing source directory for the new pin", root)
	if err != nil {
		return err
	}
	preview, err := m.previewFoundling(*r.Source, root)
	if err != nil {
		return err
	}
	if preview.Pin == *r.Pin {
		m.block(console.Block{Title: "Pin unchanged", Body: "No registration update needed. Use Connect to change or refresh this machine's local path."})
		return nil
	}
	reason, err := m.input("Why supersede this source pin?", "")
	if err != nil {
		return err
	}
	if reason == "" {
		return errors.New("pin update requires a reason")
	}
	m.showSourcePreview("Review superseding source pin", preview)
	m.block(console.Block{Title: "History to preserve", Body: "Append a superseding registration and reconnect this path. Old registrations and promoted citations remain intact; no source text is imported.", Fields: []console.Field{{Label: "Previous revision", Value: r.HeadIDs[0]}, {Label: "Previous pin", Value: r.Pin.Algorithm + ":" + r.Pin.Value}, {Label: "Reason", Value: reason}}})
	if err := m.confirm(); err != nil {
		return err
	}
	result := m.call("foundling_register", api.FoundlingRegisterInput{FoundlingID: r.FoundlingID, Name: r.Name, Description: r.Description, Source: *r.Source, Pin: preview.Pin, Root: preview.Root, ExpectedConnectionID: expected, Supersedes: []string{r.HeadIDs[0]}, Reason: reason}, true)
	if err := m.outcome("Source pin superseded and connected", result); err != nil {
		return err
	}
	m.foundlingReceipt(result.Result.(api.FoundlingMutationResult))
	return nil
}

func (m *menu) disconnectFoundling() error {
	r, err := m.pickFoundling("Disconnect which reference?")
	if err != nil {
		return err
	}
	if err := activeFoundling(r); err != nil {
		return err
	}
	reason, err := m.input("Why disconnect this reference?", "")
	if err != nil {
		return err
	}
	if reason == "" {
		return errors.New("disconnection requires a reason")
	}
	m.block(console.Block{Title: "Review foundling disconnection", Body: "Append a disconnected registration. Keep source files, local connection, previous registrations and all promoted memory. No deletion or synchronization.", Fields: []console.Field{{Label: "Reference", Value: r.Name}, {Label: "Stable ID", Value: r.FoundlingID}, {Label: "Revision", Value: r.HeadIDs[0]}, {Label: "Reason", Value: reason}}})
	if err := m.confirm(); err != nil {
		return err
	}
	v := m.call("foundling_disconnect", api.FoundlingDisconnectInput{FoundlingID: r.FoundlingID, RegistrationID: r.HeadIDs[0], Reason: reason}, true)
	return m.outcome("Foundling disconnected (history preserved)", v)
}
