package main

import (
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
