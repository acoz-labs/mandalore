package readiness

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/install"
	piplugin "github.com/acoz-labs/mandalore/plugins/pi"
)

func TestCatalogKeepsSupportDependenciesAndEvidenceSeparate(t *testing.T) {
	c, err := loadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if c.SchemaVersion != 1 || len(c.Targets) != 4 || len(c.Components) != 4 || len(c.Evidence) != 6 {
		t.Fatalf("unexpected declaration shape: %+v", c)
	}
	if c.MemoryProtocol != 1 || c.CodexHookProtocol != 1 || c.PiHarnessProtocol != 1 {
		t.Fatal("protocol declarations drifted")
	}
	files, err := piplugin.PackageFiles()
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct{ Engines map[string]string }
	if err := json.Unmarshal(files["package.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	for _, d := range c.Components {
		if d.ID == "pi" {
			if d.NativeContract != install.PiNativeVersion {
				t.Fatal("Pi contract differs from actual installer")
			}
			found := false
			for _, dep := range d.Dependencies {
				if dep.ID == "node" && dep.Constraint == manifest.Engines["node"] {
					found = true
				}
			}
			if !found {
				t.Fatal("Node requirement differs from embedded package")
			}
		}
	}
	for _, e := range c.Evidence {
		if e.Component == "pi" && e.Level != "engineering" {
			t.Fatal("Pi engineering evidence was promoted to acceptance")
		}
	}
}

func TestCatalogRuntimeEvidenceMatchesPublishedBytes(t *testing.T) {
	raw, err := os.ReadFile("../../docs/releases/1.0.0/public-verification.json")
	if err != nil {
		t.Fatal(err)
	}
	var published struct {
		SourceCommit string `json:"source_commit"`
		Assets       []struct{ Name, SHA256 string }
	}
	if err := json.Unmarshal(raw, &published); err != nil {
		t.Fatal(err)
	}
	c, err := loadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range c.Evidence {
		if e.Component != "memory-runtime" {
			continue
		}
		count++
		matched := false
		for _, a := range published.Assets {
			if a.Name == "mandalore_1.0.0_"+e.Platform.OS+"_"+e.Platform.Arch && a.SHA256 == e.Identities.RuntimeSHA256 && e.SourceCommit == published.SourceCommit {
				matched = true
			}
		}
		if !matched || e.Level != "accepted-release" || e.Scenario != "cli-mcp" {
			t.Fatal("catalog runtime evidence differs from published identity/scope", e.ID)
		}
	}
	if count != 4 {
		t.Fatal("four actually executed CLI/MCP platforms must be retained")
	}
}

func TestEvidenceMatcherRequiresEveryScenarioIdentity(t *testing.T) {
	hash := strings.Repeat("a", 64)
	e := Evidence{ID: "synthetic", Component: "pi", Scenario: "native-setup-tools", Platform: Platform{OS: "darwin", Arch: "arm64"},
		Identities: Identities{RuntimeSHA256: hash, PackageSHA256: hash, NativeSHA256: hash, NativeVersion: "0.85.1", InterpreterSHA256: hash, InterpreterVersion: "24.1.0"}}
	full := Observation{Platform: e.Platform, Identities: e.Identities}
	if got := matchEvidence(e, full); got.State != "verified" || len(got.Reasons) != 0 {
		t.Fatal("complete exact identity did not match", got)
	}
	for _, field := range []string{"runtime", "package", "native-hash", "native-version", "interpreter-hash", "interpreter-version", "platform"} {
		t.Run(field, func(t *testing.T) {
			o := full
			switch field {
			case "runtime":
				o.Identities.RuntimeSHA256 = ""
			case "package":
				o.Identities.PackageSHA256 = ""
			case "native-hash":
				o.Identities.NativeSHA256 = ""
			case "native-version":
				o.Identities.NativeVersion = ""
			case "interpreter-hash":
				o.Identities.InterpreterSHA256 = ""
			case "interpreter-version":
				o.Identities.InterpreterVersion = ""
			case "platform":
				o.Platform.OS = "linux"
			}
			if got := matchEvidence(e, o); got.State != "historical" || len(got.Reasons) == 0 {
				t.Fatal("incomplete/different identity became proof", got)
			}
		})
	}
	e.Identities.NativeSHA256 = ""
	if got := matchEvidence(e, full); got.State != "historical" {
		t.Fatal("incomplete recorded evidence became a wildcard", got)
	}
	e.Identities.NativeSHA256 = hash // Isolate the different-runtime assertion.
	full.Identities.RuntimeSHA256 = strings.Repeat("b", 64)
	if got := matchEvidence(e, full); got.State != "historical" {
		t.Fatal("same version with different bytes matched", got)
	}
}

func TestPublishedRuntimeMatchDoesNotCertifyNativeOrAnotherProcess(t *testing.T) {
	c, err := loadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range c.Evidence {
		o := Observation{Platform: e.Platform, Identities: e.Identities, ProcessSource: e.SourceCommit}
		got := matchEvidence(e, o)
		if e.Component != "memory-runtime" {
			if got.State != "historical" || len(got.Reasons) == 0 {
				t.Fatal("partial historical native identities were treated as complete", got)
			}
			continue
		}
		if got.State != "verified" {
			t.Fatal("exact published runtime bytes/platform should match its recorded scenario", got)
		}
		o.ProcessSource = strings.Repeat("b", 40)
		if got := matchEvidence(e, o); got.State != "historical" {
			t.Fatal("known process/source disagreement was ignored", got)
		}
		o.ProcessSource = e.SourceCommit
		o.Identities.RuntimeSHA256 = ""
		if got := matchEvidence(e, o); got.State != "historical" {
			t.Fatal("source label without artifact bytes became evidence", got)
		}
	}
}

func TestCatalogRejectsCorruptOrUnboundedDeclarations(t *testing.T) {
	c, err := loadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Catalog){
		func(c *Catalog) { c.SchemaVersion = 2 },
		func(c *Catalog) { c.Components = append(c.Components, c.Components[0]) },
		func(c *Catalog) { c.Evidence = append(c.Evidence, c.Evidence[0]) },
		func(c *Catalog) { c.Evidence[0].Identities.RuntimeSHA256 = "not-a-digest" },
		func(c *Catalog) { c.Evidence[0].Sources = []string{"https://example.invalid/mutable"} },
	} {
		raw, _ := json.Marshal(c)
		var changed Catalog
		if err := json.Unmarshal(raw, &changed); err != nil {
			t.Fatal(err)
		}
		change(&changed)
		raw, _ = json.Marshal(changed)
		if _, err := decodeCatalog(raw); err == nil {
			t.Fatal("accepted invalid catalog")
		}
	}
	for _, raw := range [][]byte{[]byte(`{"schema_version":1,"schema_version":1}`), make([]byte, (64<<10)+1)} {
		if _, err := decodeCatalog(raw); err == nil {
			t.Fatal("accepted duplicate/oversized catalog")
		}
	}
}
