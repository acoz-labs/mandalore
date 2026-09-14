// Package codexplugin embeds the public native integration for installed CLIs.
package codexplugin

import "embed"

// Files includes hidden native manifests without embedding the source checkout.
//
//go:embed .agents/plugins/marketplace.json all:plugins/mandalore
var Files embed.FS
