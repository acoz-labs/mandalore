package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryInventoryOnlyListsAndDoesNotClaimActiveSessions(t *testing.T) {
	home := t.TempDir()
	binary := filepath.Join(t.TempDir(), "native")
	if err := os.WriteFile(binary, []byte("synthetic selected executable"), 0700); err != nil {
		t.Fatal(err)
	}
	calls := []string{}
	r, err := inspectMemoryPlugins(context.Background(), home, binary, func(_ context.Context, _ Options, args ...string) ([]byte, error) {
		command := strings.Join(args, " ")
		calls = append(calls, command)
		switch command {
		case "plugin marketplace list --json":
			return []byte(`{"marketplaces":[]}`), nil
		case "plugin list --json":
			return []byte(`{"installed":[{"pluginId":"old@local","name":"my-friday-memory","version":"1","enabled":true},{"pluginId":"new@local","name":"mandalore","version":"2","enabled":true},{"pluginId":"disabled@local","name":"my-friday","version":"0","enabled":false},{"pluginId":"unrelated@local","name":"unrelated","enabled":true}]}`), nil
		default:
			t.Fatal("mutating native command", command)
			return nil, nil
		}
	})
	if err != nil || r.Status != "inspected" || r.PotentialWriters != 2 || len(r.Plugins) != 3 || len(calls) != 2 || !strings.Contains(r.Notice, "not tested") {
		t.Fatal(r, err, calls)
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 0 {
		t.Fatal("inventory adapter wrote profile", entries, err)
	}
}
