package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/distribution"
)

func TestReleaseInspectionIsCLIOnlyUnboundAndReadOnly(t *testing.T) {
	found := false
	for _, op := range Catalog() {
		if op.Name == "release_inspect" {
			found = true
			if !op.CLIOnly || op.RequiresBinding || !op.ReadOnly || !op.Network {
				t.Fatal("release inspection has wrong authority flags", op)
			}
		}
	}
	if !found {
		t.Fatal("release inspection missing from typed discovery")
	}
	raw, _ := json.Marshal(map[string]string{"candidate": t.TempDir()})
	out := New(nil, true).Call(context.Background(), "release_inspect", raw)
	if out.OK || out.Error.Code != "release.failed" || out.Error.WriteMayHaveOccurred || out.Error.InspectBeforeRetry {
		t.Fatal("local candidate failure has wrong effect report", out)
	}
	if strings.Contains(out.Error.Message, "binding") {
		t.Fatal("unbound release inspection required a memory binding")
	}
	for _, data := range []string{`{"version":"1.0.0","candidate":"/example"}`, `{"version":"../latest"}`, `{"unexpected":true}`} {
		out := New(nil, true).Call(context.Background(), "release_inspect", []byte(data))
		if out.OK || out.Error.Code != "input.invalid" {
			t.Fatal("invalid selection did not fail before network", out)
		}
	}
}

func TestReleaseQuotaFailurePreservesOperationEffects(t *testing.T) {
	quota := &distribution.ReleaseRateLimitError{Retry: distribution.ReleaseRetry{HTTPStatus: 429, Kind: "unspecified", RetryAfterSeconds: 60}}
	for _, result := range []*distribution.InstallResult{nil, {Phase: "staging"}, {Phase: "pending-inspection", Pending: "/synthetic/pending.json"}, {Phase: "activation", DestinationChanged: true, Pending: "/synthetic/pending.json"}} {
		e := &releaseFailure{err: quota, result: result}
		out := New(nil, false).failure(Operation{}, e)
		if out.Error.Code != "release.rate_limited" || out.Error.ReleaseRetry == nil || *out.Error.ReleaseRetry != quota.Retry || out.Error.ReleaseResult != result || out.Error.Retryable {
			t.Fatal("typed quota advice lost or enabled automatic retry", out)
		}
		mayWrite := result != nil && result.DestinationChanged
		inspect := mayWrite || result != nil && result.Pending != ""
		if out.Error.WriteMayHaveOccurred != mayWrite || out.Error.InspectBeforeRetry != inspect {
			t.Fatal("HTTP refusal replaced operation effects", out)
		}
		if strings.Contains(out.Error.Message, "no installation occurred") {
			t.Fatal("invented effects", out)
		}
	}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		out := New(nil, false).failure(Operation{}, &releaseFailure{err: errors.Join(quota, cause)})
		if out.Error.Code != "operation.cancelled" || out.Error.ReleaseRetry != nil {
			t.Fatal("cancellation lost precedence", out)
		}
	}
}

func TestReleasePlanningIsCLIOnlyUnboundAndReadOnly(t *testing.T) {
	found := false
	for _, op := range Catalog() {
		if op.Name == "release_plan" {
			found = true
			if !op.CLIOnly || op.RequiresBinding || !op.ReadOnly || !op.Network {
				t.Fatal("wrong release plan authority flags")
			}
		}
	}
	if !found {
		t.Fatal("release plan missing from operation discovery")
	}
	for _, raw := range []string{`{}`, `{"prefix":"/example","version":"../bad"}`, `{"prefix":"/example","version":"1.0.0","retained":"bad"}`} {
		out := New(nil, true).Call(context.Background(), "release_plan", []byte(raw))
		if out.OK || out.Error.Code != "input.invalid" {
			t.Fatal("invalid plan input accepted", out)
		}
	}
	raw, _ := json.Marshal(map[string]string{"prefix": t.TempDir(), "candidate": t.TempDir()})
	out := New(nil, true).Call(context.Background(), "release_plan", raw)
	if out.OK || out.Error.Code != "release.failed" || out.Error.WriteMayHaveOccurred || out.Error.InspectBeforeRetry {
		t.Fatal("planning failure has wrong effect report", out)
	}
}

func TestReleaseApplyIsExplicitCLIOnlyAndReadOnlyGuarded(t *testing.T) {
	found := false
	for _, op := range Catalog() {
		if op.Name == "release_apply" {
			found = true
			if !op.CLIOnly || op.RequiresBinding || op.ReadOnly || !op.Network {
				t.Fatal("wrong release apply authority")
			}
		}
	}
	if !found {
		t.Fatal("release apply missing from discovery")
	}
	out := New(nil, true).Call(context.Background(), "release_apply", []byte(`{}`))
	if out.OK || out.Error.Code != "operation.read_only" || out.Error.WriteMayHaveOccurred {
		t.Fatal("read-only did not deny apply before decoding", out)
	}
	out = New(nil, false).Call(context.Background(), "release_apply", []byte(`{}`))
	if out.OK || out.Error.Code != "input.invalid" || out.Error.WriteMayHaveOccurred {
		t.Fatal("invalid plan required binding or caused effects", out)
	}
}
