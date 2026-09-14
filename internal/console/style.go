package console

import (
	"fmt"
	"os"
	"strings"
)

type Tone int

const (
	Heading Tone = iota
	Accent
	Muted
	Success
	Warning
	Failure
	Plain
)

// Use the terminal's own ANSI palette, not fixed RGB backgrounds. Every
// semantic status retains its text label when color is disabled.
type Theme struct{ Color bool }

func terminalTheme() Theme {
	return Theme{Color: os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "" && os.Getenv("TERM") != "dumb"}
}

func (t Theme) Text(tone Tone, text string) string {
	var safe strings.Builder
	_, _ = (SafeWriter{Output: &safe}).Write([]byte(text))
	text = safe.String()
	if !t.Color || tone == Plain {
		return text
	}
	code := "0"
	switch tone {
	case Heading:
		code = "1"
	case Accent:
		code = "1;36"
	case Muted:
		code = "2"
	case Success:
		code = "32"
	case Warning:
		code = "33"
	case Failure:
		code = "31"
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func (c *Console) Println(tone Tone, text string) {
	// Only trusted styling bypasses SafeWriter; data is sanitized in Text first.
	fmt.Fprintln(c.output, c.theme.Text(tone, text))
}

func (c *Console) WithContext(subtitle string) *Console {
	copy := *c
	copy.subtitle = subtitle
	return &copy
}
