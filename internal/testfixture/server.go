package testfixture

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

// ServerSignet is a synthetic signet checkout on main with an empty bare
// file:// remote, both in a new test directory.
type ServerSignet struct {
	Root, SignetID, DeviceID, Remote, RemoteDir string
}

func NewServerSignet(t testing.TB, name string) ServerSignet {
	t.Helper()
	base := t.TempDir()
	store, err := memory.Create(filepath.Join(base, "signet"), "Synthetic "+name, "device-server", "Synthetic server")
	if err != nil {
		t.Fatal(err)
	}
	g, err := signetsync.Open(store.Root, store.Signet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	remote := filepath.Join(base, "remote.git")
	Git(t, base, "init", "--bare", "--initial-branch=main", remote)
	return ServerSignet{Root: store.Root, SignetID: store.Signet.ID, DeviceID: "device-server", Remote: "file://" + remote, RemoteDir: remote}
}

// Git runs git for a fixture with no user or system configuration.
func Git(t testing.TB, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
	env := []string{}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GIT_") {
			env = append(env, e)
		}
	}
	cmd.Env = append(env, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", args[0], err, out)
	}
	return strings.TrimSpace(string(out))
}
