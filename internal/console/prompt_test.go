package console

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSelectorKeyboardAndCancellation(t *testing.T) {
	m := newSelector("Signets", []string{"First", "Second", "Back"}, 0)
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = next.(promptModel)
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(promptModel)
	if !m.done || m.selected != 1 || m.cancelled {
		t.Fatalf("%+v", m)
	}
	m = newSelector("Signets", []string{"First", "Back"}, 0)
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = next.(promptModel)
	if !m.cancelled {
		t.Fatal("escape selected an action")
	}
}

func TestPromptEditingPasteAndResize(t *testing.T) {
	m := newInput("Name", "default")
	next, _ := m.Update(tea.PasteMsg{Content: "café\n\x1b[31m"})
	m = next.(promptModel)
	if strings.ContainsAny(string(m.text), "\n\x1b") {
		t.Fatal("pasted control characters")
	}
	next, _ = m.Update(tea.WindowSizeMsg{Width: 24, Height: 5})
	m = next.(promptModel)
	if strings.Contains(m.View().Content, "\x1b[31m") {
		t.Fatal("terminal injection")
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(promptModel)
	if !m.done {
		t.Fatal("input not submitted")
	}
	empty := newInput("Name", "default")
	next, _ = empty.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(promptModel).value != "default" {
		t.Fatal("lost default")
	}
}

func TestVimMenuBindingsDoNotStealTextInput(t *testing.T) {
	m := newSelector("Menu", []string{"one", "two", "three"}, 0)
	send := func(text string) {
		next, _ := m.Update(tea.KeyPressMsg{Code: []rune(text)[0], Text: text})
		m = next.(promptModel)
	}
	send("j")
	if m.selected != 1 {
		t.Fatal("j did not move down")
	}
	send("k")
	if m.selected != 0 {
		t.Fatal("k did not move up")
	}
	send("G")
	if m.selected != 2 {
		t.Fatal("G did not select last")
	}
	send("g")
	send("g")
	if m.selected != 0 {
		t.Fatal("gg did not select first")
	}
	send("l")
	if !m.done || m.cancelled {
		t.Fatal("l did not open")
	}
	m = newSelector("Menu", []string{"one"}, 0)
	send("h")
	if !m.cancelled {
		t.Fatal("h did not go back")
	}
	m = newInput("Name", "")
	for _, r := range "hjklgG" {
		send(string(r))
	}
	if string(m.text) != "hjklgG" {
		t.Fatal("vim keys stole field input")
	}
}
