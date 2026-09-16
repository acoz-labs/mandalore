package readiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"runtime"

	"github.com/acoz-labs/mandalore/internal/install"
	codexplugin "github.com/acoz-labs/mandalore/plugins/codex"
	piplugin "github.com/acoz-labs/mandalore/plugins/pi"
)

type buildKey struct{}
type buildIdentity struct{ version, source string }

// WithBuild is for the CLI entrypoint's linker stamps, not caller JSON. Missing
// context remains unknown. It does not attest an on-disk replacement executable.
func WithBuild(ctx context.Context, version, source string) context.Context {
	if !shortText(version, 128) {
		version = ""
	}
	if !hexText(source, 40) {
		source = ""
	}
	return context.WithValue(ctx, buildKey{}, buildIdentity{version, source})
}

type PackageIdentity struct {
	Harness string `json:"harness"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type Toolkit struct {
	Platform     Platform        `json:"platform"`
	Version      string          `json:"version,omitempty"`
	SourceCommit string          `json:"source_commit,omitempty"`
	Package      PackageIdentity `json:"embedded_package"`
	OnDisk       fileDigest      `json:"on_disk"`
}

type Component struct {
	ID       string          `json:"id"`
	Support  string          `json:"support"`
	Setup    string          `json:"setup"`
	Code     string          `json:"code"`
	Complete bool            `json:"complete"`
	Evidence string          `json:"evidence"`
	Matches  []EvidenceMatch `json:"matches"`
	File     *fileDigest     `json:"file,omitempty"`
}

type Action struct {
	Code    string `json:"code"`
	Summary string `json:"summary"`
}

type Report struct {
	SchemaVersion int                  `json:"schema_version"`
	Complete      bool                 `json:"complete"`
	Harness       string               `json:"harness"`
	Selection     Selection            `json:"selection"`
	Toolkit       Toolkit              `json:"toolkit"`
	Retained      *retainedObservation `json:"retained,omitempty"`
	Declarations  Catalog              `json:"declarations"`
	Components    []Component          `json:"components"`
	Untested      []string             `json:"untested"`
	NextAction    Action               `json:"next_action"`
	Prompt        string               `json:"prompt,omitempty"`
	Notice        string               `json:"notice"`
}

func embeddedPackage(harness string) (PackageIdentity, error) {
	if harness == "pi" {
		info, err := piplugin.Inspect()
		if err != nil {
			return PackageIdentity{}, ErrCatalogInvalid
		}
		return PackageIdentity{harness, info.Version, info.SHA256}, nil
	}
	files := map[string][]byte{}
	total := 0
	err := fs.WalkDir(codexplugin.Files, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := codexplugin.Files.ReadFile(name)
		if err != nil {
			return err
		}
		total += len(raw)
		if total > 1<<20 || len(files) >= 128 {
			return ErrCatalogInvalid
		}
		files[name] = raw
		return nil
	})
	if err != nil {
		return PackageIdentity{}, ErrCatalogInvalid
	}
	var manifest struct{ Name, Version string }
	if json.Unmarshal(files["plugins/mandalore/.codex-plugin/plugin.json"], &manifest) != nil || manifest.Name != "mandalore" || !shortText(manifest.Version, 128) {
		return PackageIdentity{}, ErrCatalogInvalid
	}
	raw, err := json.Marshal(files)
	if err != nil {
		return PackageIdentity{}, ErrCatalogInvalid
	}
	digest := sha256.Sum256(raw)
	return PackageIdentity{harness, manifest.Version, hex.EncodeToString(digest[:])}, nil
}

func observedComponent(id, setup, code string, complete bool) Component {
	return Component{ID: id, Support: "unknown", Setup: setup, Code: code, Complete: complete, Evidence: "none", Matches: []EvidenceMatch{}}
}

func directoryComponent(id, path string) Component {
	st, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return observedComponent(id, "missing", "directory-missing", true)
	}
	if err != nil {
		return observedComponent(id, "unknown", "directory-unreadable", false)
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return observedComponent(id, "inconsistent", "directory-type-invalid", false)
	}
	return observedComponent(id, "present", "directory-present", true)
}

func dependencyComponent(id string) Component {
	selection := executableOnPath(id)
	switch selection.Source {
	case "path-missing":
		return observedComponent(id, "missing", "dependency-missing", true)
	case "path-unresolved":
		return observedComponent(id, "unknown", "dependency-unresolved", false)
	default:
		return observedComponent(id, "present", "dependency-present-version-untested", true)
	}
}

func fileComponent(ctx context.Context, id, path string, limit int64) (Component, fileDigest, error) {
	if path == "" {
		return observedComponent(id, "missing", "executable-missing", true), fileDigest{}, nil
	}
	digest, err := hashFile(ctx, path, limit, true)
	if err != nil {
		if os.IsNotExist(err) {
			return observedComponent(id, "missing", "executable-missing", true), fileDigest{}, nil
		}
		m, err := metadataProblem(err, false)
		return observedComponent(id, m.Setup, m.Code, m.Complete), fileDigest{}, err
	}
	if !digest.Executable || digest.Size == 0 {
		return observedComponent(id, "inconsistent", "executable-mode-or-size-invalid", false), digest, nil
	}
	c := observedComponent(id, "present", "on-disk-bytes-measured", true)
	c.File = &digest
	return c, digest, nil
}

// Assess performs fixed local metadata observations only. It opens no memory
// service, scans no native profile, executes no selected binary and writes no
// state. Missing setup is a successful observation, not an operation failure.
func Assess(ctx context.Context, in Input) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	selection, err := resolveSelection(in)
	if err != nil {
		return Report{}, err
	}
	catalog, err := loadCatalog()
	if err != nil {
		return Report{}, err
	}
	pkg, err := embeddedPackage(in.Harness)
	if err != nil {
		return Report{}, err
	}
	stamp, _ := ctx.Value(buildKey{}).(buildIdentity)
	r := Report{SchemaVersion: 1, Complete: true, Harness: in.Harness, Selection: selection, Declarations: catalog, Components: []Component{},
		Toolkit:  Toolkit{Platform: Platform{runtime.GOOS, runtime.GOARCH}, Version: stamp.version, SourceCommit: stamp.source, Package: pkg},
		Untested: []string{"native version and wrapper/interpreter target", "Git feature support and remote delivery", "native registration, full package/cache and active-session loading", "provider authentication, hook trust and live model behavior", "memory record graph and foundling content", "physical CPU/translation and loaded-image identity"},
		Notice:   "Non-executing assessment-time snapshot, not a readiness lease. Support, observed setup and recorded scenario evidence are independent. On-disk fingerprints do not attest loaded programs. No installation, repair, authentication, memory writes or synchronization performed.",
	}
	add := func(c Component) { r.Components = append(r.Components, c); r.Complete = r.Complete && c.Complete }
	executable, exeErr := os.Executable()
	toolkit := observedComponent("memory-runtime", "unknown", "process-path-unavailable", false)
	if exeErr == nil {
		toolkit, r.Toolkit.OnDisk, err = fileComponent(ctx, "memory-runtime", executable, 128<<20)
		if err != nil {
			return Report{}, err
		}
	}
	toolkit.Support = "unsupported"
	if supportedPlatform(r.Toolkit.Platform) {
		toolkit.Support = "supported"
	}
	if artifactSupport(catalog, r.Toolkit.OnDisk.SHA256, r.Toolkit.Platform) == "unsupported" {
		toolkit.Support = "unsupported"
		toolkit.Code = "runtime-artifact-target-differs"
	}
	applyEvidence(&toolkit, catalog, Observation{Platform: r.Toolkit.Platform, ProcessSource: stamp.source, Identities: Identities{RuntimeSHA256: r.Toolkit.OnDisk.SHA256, PackageSHA256: pkg.SHA256}}, in.Harness)
	add(toolkit)
	git := dependencyComponent("git")
	git.ID = "git-sync"
	add(git)
	native, digest, err := fileComponent(ctx, in.Harness, selection.NativeBinary.Path, 512<<20)
	if err != nil {
		return Report{}, err
	}
	if selection.NativeBinary.Source == "path-unresolved" {
		native = observedComponent(in.Harness, "unknown", "dependency-unresolved", false)
	}
	applyEvidence(&native, catalog, Observation{Platform: r.Toolkit.Platform, ProcessSource: stamp.source, Identities: Identities{RuntimeSHA256: r.Toolkit.OnDisk.SHA256, PackageSHA256: pkg.SHA256, NativeSHA256: digest.SHA256}}, in.Harness)
	add(native)
	if in.Harness == "pi" {
		r.Untested = append(r.Untested, "Node version requirement")
		add(dependencyComponent("node"))
	}
	add(directoryComponent("native-profile", selection.NativeHome.Path))
	add(directoryComponent("installation-state", selection.StateDir.Path))
	bound, err := inspectBinding(ctx, selection.Binding.Path, in.Harness, readMetadata)
	if err != nil {
		return Report{}, err
	}
	add(observedComponent("binding", bound.Setup, bound.Code, bound.Complete))
	if selection.ConnectionRoot.Path != "" {
		nativePath := selection.NativeBinary.Path
		if digest.Path != "" {
			nativePath = digest.Path
		}
		retained, err := inspectRetained(ctx, install.ReceiptSelection{Harness: in.Harness, Root: selection.ConnectionRoot.Path, StateDir: selection.StateDir.Path, NativeHome: selection.NativeHome.Path, NativeBinary: nativePath, Binding: selection.Binding.Path}, digest, bound, readMetadata, hashFile)
		if err != nil {
			return Report{}, err
		}
		r.Retained = &retained
		c := observedComponent("retained-runtime", retained.Setup, retained.Code, retained.Complete)
		if retained.Runtime.SHA256 != "" {
			c.Support = artifactSupport(catalog, retained.Runtime.SHA256, r.Toolkit.Platform)
			for _, e := range catalog.Evidence {
				if e.Component != "memory-runtime" {
					continue
				}
				m := matchEvidence(e, Observation{Platform: r.Toolkit.Platform, Identities: Identities{RuntimeSHA256: retained.Runtime.SHA256}})
				c.Matches = append(c.Matches, m)
			}
			c.Evidence = matchState(c.Matches)
		}
		add(c)
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	r.NextAction = nextAction(r.Components)
	if in.IncludePrompt {
		r.Prompt = assessmentPrompt(r)
	}
	return r, nil
}

func artifactSupport(catalog Catalog, digest string, platform Platform) string {
	known := false
	for _, e := range catalog.Evidence {
		if e.Component != "memory-runtime" || e.Identities.RuntimeSHA256 != digest {
			continue
		}
		known = true
		if e.Platform == platform {
			return "supported"
		}
	}
	if known {
		return "unsupported"
	}
	return "unknown"
}

func applyEvidence(c *Component, catalog Catalog, o Observation, harness string) {
	for _, e := range catalog.Evidence {
		if e.Component != c.ID {
			continue
		}
		m := matchEvidence(e, o)
		// Known embedded-package disagreement must not certify an on-disk
		// runtime as this process's toolkit, even if source stamps are absent.
		for _, related := range catalog.Evidence {
			if related.Component == harness && related.Identities.RuntimeSHA256 == o.Identities.RuntimeSHA256 && related.Identities.PackageSHA256 != "" && related.Identities.PackageSHA256 != o.Identities.PackageSHA256 {
				m.State = "historical"
				m.Reasons = append(m.Reasons, "process-package-differs")
				break
			}
		}
		c.Matches = append(c.Matches, m)
	}
	c.Evidence = matchState(c.Matches)
}

func matchState(matches []EvidenceMatch) string {
	state := "none"
	for _, m := range matches {
		if m.State == "verified" {
			return "verified"
		}
		state = "historical"
	}
	return state
}

func nextAction(components []Component) Action {
	for _, c := range components {
		if c.Setup == "inconsistent" || !c.Complete {
			return Action{"inspect-selection", "Inspect the selected paths and incomplete findings before considering changes."}
		}
	}
	for _, c := range components {
		if c.Support == "unsupported" {
			return Action{"review-support", "Review the unsupported target or artifact selection; do not assume an upgrade is required."}
		}
	}
	for _, c := range components {
		if c.Setup == "missing" && (c.ID == "codex" || c.ID == "pi" || c.ID == "node" || c.ID == "git-sync") {
			return Action{"select-dependencies", "Select or configure the missing dependencies in a separately authorized task."}
		}
	}
	for _, c := range components {
		if c.ID == "binding" && c.Setup != "verified-static" {
			return Action{"inspect-binding", "Confirm the intended signet and inspect or establish its machine-local binding."}
		}
	}
	for _, c := range components {
		if c.Setup == "missing" && (c.ID == "native-profile" || c.ID == "installation-state") {
			return Action{"inspect-connection", "Review the missing profile or installation state before explicitly setting up or checking a native connection."}
		}
	}
	return Action{"verify-native", "If desired, explicitly run native checks; they may execute the selected program and create native logs or cache files."}
}

func assessmentPrompt(r Report) string {
	// Only fixed text, validated harness enum and a fixed-code action enter this
	// guidance. No private paths, receipt strings, raw errors or report replay.
	complete := "Some observations were incomplete."
	if r.Complete {
		complete = "The fixed observations completed; this does not establish overall readiness."
	}
	return "Help me assess Mandalore's " + r.Harness + " memory connection. " + complete + " " + r.NextAction.Summary + " Confirm the intended harness and signet binding with me, then recheck current state. Treat this as an assessment-time snapshot, not permission to install, repair, access credentials, synchronize or change configuration. Use current task authorization for any follow-up. Historical artifact/scenario evidence does not prove this live session works; unknown evidence alone is not a reason to repair or upgrade. Keep ordinary learning behavior unchanged."
}
