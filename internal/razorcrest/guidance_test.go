package razorcrest

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/api"
)

// Exercise the published wire metadata, including capability-specific init
// guidance. A description must never direct a remote client to an unavailable
// local operation, even if a future catalog update adds such a reference.
func TestRemoteGuidanceOnlyReferencesAvailableTools(t *testing.T) {
	for _, write := range []bool{false, true} {
		for _, canon := range []bool{false, true} {
			s, token, secret := serviceFixture(t, write)
			if canon {
				s.config.CanonFoundlings = []string{"foundling-example"}
				// Rebuild after changing the configured capability, before connecting.
				rebuilt, err := New(s.config)
				if err != nil {
					t.Fatal(err)
				}
				rebuilt.auth = s.auth
				s = rebuilt
			}
			ctx, client := serviceClient(t, s, token, secret)
			list, err := client.ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			names := map[string]bool{}
			published := client.InitializeResult().Instructions
			for _, tool := range list.Tools {
				names[tool.Name] = true
				raw, err := json.Marshal(tool)
				if err != nil {
					t.Fatal(err)
				}
				published += " " + string(raw)
			}
			for _, op := range api.Catalog() {
				if !names[op.Name] && regexp.MustCompile(`\b`+regexp.QuoteMeta(op.Name)+`\b`).MatchString(published) {
					t.Errorf("write=%t canon=%t: guidance references unavailable %s", write, canon, op.Name)
				}
			}
			if strings.Contains(published, "harness-prefixed") || strings.Contains(published, "supplied by lifecycle context") {
				t.Error("remote schema retained a native-only session requirement")
			}
			if !canon && strings.Contains(published, "razor_session_open") {
				t.Error("canon guidance advertised without canon capability")
			}
		}
	}
}

func TestRemoteSchemaDoesNotMutateNativeCatalog(t *testing.T) {
	for _, op := range api.Catalog() {
		if !strings.HasPrefix(op.Name, "foundling_") {
			continue
		}
		before, err := json.Marshal(op.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		remote := remoteSchema(op.InputSchema)
		if field, ok := remote.Properties["session_id"]; ok && !strings.Contains(field.Description, "razor_session_open") {
			t.Errorf("%s lacks remote session source", op.Name)
		}
		after, _ := json.Marshal(op.InputSchema)
		if string(before) != string(after) {
			t.Errorf("%s mutated shared catalog", op.Name)
		}
	}
}
