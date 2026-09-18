package install

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// These helpers invoke only the selected runtime's existing typed connection
// operations. The preparing parent must never substitute its embedded plugin.
func PrepareViaRuntime(ctx context.Context, o Options) (Plan, error) {
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}
	var err error
	for _, path := range []*string{&o.StateDir, &o.NativeHome, &o.NativeBinary, &o.Binary, &o.Binding} {
		*path, err = canonical(*path)
		if err != nil {
			return Plan{}, err
		}
	}
	want, err := digest(o.Binary)
	if err != nil {
		return Plan{}, err
	}
	b, _ := json.Marshal(o)
	if len(b) > 32768 {
		return Plan{}, errors.New("connection options exceed the typed input limit")
	}
	raw, runErr := executeBounded(ctx, o.Binary, filepath.Dir(o.Binding), environment(nil), bytes.NewReader(b), "call", "connection_plan", "--read-only")
	if runErr != nil {
		if ctx.Err() != nil {
			return Plan{}, ctx.Err()
		}
		return Plan{}, errors.New("selected runtime could not prepare this connection; raw output suppressed")
	}
	var reply struct {
		Protocol int             `json:"protocol_version"`
		OK       bool            `json:"ok"`
		Result   json.RawMessage `json:"result"`
	}
	if decodeNative(raw, &reply) != nil || reply.Protocol != 1 || !reply.OK {
		return Plan{}, errors.New("selected runtime returned an invalid connection preview")
	}
	var p Plan
	if strictjson.Decode(reply.Result, &p, 32768) != nil {
		return Plan{}, errors.New("selected runtime connection plan has invalid fields or exceeds its limit")
	}
	if got, e := digest(o.Binary); e != nil || got != want {
		return Plan{}, errors.New("selected runtime changed while preparing the connection")
	}
	if p.Options != o || p.BinarySHA256 != want || p.SchemaVersion != 1 || p.SignetID == "" || p.Marketplace != "mandalore" || p.PluginID != "mandalore@mandalore" || p.PackageVersion == "" {
		return Plan{}, errors.New("selected runtime preview is not bound to the requested connection")
	}
	decoded, e := hex.DecodeString(p.PackageSHA256)
	if e != nil || len(decoded) != 32 || strings.ToLower(p.PackageSHA256) != p.PackageSHA256 {
		return Plan{}, errors.New("selected runtime preview has an invalid embedded package identity")
	}
	if got, e := digestLimit(o.NativeBinary, maxNativeBinary); e != nil || got != p.NativeSHA256 {
		return Plan{}, errors.New("native executable differs from the preview")
	}
	if got, e := digestLimit(o.Binding, 32768); e != nil || got != p.BindingSHA256 {
		return Plan{}, errors.New("binding differs from the preview")
	}
	key := planKey(p)
	base, _, _ := strings.Cut(p.PackageVersion, "+")
	if p.Root != filepath.Join(o.StateDir, "connections", key) || p.Runtime != filepath.Join(o.StateDir, "runtimes", "sha256-"+want, "mandalore") || p.Version != base+"+codex."+key {
		return Plan{}, errors.New("selected runtime preview contains inconsistent generated paths or cache identity")
	}
	return p, nil
}

func ApplyViaRuntime(ctx context.Context, p Plan) (Result, error) {
	return ApplyViaRuntimeAcknowledged(ctx, p, false)
}

func ApplyViaRuntimeAcknowledged(ctx context.Context, p Plan, sessionsStopped bool) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	// Re-prepare using that SAME executable, including its own embedded package,
	// and compare the complete reviewed plan immediately before native mutation.
	fresh, err := PrepareViaRuntime(ctx, p.Options)
	if err != nil {
		return Result{}, err
	}
	if !reflect.DeepEqual(fresh, p) {
		return Result{}, errors.New("selected runtime connection plan is stale; preview again")
	}
	guarded, err := runtimeSupportsSessionGuard(ctx, p)
	if err != nil {
		return Result{}, err
	}
	if !guarded && !sessionsStopped {
		return Result{Connection: p, RequiresFreshSession: true, Phase: "deferred", Notice: "Selected runtime does not advertise the stopped-session guard. No connection apply was invoked. Exit all Codex sessions using this profile, then explicitly acknowledge sessions_stopped; idle is insufficient."}, errors.New("legacy runtime connection activation deferred pending stopped-session acknowledgement")
	}
	var input any = p
	if guarded {
		input = ApplyInput{Plan: p, SessionsStopped: sessionsStopped}
	}
	b, _ := json.Marshal(input)
	if len(b) > 32768 {
		return Result{}, errors.New("connection plan exceeds the typed input limit")
	}
	raw, runErr := executeBounded(ctx, p.Binary, filepath.Dir(p.Binding), environment(nil), bytes.NewReader(b), "call", "connection_apply")
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	var reply struct {
		Protocol int             `json:"protocol_version"`
		OK       bool            `json:"ok"`
		Result   json.RawMessage `json:"result"`
		Error    struct {
			Connection json.RawMessage `json:"connection_result"`
		} `json:"error"`
	}
	if decodeNative(raw, &reply) != nil || reply.Protocol != 1 {
		return Result{}, errors.New("selected runtime apply did not return a usable receipt; inspect native state before retrying; raw output suppressed")
	}
	data := reply.Result
	if !reply.OK {
		data = reply.Error.Connection
	}
	var result Result
	if strictjson.Decode(data, &result, 65536) != nil || !reflect.DeepEqual(result.Connection, p) {
		return Result{}, errors.New("selected runtime apply returned no matching connection receipt; inspect native state before retrying")
	}
	if !reply.OK || runErr != nil {
		return result, errors.New("selected runtime did not complete the native connection; inspect the reported phase before retrying")
	}
	if !result.Installed || !result.RequiresFreshSession || result.Phase != "verified" {
		return result, errors.New("selected runtime did not verify a complete native connection")
	}
	return result, nil
}

// Negotiate the additive input through the selected trusted runtime's catalog.
// Legacy strict decoders must never receive fields they did not advertise.
// A missing/unreadable catalog is not permission to attempt an unguarded apply.
func runtimeSupportsSessionGuard(ctx context.Context, p Plan) (bool, error) {
	raw, err := execute(ctx, p.Binary, filepath.Dir(p.Binding), environment(nil), nil, "operations")
	if err != nil {
		return false, errors.New("cannot inspect selected runtime apply contract; no connection apply invoked")
	}
	var reply struct {
		Protocol int  `json:"protocol_version"`
		OK       bool `json:"ok"`
		Result   struct {
			Operations []struct {
				Name  string          `json:"name"`
				Input json.RawMessage `json:"input_schema"`
			} `json:"operations"`
		} `json:"result"`
	}
	if decodeNative(raw, &reply) != nil || reply.Protocol != 1 || !reply.OK {
		return false, errors.New("invalid selected runtime operation catalog; no connection apply invoked")
	}
	found, guarded := false, false
	for _, op := range reply.Result.Operations {
		if op.Name != "connection_apply" {
			continue
		}
		var input struct {
			Type       string                     `json:"type"`
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if found || json.Unmarshal(op.Input, &input) != nil || input.Type != "object" || input.Properties == nil {
			return false, errors.New("ambiguous selected runtime apply contract")
		}
		found = true
		rawField, exists := input.Properties["sessions_stopped"]
		var field struct {
			Type string `json:"type"`
		}
		if exists && (json.Unmarshal(rawField, &field) != nil || field.Type != "boolean") {
			return false, errors.New("unsupported stopped-session acknowledgement contract")
		}
		guarded = exists
	}
	if !found {
		return false, errors.New("selected runtime has no connection apply contract")
	}
	if got, e := digest(p.Binary); e != nil || got != p.BinarySHA256 {
		return false, errors.New("selected runtime changed during contract inspection")
	}
	return guarded, nil
}
