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
	return connectionHarnessDefaults(profile, "codex")
}

func connectionHarnessDefaults(profile *install.Profile, harness string) error {
	if harness != "codex" && harness != "pi" && harness != "claude-code" {
		return errors.New("unsupported connection harness")
	}
	if profile.StateDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		profile.StateDir = filepath.Join(base, "mandalore", "installation")
	}
	if profile.NativeHome == "" {
		key := "CODEX_HOME"
		if harness == "pi" {
			key = "PI_CODING_AGENT_DIR"
		} else if harness == "claude-code" {
			key = "CLAUDE_CONFIG_DIR"
		}
		profile.NativeHome = os.Getenv(key)
		if profile.NativeHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			profile.NativeHome = filepath.Join(home, ".codex")
			if harness == "pi" {
				profile.NativeHome = filepath.Join(home, ".pi", "agent")
			} else if harness == "claude-code" {
				profile.NativeHome = filepath.Join(home, ".claude")
			}
		}
	}
	if profile.NativeBinary == "" {
		executable := harness
		if harness == "claude-code" {
			executable = "claude"
		}
		path, err := exec.LookPath(executable)
		if err != nil {
			return errors.New("Selected harness is not on PATH; supply --native-binary with its absolute path")
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
	return unwrapConnectionPlan[install.Plan](raw)
}

func unwrapConnectionPlan[P any](raw []byte) ([]byte, error) {
	var object map[string]any
	if err := strictjson.Decode(raw, &object, api.MaxInputBytes); err != nil {
		return nil, err
	}
	if _, wrapped := object["protocol_version"]; !wrapped {
		return raw, nil
	}
	var envelope struct {
		Protocol int  `json:"protocol_version"`
		OK       bool `json:"ok"`
		Result   P    `json:"result"`
	}
	if err := strictjson.Decode(raw, &envelope, api.MaxInputBytes); err != nil || !envelope.OK || envelope.Protocol != api.ProtocolVersion {
		return nil, strictjson.ErrInvalid
	}
	return json.Marshal(envelope.Result)
}

func runConnection(ctx context.Context, args []string, input io.Reader, out io.Writer) int {
	if len(args) > 0 && args[0] == "assess" {
		return runAssessment(ctx, args[1:], out)
	}
	if len(args) == 0 {
		return bad(out, "Choose connection assess, plan, apply, armorer or repair; doctor remains an alias. Use --help.")
	}
	if args[0] == "--help" || args[0] == "-h" {
		_, err := io.WriteString(out, help)
		if err != nil {
			return 1
		}
		return 0
	}
	sub := args[0]
	// Preserve the versioned diagnostic operation and legacy flags/results.
	if sub == "armorer" {
		sub = "doctor"
	}
	if sub != "plan" && sub != "apply" && sub != "doctor" && sub != "repair" {
		return bad(out, "Unknown connection command; use --help.")
	}
	f := flag.NewFlagSet("connection "+sub, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	readOnly := f.Bool("read-only", false, "Reject mutations")
	harness := f.String("harness", "codex", "Native harness: codex, pi or claude-code")
	var memoryReadOnly bool
	var sessionSync bool
	var profile install.Profile
	var binary, path, root string
	var applyRepair bool
	var sessionsStopped bool
	if sub == "apply" || sub == "repair" {
		f.BoolVar(&sessionsStopped, "sessions-stopped", false, "Confirm all selected native sessions using the selected profile have exited")
	}
	if sub == "plan" || sub == "doctor" {
		f.StringVar(&profile.StateDir, "state-dir", "", "Machine-local installation state directory")
		f.StringVar(&profile.NativeHome, "native-home", "", "Existing native profile")
		f.StringVar(&profile.NativeBinary, "native-binary", "", "Absolute native executable")
	}
	if sub == "plan" {
		f.BoolVar(&sessionSync, "session-sync", false, "Enable reviewed session-authorized automatic refresh and delivery")
		f.BoolVar(&memoryReadOnly, "memory-read-only", false, "Enforce read-only memory in Pi and Claude Code connections")
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
	if *harness != "codex" && *harness != "pi" && *harness != "claude-code" {
		return bad(out, "Choose --harness codex, pi or claude-code.")
	}
	if memoryReadOnly && sessionSync {
		return bad(out, "Read-only memory cannot enable session transport.")
	}
	if memoryReadOnly && *harness != "pi" && *harness != "claude-code" {
		return bad(out, "--memory-read-only is supported by Pi and Claude Code connections.")
	}
	if sessionsStopped && (*harness == "pi" || sub == "repair" && !applyRepair) {
		return bad(out, "--sessions-stopped applies only to Codex or Claude Code apply or repair --apply.")
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
		if err := connectionHarnessDefaults(&profile, *harness); err != nil {
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
			transportVersion := 0
			if sessionSync {
				transportVersion = 1
			}
			value = install.Options{SessionTransportVersion: transportVersion, StateDir: profile.StateDir, NativeHome: profile.NativeHome, NativeBinary: profile.NativeBinary, Binary: binary, Binding: path}
			if *harness == "pi" {
				value = install.PiOptions{Options: value.(install.Options), ReadOnly: memoryReadOnly}
			} else if *harness == "claude-code" {
				value = install.ClaudeOptions{Options: value.(install.Options), ReadOnly: memoryReadOnly}
			}
		}
	case "repair":
		name = "connection_repair_plan"
		value = install.RepairInput{Root: root, NativeBinary: profile.NativeBinary}
	}
	if *harness == "pi" {
		name = "pi_" + name
	} else if *harness == "claude-code" {
		name = "claude_code_" + name
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
		if *harness == "pi" {
			raw, err = unwrapConnectionPlan[install.PiPlan](raw)
		} else if *harness == "claude-code" {
			raw, err = unwrapConnectionPlan[install.ClaudePlan](raw)
		} else {
			raw, err = unwrapPlan(raw)
		}
		if err != nil {
			return bad(out, "Expected a valid plan object or successful connection-plan envelope.")
		}
	}
	a := api.New(nil, *readOnly)
	if sub == "apply" && *harness == "codex" && sessionsStopped {
		var p install.Plan
		if err := strictjson.Decode(raw, &p, api.MaxInputBytes); err != nil {
			return bad(out, "Expected an exact approved connection plan.")
		}
		raw, _ = json.Marshal(install.ApplyInput{Plan: p, SessionsStopped: true})
	}
	if sub == "apply" && *harness == "claude-code" {
		var p install.ClaudePlan
		if err := strictjson.Decode(raw, &p, api.MaxInputBytes); err != nil {
			return bad(out, "Expected an exact approved connection plan.")
		}
		raw, _ = json.Marshal(install.ClaudeApplyInput{Plan: p, SessionsStopped: sessionsStopped})
	}
	result := a.Call(ctx, name, raw)
	if sub == "repair" && applyRepair && result.OK {
		raw, _ = json.Marshal(result.Result)
		if *harness == "codex" && sessionsStopped {
			raw, _ = json.Marshal(install.ApplyInput{Plan: result.Result.(install.Plan), SessionsStopped: true})
		}
		applyName := "connection_apply"
		if *harness == "pi" {
			applyName = "pi_" + applyName
		} else if *harness == "claude-code" {
			applyName = "claude_code_" + applyName
			raw, _ = json.Marshal(install.ClaudeApplyInput{Plan: result.Result.(install.ClaudePlan), SessionsStopped: sessionsStopped})
		}
		result = a.Call(ctx, applyName, raw)
	}
	return emit(out, result)
}
