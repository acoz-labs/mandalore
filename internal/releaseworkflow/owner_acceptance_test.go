package releaseworkflow

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const ownerCandidate = "5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f"
const ownerArtifact = "mandalore:" + ownerCandidate + ":sha256:c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237"

func TestOwnerAcceptanceScope(t *testing.T) {
	for _, scenario := range []string{"exact", "case-insensitive owner", "no owner", "wrong owner", "no confirmation", "false confirmation", "wrong repo", "wrong sha", "wrong artifact", "wrong issue", "leading-zero issue"} {
		t.Run(scenario, func(t *testing.T) {
			args := []string{"../../bin/mvp-owner-acceptance", "acoz-labs/mandalore", ownerCandidate, "10", ownerArtifact, "example-owner"}
			owner, confirmation := "example-owner", "true"
			switch scenario {
			case "case-insensitive owner":
				owner = "Example-Owner"
			case "no owner":
				owner = ""
			case "wrong owner":
				owner = "another-owner"
			case "no confirmation":
				confirmation = ""
			case "false confirmation":
				confirmation = "false"
			case "wrong repo":
				args[1] = "example/another"
			case "wrong sha":
				args[2] = strings.Repeat("a", 40)
			case "wrong artifact":
				args[4] += "changed"
			case "wrong issue":
				args[3] = "42"
			case "leading-zero issue":
				args[3] = "010"
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", args...)
			cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "MVP_ACCEPTANCE_OWNER=" + owner, "MVP_OWNER_REVIEW_CONFIRMED=" + confirmation}
			out, err := cmd.CombinedOutput()
			want := scenario == "exact" || scenario == "case-insensitive owner"
			if (err == nil) != want {
				t.Fatalf("eligibility=%v want %v: %s", err, want, out)
			}
		})
	}
}

func TestOwnerAcceptanceRecorder(t *testing.T) {
	for _, scenario := range []string{"owner candidate author", "owner linked author", "no confirmation", "linked no confirmation", "unlisted owner", "normal independent", "wrong nomination", "gate failed"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, "bin"), 0700); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"record-product-acceptance", "mvp-owner-acceptance"} {
				b, err := os.ReadFile("../../bin/" + name)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "bin", name), b, 0700); err != nil {
					t.Fatal(err)
				}
			}
			fakeGH := `#!/usr/bin/env bash
set -eu
case "$*" in
  *'commits/'*'/pulls?'*) printf '[{"merged_at":"2026-09-14","user":{"login":"%s"}}]\n' "$TEST_CANDIDATE_AUTHOR" ;;
  'api repos/acoz-labs/mandalore/issues/10') printf '%s\n' '{"state":"open","labels":[{"name":"delivery:acceptance"}],"body":"- Phase 3 — Implementation: https://github.com/acoz-labs/mandalore/pull/26"}' ;;
  'api repos/acoz-labs/mandalore/pulls/26') printf '{"merged_at":"2026-09-14","merge_commit_sha":"%s","user":{"login":"example-owner"}}\n' "$TEST_SHA" ;;
  'api repos/acoz-labs/mandalore/compare/'*) printf '%s\n' '{"status":"identical"}' ;;
  *'issues/10/comments?'*) printf '[[{"user":{"login":"github-actions[bot]"},"body":"<!-- release-candidate sha=%s artifact=%s source=artifact source-id=123 intent=application -->"}]]\n' "$TEST_SHA" "$TEST_ARTIFACT_DIGEST" ;;
  'label create '*|'issue comment '*|'issue edit '*|'api -X '* ) printf '%s\n' "$*" >> "$TEST_LOG"; printf '{}\n' ;;
  *) printf 'Unexpected test call: %s\n' "$*" >&2; exit 3 ;;
esac
`
			if err := os.WriteFile(filepath.Join(dir, "bin/gh"), []byte(fakeGH), 0700); err != nil {
				t.Fatal(err)
			}
			gate := "#!/bin/sh\ntest \"$TEST_GATE_FAIL\" != true\n"
			if err := os.WriteFile(filepath.Join(dir, "bin/release-gate"), []byte(gate), 0700); err != nil {
				t.Fatal(err)
			}
			actor, allowed, confirmation, author := "example-owner", "example-owner", "true", "example-owner"
			digest, gateFail := blob(ownerArtifact), "false"
			switch scenario {
			case "owner linked author":
				author = "example-contributor"
			case "no confirmation":
				confirmation = "false"
			case "linked no confirmation":
				author, confirmation = "example-contributor", "false"
			case "unlisted owner":
				allowed = "example-reviewer"
			case "normal independent":
				actor, allowed, confirmation = "example-reviewer", "example-reviewer", "false"
			case "wrong nomination":
				digest = strings.Repeat("a", 40)
			case "gate failed":
				gateFail = "true"
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", "bin/record-product-acceptance", ownerCandidate, "10", "approved", "Personally reviewed synthetic scenarios", "https://example.invalid/evidence")
			cmd.Dir = dir
			cmd.Env = []string{"PATH=" + filepath.Join(dir, "bin") + ":" + os.Getenv("PATH"), "GITHUB_REPOSITORY=acoz-labs/mandalore", "GITHUB_ACTOR=" + actor, "GITHUB_RUN_ID=123", "ACCEPTANCE_ACTORS=" + allowed, "MVP_ACCEPTANCE_OWNER=example-owner", "MVP_OWNER_REVIEW_CONFIRMED=" + confirmation, "RELEASE_ARTIFACT=" + ownerArtifact, "TEST_CANDIDATE_AUTHOR=" + author, "TEST_SHA=" + ownerCandidate, "TEST_ARTIFACT_DIGEST=" + digest, "TEST_LOG=" + filepath.Join(dir, "calls"), "TEST_GATE_FAIL=" + gateFail}
			out, err := cmd.CombinedOutput()
			want := scenario == "owner candidate author" || scenario == "owner linked author" || scenario == "normal independent"
			if (err == nil) != want {
				t.Fatalf("recorder=%v want success=%v: %s", err, want, out)
			}
			calls, readErr := os.ReadFile(filepath.Join(dir, "calls"))
			if !want {
				if !os.IsNotExist(readErr) {
					t.Fatal("refused recording made external mutations", string(calls), readErr)
				}
				return
			}
			if readErr != nil || strings.Count(string(calls), "api -X POST repos/acoz-labs/mandalore/statuses/") != 4 {
				t.Fatal("missing acceptance status writes", string(calls), readErr)
			}
			exception := strings.Contains(string(calls), "Owner acceptance under the candidate-specific exception")
			if exception != (scenario != "normal independent") {
				t.Fatal("incorrect acceptance authority disclosure", string(calls))
			}
		})
	}
}
