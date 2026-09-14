package distribution

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

const (
	pluginManifestPath = "plugins/mandalore/.codex-plugin/plugin.json"
	marketplacePath    = ".agents/plugins/marketplace.json"
	maxPluginFiles     = 128
)

type PluginPackage struct {
	Files   map[string][]byte
	Archive []byte
	SHA256  string
}

func validPluginPath(name string) bool {
	return len(name) <= 512 && fs.ValidPath(name) && !strings.Contains(name, "\\") &&
		strings.IndexFunc(name, unicode.IsControl) < 0 &&
		(name == marketplacePath || strings.HasPrefix(name, "plugins/mandalore/"))
}

func pluginMode(name string) fs.FileMode {
	if strings.HasPrefix(name, "plugins/mandalore/scripts/") && strings.HasSuffix(name, ".sh") {
		return 0755
	}
	return 0644
}

// fs.WalkDir follows a symlink passed as its starting root. Inspect every root
// component through its parent entries before walking, including .agents parents.
// The release builder must additionally supply an isolated, symlink-free tracked export.
func checkPluginRoot(source fs.FS, root string) error {
	parts := strings.Split(root, "/")
	parent := "."
	for i, part := range parts {
		entries, err := fs.ReadDir(source, parent)
		if err != nil {
			return err
		}
		found := false
		for _, entry := range entries {
			if entry.Name() != part {
				continue
			}
			found = true
			wantDir := i < len(parts)-1 || root == "plugins/mandalore"
			if entry.Type()&fs.ModeSymlink != 0 || entry.IsDir() != wantDir || (!wantDir && !entry.Type().IsRegular()) {
				return errors.New("plugin embed root must not be redirected or have an unexpected type")
			}
			break
		}
		if !found {
			return errors.New("plugin embed root is missing")
		}
		if parent == "." {
			parent = part
		} else {
			parent += "/" + part
		}
	}
	return nil
}

func pluginMetadata(files map[string][]byte, version string) (map[string]any, error) {
	var p, market map[string]any
	if err := strictjson.Decode(files[pluginManifestPath], &p, MaxBootstrapBytes); err != nil {
		return nil, err
	}
	if p["name"] != "mandalore" || p["version"] != version {
		return nil, errors.New("plugin manifest identity does not match the release")
	}
	if err := strictjson.Decode(files[marketplacePath], &market, MaxBootstrapBytes); err != nil {
		return nil, err
	}
	entries, ok := market["plugins"].([]any)
	if !ok || market["name"] != "mandalore" || len(entries) != 1 {
		return nil, errors.New("release marketplace must contain only the mandalore plugin")
	}
	entry, ok := entries[0].(map[string]any)
	if !ok || entry["name"] != "mandalore" {
		return nil, errors.New("invalid release marketplace plugin identity")
	}
	source, ok := entry["source"].(map[string]any)
	if !ok || source["source"] != "local" || source["path"] != "./plugins/mandalore" {
		return nil, errors.New("release marketplace source must use its packaged plugin")
	}
	return p, nil
}

// PreparePlugin reads only the public embed roots and stamps a private in-memory
// copy. Its Files are the exact input for the isolated release-source export.
// Archive metadata is normalized; map JSON matches install's package identity.
func PreparePlugin(source fs.FS, version string) (PluginPackage, error) {
	if !ValidVersion(version) {
		return PluginPackage{}, errors.New("invalid plugin release version")
	}
	files := map[string][]byte{}
	total := 0
	for _, root := range []string{marketplacePath, "plugins/mandalore"} {
		if err := checkPluginRoot(source, root); err != nil {
			return PluginPackage{}, err
		}
		err := fs.WalkDir(source, root, func(name string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !validPluginPath(name) || !d.Type().IsRegular() || len(files) >= maxPluginFiles {
				return errors.New("invalid plugin input path, type or count")
			}
			f, err := source.Open(name)
			if err != nil {
				return err
			}
			b, readErr := io.ReadAll(io.LimitReader(f, int64(MaxPluginBytes-total+1)))
			closeErr := f.Close()
			if readErr != nil || closeErr != nil {
				return errors.New("cannot read bounded plugin input")
			}
			total += len(b)
			if total > MaxPluginBytes {
				return errors.New("plugin contents exceed size limit")
			}
			files[name] = b
			return nil
		})
		if err != nil {
			return PluginPackage{}, err
		}
	}
	var old map[string]any
	if err := strictjson.Decode(files[pluginManifestPath], &old, MaxBootstrapBytes); err != nil {
		return PluginPackage{}, err
	}
	oldVersion, ok := old["version"].(string)
	if !ok || !ValidVersion(oldVersion) {
		return PluginPackage{}, errors.New("source plugin version is invalid")
	}
	m, err := pluginMetadata(files, oldVersion)
	if err != nil {
		return PluginPackage{}, err
	}
	m["version"] = version
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return PluginPackage{}, err
	}
	files[pluginManifestPath] = append(data, '\n')
	keys := make([]string, 0, len(files))
	for name := range files {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	for _, name := range keys {
		h := &zip.FileHeader{Name: name, Method: zip.Store, Modified: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)}
		h.SetMode(pluginMode(name))
		f, err := w.CreateHeader(h)
		if err != nil {
			return PluginPackage{}, err
		}
		if _, err := f.Write(files[name]); err != nil {
			return PluginPackage{}, err
		}
	}
	if err := w.Close(); err != nil {
		return PluginPackage{}, err
	}
	if archive.Len() > MaxPluginBytes {
		return PluginPackage{}, errors.New("plugin archive exceeds size limit")
	}
	b, err := json.Marshal(files)
	if err != nil {
		return PluginPackage{}, err
	}
	return PluginPackage{Files: files, Archive: archive.Bytes(), SHA256: Digest(b)}, nil
}

// VerifyPlugin never extracts or executes an archive. Bounded content identity is
// compared with the embedded-package digest, separately from the ZIP asset digest.
func VerifyPlugin(data []byte, version, expectedSHA256 string) error {
	if len(data) > MaxPluginBytes || !ValidVersion(version) || !validHex(expectedSHA256, 64) {
		return errors.New("invalid plugin verification input")
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return errors.New("invalid plugin ZIP")
	}
	if len(z.File) == 0 || len(z.File) > maxPluginFiles {
		return errors.New("invalid plugin archive entry count")
	}
	files := map[string][]byte{}
	total := 0
	for _, entry := range z.File {
		if !validPluginPath(entry.Name) || !entry.Mode().IsRegular() || entry.Mode().Perm() != pluginMode(entry.Name) || entry.UncompressedSize64 > MaxPluginBytes {
			return errors.New("invalid plugin archive path, mode or size")
		}
		if _, exists := files[entry.Name]; exists {
			return errors.New("duplicate plugin archive entry")
		}
		f, err := entry.Open()
		if err != nil {
			return errors.New("cannot open plugin archive entry")
		}
		b, readErr := io.ReadAll(io.LimitReader(f, int64(MaxPluginBytes-total+1)))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil {
			return errors.New("cannot read bounded plugin archive entry")
		}
		total += len(b)
		if total > MaxPluginBytes || uint64(len(b)) != entry.UncompressedSize64 {
			return errors.New("plugin archive expanded size is invalid")
		}
		files[entry.Name] = b
	}
	if _, err := pluginMetadata(files, version); err != nil {
		return err
	}
	b, err := json.Marshal(files)
	if err != nil || Digest(b) != expectedSHA256 {
		return errors.New("plugin contents do not match embedded package identity")
	}
	return nil
}
