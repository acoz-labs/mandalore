// Package launch selects machine-local memory environments. It never installs
// native integrations or synchronizes a signet; native lifecycle hooks own both
// context and foreground transport.
package launch

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

const maxConfig = 256 << 10

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

type Agent struct {
	NativeHome     string `json:"native_home"`
	NativeBinary   string `json:"native_binary"`
	StateDir       string `json:"state_dir"`
	ConnectionRoot string `json:"connection_root"`
}
type Entry struct {
	Binding       string           `json:"binding"`
	SignetID      string           `json:"signet_id,omitempty"`
	BindingSHA256 string           `json:"binding_sha256,omitempty"`
	DefaultAgent  string           `json:"default_agent"`
	Agents        map[string]Agent `json:"agents"`
}
type Config struct {
	SchemaVersion int              `json:"schema_version"`
	Entries       map[string]Entry `json:"entries"`
}
type profileOwner struct {
	SchemaVersion int    `json:"schema_version"`
	SignetID      string `json:"signet_id"`
	Harness       string `json:"harness"`
}

func DefaultPath() (string, error) {
	base, e := os.UserConfigDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(base, "mandalore", "launch.json"), nil
}
func readFile(path string, limit int64) ([]byte, error) {
	st, e := os.Lstat(path)
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() || st.Size() > limit {
		return nil, errors.New("configuration must be a bounded regular file")
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, limit+1))
}
func supported(agent string) bool { return agent == "codex" || agent == "pi" || agent == "claude-code" }
func Load(path string) (Config, error) {
	c := Config{SchemaVersion: 1, Entries: map[string]Entry{}}
	raw, e := readFile(path, maxConfig)
	if e != nil {
		return c, errors.New("launch configuration unavailable; use mandalore launch configure NAME --confirm-profile < entry.json")
	}
	if strictjson.Decode(raw, &c, maxConfig) != nil || c.SchemaVersion != 1 || c.Entries == nil || len(c.Entries) > 128 {
		return Config{}, errors.New("invalid launch configuration; inspect schema_version and named entries")
	}
	for name, entry := range c.Entries {
		if !namePattern.MatchString(name) || validateEntry(entry) != nil {
			return Config{}, errors.New("invalid launch entry; inspect names, default agent and explicit profile paths")
		}
	}
	return c, nil
}
func validateEntry(e Entry) error {
	if !supported(e.DefaultAgent) || len(e.Agents) == 0 || len(e.Agents) > 3 {
		return errors.New("choose a configured default agent: codex, pi or claude-code")
	}
	if _, ok := e.Agents[e.DefaultAgent]; !ok {
		return errors.New("default agent has no configured profile")
	}
	homes := map[string]bool{}
	for harness, a := range e.Agents {
		if homes[a.NativeHome] {
			return errors.New("each agent requires a distinct native profile")
		}
		homes[a.NativeHome] = true
		if !supported(harness) {
			return errors.New("unsupported agent; choose codex, pi or claude-code")
		}
		for _, p := range []string{e.Binding, a.NativeHome, a.NativeBinary, a.StateDir, a.ConnectionRoot} {
			if !filepath.IsAbs(p) || filepath.Clean(p) != p || strings.ContainsAny(p, "\x00\r\n") {
				return errors.New("all binding, executable, profile and connection paths must be explicit absolute paths")
			}
		}
	}
	return nil
}
func selection(e Entry, h string) install.ReceiptSelection {
	a := e.Agents[h]
	return install.ReceiptSelection{Harness: h, Root: a.ConnectionRoot, StateDir: a.StateDir, NativeHome: a.NativeHome, NativeBinary: a.NativeBinary, Binding: e.Binding}
}
func markerPath(home string) string { return filepath.Join(home, ".mandalore-launch-profile.json") }
func checkOwner(home, harness, signet string, allowAbsent bool) error {
	raw, e := readFile(markerPath(home), 4096)
	if allowAbsent && os.IsNotExist(e) {
		return nil
	}
	var owner profileOwner
	if e != nil || strictjson.Decode(raw, &owner, 4096) != nil || owner != (profileOwner{1, signet, harness}) {
		return errors.New("native profile is unclaimed or belongs to another signet/agent; use a dedicated profile and explicitly configure it")
	}
	return nil
}
func claimProfile(home, harness, signet string) error {
	if e := checkOwner(home, harness, signet, true); e != nil {
		return e
	}
	raw, _ := json.Marshal(profileOwner{1, signet, harness})
	f, e := os.CreateTemp(home, ".mandalore-launch-claim-*")
	if e != nil {
		return errors.New("cannot claim native profile")
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, e = f.Write(raw); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Link(f.Name(), markerPath(home)); os.IsExist(e) {
		return checkOwner(home, harness, signet, false)
	}
	return e
}

// Configure explicitly claims each existing profile for this signet. The caller
// must confirm its native history already belongs to that environment. Claims
// are permanent local guards; an interrupted configuration can be retried.
func Configure(path, name string, entry Entry, confirmed bool) (Config, error) {
	return configure(path, name, entry, confirmed, install.InspectLaunch)
}
func configure(path, name string, entry Entry, confirmed bool, inspect func(install.ReceiptSelection) (install.LaunchConnection, error)) (Config, error) {
	if !confirmed {
		return Config{}, errors.New("configuration requires --confirm-profile: confirm these dedicated profiles contain only this signet's native history; no profiles or authentication will be copied")
	}
	if !namePattern.MatchString(name) || name == "configure" || name == "list" || name == "schema" {
		return Config{}, errors.New("choose a non-reserved lowercase environment name")
	}
	if e := validateEntry(entry); e != nil {
		return Config{}, e
	}
	if !filepath.IsAbs(path) {
		return Config{}, errors.New("launch configuration path must be absolute")
	}
	var connections []struct {
		h string
		c install.LaunchConnection
	}
	for h := range entry.Agents {
		c, e := inspect(selection(entry, h))
		if e != nil {
			return Config{}, errors.New("existing connection is not ready; inspect the selected profile with mandalore connection armorer and explicitly connect it")
		}
		if entry.SignetID != "" && entry.SignetID != c.SignetID || entry.BindingSHA256 != "" && entry.BindingSHA256 != c.BindingSHA256 {
			return Config{}, errors.New("configured binding identity differs from the selected connection")
		}
		entry.SignetID, entry.BindingSHA256 = c.SignetID, c.BindingSHA256
		service, e := binding.OpenGuarded(entry.Binding, h, binding.Guard{SHA256: c.BindingSHA256, SignetID: c.SignetID})
		if e != nil {
			return Config{}, e
		}
		// Reuse the binding's symlink-aware separation check without creating a binding.
		if e = binding.ValidateBindingDestination(service.Root(), path); e != nil && !strings.Contains(e.Error(), "binding already exists") {
			return Config{}, errors.New("launcher configuration must remain outside the portable signet")
		}
		if e = checkOwner(c.NativeHome, h, c.SignetID, true); e != nil {
			return Config{}, e
		}
		connections = append(connections, struct {
			h string
			c install.LaunchConnection
		}{h, c})
	}
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return Config{}, e
	}
	lock, e := os.OpenFile(path+".lock", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return Config{}, errors.New("launch configuration is locked; inspect the owner before retrying")
	}
	lock.Close()
	defer os.Remove(path + ".lock")
	config := Config{1, map[string]Entry{}}
	if _, e := os.Lstat(path); e == nil {
		config, e = Load(path)
		if e != nil {
			return Config{}, e
		}
	} else if !os.IsNotExist(e) {
		return Config{}, errors.New("cannot inspect launch configuration")
	}
	for _, c := range connections {
		if e := claimProfile(c.c.NativeHome, c.h, c.c.SignetID); e != nil {
			return Config{}, e
		}
	}
	config.Entries[name] = entry
	raw, e := json.MarshalIndent(config, "", "  ")
	if e != nil || len(raw) > maxConfig {
		return Config{}, errors.New("launch configuration exceeds limit")
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".launch-*")
	if e != nil {
		return Config{}, e
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, e = f.Write(append(raw, '\n')); e != nil {
		return Config{}, e
	}
	if e = f.Sync(); e != nil {
		return Config{}, e
	}
	if e = f.Close(); e != nil {
		return Config{}, e
	}
	if e = os.Rename(f.Name(), path); e != nil {
		return Config{}, e
	}
	return config, nil
}
