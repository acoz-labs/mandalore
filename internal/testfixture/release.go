package testfixture

import (
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/distribution"
)

// Release20 is an accepted 2.0 manifest and the seven assets it publishes:
// four CLI targets, install.sh, manifest.json and SHA256SUMS.
func Release20(t testing.TB) (distribution.ParsedManifest, []distribution.ReleaseAsset) {
	t.Helper()
	m := distribution.Manifest{FormatVersion: 1, Product: "mandalore", Version: "2.0.0", Tag: "v2.0.0", SourceCommit: strings.Repeat("a", 40),
		GoVersion: "go1.26.10", ProtocolVersion: 1, SignetReadVersions: []int{1, 2}, SignetWriteVersions: []int{1, 2}}
	for i, target := range [][2]string{{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}} {
		m.Assets = append(m.Assets, distribution.Asset{Kind: "cli", OS: target[0], Arch: target[1], Name: "mandalore_2.0.0_" + target[0] + "_" + target[1], Size: 20, SHA256: strings.Repeat(string(rune('0'+i)), 64)})
	}
	m.Assets = append(m.Assets, distribution.Asset{Kind: "bootstrap", Name: "install.sh", Size: 40, SHA256: strings.Repeat("e", 64)})
	raw, err := distribution.EncodeManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := distribution.ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	sums, err := distribution.Checksums(raw)
	if err != nil {
		t.Fatal(err)
	}
	assets := []distribution.ReleaseAsset{{Name: "manifest.json", Size: int64(len(raw)), SHA256: parsed.SHA256}, {Name: "SHA256SUMS", Size: int64(len(sums)), SHA256: distribution.Digest(sums)}}
	for _, a := range m.Assets {
		assets = append(assets, distribution.ReleaseAsset{Name: a.Name, Size: a.Size, SHA256: a.SHA256})
	}
	return parsed, assets
}
