package launch

// EntrySchema describes the CLI-only, machine-local configuration input. These
// paths and defaults never become portable signet data or agent memory tools.
func EntrySchema() map[string]any {
	path := map[string]any{"type": "string", "minLength": 1, "description": "Canonical absolute machine-local path"}
	agent := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"native_home", "native_binary", "state_dir", "connection_root"}, "properties": map[string]any{"native_home": path, "native_binary": path, "state_dir": path, "connection_root": path}}
	return map[string]any{"type": "object", "additionalProperties": false, "required": []string{"binding", "default_agent", "agents"}, "properties": map[string]any{"binding": path, "signet_id": map[string]any{"type": "string"}, "binding_sha256": map[string]any{"type": "string", "pattern": "^[0-9a-f]{64}$"}, "default_agent": map[string]any{"type": "string", "enum": []string{"codex", "pi", "claude-code", "hermes"}}, "agents": map[string]any{"type": "object", "minProperties": 1, "additionalProperties": false, "properties": map[string]any{"codex": agent, "pi": agent, "claude-code": agent, "hermes": agent}}}}
}
