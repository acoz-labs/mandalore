// Package distribution owns versioned product assets, not memory semantics or
// native account configuration. A digest proves byte identity, not publisher trust.
package distribution

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

const (
	MaxManifestBytes  = 64 << 10
	MaxBinaryBytes    = 128 << 20
	MaxBootstrapBytes = 64 << 10
)

// Manifest v1 describes exactly four executable targets and the bootstrap.
// Manifest/checksum digests are external to avoid self-referential asset hashes.
type Manifest struct {
	FormatVersion       int     `json:"format_version"`
	Product             string  `json:"product"`
	Version             string  `json:"version"`
	Tag                 string  `json:"tag"`
	SourceCommit        string  `json:"source_commit"`
	GoVersion           string  `json:"go_version"`
	ProtocolVersion     int     `json:"protocol_version"`
	SignetReadVersions  []int   `json:"signet_read_versions"`
	SignetWriteVersions []int   `json:"signet_write_versions"`
	Assets              []Asset `json:"assets"`
}

type Asset struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	OS     string `json:"os,omitempty"`
	Arch   string `json:"arch,omitempty"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// ParsedManifest binds the exact received JSON, not a normalized reconstruction.
// Legacy is set only for a 1.x runtime already retained in an owned prefix.
type ParsedManifest struct {
	Manifest Manifest       `json:"manifest"`
	SHA256   string         `json:"sha256"`
	Legacy   *LegacyRelease `json:"legacy_1x,omitempty"`
}

// LegacyRelease holds the 1.x manifest claims that 2.0 retired (#163). They are
// kept so the exact bytes stay accounted for; nothing installs the plugin.
type LegacyRelease struct {
	PluginSHA256 string `json:"plugin_sha256"`
	CodexPlugin  Asset  `json:"codex_plugin"`
}

// legacyManifest is the 1.x manifest v1 shape: Manifest plus plugin_sha256.
type legacyManifest struct {
	FormatVersion       int     `json:"format_version"`
	Product             string  `json:"product"`
	Version             string  `json:"version"`
	Tag                 string  `json:"tag"`
	SourceCommit        string  `json:"source_commit"`
	GoVersion           string  `json:"go_version"`
	ProtocolVersion     int     `json:"protocol_version"`
	SignetReadVersions  []int   `json:"signet_read_versions"`
	SignetWriteVersions []int   `json:"signet_write_versions"`
	PluginSHA256        string  `json:"plugin_sha256"`
	Assets              []Asset `json:"assets"`
}

const maxLegacyPluginBytes = 4 << 20

func (p ParsedManifest) Identity() string {
	return "mandalore:" + p.Manifest.SourceCommit + ":sha256:" + p.SHA256
}

var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)
var goVersionPattern = regexp.MustCompile(`^go[1-9][0-9]*\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func ValidVersion(v string) bool {
	if len(v) > 96 || !versionPattern.MatchString(v) {
		return false
	}
	base, _, _ := strings.Cut(v, "+")
	_, prerelease, ok := strings.Cut(base, "-")
	if !ok {
		return true
	}
	for _, part := range strings.Split(prerelease, ".") {
		if len(part) > 1 && part[0] == '0' && strings.IndexFunc(part, func(r rune) bool { return r < '0' || r > '9' }) < 0 {
			return false
		}
	}
	return true
}

func validHex(s string, n int) bool {
	if len(s) != n || strings.ToLower(s) != s {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func supportedTarget(goos, arch string) bool {
	return (goos == "darwin" || goos == "linux") && (arch == "amd64" || arch == "arm64")
}

func (m Manifest) Validate() error {
	if m.FormatVersion != 1 || m.Product != "mandalore" {
		return errors.New("unsupported release manifest format or product")
	}
	if !ValidVersion(m.Version) || m.Tag != "v"+m.Version {
		return errors.New("release version and tag must be matching canonical SemVer")
	}
	if !validHex(m.SourceCommit, 40) || len(m.GoVersion) > 32 || !goVersionPattern.MatchString(m.GoVersion) {
		return errors.New("release source or pinned toolchain is invalid")
	}
	// Retain support for immutable format1 releases and the explicit format2
	// extension. Unknown, duplicate or noncanonical claims still fail closed.
	if m.ProtocolVersion != 1 || !supportedSignetVersions(m.SignetReadVersions) || !supportedSignetVersions(m.SignetWriteVersions) {
		return errors.New("release protocol or signet compatibility is unsupported by this installer")
	}
	if len(m.Assets) != 5 {
		return errors.New("release requires four CLI targets and one bootstrap")
	}
	seen := map[string]bool{}
	for _, a := range m.Assets {
		var name string
		var limit int64
		switch a.Kind {
		case "cli":
			if !supportedTarget(a.OS, a.Arch) {
				return errors.New("release CLI target is unsupported")
			}
			name, limit = "mandalore_"+m.Version+"_"+a.OS+"_"+a.Arch, MaxBinaryBytes
		case "bootstrap":
			name, limit = "install.sh", MaxBootstrapBytes
		default:
			return errors.New("release contains an unsupported asset kind")
		}
		if a.Kind != "cli" && (a.OS != "" || a.Arch != "") {
			return errors.New("portable release asset must not select a native target")
		}
		if a.Name != name || seen[name] || a.Size < 1 || a.Size > limit || !validHex(a.SHA256, 64) {
			return errors.New("release asset name, uniqueness, size or digest is invalid")
		}
		seen[name] = true
	}
	// Exactly five unique allowed names implies the complete fixed v1 matrix.
	return nil
}

func supportedSignetVersions(v []int) bool {
	return (len(v) == 1 && v[0] == 1) || (len(v) == 2 && v[0] == 1 && v[1] == 2)
}

func ParseManifest(data []byte) (ParsedManifest, error) {
	var m Manifest
	if err := strictjson.Decode(data, &m, MaxManifestBytes); err != nil {
		return ParsedManifest{}, err
	}
	if err := m.Validate(); err != nil {
		return ParsedManifest{}, err
	}
	return ParsedManifest{Manifest: m, SHA256: Digest(data)}, nil
}

// parseRetainedManifest reads the manifest of a runtime already retained in an
// owned prefix, so 2.0 can update or roll back a 1.x install in place. Besides
// the current shape it accepts only a 1.x release with its plugin_sha256 and one
// codex-plugin asset; every other rule still applies. New candidates and
// published releases use ParseManifest alone.
func parseRetainedManifest(data []byte) (ParsedManifest, error) {
	if p, err := ParseManifest(data); err == nil {
		return p, nil
	}
	var l legacyManifest
	if err := strictjson.Decode(data, &l, MaxManifestBytes); err != nil {
		return ParsedManifest{}, err
	}
	m := Manifest{FormatVersion: l.FormatVersion, Product: l.Product, Version: l.Version, Tag: l.Tag, SourceCommit: l.SourceCommit, GoVersion: l.GoVersion,
		ProtocolVersion: l.ProtocolVersion, SignetReadVersions: l.SignetReadVersions, SignetWriteVersions: l.SignetWriteVersions}
	var codex []Asset
	for _, a := range l.Assets {
		if a.Kind == "codex-plugin" {
			codex = append(codex, a)
		} else {
			m.Assets = append(m.Assets, a)
		}
	}
	if !strings.HasPrefix(m.Version, "1.") || !validHex(l.PluginSHA256, 64) || len(codex) != 1 {
		return ParsedManifest{}, errors.New("retained manifest is neither a current nor a 1.x release manifest")
	}
	c := codex[0]
	if c.Name != "mandalore_"+m.Version+"_codex.zip" || c.OS != "" || c.Arch != "" || c.Size < 1 || c.Size > maxLegacyPluginBytes || !validHex(c.SHA256, 64) {
		return ParsedManifest{}, errors.New("retained 1.x Codex package claim is invalid")
	}
	if err := m.Validate(); err != nil {
		return ParsedManifest{}, err
	}
	return ParsedManifest{Manifest: m, SHA256: Digest(data), Legacy: &LegacyRelease{PluginSHA256: l.PluginSHA256, CodexPlugin: c}}, nil
}

func EncodeManifest(m Manifest) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (m Manifest) Binary(goos, arch string) (Asset, error) {
	if err := m.Validate(); err != nil {
		return Asset{}, err
	}
	for _, a := range m.Assets {
		if a.Kind == "cli" && a.OS == goos && a.Arch == arch {
			return a, nil
		}
	}
	return Asset{}, fmt.Errorf("no supported release binary for target %q/%q", goos, arch)
}
