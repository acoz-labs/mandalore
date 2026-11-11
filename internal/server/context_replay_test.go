package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	signetsync "github.com/acoz-labs/mandalore/internal/sync"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

// TestWriteEvaluationSignet writes the synthetic evaluation set as a served
// signet for the hook replay (docs/evidence/2.5/hook-replay.py): a checkout
// on main in MANDALORE_EVAL_SIGNET_DIR/signet, an empty bare remote beside it,
// and signet.json with what a server config needs.
func TestWriteEvaluationSignet(t *testing.T) {
	dir := os.Getenv("MANDALORE_EVAL_SIGNET_DIR")
	if dir == "" {
		t.Skip("set MANDALORE_EVAL_SIGNET_DIR to write the evaluation signet for the hook replay")
	}
	root := filepath.Join(dir, "signet")
	m, _ := testfixture.WriteEvaluation(t, "testdata/context-eval.json", root)
	g, err := signetsync.Open(root, m.ID())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	remote := filepath.Join(dir, "remote.git")
	testfixture.Git(t, dir, "init", "--bare", "--initial-branch=main", remote)
	out, _ := json.Marshal(map[string]string{"root": root, "signet_id": m.ID(), "device_id": "device-evaluation", "remote": "file://" + remote})
	if err := os.WriteFile(filepath.Join(dir, "signet.json"), out, 0o600); err != nil {
		t.Fatal(err)
	}
}
