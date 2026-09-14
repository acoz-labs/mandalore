package releaseworkflow

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Extract one literal run block to execute the actual receipt code. This is not
// a YAML validator; workflow syntax/structure is checked separately with a parser.
func workflowRunBody(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("../../.github/workflows/release-artifact.yml")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(b), "\n")
	selected, body := false, false
	var out []string
	for _, line := range lines {
		if strings.HasPrefix(line, "      - name: ") {
			if selected {
				break
			}
			selected = strings.TrimPrefix(line, "      - name: ") == name
		}
		if !selected {
			continue
		}
		if line == "        run: |" {
			body = true
			continue
		}
		if body {
			if strings.TrimSpace(line) == "" {
				out = append(out, "")
				continue
			}
			if !strings.HasPrefix(line, "          ") {
				break
			}
			out = append(out, strings.TrimPrefix(line, "          "))
		}
	}
	if len(out) == 0 {
		t.Fatal("literal workflow run block not found", name)
	}
	return strings.Join(out, "\n")
}

func TestPublicationRecoveryReceiptRedactsPathsAndHandlesInterruptedOutput(t *testing.T) {
	for _, kind := range []string{"published", "ledger complete", "failed publish", "missing", "empty", "truncated", "scalar"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			raw := `{"phase":"publication-verified","staging_directory":"/synthetic/private-path","publication":{"release_id":42}}`
			promote, finalize := "success", "skipped"
			want := "publication-verified"
			switch kind {
			case "ledger complete":
				finalize = "success"
				want = "ledger-complete"
			case "failed publish":
				promote = "failure"
				raw = `{"phase":"authorized","staging_directory":"/synthetic/private-path","publication":{"pending_operation":"publish-draft"}}`
				want = "authorized"
			case "missing", "empty", "truncated", "scalar":
				promote = "cancelled"
				want = "unconfirmed"
			}
			switch kind {
			case "empty":
				raw = ""
			case "truncated":
				raw = `{"staging_directory":"/synthetic/private-path"`
			case "scalar":
				raw = `"/synthetic/private-path"`
			}
			if kind != "missing" {
				if err := os.WriteFile(filepath.Join(dir, "mandalore-promotion-local.json"), []byte(raw), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", "-c", workflowRunBody(t, "Prepare redacted recovery receipt"))
			cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "RUNNER_TEMP=" + dir, "PROMOTE_OUTCOME=" + promote, "FINALIZE_OUTCOME=" + finalize}
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatal(err, string(out))
			}
			b, err := os.ReadFile(filepath.Join(dir, "mandalore-promotion.json"))
			if err != nil {
				t.Fatal(err)
			}
			var result map[string]any
			if err := json.Unmarshal(b, &result); err != nil {
				t.Fatal(err)
			}
			if result["phase"] != want || strings.Contains(string(b), "private-path") || strings.Contains(string(b), "staging_directory") {
				t.Fatal("incorrect or unredacted recovery evidence", string(b))
			}
			if kind == "failed publish" && !strings.Contains(string(b), "publish-draft") {
				t.Fatal("uncertain remote effects lost")
			}
		})
	}
}
