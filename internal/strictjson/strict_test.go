package strictjson

import (
	"strings"
	"testing"
)

func TestStrictInput(t *testing.T) {
	for _, raw := range []string{`{"value":"first","value":"last"}`, `{"value":null}`, `{"VALUE":"case"}`, `{"unknown":"secret-canary"}`, `{} {}`, `null`, `[]`, strings.Repeat(" ", 1025), `{"value":3}`, `{"value":"unterminated}`} {
		var out struct {
			Value string `json:"value"`
		}
		if err := Decode([]byte(raw), &out, 1024); err == nil {
			t.Errorf("accepted invalid input %q", raw)
		}
	}
	var out struct {
		Value string `json:"value"`
	}
	if err := Decode([]byte(`{"value":"valid"}`), &out, 1024); err != nil || out.Value != "valid" {
		t.Fatal(out, err)
	}
}

func TestNestedDuplicateKeys(t *testing.T) {
	var out struct {
		Nested map[string]string `json:"nested"`
	}
	if err := Decode([]byte(`{"nested":{"key":"first","key":"last"}}`), &out, 1024); err == nil {
		t.Fatal("nested duplicate accepted")
	}
}
