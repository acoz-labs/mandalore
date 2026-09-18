package readiness

import (
	_ "embed"
	"encoding/hex"
	"errors"
	"net/url"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// ErrCatalogInvalid never includes raw declaration contents.
var ErrCatalogInvalid = errors.New("readiness declarations are invalid")

//go:embed catalog.json
var catalogJSON []byte

type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type Dependency struct {
	ID         string `json:"id"`
	Constraint string `json:"constraint"`
}

type Declaration struct {
	ID             string       `json:"id"`
	Dependencies   []Dependency `json:"dependencies"`
	NativeContract string       `json:"native_contract,omitempty"`
}

type Identities struct {
	RuntimeSHA256      string `json:"runtime_sha256,omitempty"`
	PackageSHA256      string `json:"package_sha256,omitempty"`
	NativeSHA256       string `json:"native_sha256,omitempty"`
	NativeVersion      string `json:"native_version,omitempty"`
	InterpreterSHA256  string `json:"interpreter_sha256,omitempty"`
	InterpreterVersion string `json:"interpreter_version,omitempty"`
}

type Evidence struct {
	ID             string     `json:"id"`
	Component      string     `json:"component"`
	Scenario       string     `json:"scenario"`
	Platform       Platform   `json:"platform"`
	SourceCommit   string     `json:"source_commit"`
	ManifestSHA256 string     `json:"manifest_sha256"`
	Identities     Identities `json:"identities"`
	Level          string     `json:"level"`
	Sources        []string   `json:"sources"`
}

type Catalog struct {
	SchemaVersion       int           `json:"schema_version"`
	Targets             []Platform    `json:"targets"`
	SignetReadVersions  []int         `json:"signet_read_versions"`
	SignetWriteVersions []int         `json:"signet_write_versions"`
	MemoryProtocol      int           `json:"memory_protocol"`
	CodexHookProtocol   int           `json:"codex_hook_protocol"`
	PiHarnessProtocol   int           `json:"pi_harness_protocol"`
	Components          []Declaration `json:"components"`
	Evidence            []Evidence    `json:"evidence"`
}

var declarationID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,95}$`)
var exactVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+([+-][a-zA-Z0-9.-]+)?$`)

func loadCatalog() (Catalog, error) { return decodeCatalog(catalogJSON) }

func decodeCatalog(raw []byte) (Catalog, error) {
	var c Catalog
	if strictjson.Decode(raw, &c, 64<<10) != nil || c.SchemaVersion != 1 ||
		c.MemoryProtocol != 1 || c.CodexHookProtocol != 1 || c.PiHarnessProtocol != 1 ||
		len(c.Targets) != 4 || len(c.Components) != 4 || len(c.Evidence) < 1 || len(c.Evidence) > 32 ||
		len(c.SignetReadVersions) != 2 || c.SignetReadVersions[0] != 1 || c.SignetReadVersions[1] != 2 ||
		len(c.SignetWriteVersions) != 2 || c.SignetWriteVersions[0] != 1 || c.SignetWriteVersions[1] != 2 {
		return Catalog{}, ErrCatalogInvalid
	}
	targets := map[Platform]bool{}
	for _, p := range c.Targets {
		if !supportedPlatform(p) || targets[p] {
			return Catalog{}, ErrCatalogInvalid
		}
		targets[p] = true
	}
	components := map[string]bool{}
	for _, d := range c.Components {
		if !knownComponent(d.ID) || components[d.ID] || len(d.Dependencies) > 8 ||
			(d.NativeContract != "" && !exactVersion.MatchString(d.NativeContract)) {
			return Catalog{}, ErrCatalogInvalid
		}
		components[d.ID] = true
		deps := map[string]bool{}
		for _, r := range d.Dependencies {
			if !declarationID.MatchString(r.ID) || deps[r.ID] || !shortText(r.Constraint, 256) {
				return Catalog{}, ErrCatalogInvalid
			}
			deps[r.ID] = true
		}
	}
	ids := map[string]bool{}
	for _, e := range c.Evidence {
		if !declarationID.MatchString(e.ID) || ids[e.ID] || !components[e.Component] || !targets[e.Platform] ||
			!validScenario(e.Component, e.Scenario) || !hexText(e.SourceCommit, 40) || !hexText(e.ManifestSHA256, 64) ||
			!hexText(e.Identities.RuntimeSHA256, 64) || (e.Level != "engineering" && e.Level != "accepted-release") ||
			len(e.Sources) < 1 || len(e.Sources) > 3 {
			return Catalog{}, ErrCatalogInvalid
		}
		ids[e.ID] = true
		for _, h := range []string{e.Identities.PackageSHA256, e.Identities.NativeSHA256, e.Identities.InterpreterSHA256} {
			if h != "" && !hexText(h, 64) {
				return Catalog{}, ErrCatalogInvalid
			}
		}
		for _, v := range []string{e.Identities.NativeVersion, e.Identities.InterpreterVersion} {
			if len(v) > 64 || v != "" && !exactVersion.MatchString(v) {
				return Catalog{}, ErrCatalogInvalid
			}
		}
		for _, source := range e.Sources {
			if !evidenceURL(source) {
				return Catalog{}, ErrCatalogInvalid
			}
		}
	}
	return c, nil
}

func supportedPlatform(p Platform) bool {
	return (p.OS == "darwin" || p.OS == "linux") && (p.Arch == "amd64" || p.Arch == "arm64")
}

func knownComponent(id string) bool {
	return id == "memory-runtime" || id == "git-sync" || id == "codex" || id == "pi"
}

func validScenario(component, scenario string) bool {
	return component == "memory-runtime" && scenario == "cli-mcp" ||
		component == "codex" && scenario == "native-recall" ||
		component == "pi" && scenario == "native-setup-tools"
}

func hexText(value string, n int) bool {
	if len(value) != n || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func shortText(value string, max int) bool {
	return strings.TrimSpace(value) != "" && len(value) <= max && strings.IndexFunc(value, unicode.IsControl) < 0
}

func evidenceURL(raw string) bool {
	if !shortText(raw, 512) {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.RawPath != "" || path.Clean(u.Path) != u.Path {
		return false
	}
	parts := strings.Split(u.Path, "/")
	return len(parts) >= 7 && parts[1] == "acoz-labs" && parts[2] == "mandalore" && parts[3] == "blob" && hexText(parts[4], 40) && parts[5] == "docs"
}
