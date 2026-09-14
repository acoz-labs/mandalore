// Package console is presentation only. It cannot run agent operations.
package console

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/rivo/uniseg"
)

var ErrBack = errors.New("back")

type Console struct {
	input, output *os.File
	theme         Theme
	subtitle      string
	ctx           context.Context
}

// Keep operation summaries readable without allowing data-derived terminal
// control sequences. The TUI renderer itself writes to the original terminal.
type SafeWriter struct{ Output io.Writer }

func (w SafeWriter) Write(data []byte) (int, error) {
	value := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, string(data))
	_, err := io.WriteString(w.Output, value)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func Available(input io.Reader, output io.Writer) bool {
	in, ok := input.(*os.File)
	if !ok {
		return false
	}
	out, ok := output.(*os.File)
	return ok && term.IsTerminal(in.Fd()) && term.IsTerminal(out.Fd()) && os.Getenv("TERM") != "dumb" && os.Getenv("TERM") != "" && os.Getenv("MANDALORE_PLAIN") != "1"
}
func New(input io.Reader, output io.Writer) *Console {
	if !Available(input, output) {
		return nil
	}
	return &Console{input: input.(*os.File), output: output.(*os.File), theme: terminalTheme(), ctx: context.Background()}
}
func (c *Console) WithCancellation(ctx context.Context) *Console {
	copy := *c
	copy.ctx = ctx
	return &copy
}
func (c *Console) run(m promptModel) (promptModel, error) {
	m.theme, m.subtitle = c.theme, c.subtitle
	result, err := tea.NewProgram(m, tea.WithInput(c.input), tea.WithOutput(c.output), tea.WithContext(c.ctx)).Run()
	if errors.Is(err, tea.ErrInterrupted) {
		return m, io.EOF
	}
	if err != nil {
		return m, err
	}
	m = result.(promptModel)
	// SIGTERM can end a Bubble Tea program with no error and an unfinished
	// model. Never interpret that as selecting the highlighted action.
	if !m.done || m.quit {
		return m, io.EOF
	}
	if m.cancelled {
		return m, ErrBack
	}
	return m, nil
}
func (c *Console) Select(title string, items []string, selected int) (int, error) {
	m, err := c.run(newSelector(title, items, selected))
	return m.selected, err
}
func (c *Console) Input(title, def string) (string, error) {
	m, err := c.run(newInput(title, def))
	return m.value, err
}

type promptModel struct {
	title                        string
	items                        []string
	selected                     int
	text                         []rune
	cursor                       int
	def, value                   string
	input, done, cancelled, quit bool
	width, height                int
	goPending                    bool
	theme                        Theme
	subtitle                     string
}

func newSelector(title string, items []string, selected int) promptModel {
	return promptModel{title: title, items: items, selected: selected, width: 80, height: 24}
}
func newInput(title, def string) promptModel {
	return promptModel{title: title, def: def, input: true, width: 80, height: 24}
}
func (m promptModel) Init() tea.Cmd { return nil }

func clean(text string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
}
func (m *promptModel) insert(text string) {
	runes := []rune(clean(text))
	remaining := 4096 - len(m.text)
	if len(runes) > remaining {
		runes = runes[:remaining]
	}
	tail := append([]rune{}, m.text[m.cursor:]...)
	m.text = append(append(m.text[:m.cursor], runes...), tail...)
	m.cursor += len(runes)
}
func (m promptModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = max(8, msg.Width)
		}
		if msg.Height > 0 {
			m.height = max(4, msg.Height)
		}
	case tea.PasteMsg:
		if m.input {
			m.insert(msg.Content)
		}
	case tea.KeyPressMsg:
		key := msg.Keystroke()
		if key == "ctrl+c" || key == "ctrl+d" {
			m.quit = true
			m.done = true
			return m, tea.Quit
		}
		if msg.Code == tea.KeyEscape {
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		}
		if msg.Code == tea.KeyEnter {
			m.done = true
			if m.input {
				m.value = strings.TrimSpace(string(m.text))
				if m.value == "" {
					m.value = m.def
				}
			}
			return m, tea.Quit
		}
		if !m.input {
			if key == "h" {
				m.cancelled = true
				m.done = true
				return m, tea.Quit
			}
			if key == "l" {
				m.done = true
				return m, tea.Quit
			}
			if key == "g" {
				if m.goPending {
					m.selected = 0
					m.goPending = false
				} else {
					m.goPending = true
				}
				return m, nil
			}
			m.goPending = false
			if key == "G" || msg.Text == "G" {
				m.selected = len(m.items) - 1
				return m, nil
			}
			switch key {
			case "up", "k", "shift+tab":
				m.selected = max(0, m.selected-1)
			case "down", "j", "tab":
				m.selected = min(len(m.items)-1, m.selected+1)
			case "home":
				m.selected = 0
			case "end":
				m.selected = len(m.items) - 1
			case "q":
				m.quit = true
				m.done = true
				return m, tea.Quit
			}
		} else {
			switch key {
			case "left":
				m.cursor = max(0, m.cursor-1)
			case "right":
				m.cursor = min(len(m.text), m.cursor+1)
			case "home", "ctrl+a":
				m.cursor = 0
			case "end", "ctrl+e":
				m.cursor = len(m.text)
			case "ctrl+u":
				m.text = nil
				m.cursor = 0
			case "backspace":
				if m.cursor > 0 {
					m.text = append(m.text[:m.cursor-1], m.text[m.cursor:]...)
					m.cursor--
				}
			case "delete":
				if m.cursor < len(m.text) {
					m.text = append(m.text[:m.cursor], m.text[m.cursor+1:]...)
				}
			case "tab":
				if len(m.text) == 0 {
					m.insert(m.def)
				}
			default:
				if msg.Text != "" && msg.Mod&tea.ModCtrl == 0 && msg.Mod&tea.ModAlt == 0 {
					m.insert(msg.Text)
				}
			}
		}
	}
	return m, nil
}
func fit(text string, width int) string {
	text = clean(text)
	if uniseg.StringWidth(text) <= width {
		return text
	}
	var b strings.Builder
	g := uniseg.NewGraphemes(text)
	used := 0
	for g.Next() {
		piece := g.Str()
		w := uniseg.StringWidth(piece)
		if used+w > max(0, width-1) {
			break
		}
		b.WriteString(piece)
		used += w
	}
	return b.String() + "…"
}
func (m promptModel) View() tea.View {
	if m.done {
		if m.quit || m.cancelled {
			return tea.NewView("")
		}
		if m.input {
			return tea.NewView(m.theme.Text(Muted, fit(m.title+": "+m.value, m.width)) + "\n")
		}
		return tea.NewView(m.theme.Text(Muted, fit(m.title+" › "+m.items[m.selected], m.width)) + "\n")
	}
	width := max(8, m.width-2)
	lines := []string{m.theme.Text(Heading, fit(m.title, width))}
	if m.subtitle != "" {
		lines = append(lines, m.theme.Text(Muted, fit(m.subtitle, width)))
	}
	lines = append(lines, "")
	if m.input {
		before := string(m.text[:m.cursor])
		after := string(m.text[m.cursor:])
		for uniseg.StringWidth(before) > max(1, width/2) {
			_, size := firstRune(before)
			before = before[size:]
		}
		lines = append(lines, m.theme.Text(Accent, fit("› "+before+"│"+after, width)))
		if m.def != "" {
			lines = append(lines, m.theme.Text(Muted, fit("Default: "+m.def, width)))
		}
		lines = append(lines, m.theme.Text(Muted, fit("Enter save · Tab edit default · Esc back · Ctrl+C exit", width)))
	} else {
		count := max(1, m.height-len(lines)-4)
		start := max(0, m.selected-count+1)
		end := min(len(m.items), start+count)
		for n := start; n < end; n++ {
			mark := "  "
			if n == m.selected {
				mark = "› "
			}
			line := fit(mark+m.items[n], width)
			if n == m.selected {
				line = m.theme.Text(Accent, line)
			}
			lines = append(lines, line)
		}
		if end-start < len(m.items) {
			lines = append(lines, m.theme.Text(Muted, fit(fmt.Sprintf("%d of %d", m.selected+1, len(m.items)), width)))
		}
		lines = append(lines, "", m.theme.Text(Muted, fit("↑/↓ or j/k move · Enter/l select · Esc/h back · gg/G ends", width)))
	}
	return tea.NewView(strings.Join(lines, "\n") + "\n")
}
func firstRune(text string) (rune, int) {
	for _, r := range text {
		return r, len(string(r))
	}
	return 0, 0
}
