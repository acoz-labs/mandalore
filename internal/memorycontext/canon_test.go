package memorycontext

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/sessionsync"
)

func TestCanonOrientationProvidesHarnessScopedSelectionWithoutFetchingOnTurns(t *testing.T) {
	root := filepath.Join(t.TempDir(), "primary")
	if _, err := memory.Create(root, "Synthetic", "device-test", "Fixture"); err != nil {
		t.Fatal(err)
	}
	author := memory.Authorship{DeviceID: "device-test", Actor: "Synthetic", Harness: "codex"}
	svc, err := memory.OpenService(root, author)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.WriteFoundling(memory.FoundlingWrite{Mode: "canon", Branch: "main", SourceSignetID: "signet-work", Name: "Work conventions", Description: "Synthetic work evidence", Source: memory.FoundlingSource{Kind: "git", Locator: "https://example.invalid/work.git"}, Pin: memory.SourcePin{Algorithm: "git-sha1", Value: strings.Repeat("a", 40)}, State: "active", Reason: "Explicit reference"}); err != nil {
		t.Fatal(err)
	}
	boundary := sessionsync.Boundary{Kind: "turn", SessionID: "same-native-id"}
	first := CanonOrientation(context.Background(), svc, boundary)
	if !strings.Contains(first, `"codex:same-native-id"`) || !strings.Contains(first, "unavailable") {
		t.Fatal(first)
	}
	author.Harness = "claude-code"
	other, err := memory.OpenService(root, author)
	if err != nil {
		t.Fatal(err)
	}
	second := CanonOrientation(context.Background(), other, boundary)
	if !strings.Contains(second, `"claude-code:same-native-id"`) || strings.Contains(second, `"codex:same-native-id"`) {
		t.Fatal(second)
	}
	boundary.Kind = "compact"
	compact := CanonOrientation(context.Background(), svc, boundary)
	if compact != first {
		t.Fatal("compact changed snapshot behavior", compact, first)
	}
}
