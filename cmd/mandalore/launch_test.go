package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLaunchCLIHelpSchemaAndFailures(t *testing.T) {
	for _, tc := range []struct {
		args     []string
		input    string
		ok       bool
		contains string
	}{
		{[]string{"launch", "--help"}, "", true, "--confirm-profile"},
		{[]string{"launch", "schema"}, "", true, "connection_root"},
		{[]string{"launch", "list", "--config", "/missing/launch.json"}, "", false, "configure"},
		{[]string{"launch", "work", "--agent", "pi", "--unknown"}, "", false, "Invalid launch"},
		{[]string{"launch", "configure", "work", "--confirm-profile"}, `{"binding":"/example","binding":"/other"}`, false, "strict launch"},
		{[]string{"launch", "work", "not-after-boundary"}, "", false, "native arguments after --"},
	} {
		var out, errout bytes.Buffer
		code := run(context.Background(), tc.args, strings.NewReader(tc.input), &out, &errout)
		if (code == 0) != tc.ok || !strings.Contains(out.String()+errout.String(), tc.contains) {
			t.Errorf("args %v: code %d stdout %s stderr %s", tc.args, code, out.String(), errout.String())
		}
	}
}
