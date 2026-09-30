package launch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"syscall"

	"github.com/acoz-labs/mandalore/internal/install"
)

type Plan struct {
	SchemaVersion    int      `json:"schema_version"`
	Name             string   `json:"name"`
	Agent            string   `json:"agent"`
	SignetID         string   `json:"signet_id"`
	Access           string   `json:"memory_access"`
	Executable       string   `json:"executable"`
	NativeHome       string   `json:"native_home"`
	Arguments        []string `json:"arguments"`
	WorkingDirectory string   `json:"working_directory"`
	Notice           string   `json:"notice"`
	connection       install.LaunchConnection
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateArguments rejects alternate configuration and imported session paths;
// normal native prompts/options retain their exact byte boundaries.
func ValidateArguments(harness string, args []string) error {
	denied := map[string]bool{}
	switch harness {
	case "codex":
		for _, v := range []string{"--ignore-user-config", "-c", "--config", "-p", "--profile", "-C", "--cd", "--remote", "--remote-env", "--enable", "--disable"} {
			denied[v] = true
		}
	case "pi":
		for _, v := range []string{"--fork", "-ne", "-ns", "-np", "--session", "--session-dir", "--extension", "-e", "--no-extensions", "--package", "--no-skills", "--skill", "--no-prompt-templates", "--prompt-template"} {
			denied[v] = true
		}
	case "claude-code":
		for _, v := range []string{"--restricted", "--safe-mode", "--bare", "--plugin-url", "--settings", "--setting-sources", "--plugin-dir", "--mcp-config", "--strict-mcp-config", "--add-dir", "--agents", "--agent", "--disable-slash-commands", "--worktree", "-w"} {
			denied[v] = true
		}
	default:
		return errors.New("unsupported agent")
	}
	for i, arg := range args {
		if arg == "--" {
			break
		}
		flag, _, _ := strings.Cut(arg, "=")
		blocked := denied[flag]
		if harness == "codex" {
			for _, prefix := range []string{"-c", "-p", "-C"} {
				if strings.HasPrefix(arg, prefix) && !strings.HasPrefix(arg, "--") {
					blocked = true
				}
			}
		}
		if harness == "pi" && strings.HasPrefix(arg, "-e") && !strings.HasPrefix(arg, "--") {
			blocked = true
		}
		if blocked {
			return errors.New("native argument overrides the selected memory/profile environment; configure a separate entry instead")
		}
		if (harness == "codex" && (arg == "resume" || arg == "fork")) || (harness == "claude-code" && (flag == "--resume" || flag == "-r")) || (harness == "pi" && flag == "--session-id") {
			value, has := strings.CutPrefix(arg, flag+"=")
			if !has && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
				has = true
			}
			if has && !uuidPattern.MatchString(value) {
				return errors.New("resume requires a native UUID from the selected profile; imported paths and named foreign sessions are refused")
			}
		}
	}
	return nil
}

func Resolve(config Config, name, agent string, args []string) (Plan, error) {
	return resolve(config, name, agent, args, install.InspectLaunch)
}
func resolve(config Config, name, agent string, args []string, inspect func(install.ReceiptSelection) (install.LaunchConnection, error)) (Plan, error) {
	e, ok := config.Entries[name]
	if !ok {
		return Plan{}, errors.New("unknown launch environment; inspect mandalore launch list or configure this name")
	}
	if agent == "" {
		agent = e.DefaultAgent
	}
	if !supported(agent) {
		return Plan{}, errors.New("unsupported agent; choose codex, pi or claude-code")
	}
	a, ok := e.Agents[agent]
	if !ok {
		return Plan{}, errors.New("agent has no configured profile in this environment; explicitly configure it first")
	}
	if err := ValidateArguments(agent, args); err != nil {
		return Plan{}, err
	}
	c, err := inspect(selection(e, agent))
	if err != nil {
		return Plan{}, fmt.Errorf("selected connection validation failed: %w", err)
	}
	if c.SignetID != e.SignetID || c.BindingSHA256 != e.BindingSHA256 {
		return Plan{}, errors.New("binding identity changed; explicitly reconnect and reconfigure instead of selecting another signet")
	}
	if err = checkOwner(a.NativeHome, agent, e.SignetID, false); err != nil {
		return Plan{}, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return Plan{}, errors.New("caller working directory is unavailable")
	}
	access := "read/write"
	if c.ReadOnly {
		access = "read-only"
	}
	executable := c.Executable
	if executable == "" {
		executable = a.NativeBinary
	}
	return Plan{1, name, agent, c.SignetID, access, executable, a.NativeHome, append([]string{}, args...), cwd, "Selection verified locally. Native registration is checked at launch; native hooks report synchronization and canon freshness. This is not an OS sandbox; workspace files and native authentication remain native-owned.", c}, nil
}

func (p Plan) Environment(in []string) []string {
	overrides := map[string]string{"MANDALORE_BINDING": p.connection.Binding, "MANDALORE_BIN": p.connection.Runtime}
	switch p.Agent {
	case "codex":
		overrides["CODEX_HOME"] = p.NativeHome
	case "pi":
		overrides["PI_CODING_AGENT_DIR"] = p.NativeHome
	case "claude-code":
		overrides["CLAUDE_CONFIG_DIR"] = p.NativeHome
	}
	result := []string{}
	for _, item := range in {
		key, _, _ := strings.Cut(item, "=")
		if strings.HasPrefix(key, "MANDALORE_") || key == "CODEX_HOME" || key == "PI_CODING_AGENT_DIR" || key == "CLAUDE_CONFIG_DIR" || key == "CODEX_SQLITE_HOME" || key == "CLAUDE_CODE_SAFE_MODE" || key == "CLAUDE_CODE_PLUGIN_DIRS" || key == "CLAUDE_CODE_PLUGIN_CACHE_DIR" || key == "CLAUDE_CODE_PLUGIN_SEED_DIR" {
			continue
		}
		result = append(result, item)
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}

// Execute replaces the launcher, preserving terminal ownership, caller cwd,
// process ID, signal delivery and exact native exit status. No shell is involved.
func (p Plan) Execute(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := install.ValidateLaunchNative(ctx, p.Agent, p.connection); err != nil {
		return err
	}
	// Recheck immutable connection bytes after native inventory and before exec.
	e := Entry{Binding: p.connection.Binding, Agents: map[string]Agent{p.Agent: {NativeHome: p.NativeHome, NativeBinary: p.connection.NativeBinary, StateDir: p.connection.StateDir, ConnectionRoot: p.connection.Root}}}
	c, err := install.InspectLaunch(selection(e, p.Agent))
	if err != nil || c != p.connection {
		return errors.New("connection changed during launch; inspect and retry")
	}
	if err = checkOwner(p.NativeHome, p.Agent, p.SignetID, false); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = replaceProcess(p.Executable, p.Arguments, p.Environment(os.Environ())); err != nil {
		return errors.New("native executable could not start; inspect executable/interpreter permissions and availability")
	}
	return nil
}

func replaceProcess(executable string, args, env []string) error {
	return syscall.Exec(executable, append([]string{executable}, args...), env)
}
