package console

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

func TestReportAlignmentAndNarrowWrapping(t *testing.T) {
	path := "/very/long/project/directory/with-important-details/agent.json"
	b := Block{Title: "Storage", Body: "These are local paths, not proof of a remote backup.", Fields: []Field{{"Source", path}, {"Memory", "/memory"}}}
	for _, width := range []int{8, 10, 12, 24, 40, 100} {
		plain := RenderBlock(b, Theme{}, width)
		colored := RenderBlock(b, Theme{Color: true}, width)
		if ansi.Strip(colored) != plain {
			t.Fatal("styling changed report text")
		}
		for _, line := range strings.Split(plain, "\n") {
			if uniseg.StringWidth(line) > width {
				t.Fatalf("width %d: %q", width, line)
			}
		}
		// Wrapping must not truncate path characters or substitute an ellipsis.
		compact := strings.NewReplacer("\n", "", " ", "").Replace(plain)
		if !strings.Contains(compact, path) {
			t.Fatal("lost path data")
		}
	}
	if text := RenderBlock(Block{Title: "Signet", Fields: []Field{{"Name", "pilot"}, {"Harness", "codex"}}}, Theme{}, 80); !strings.Contains(text, "Name:    pilot") || !strings.Contains(text, "Harness: codex") {
		t.Fatal(text)
	}
}

func TestReportSanitizesAndWrapsUnicode(t *testing.T) {
	b := Block{Title: "Health\x1b]52;bad\a", Body: strings.Repeat("界", 30), Fields: []Field{{"Name", "café\nforged heading\x1b[31m"}}}
	text := RenderBlock(b, Theme{}, 24)
	if strings.Contains(text, "\x1b") || strings.Contains(text, "\nforged") {
		t.Fatal("untrusted control escaped")
	}
	for _, line := range strings.Split(text, "\n") {
		if uniseg.StringWidth(line) > 24 {
			t.Fatal(line)
		}
	}
}

func TestLiteralFieldSpacesSurviveWrapping(t *testing.T) {
	value := "/a directory/two  spaces/" + strings.Repeat("z", 50)
	text := RenderBlock(Block{Title: "Storage", Fields: []Field{{"Path", value}}}, Theme{}, 24)
	var recovered strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "    ") {
			recovered.WriteString(strings.TrimPrefix(line, "    "))
		}
	}
	if recovered.String() != value {
		t.Fatalf("literal changed: %q", recovered.String())
	}
}
