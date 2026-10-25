package main

import (
	"runtime"

	"github.com/acoz-labs/mandalore/internal/api"
)

func runtimeMetadata() (map[string]any, error) {
	// Released installers strictly decode this wire contract. Expose new
	// capabilities through operation schemas, not additional version fields.
	return map[string]any{
		"name": "mandalore", "version": version, "source_commit": sourceCommit,
		"protocol_version": api.ProtocolVersion,
		"os":               runtime.GOOS, "arch": runtime.GOARCH, "go_version": runtime.Version(),
		"signet_read_versions": []int{1, 2}, "signet_write_versions": []int{1, 2},
	}, nil
}
