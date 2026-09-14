package main

import (
	"encoding/json"
	"github.com/acoz-labs/mandalore/internal/distribution"
	"strings"
	"testing"
)

func TestReleaseInspectionCLIValidationAndNoBinding(t *testing.T) {
	for _, args := range [][]string{{"release"}, {"release", "unknown"}, {"release", "inspect", "--unknown"}, {"release", "inspect", "--version", "../latest"}, {"release", "inspect", "--candidate", "/example", "--version", "1.0.0"}} {
		out, code := cli(t, args, "")
		if code != 2 || out.Error.Code != "input.invalid" {
			t.Fatal("invalid release request did not fail before network", args, out)
		}
	}
	out, code := cli(t, []string{"release", "inspect", "--candidate", t.TempDir(), "--read-only"}, "")
	if code != 1 || out.Error.Code != "release.failed" || out.Error.WriteMayHaveOccurred || strings.Contains(out.Error.Message, "binding") {
		t.Fatal("read-only local inspection was not unbound", out, code)
	}
}

func TestReleasePlanUnwrapPreservesLargeFilesystemIDs(t *testing.T) {
	p := distribution.InstallPlan{Observed: distribution.InstallObservation{Directories: []distribution.PathObservation{{Path: "/synthetic", Exists: true, Device: 1<<60 + 11, Inode: 1<<62 + 123}}}}
	for _, wrapped := range []bool{false, true} {
		var input any = p
		if wrapped {
			input = map[string]any{"protocol_version": 1, "ok": true, "result": p}
		}
		raw, _ := json.Marshal(input)
		got, err := unwrapReleasePlan(raw)
		if err != nil {
			t.Fatal(err)
		}
		var decoded distribution.InstallPlan
		if err := json.Unmarshal(got, &decoded); err != nil || decoded.Observed.Directories[0] != p.Observed.Directories[0] {
			t.Fatal("filesystem observation lost integer precision", err)
		}
	}
}

func TestReleasePlanCLIValidationAndNoBinding(t *testing.T) {
	for _, args := range [][]string{{"release", "plan"}, {"release", "plan", "--prefix", "/example", "--version", "../bad"}, {"release", "plan", "--prefix", "/example", "--retained", "bad"}} {
		out, code := cli(t, args, "")
		if code != 2 || out.Error.Code != "input.invalid" {
			t.Fatal("invalid release plan accepted", out, code)
		}
	}
	out, code := cli(t, []string{"release", "plan", "--prefix", t.TempDir(), "--candidate", t.TempDir(), "--read-only"}, "")
	if code != 1 || out.Error.Code != "release.failed" || out.Error.WriteMayHaveOccurred || strings.Contains(out.Error.Message, "binding") {
		t.Fatal("CLI plan required binding or wrote", out, code)
	}
}

func TestReleaseApplyCLIRequiresPlanAndHonorsReadOnly(t *testing.T) {
	out, code := cli(t, []string{"release", "apply", "--read-only"}, "")
	if code != 2 || out.Error.Code != "operation.read_only" {
		t.Fatal("apply read-only guard missing", out, code)
	}
	for _, raw := range []string{"", `{}`, `{"protocol_version":1,"ok":false,"result":{}}`} {
		out, code := cli(t, []string{"release", "apply"}, raw)
		if code != 2 || out.Error.Code != "input.invalid" || out.Error.WriteMayHaveOccurred {
			t.Fatal("invalid plan was accepted", out, code)
		}
	}
}
