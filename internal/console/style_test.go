package console

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

func TestThemePreservesTextAndSanitizesBeforeStyling(t *testing.T) {
	for _, tone := range []Tone{Heading, Accent, Muted, Success, Warning, Failure} {
		plain := Theme{}.Text(tone, "Healthy: local checks passed")
		styled := Theme{Color: true}.Text(tone, plain)
		if !strings.Contains(styled, "\x1b[") || ansi.Strip(styled) != plain {
			t.Fatalf("styling changed text: %q", styled)
		}
		if strings.Contains(Theme{Color: true}.Text(tone, "bad\x1b]52;c;payload\a"), "\x1b]52") {
			t.Fatal("untrusted terminal command survived")
		}
	}
}

func TestThemeHonorsNoColorAndDumbTerminal(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "")
	if !terminalTheme().Color {
		t.Fatal("normal terminal lacks color")
	}
	t.Setenv("NO_COLOR", "1")
	if terminalTheme().Color {
		t.Fatal("NO_COLOR ignored")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if terminalTheme().Color {
		t.Fatal("dumb terminal got color")
	}
}

func TestStyledViewsKeepWidthAndMonochromeNavigation(t *testing.T) {
	m := newSelector("Signet", []string{"First", "Second", "Back"}, 1)
	m.subtitle = "codex · pinned: abc1234"
	m.width = 24
	plain := m.View().Content
	m.theme = Theme{Color: true}
	styled := m.View().Content
	if ansi.Strip(styled) != plain || !strings.Contains(plain, "› Second") {
		t.Fatal("color changed navigation labels")
	}
	for _, line := range strings.Split(ansi.Strip(styled), "\n") {
		if uniseg.StringWidth(line) > m.width {
			t.Fatalf("overflow: %q", line)
		}
	}
	m = newInput("A long field heading", "a long field value")
	m.width = 16
	m.done, m.value = true, "a long field value"
	for _, line := range strings.Split(m.View().Content, "\n") {
		if uniseg.StringWidth(line) > m.width {
			t.Fatalf("submitted field overflow: %q", line)
		}
	}
}
