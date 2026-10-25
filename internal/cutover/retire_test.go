package cutover_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/cutover"
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
	_, err := cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles, func(context.Context, cutover.LegacyConnection) error {
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
	if _, err := cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles, func(context.Context, cutover.LegacyConnection) error { return nil }); err == nil {
		t.Fatal("retire trusted a disable that changed nothing")
	}

	var disabled []cutover.LegacyConnection
	r, err := cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles, func(_ context.Context, c cutover.LegacyConnection) error {
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
	if _, err := cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles, func(context.Context, cutover.LegacyConnection) error {
		t.Fatal("disabled an already disabled connection")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := cutover.Retire(context.Background(), clone, "ftp://memory.example.com", nil, nil); err == nil {
		t.Fatal("retire accepted a non-HTTP server URL")
	}
}

// A failure after one profile's disable reports what was already disabled,
// and a rerun lists that profile as clear.
func TestRetireReportsDisablesBeforeAFailure(t *testing.T) {
	l := testfixture.NewLegacySignet(t, "device-a")
	clone := l.Clones["device-a"].Root
	restoreWrite(t, clone)
	first, second := filepath.Join(t.TempDir(), "first"), filepath.Join(t.TempDir(), "second")
	claudeProfile(t, first, true)
	claudeProfile(t, second, true)
	profiles := []cutover.NativeProfile{{Harness: "claude-code", Home: first, Binary: "/fixture/bin/claude"}, {Harness: "claude-code", Home: second, Binary: "/fixture/bin/claude"}}
	calls := 0
	_, err := cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles, func(_ context.Context, c cutover.LegacyConnection) error {
		if calls++; calls == 2 {
			return errors.New("synthetic failure")
		}
		claudeProfile(t, c.Home, false)
		return nil
	})
	var partial *cutover.PartialError
	if !errors.As(err, &partial) || !strings.Contains(partial.Written, "mandalore@mandalore in ") || !strings.Contains(partial.Written, "first") {
		t.Fatalf("failure after a disable: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(clone, cutover.RetiredNote)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("RETIRED.md written after a failed disable")
	}
	r, err := cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles, func(_ context.Context, c cutover.LegacyConnection) error {
		claudeProfile(t, c.Home, false)
		return nil
	})
	if err != nil || len(r.Disabled) != 1 || len(r.Clear) != 2 || !r.ReadOnly {
		t.Fatalf("rerun: %+v %v", r, err)
	}
}

// A partial retire names every profile with the state its configuration shows
// after the disable, not what the disable command reported.
func TestRetireLabelsEachProfileWithItsRealState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	l := testfixture.NewLegacySignet(t, "device-a")
	clone := l.Clones["device-a"].Root
	restoreWrite(t, clone)
	base := t.TempDir()
	homes := map[string]string{}
	profiles := []cutover.NativeProfile{}
	for _, name := range []string{"works", "noop", "fails", "later"} {
		homes[name] = filepath.Join(base, name)
		claudeProfile(t, homes[name], true)
		profiles = append(profiles, cutover.NativeProfile{Harness: "claude-code", Home: homes[name], Binary: "/fixture/bin/claude"})
	}
	// works disables for real; noop exits 0 but changes nothing.
	disable := func(_ context.Context, c cutover.LegacyConnection) error {
		if strings.HasSuffix(c.Home, "works") {
			claudeProfile(t, c.Home, false)
		}
		return nil
	}
	_, err := cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles, disable)
	var partial *cutover.PartialError
	if !errors.As(err, &partial) {
		t.Fatalf("partial retire: %v", err)
	}
	for _, want := range []string{
		"mandalore@mandalore in " + homes["works"] + ": disabled",
		"mandalore@mandalore in " + homes["noop"] + ": unchanged, still enabled",
		homes["fails"] + ": not reached",
		homes["later"] + ": not reached",
	} {
		if !strings.Contains(partial.Err.Error(), want) {
			t.Errorf("message %q lacks %q", partial.Err, want)
		}
	}
	if !strings.Contains(partial.Written, "works") || strings.Contains(partial.Written, "noop") {
		t.Errorf("written %q must list only the real disable", partial.Written)
	}

	// Now noop works, and fails' disable errors without changing anything.
	_, err = cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles, func(_ context.Context, c cutover.LegacyConnection) error {
		if strings.HasSuffix(c.Home, "fails") {
			return errors.New("synthetic failure")
		}
		claudeProfile(t, c.Home, false)
		return nil
	})
	if !errors.As(err, &partial) {
		t.Fatalf("second partial retire: %v", err)
	}
	for _, want := range []string{
		homes["works"] + ": already clear",
		"mandalore@mandalore in " + homes["noop"] + ": disabled",
		"mandalore@mandalore in " + homes["fails"] + ": failed, still enabled",
		homes["later"] + ": not reached",
		"synthetic failure",
	} {
		if !strings.Contains(partial.Err.Error(), want) {
			t.Errorf("message %q lacks %q", partial.Err, want)
		}
	}
	if _, err := os.Lstat(filepath.Join(clone, cutover.RetiredNote)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("RETIRED.md written after a partial retire")
	}

	// Nothing changes at all: a refusal that still names the enabled profile.
	_, err = cutover.Retire(context.Background(), clone, "https://memory.example.com", profiles[2:3], func(context.Context, cutover.LegacyConnection) error { return nil })
	if err == nil || errors.As(err, &partial) || !strings.Contains(err.Error(), homes["fails"]+": unchanged, still enabled") {
		t.Fatalf("no-op retire: %v", err)
	}
}
