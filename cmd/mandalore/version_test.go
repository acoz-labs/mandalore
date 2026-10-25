package main

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

func TestVersionWireContractIsStrict(t *testing.T) {
	var out, errout bytes.Buffer
	if code := run(context.Background(), []string{"version"}, strings.NewReader(""), &out, &errout); code != 0 {
		t.Fatalf("version exit %d: %s", code, errout.String())
	}
	// Frozen 2.0 installer wire contract (#163 dropped the 1.x plugin fields, so
	// 1.x installers can't update to 2.0; the bootstrap runs the new binary). Do
	// not replace this with the runtime's metadata type: released updaters cannot
	// accept additive fields.
	var released struct {
		Protocol int  `json:"protocol_version"`
		OK       bool `json:"ok"`
		Result   struct {
			Name     string `json:"name"`
			Version  string `json:"version"`
			Source   string `json:"source_commit"`
			OS       string `json:"os"`
			Arch     string `json:"arch"`
			Go       string `json:"go_version"`
			Protocol int    `json:"protocol_version"`
			Read     []int  `json:"signet_read_versions"`
			Write    []int  `json:"signet_write_versions"`
		} `json:"result"`
	}
	if err := strictjson.Decode(out.Bytes(), &released, 16<<10); err != nil {
		t.Fatalf("the installer cannot decode actual version output: %v", err)
	}
	if !released.OK || released.Protocol != 1 || released.Result.Name != "mandalore" {
		t.Fatalf("invalid version envelope: %+v", released)
	}
}

func TestVersionReportsActualToolchainAndBuildSource(t *testing.T) {
	v, code := cli(t, []string{"version"}, "")
	if code != 0 || !v.OK {
		t.Fatal(v, code)
	}
	r := v.Result.(map[string]any)
	if r["go_version"] != runtime.Version() {
		t.Fatal("runtime does not report its actual toolchain", r)
	}
	if r["source_commit"] != "" || r["version"] != "0.0.0-dev" {
		t.Fatal("unstamped development build claims release provenance", r)
	}
	for _, key := range []string{"signet_read_versions", "signet_write_versions"} {
		values, ok := r[key].([]any)
		if !ok || len(values) != 2 || values[0] != float64(1) || values[1] != float64(2) {
			t.Fatal("missing schema compatibility", r)
		}
	}
}
