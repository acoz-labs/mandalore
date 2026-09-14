package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

func connectionDefaults(profile *install.Profile) error {
	if profile.StateDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		profile.StateDir = filepath.Join(base, "mandalore", "installation")
	}
	if profile.NativeHome == "" {
		profile.NativeHome = os.Getenv("CODEX_HOME")
		if profile.NativeHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			profile.NativeHome = filepath.Join(home, ".codex")
		}
	}
	if profile.NativeBinary == "" {
		path, err := exec.LookPath("codex")
		if err != nil {
			return errors.New("Codex is not on PATH; supply --native-binary with its absolute path")
		}
		profile.NativeBinary, err = filepath.Abs(path)
		if err != nil {
			return err
		}
	}
	return nil
}

// The human CLI accepts its own successful plan envelope, while the typed
// connection_apply operation accepts the documented raw plan object.
func unwrapPlan(raw []byte) ([]byte, error) {
	var object map[string]any
	if err := strictjson.Decode(raw, &object, api.MaxInputBytes); err != nil {
		return nil, err
	}
	if _, wrapped := object["protocol_version"]; !wrapped {
		return raw, nil
	}
	var envelope struct {
		Protocol int          `json:"protocol_version"`
		OK       bool         `json:"ok"`
		Result   install.Plan `json:"result"`
	}
	if err := strictjson.Decode(raw, &envelope, api.MaxInputBytes); err != nil || !envelope.OK || envelope.Protocol != api.ProtocolVersion {
		return nil, strictjson.ErrInvalid
	}
	return json.Marshal(envelope.Result)
}

func runConnection(ctx context.Context, args []string, input io.Reader, out io.Writer) int {
	if len(args) == 0 {
		return bad(out, "Choose connection plan, apply, doctor or repair; use --help.")
	}
	if args[0] == "--help" || args[0] == "-h" {
		_, err := io.WriteString(out, help)
		if err != nil {
			return 1
		}
		return 0
	}
	sub := args[0]
	if sub != "plan" && sub != "apply" && sub != "doctor" && sub != "repair" {
		return bad(out, "Unknown connection command; use --help.")
	}
	f := flag.NewFlagSet("connection "+sub, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	readOnly := f.Bool("read-only", false, "Reject mutations")
	var profile install.Profile
	var binary, path, root string
	var applyRepair bool
	if sub == "plan" || sub == "doctor" {
		f.StringVar(&profile.StateDir, "state-dir", "", "Machine-local installation state directory")
		f.StringVar(&profile.NativeHome, "native-home", "", "Existing native Codex profile")
		f.StringVar(&profile.NativeBinary, "native-binary", "", "Absolute native Codex executable")
	}
	if sub == "plan" {
		f.StringVar(&binary, "binary", "", "Trusted Mandalore executable to stage; default running CLI")
		f.StringVar(&path, "binding", "", "Explicit machine-local signet binding")
	}
	if sub == "repair" {
		f.StringVar(&root, "connection-root", "", "Retained managed marketplace root")
		f.StringVar(&profile.NativeBinary, "native-binary", "", "Explicit replacement native executable, if needed")
		f.BoolVar(&applyRepair, "apply", false, "Apply the generated repair plan")
	}
	if err := f.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err := io.WriteString(out, help)
			if err != nil {
				return 1
			}
			return 0
		}
		return bad(out, "Invalid connection flags or missing values; use --help.")
	}
	if f.NArg() != 0 {
		return bad(out, "Unexpected connection arguments.")
	}
	if *readOnly && (sub == "apply" || applyRepair) {
		return emit(out, api.Failure("operation.read_only", "Mutations are disabled for this task.", false))
	}
	if ctx.Err() != nil {
		return emit(out, api.Failure("operation.cancelled", "Cancelled before connection work.", false))
	}
	var value any
	name := "connection_" + sub
	switch sub {
	case "plan", "doctor":
		if err := connectionDefaults(&profile); err != nil {
			return emit(out, api.Failure("connection.failed", err.Error(), false))
		}
		value = profile
		if sub == "plan" {
			var err error
			if binary == "" {
				binary, err = os.Executable()
			}
			if err == nil && path == "" {
				path, err = binding.DefaultPath()
			}
			if err != nil {
				return bad(out, "Cannot select local runtime or binding defaults; supply explicit absolute paths.")
			}
			value = install.Options{StateDir: profile.StateDir, NativeHome: profile.NativeHome, NativeBinary: profile.NativeBinary, Binary: binary, Binding: path}
		}
	case "repair":
		name = "connection_repair_plan"
		value = install.RepairInput{Root: root, NativeBinary: profile.NativeBinary}
	}
	var raw []byte
	if value != nil {
		raw, _ = json.Marshal(value)
	} else {
		var err error
		raw, err = io.ReadAll(io.LimitReader(input, api.MaxInputBytes+1))
		if err != nil {
			return bad(out, "Could not read the approved connection plan.")
		}
		raw, err = unwrapPlan(raw)
		if err != nil {
			return bad(out, "Expected a valid plan object or successful connection-plan envelope.")
		}
	}
	a := api.New(nil, *readOnly)
	result := a.Call(ctx, name, raw)
	if sub == "repair" && applyRepair && result.OK {
		raw, _ = json.Marshal(result.Result)
		result = a.Call(ctx, "connection_apply", raw)
	}
	return emit(out, result)
}
