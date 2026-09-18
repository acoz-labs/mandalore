package foundlings

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// ConfiguredRootSnapshot is local path metadata, not source availability or
// registration authority. It includes disconnected/stale connections. Consumers
// must check path identity/overlap themselves and reread to detect changes.
type ConfiguredRootSnapshot struct {
	Roots  []string
	SHA256 string
}

// ConfiguredRoots reads only bounded local connection metadata. In particular it
// does not inspect reference contents, create ignored state, or hide a configured
// path just because the reference is disconnected. Unknown files fail closed.
func (m *Manager) ConfiguredRoots(ctx context.Context) (ConfiguredRootSnapshot, error) {
	var result ConfiguredRootSnapshot
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if _, err := m.store(); err != nil {
		return result, ErrConnection
	}
	h := sha256.New()
	root, err := m.localRoot(false)
	if os.IsNotExist(err) {
		result.SHA256 = hex.EncodeToString(h.Sum(nil))
		return result, nil
	}
	if err != nil {
		return result, ErrConnection
	}
	defer root.Close()
	const directory = ".mandalore/foundlings"
	dir, err := root.Open(directory)
	if err != nil {
		return result, ErrConnection
	}
	defer dir.Close()
	dirInfo, err := dir.Stat()
	if err != nil {
		return result, ErrConnection
	}
	// 256 connections, each at most 16 KiB, bounds enumeration and reads.
	entries, err := dir.ReadDir(257)
	if err != nil && err != io.EOF || len(entries) > 256 {
		return result, ErrConnection
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	seen := map[string]bool{}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return ConfiguredRootSnapshot{}, err
		}
		name := entry.Name()
		id, ok := strings.CutSuffix(name, ".json")
		if !ok || !registrationID.MatchString(id) {
			return ConfiguredRootSnapshot{}, ErrConnection
		}
		path := directory + "/" + name
		info, err := root.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 16384 {
			return ConfiguredRootSnapshot{}, ErrConnection
		}
		f, err := root.Open(path)
		if err != nil {
			return ConfiguredRootSnapshot{}, ErrConnection
		}
		opened, statErr := f.Stat()
		if statErr != nil || !os.SameFile(info, opened) {
			f.Close()
			return ConfiguredRootSnapshot{}, ErrConnection
		}
		data, readErr := io.ReadAll(io.LimitReader(f, 16385))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil || len(data) > 16384 {
			return ConfiguredRootSnapshot{}, ErrConnection
		}
		c, err := decodeConnection(data, id, m.memory.ID())
		if err != nil {
			return ConfiguredRootSnapshot{}, ErrConnection
		}
		fmt.Fprintf(h, "%d:%s:%d:", len(name), name, len(data))
		h.Write(data)
		if !seen[c.Root] {
			result.Roots = append(result.Roots, c.Root)
			seen[c.Root] = true
		}
	}
	current, err := root.Lstat(directory)
	if err != nil || !current.IsDir() || !os.SameFile(dirInfo, current) {
		return ConfiguredRootSnapshot{}, ErrConnection
	}
	if err := ctx.Err(); err != nil {
		return ConfiguredRootSnapshot{}, err
	}
	sort.Strings(result.Roots)
	result.SHA256 = hex.EncodeToString(h.Sum(nil))
	return result, nil
}
