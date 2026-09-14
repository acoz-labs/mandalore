package releaseworkflow

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/distribution"
)

type fixtureState struct {
	SHA, Identity, Tag, Scenario string
	Issue                        map[string]any
	Comments                     []map[string]any
	Statuses                     []map[string]any
	Release                      map[string]any
}

func blob(s string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("blob %d\x00%s", len(s), s))))
}

func fixture(t *testing.T, scenario string) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"git", "jq", "bash"} {
		if _, err := exec.LookPath(name); err != nil {
			t.Fatal("finalizer integration requires", name)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../bin/finalize-release")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin/finalize-release"), raw, 0700); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gh", "release-publication", "record-production-deployment"} {
		wrapper := "#!/bin/sh\nexport MANDALORE_FINALIZER_HELPER=" + name + "\nexec \"$FINALIZER_TEST_EXECUTABLE\" -test.run=^TestFinalizerHelperProcess$ -- \"$@\"\n"
		if err := os.WriteFile(filepath.Join(dir, "bin", name), []byte(wrapper), 0700); err != nil {
			t.Fatal(err)
		}
	}
	env := []string{"PATH=" + filepath.Join(dir, "bin") + ":" + os.Getenv("PATH"), "HOME=" + dir, "TMPDIR=" + dir, "LANG=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GORACE=atexit_sleep_ms=0", "FINALIZER_TEST_EXECUTABLE=" + executable, "FINALIZER_FIXTURE=" + dir, "GITHUB_REPOSITORY=acoz-labs/mandalore", "GITHUB_SERVER_URL=https://github.com", "GITHUB_RUN_ID=123", "RELEASE_ISSUES=11", "RELEASE_VERSION=1.0.0", "RELEASE_SUMMARY=Synthetic release", "RELEASE_ROLLBACK=Synthetic rollback", "SMOKE_SUMMARY=Synthetic checks", "PRODUCTION_DEPLOYMENT_ID=45", "RELEASE_CONTROL_SHA=" + strings.Repeat("c", 40)}
	runGit := func(args ...string) string {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = dir
		cmd.Env = env
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err, string(b))
		}
		return strings.TrimSpace(string(b))
	}
	runGit("init", "-q")
	runGit("-c", "user.name=Example Contributor", "-c", "user.email=contributor@example.invalid", "commit", "--allow-empty", "-qm", "Synthetic source")
	sha := runGit("rev-parse", "HEAD")
	identity := "mandalore:" + sha + ":sha256:" + strings.Repeat("a", 64)
	implementation := blob("36\n")
	artifact := blob(identity)
	body := "<!-- software-lifecycle:start -->\n- Phase 3 — Implementation: https://github.com/acoz-labs/mandalore/pull/36\n- Acceptance: pending\n- Release: pending\n<!-- software-lifecycle:end -->"
	state := fixtureState{SHA: sha, Identity: identity, Tag: "v1.0.0", Scenario: scenario, Issue: map[string]any{"state": "OPEN", "labels": []any{map[string]any{"name": "delivery:ready-for-release"}}, "body": body}}
	for _, text := range []string{
		"<!-- release-candidate sha=" + sha + " artifact=" + artifact + " source=artifact source-id=123 intent=application -->",
		"<!-- product-acceptance issue=11 sha=" + sha + " artifact=" + artifact + " implementation=" + implementation + " acceptor=example-acceptor decision=approved -->",
	} {
		state.Comments = append(state.Comments, map[string]any{"body": text, "user": map[string]any{"login": "github-actions[bot]"}})
	}
	for _, name := range []string{"product-acceptance-issue/11", "product-acceptance-artifact/" + artifact + "/issue-11/implementation-" + implementation + "/acceptor-example-acceptor"} {
		state.Statuses = append(state.Statuses, map[string]any{"context": name, "state": "success", "creator": map[string]any{"login": "github-actions[bot]"}, "description": "Accepted by example-acceptor", "target_url": "https://github.com/acoz-labs/mandalore/actions/runs/123"})
	}
	state.Release = map[string]any{"tagName": state.Tag, "url": "https://github.com/acoz-labs/mandalore/releases/tag/" + state.Tag, "body": "- Artifact: `" + identity + "`"}
	switch scenario {
	case "no acceptance":
		state.Comments = state.Comments[:1]
	case "foreign status":
		state.Statuses[0]["creator"] = map[string]any{"login": "example-contributor"}
	case "unlinked implementation":
		state.Issue["body"] = "no implementation link"
	case "wrong nomination":
		state.Comments[0]["body"] = "no matching nomination"
	case "production":
		state.Issue["labels"] = []any{map[string]any{"name": "delivery:ready-for-production"}}
		state.Release = nil
	}
	b, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(dir, "state.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	return dir, append(env, "RELEASE_SHA="+sha, "RELEASE_ARTIFACT="+identity)
}

func runFinalizer(t *testing.T, dir string, env []string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", append([]string{"bin/finalize-release"}, args...)...)
	cmd.Dir = dir
	cmd.Env = env
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func fixtureLog(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "calls.log"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(b)
}

func TestArtifactFinalizerVerifiesBeforeLedgerWritesAndSupportsRetry(t *testing.T) {
	dir, env := fixture(t, "success")
	out, err := runFinalizer(t, dir, env, "artifact")
	if err != nil {
		t.Fatal(err, out)
	}
	log := fixtureLog(t, dir)
	verified, closed := strings.Index(log, "publication-verified"), strings.Index(log, "gh issue close")
	if verified < 0 || closed < verified || strings.Contains(log, "gh release create") || strings.Contains(out, "artifact-20") {
		t.Fatal("ledger closed before exact product verification", log, out)
	}
	for _, marker := range []string{"gh label create", "gh issue comment", "gh api -X DELETE", "gh issue edit"} {
		if i := strings.Index(log, marker); i >= 0 && i < verified {
			t.Fatal("ledger mutated before verification", marker, log)
		}
	}
	// No local product tag is fabricated: a fresh verified GitHub release, not
	// presence of a local tag, is authority for an already-closed retry.
	if out, err := runFinalizer(t, dir, env, "artifact", "--preflight"); err != nil {
		t.Fatal("closed retry preflight", err, out)
	}
	if out, err := runFinalizer(t, dir, env, "artifact"); err != nil {
		t.Fatal("closed retry", err, out)
	}
}

func TestArtifactFinalizerDenialsNeverMutateLedger(t *testing.T) {
	for _, scenario := range []string{"no acceptance", "foreign status", "unlinked implementation", "wrong nomination", "verification failure", "partial verification", "wrong verified identity", "wrong verified source", "wrong verified URL", "missing asset", "omitted issue"} {
		t.Run(scenario, func(t *testing.T) {
			dir, env := fixture(t, scenario)
			out, err := runFinalizer(t, dir, env, "artifact")
			if err == nil {
				t.Fatal("expected refusal", out)
			}
			log := fixtureLog(t, dir)
			verificationScenario := strings.Contains(scenario, "verifi") || scenario == "missing asset"
			if verificationScenario && !strings.Contains(log, "release-publication verify") {
				t.Fatal("test refused before exercising the intended published-verification boundary", log, out)
			}
			for _, marker := range []string{"gh label create", "gh issue close", "gh issue edit", "gh issue comment", "gh api -X DELETE", "gh release create"} {
				if strings.Contains(log, marker) {
					t.Fatal("denial mutated external state", log, out)
				}
			}
		})
	}
}

func TestArtifactFinalizerRecoversPartialLedgerWithoutRepublishing(t *testing.T) {
	dir, env := fixture(t, "ledger interruption")
	if out, err := runFinalizer(t, dir, env, "artifact"); err == nil {
		t.Fatal("expected interrupted ledger", out)
	}
	log := fixtureLog(t, dir)
	if !strings.Contains(log, "publication-verified") || !strings.Contains(log, "gh issue comment") || strings.Contains(log, "gh issue close") {
		t.Fatal("fixture did not interrupt a partially written ledger", log)
	}
	if out, err := runFinalizer(t, dir, env, "artifact", "--preflight"); err != nil {
		t.Fatal("partial retry preflight", err, out)
	}
	if out, err := runFinalizer(t, dir, env, "artifact"); err != nil {
		t.Fatal("partial retry", err, out)
	}
	log = fixtureLog(t, dir)
	if strings.Count(log, "publication-verified") != 3 || strings.Count(log, "gh issue comment") != 1 || strings.Count(log, "gh issue close") != 1 || strings.Contains(log, "gh release create") {
		t.Fatal("retry skipped verification or duplicated release/ledger comment", log)
	}
}

func TestClosedArtifactRetryStillRequiresFreshPublicationVerification(t *testing.T) {
	dir, env := fixture(t, "success")
	if out, err := runFinalizer(t, dir, env, "artifact"); err != nil {
		t.Fatal(err, out)
	}
	b, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s fixtureState
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	s.Scenario = "verification failure"
	b, _ = json.Marshal(s)
	if err := os.WriteFile(filepath.Join(dir, "state.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	start := len(fixtureLog(t, dir))
	if out, err := runFinalizer(t, dir, env, "artifact", "--preflight"); err == nil {
		t.Fatal("closed issue bypassed fresh verification", out)
	}
	log := fixtureLog(t, dir)[start:]
	if !strings.Contains(log, "release-publication verify") || strings.Contains(log, "gh issue edit") || strings.Contains(log, "gh label create") {
		t.Fatal("closed retry verification ordering", log)
	}
}

func TestArtifactPreflightDoesNotRequirePublicationForAcceptedOpenIssues(t *testing.T) {
	dir, env := fixture(t, "verification failure")
	out, err := runFinalizer(t, dir, env, "artifact", "--preflight")
	if err != nil {
		t.Fatal(err, out)
	}
	if strings.Contains(fixtureLog(t, dir), "release-publication verify") {
		t.Fatal("pre-publication preflight demanded a published candidate")
	}
}

func TestProductionFinalizerPreservesDeploymentAndCalendarRelease(t *testing.T) {
	dir, env := fixture(t, "production")
	out, err := runFinalizer(t, dir, env, "production")
	if err != nil {
		t.Fatal(err, out)
	}
	log := fixtureLog(t, dir)
	if !strings.Contains(log, "record-production-deployment verify") || !strings.Contains(log, "gh release create prod-") || strings.Contains(log, "release-publication") || !strings.Contains(log, "gh issue close") {
		t.Fatal("production behavior changed", log, out)
	}
}

func TestFinalizerHelperProcess(t *testing.T) {
	helper := os.Getenv("MANDALORE_FINALIZER_HELPER")
	if helper == "" {
		return
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	args = args[1:]
	dir := os.Getenv("FINALIZER_FIXTURE")
	b, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		panic(err)
	}
	var s fixtureState
	if err := json.Unmarshal(b, &s); err != nil {
		panic(err)
	}
	log := func(text string) {
		f, err := os.OpenFile(filepath.Join(dir, "calls.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			panic(err)
		}
		fmt.Fprintln(f, text)
		f.Close()
	}
	log(helper + " " + strings.Join(args, " "))
	emit := func(v any) { json.NewEncoder(os.Stdout).Encode(v) }
	save := func() {
		b, _ := json.Marshal(s)
		if err := os.WriteFile(filepath.Join(dir, "state.json"), b, 0600); err != nil {
			panic(err)
		}
	}
	arg := func(name string) string {
		for i, a := range args {
			if a == name && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	if helper == "record-production-deployment" {
		emit(map[string]any{"deployment_id": "45", "receipt": map[string]any{"control_sha": strings.Repeat("c", 40)}})
		os.Exit(0)
	}
	if helper == "release-publication" {
		selection, err := distribution.SelectPublication(arg("--version"), arg("--identity"))
		if err != nil {
			os.Exit(64)
		}
		if args[0] == "selection" {
			emit(selection)
			os.Exit(0)
		}
		if args[0] != "verify" {
			os.Exit(64)
		}
		if s.Scenario == "verification failure" {
			os.Exit(1)
		}
		r := map[string]any{"identity": s.Identity, "source_commit": s.SHA, "release_id": 42, "release_url": "https://github.com/acoz-labs/mandalore/releases/tag/v1.0.0", "phase": "publication-verified", "assets": make([]int, 8)}
		switch s.Scenario {
		case "partial verification":
			r["phase"] = "published"
		case "wrong verified identity":
			r["identity"] = "other"
		case "wrong verified source":
			r["source_commit"] = strings.Repeat("f", 40)
		case "wrong verified URL":
			r["release_url"] = "https://example.invalid"
		case "missing asset":
			r["assets"] = []int{1}
		}
		log("publication-verified")
		emit(r)
		os.Exit(0)
	}
	if helper != "gh" {
		os.Exit(97)
	}
	command := strings.Join(args, " ")
	switch {
	case strings.HasPrefix(command, "issue list "):
		label := arg("--label")
		if label == "delivery:acceptance" {
			os.Exit(0)
		}
		if s.Scenario == "omitted issue" {
			fmt.Println("11\n12")
			os.Exit(0)
		}
		if s.Issue["state"] == "OPEN" {
			for _, v := range s.Issue["labels"].([]any) {
				if v.(map[string]any)["name"] == label {
					fmt.Println("11")
					break
				}
			}
		}
	case strings.HasPrefix(command, "issue view "):
		emit(s.Issue)
	case strings.Contains(command, "/statuses?"):
		emit([]any{s.Statuses})
	case strings.Contains(command, "/comments?"):
		emit([]any{s.Comments})
	case strings.HasPrefix(command, "release view "):
		if s.Release == nil {
			os.Exit(1)
		}
		emit(s.Release)
	case strings.HasPrefix(command, "release create "):
		if s.Scenario != "production" {
			fmt.Fprintln(os.Stderr, "artifact finalizer must never create a release")
			os.Exit(98)
		}
		s.Release = map[string]any{"tagName": args[2], "url": "https://github.com/acoz-labs/mandalore/releases/tag/" + args[2], "body": arg("--notes")}
		save()
	case strings.HasPrefix(command, "label create "):
	case strings.HasPrefix(command, "issue comment "):
		s.Comments = append(s.Comments, map[string]any{"body": arg("--body"), "user": map[string]any{"login": "github-actions[bot]"}})
		save()
	case strings.HasPrefix(command, "api -X DELETE "):
		label := strings.SplitN(args[len(args)-1], "/labels/", 2)[1]
		var labels []any
		for _, v := range s.Issue["labels"].([]any) {
			if v.(map[string]any)["name"] != label {
				labels = append(labels, v)
			}
		}
		if labels == nil {
			labels = []any{}
		}
		s.Issue["labels"] = labels
		save()
	case strings.HasPrefix(command, "issue edit "):
		if label := arg("--add-label"); label != "" {
			s.Issue["labels"] = append(s.Issue["labels"].([]any), map[string]any{"name": label})
		}
		if file := arg("--body-file"); file != "" {
			if s.Scenario == "ledger interruption" {
				s.Scenario = "success"
				save()
				os.Exit(1)
			}
			b, err := os.ReadFile(file)
			if err != nil {
				panic(err)
			}
			s.Issue["body"] = string(b)
		}
		save()
	case strings.HasPrefix(command, "issue close "):
		s.Issue["state"] = "CLOSED"
		save()
	default:
		fmt.Fprintln(os.Stderr, "unexpected fake GitHub command", command)
		os.Exit(97)
	}
	os.Exit(0)
}
