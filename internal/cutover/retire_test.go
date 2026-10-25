package cutover_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/acoz-labs/mandalore/internal/cutover"
	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

// claudeProfile is a synthetic Claude Code profile with the 1.x Mandalore
// plugin registered and enabled or disabled.
func claudeProfile(t *testing.T, home string, enabled bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(home, "plugins"), 0700); err != nil {
		t.Fatal(err)
	}
	settings := `{"enabledPlugins":{"mandalore@mandalore":false}}`
	if enabled {
		settings = `{"enabledPlugins":{"mandalore@mandalore":true}}`
	}
	for name, body := range map[string]string{
		"settings.json":                   settings,
		"plugins/known_marketplaces.json": `{"mandalore":{"source":{"source":"directory","path":"/fixture/state/claude-code/connections/one"}}}`,
	} {
		if err := os.WriteFile(filepath.Join(home, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func restoreWrite(t *testing.T, root string) {
	t.Cleanup(func() {
		_ = filepath.WalkDir(root, func(path string, _ os.DirEntry, err error) error {
			if err == nil {
				_ = os.Chmod(path, 0700)
			}
			return nil
		})
	})
}

func TestRetireDisablesDetectedConnectionsFirst(t *testing.T) {
	l := testfixture.NewLegacySignet(t, "device-a")
	clone := l.Clones["device-a"].Root
	restoreWrite(t, clone)
	home := filepath.Join(t.TempDir(), "claude")
	claudeProfile(t, home, true)
	profiles := []cutover.NativeProfile{{Harness: "claude-code", Home: home, Binary: "/fixture/bin/claude"}}

	// A failed disable leaves the clone as it was.
	_, err := cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles, func(context.Context, install.LegacyConnection) error {
		return errors.New("synthetic failure")
	})
	if err == nil {
		t.Fatal("retire continued after a failed disable")
	}
	if _, err := os.Lstat(filepath.Join(clone, cutover.RetiredNote)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("retire wrote RETIRED.md after a failed disable")
	}
	if info, _ := os.Stat(clone); info.Mode().Perm()&0200 == 0 {
		t.Fatal("retire made the clone read-only after a failed disable")
	}

	// A disable that reports success but leaves the plugin enabled is caught.
	if _, err := cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles, func(context.Context, install.LegacyConnection) error { return nil }); err == nil {
		t.Fatal("retire trusted a disable that changed nothing")
	}

	var disabled []install.LegacyConnection
	r, err := cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles, func(_ context.Context, c install.LegacyConnection) error {
		disabled = append(disabled, c)
		claudeProfile(t, home, false) // what `claude plugin disable` does
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(disabled) != 1 || disabled[0].Target != "mandalore@mandalore" || len(r.Disabled) != 1 || !r.ReadOnly {
		t.Fatalf("disabled %v, result %+v", disabled, r)
	}
	// Rerunning finishes or confirms the retirement without disabling again.
	if _, err := cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles, func(context.Context, install.LegacyConnection) error {
		t.Fatal("disabled an already disabled connection")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := cutover.Retire(context.Background(), clone, "ftp://memory.example.com", nil, nil); err == nil {
		t.Fatal("retire accepted a non-HTTP server URL")
	}
}
