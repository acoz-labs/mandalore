package releaseworkflow

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

const roadmapCandidate = "b89458617b49d2eb0a8686dcd75c26aa7ea6f6ba"
const roadmapArtifact = "mandalore:" + roadmapCandidate + ":sha256:7995f820a7182a95b828851afce12bacea7108f82afac9369d307a43d961f45a"

var roadmapIssues = []int{13, 56, 57, 61, 65, 74, 85, 88, 93}

const correctedCandidate = "51aee17afec015ba2ad44584f8190b4bb6d901a8"
const correctedArtifact = "mandalore:" + correctedCandidate + ":sha256:7adacb6e7dc990a12df2cc25e72404dde0156835d771474c9745d3b517a0780c"

var correctedIssues = []int{13, 56, 57, 61, 65, 74, 85, 88, 93, 99, 100}

func TestRoadmapOwnerAcceptanceScope(t *testing.T) {
	run := func(t *testing.T, args []string, owner, confirmation string, want bool) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "bash", append([]string{"../../bin/mvp-owner-acceptance"}, args...)...)
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "MVP_ACCEPTANCE_OWNER=" + owner, "MVP_OWNER_REVIEW_CONFIRMED=" + confirmation}
		out, err := cmd.CombinedOutput()
		if (err == nil) != want {
			t.Fatalf("eligibility=%v want %v: %s", err, want, out)
		}
	}
	for _, pair := range []struct{ source, artifact string }{{ownerCandidate, ownerArtifact}, {armorerCandidate, armorerArtifact}, {roadmapCandidate, roadmapArtifact}, {correctedCandidate, correctedArtifact}} {
		for issue := 0; issue <= 110; issue++ {
			t.Run(pair.source+"/"+strconv.Itoa(issue), func(t *testing.T) {
				want := issue >= 2 && issue <= 12 || pair.source == armorerCandidate && issue == 45
				if pair.source == roadmapCandidate {
					want = false
					for _, allowed := range roadmapIssues {
						want = want || issue == allowed
					}
				}
				if pair.source == correctedCandidate {
					want = false
					for _, allowed := range correctedIssues {
						want = want || issue == allowed
					}
				}
				run(t, []string{"acoz-labs/mandalore", pair.source, strconv.Itoa(issue), pair.artifact, "example-owner"}, "example-owner", "true", want)
			})
		}
	}
	for _, scenario := range []string{"old-source", "old-artifact", "armorer-source", "armorer-artifact", "roadmap-source", "roadmap-artifact", "earlier-1.1", "wrong-artifact", "wrong-source", "wrong-repo", "leading-zero", "negative-issue", "spaced-issue", "no-owner", "wrong-owner", "no-reviewer", "no-confirmation", "false-confirmation", "uppercase-confirmation", "missing-arg", "extra-arg", "case-insensitive"} {
		t.Run(scenario, func(t *testing.T) {
			args := []string{"acoz-labs/mandalore", correctedCandidate, "93", correctedArtifact, "example-owner"}
			owner, confirmation := "example-owner", "true"
			switch scenario {
			case "old-source":
				args[1], args[2] = ownerCandidate, "10"
			case "old-artifact":
				args[3] = ownerArtifact
			case "armorer-source":
				args[1], args[2] = armorerCandidate, "45"
			case "armorer-artifact":
				args[3] = armorerArtifact
			case "roadmap-source":
				args[1] = roadmapCandidate
			case "roadmap-artifact":
				args[3] = roadmapArtifact
			case "earlier-1.1":
				args[1] = "bb16a57eb16dcd5e3da1386d6eb72c64169be6b5"
				args[3] = "mandalore:" + args[1] + ":sha256:ab4af42cee70495b66f4a64b3d754dc0e83d50c789ac1d8b6c4a419e7ffcb986"
			case "wrong-artifact":
				args[3] += "changed"
			case "wrong-source":
				args[1] = strings.Repeat("a", 40)
			case "wrong-repo":
				args[0] = "example/another"
			case "leading-zero":
				args[2] = "093"
			case "negative-issue":
				args[2] = "-93"
			case "spaced-issue":
				args[2] = "93 "
			case "no-owner":
				owner = ""
			case "wrong-owner":
				owner = "another-owner"
			case "no-reviewer":
				args[4] = ""
			case "no-confirmation":
				confirmation = ""
			case "false-confirmation":
				confirmation = "false"
			case "uppercase-confirmation":
				confirmation = "TRUE"
			case "missing-arg":
				args = args[:4]
			case "extra-arg":
				args = append(args, "unexpected")
			case "case-insensitive":
				owner = "Example-Owner"
			}
			run(t, args, owner, confirmation, scenario == "case-insensitive")
		})
	}
}
