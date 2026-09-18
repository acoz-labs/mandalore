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

func TestPlainChoicesWrapAsProseNotLiteralPaths(t *testing.T) {
	text := RenderBlock(Block{Title: "Choose", Choices: []string{"Connect an existing local clone", "Inspect selected memory and sync status"}}, Theme{}, 24)
	for _, word := range []string{"existing", "Inspect", "selected", "memory", "status"} {
		if !strings.Contains(text, word) {
			t.Fatalf("split prose word %q: %s", word, text)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if uniseg.StringWidth(line) > 24 {
			t.Fatal("overflow", line)
		}
	}
}

func TestCommandRemainsOneUnstyledLogicalLine(t *testing.T) {
	command := "mandalore release apply < '/synthetic/owner'\"'\"'s two  spaces/" + strings.Repeat("long-", 25) + "pending.json'"
	for _, width := range []int{24, 32, 80} {
		for _, colored := range []bool{false, true} {
			got := RenderBlock(Block{Title: "Recovery", Body: "Inspect the plan first.", Command: command}, Theme{Color: colored}, width)
			if strings.Count(got, "\n"+command+"\n") != 1 {
				t.Fatalf("literal command altered at width %d: %q", width, got)
			}
		}
	}
}

func TestUnsafeCommandSuppressedNotRewritten(t *testing.T) {
	for _, bad := range []string{"\n", "\r", "\t", "\x1b[31m", "\x00", "\xff", "\u202e", "\u2028", "\u2029"} {
		got := RenderBlock(Block{Title: "Recovery", Command: "mandalore " + bad + " release apply"}, Theme{}, 32)
		if strings.Contains(got, "mandalore") || !strings.Contains(strings.ReplaceAll(got, "\n", " "), "unavailable") {
			t.Fatalf("unsafe command not refused: %q", got)
		}
	}
	if got := RenderBlock(Block{Title: "Complete"}, Theme{}, 32); strings.Contains(got, "Command") {
		t.Fatal(got)
	}
}
