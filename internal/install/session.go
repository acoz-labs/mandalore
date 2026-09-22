package install

import (
	"encoding/json"
	"errors"
	"github.com/acoz-labs/mandalore/internal/sessionsync"
	"os"
	"path/filepath"
)

// Policy bytes are generated from reviewed immutable connection identities.
// Legacy plans omit the version and generate no session transport authority.
func sessionPolicy(o Options, runtime, digest, bindingDigest, signet string) ([]byte, error) {
	return json.MarshalIndent(sessionsync.Policy{SchemaVersion: 1, Mode: "enabled-session", Binding: o.Binding, BindingSHA256: bindingDigest, SignetID: signet, Runtime: runtime, RuntimeSHA256: digest, StateDir: o.StateDir}, "", "  ")
}
func sessionArgs(o Options, root, runtime, digest, bindingDigest, signet string) (string, error) {
	if o.SessionTransportVersion == 0 {
		return "", nil
	}
	raw, err := sessionPolicy(o, runtime, digest, bindingDigest, signet)
	if err != nil {
		return "", err
	}
	return " --session-policy " + quote(filepath.Join(root, "session-policy.json")) + " --session-policy-sha256 " + quote(hash(raw)), nil
}

func validateSessionReceipt(o Options, runtime, digest, bindingDigest, signet string, files map[string]string, policyName string) error {
	if o.SessionTransportVersion == 0 {
		if _, ok := files[policyName]; ok {
			return errors.New("legacy receipt cannot grant session transport")
		}
		return nil
	}
	if o.SessionTransportVersion != 1 {
		return errors.New("unsupported session transport policy")
	}
	raw, err := sessionPolicy(o, runtime, digest, bindingDigest, signet)
	if err != nil || files[policyName] != hash(raw) {
		return errors.New("session policy differs from reviewed connection")
	}
	return nil
}

func sessionRuntimeGuard(runtime, digest string) string {
	return "#!/bin/sh\nmandalore_runtime=" + quote(runtime) + "\n" +
		"if [ -x /usr/bin/sha256sum ]; then mandalore_digest=$(/usr/bin/sha256sum \"$mandalore_runtime\" 2>/dev/null); elif [ -x /usr/bin/shasum ]; then mandalore_digest=$(/usr/bin/shasum -a 256 \"$mandalore_runtime\" 2>/dev/null); else exit 1; fi\n" +
		"mandalore_digest=${mandalore_digest%% *}\n[ \"$mandalore_digest\" = " + quote(digest) + " ] || exit 1\n"
}

// ExistingSessionMode reads owned registration metadata without executing native
// tools. Codex has no static native listing contract, so retained matching
// generations conservatively preserve legacy authority until explicitly changed.
func ExistingSessionMode(harness string, p Profile) (version int, readOnly bool, exists bool, err error) {
	switch harness {
	case "pi":
		settings, e := inspectPiSettings(p.NativeHome)
		if e != nil {
			return 0, false, false, e
		}
		root, e := piSelectedRegistration(settings, p.StateDir)
		if e != nil || root == "" {
			return 0, false, false, e
		}
		r, e := ownedPi(root, p.StateDir, p.NativeHome, false)
		return r.Plan.SessionTransportVersion, r.Plan.ReadOnly, true, e
	case "claude-code":
		settings, e := inspectClaudeSettings(p.NativeHome)
		if e != nil || settings.Root == "" {
			return 0, false, false, e
		}
		r, e := ownedClaude(settings.Root, p.StateDir, p.NativeHome, false)
		return r.Plan.SessionTransportVersion, r.Plan.ReadOnly, true, e
	case "codex":
		entries, e := os.ReadDir(filepath.Join(p.StateDir, "connections"))
		if os.IsNotExist(e) {
			return 0, false, false, nil
		}
		if e != nil {
			return 0, false, false, e
		}
		if len(entries) > 256 {
			return 0, false, false, errors.New("retained connection inventory exceeds limit; inspect explicit prior mode")
		}
		mode := 1
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			r, e := loadReceipt(filepath.Join(p.StateDir, "connections", entry.Name()))
			if e != nil {
				continue
			}
			if r.Plan.NativeHome == p.NativeHome {
				exists = true
				if r.Plan.SessionTransportVersion == 0 {
					mode = 0
				}
			}
		}
		return mode, false, exists, nil
	default:
		return 0, false, false, errors.New("unsupported native harness")
	}
}
