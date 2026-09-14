package distribution

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func fixtureManifest() Manifest {
	m := Manifest{
		FormatVersion: 1, Product: "mandalore", Version: "1.0.0", Tag: "v1.0.0",
		SourceCommit: strings.Repeat("a", 40), GoVersion: "go1.26.4",
		ProtocolVersion: 1, SignetReadVersions: []int{1}, SignetWriteVersions: []int{1},
		PluginSHA256: strings.Repeat("b", 64),
	}
	for _, target := range [][2]string{{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}} {
		m.Assets = append(m.Assets, Asset{Kind: "cli", OS: target[0], Arch: target[1],
			Name: "mandalore_1.0.0_" + target[0] + "_" + target[1], Size: 20, SHA256: strings.Repeat("c", 64)})
	}
	m.Assets = append(m.Assets,
		Asset{Kind: "codex-plugin", Name: "mandalore_1.0.0_codex.zip", Size: 30, SHA256: strings.Repeat("d", 64)},
		Asset{Kind: "bootstrap", Name: "install.sh", Size: 40, SHA256: strings.Repeat("e", 64)})
	return m
}

func TestManifestRoundTripBindsExactBytes(t *testing.T) {
	m := fixtureManifest()
	b, err := EncodeManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseManifest(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Manifest, m) {
		t.Fatalf("round trip changed manifest: %+v", got.Manifest)
	}
	sum := sha256.Sum256(b)
	want := hex.EncodeToString(sum[:])
	if got.SHA256 != want || got.Identity() != "mandalore:"+m.SourceCommit+":sha256:"+want {
		t.Fatalf("candidate identity does not bind exact source and bytes: %+v", got)
	}
	other, err := ParseManifest(append(b, '\n'))
	if err != nil {
		t.Fatal(err)
	}
	if other.SHA256 == got.SHA256 {
		t.Fatal("different valid manifest bytes shared a candidate identity")
	}
	again, err := EncodeManifest(m)
	if err != nil || !bytes.Equal(b, again) {
		t.Fatal("encoding is not deterministic")
	}
}

func TestManifestRejectsInvalidContracts(t *testing.T) {
	cases := map[string]func(*Manifest){
		"format":                func(m *Manifest) { m.FormatVersion = 2 },
		"product":               func(m *Manifest) { m.Product = "my-friday" },
		"short version":         func(m *Manifest) { m.Version = "1.0" },
		"leading zero":          func(m *Manifest) { m.Version = "01.0.0" },
		"tag mismatch":          func(m *Manifest) { m.Tag = "v9.0.0" },
		"version controls":      func(m *Manifest) { m.Version = "1.0.0\n" },
		"version traversal":     func(m *Manifest) { m.Version = "1.0.0/../../x" },
		"source abbreviated":    func(m *Manifest) { m.SourceCommit = "abcdef0" },
		"source uppercase":      func(m *Manifest) { m.SourceCommit = strings.Repeat("A", 40) },
		"source not hex":        func(m *Manifest) { m.SourceCommit = strings.Repeat("z", 40) },
		"toolchain unpinned":    func(m *Manifest) { m.GoVersion = "latest" },
		"protocol incompatible": func(m *Manifest) { m.ProtocolVersion = 2 },
		"read incompatible":     func(m *Manifest) { m.SignetReadVersions = []int{2} },
		"write incompatible":    func(m *Manifest) { m.SignetWriteVersions = []int{2} },
		"schema duplicate":      func(m *Manifest) { m.SignetReadVersions = []int{1, 1} },
		"schema missing":        func(m *Manifest) { m.SignetWriteVersions = nil },
		"plugin digest":         func(m *Manifest) { m.PluginSHA256 = "invalid" },
		"missing target":        func(m *Manifest) { m.Assets = m.Assets[1:] },
		"duplicate asset":       func(m *Manifest) { m.Assets = append(m.Assets, m.Assets[0]) },
		"duplicate target":      func(m *Manifest) { m.Assets[1] = m.Assets[0] },
		"renamed binary":        func(m *Manifest) { m.Assets[0].Name = "mandalore" },
		"foreign target":        func(m *Manifest) { m.Assets[0].OS = "windows" },
		"foreign arch":          func(m *Manifest) { m.Assets[0].Arch = "386" },
		"traversal asset":       func(m *Manifest) { m.Assets[0].Name = "../mandalore" },
		"absolute asset":        func(m *Manifest) { m.Assets[0].Name = "/mandalore" },
		"controls asset":        func(m *Manifest) { m.Assets[0].Name += "\x1b[0m" },
		"zero size":             func(m *Manifest) { m.Assets[0].Size = 0 },
		"negative size":         func(m *Manifest) { m.Assets[0].Size = -1 },
		"oversize binary":       func(m *Manifest) { m.Assets[0].Size = 128<<20 + 1 },
		"oversize plugin":       func(m *Manifest) { m.Assets[4].Size = 4<<20 + 1 },
		"oversize bootstrap":    func(m *Manifest) { m.Assets[5].Size = 64<<10 + 1 },
		"asset digest":          func(m *Manifest) { m.Assets[0].SHA256 = strings.Repeat("A", 64) },
		"unknown kind":          func(m *Manifest) { m.Assets[4].Kind = "script" },
		"plugin platform":       func(m *Manifest) { m.Assets[4].OS = "linux" },
		"bootstrap arch":        func(m *Manifest) { m.Assets[5].Arch = "arm64" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			m := fixtureManifest()
			change(&m)
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseManifest(b); err == nil {
				t.Fatal("invalid manifest accepted")
			}
			if _, err := EncodeManifest(m); err == nil {
				t.Fatal("invalid manifest encoded")
			}
		})
	}
}

func TestManifestStrictBoundedJSON(t *testing.T) {
	b, err := json.Marshal(fixtureManifest())
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{
		"unknown":          append([]byte(`{"unexpected":1,`), b[1:]...),
		"duplicate":        append([]byte(`{"product":"mandalore",`), b[1:]...),
		"nested duplicate": bytes.Replace(b, []byte(`"size":20`), []byte(`"size":20,"size":20`), 1),
		"trailing object":  append(append([]byte{}, b...), []byte(`{}`)...),
		"oversized":        append(append([]byte{}, b...), bytes.Repeat([]byte(" "), 64<<10)...),
		"invalid utf8":     append(append([]byte{}, b...), 0xff),
		"empty":            nil,
		"null":             []byte("null"),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseManifest(data); err == nil {
				t.Fatal("invalid JSON accepted")
			}
		})
	}
}

func TestCanonicalReleaseVersions(t *testing.T) {
	for _, v := range []string{"1.0.0", "0.1.0", "1.2.3-rc.1", "1.2.3-alpha-beta.0", "1.2.3+build.01", "1.2.3-rc.1+build.2"} {
		if !ValidVersion(v) {
			t.Errorf("valid version rejected: %q", v)
		}
	}
	for _, v := range []string{"", "v1.2.3", "1.2", "01.2.3", "1.02.3", "1.2.03", "1.2.3-01", "1.2.3-rc..1", "1.2.3+", "1.2.3-", "1.2.3\n", "1.2.3/evil", "1.2.3-" + strings.Repeat("a", 96)} {
		if ValidVersion(v) {
			t.Errorf("invalid version accepted: %q", v)
		}
	}
}

func TestManifestAssetSelectionNeverFallsBack(t *testing.T) {
	m := fixtureManifest()
	a, err := m.Binary("linux", "arm64")
	if err != nil || a.Name != "mandalore_1.0.0_linux_arm64" {
		t.Fatalf("wrong native asset: %+v %v", a, err)
	}
	for _, target := range [][2]string{{"windows", "amd64"}, {"linux", "386"}, {"", ""}} {
		if _, err := m.Binary(target[0], target[1]); err == nil {
			t.Fatal("unsupported target silently substituted")
		}
	}
}
