package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

type MemoryPlugin struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
}

type MemoryInventory struct {
	Status           string         `json:"status"`
	BinarySHA256     string         `json:"binary_sha256,omitempty"`
	Plugins          []MemoryPlugin `json:"plugins"`
	PotentialWriters int            `json:"potential_writers"`
	Notice           string         `json:"notice,omitempty"`
}

// InspectMemoryPlugins invokes only the selected native CLI's listing commands.
// Installation is not proof of an active session or the bank a plugin targets.
// Native logging/caching is owned by the selected CLI, not edited by Mandalore.
func InspectMemoryPlugins(ctx context.Context, home, binary string) (MemoryInventory, error) {
	return inspectMemoryPlugins(ctx, home, binary, native)
}

func inspectMemoryPlugins(ctx context.Context, home, binary string, run runner) (MemoryInventory, error) {
	result := MemoryInventory{Status: "not-tested", Plugins: []MemoryPlugin{}}
	var err error
	home, err = canonical(home)
	if err != nil {
		return result, err
	}
	binary, err = canonical(binary)
	if err != nil {
		return result, err
	}
	if st, err := os.Lstat(home); err != nil || !st.IsDir() {
		return result, errors.New("native inventory requires an existing explicit profile directory")
	}
	before, err := digestLimit(binary, maxNativeBinary)
	if err != nil {
		return result, err
	}
	_, plugins, err := inventory(ctx, Options{NativeHome: home, NativeBinary: binary, Binding: filepath.Join(home, "inventory-context")}, run)
	if err != nil {
		return result, err
	}
	for _, p := range plugins {
		if p.Name != "my-friday" && p.Name != "my-friday-memory" && p.Name != "mandalore" {
			continue
		}
		if len(result.Plugins) >= 32 || len(p.ID) > 512 || len(p.Version) > 256 {
			return result, errors.New("memory plugin inventory exceeds bounded report limits")
		}
		result.Plugins = append(result.Plugins, MemoryPlugin{ID: p.ID, Name: p.Name, Version: p.Version, Enabled: p.Enabled})
		if p.Enabled {
			result.PotentialWriters++
		}
	}
	sort.Slice(result.Plugins, func(i, j int) bool {
		a, b := result.Plugins[i], result.Plugins[j]
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		return !a.Enabled && b.Enabled
	})
	after, err := digestLimit(binary, maxNativeBinary)
	if err != nil || before != after {
		return result, errors.New("native binary changed during inventory")
	}
	result.Status, result.BinarySHA256 = "inspected", before
	result.Notice = "Enabled memory integrations are potential writers only. Active sessions, same-bank targeting, other machines and authentication are not tested. Listing may produce native-owned logs or caches."
	return result, nil
}
