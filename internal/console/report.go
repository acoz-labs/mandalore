package console

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/rivo/uniseg"
)

type Field struct{ Label, Value string }
type Block struct {
	Title, Body string
	Fields      []Field
	Choices     []string
	Tone        Tone
}

// Measure for each report, so resizing between operations is respected. Cap
// prose width for readability; pipes use a predictable, unstyled 80 columns.
func ReportWidth(out io.Writer) int {
	if f, ok := out.(*os.File); ok {
		if w, _, err := term.GetSize(f.Fd()); err == nil && w > 1 {
			return min(w-1, 100)
		}
	}
	return 80
}

func RenderBlock(b Block, theme Theme, width int) string {
	width = max(8, width)
	var lines []string
	appendText := func(text string, indent int, tone Tone) {
		for _, line := range strings.Split(ansi.Wrap(text, width-indent, ""), "\n") {
			lines = append(lines, strings.Repeat(" ", indent)+theme.Text(tone, line))
		}
	}
	lines = append(lines, "")
	appendText(clean(b.Title), 0, Heading)
	if b.Body != "" {
		appendText(clean(b.Body), 2, b.Tone)
	}
	for n, choice := range b.Choices {
		appendText(fmt.Sprintf("%d. %s", n+1, clean(choice)), 2, Plain)
	}
	labelWidth := 0
	for _, f := range b.Fields {
		labelWidth = max(labelWidth, uniseg.StringWidth(clean(f.Label))+1)
	}
	for _, f := range b.Fields {
		label, value := clean(f.Label)+":", clean(f.Value)
		padding := strings.Repeat(" ", labelWidth-uniseg.StringWidth(label)+1)
		if 2+labelWidth+1+uniseg.StringWidth(value) <= width {
			lines = append(lines, "  "+theme.Text(Muted, label+padding)+value)
		} else {
			appendText(label, 2, Muted)
			// Paths and commands are literal data: preserve spaces while wrapping,
			// rather than treating their contents as prose or truncating them.
			for _, line := range strings.Split(ansi.Hardwrap(value, width-4, true), "\n") {
				lines = append(lines, "    "+line)
			}
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func (c *Console) Block(b Block) error {
	_, err := fmt.Fprint(c.output, RenderBlock(b, c.theme, ReportWidth(c.output)))
	return err
}
