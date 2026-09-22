package install

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	claudeplugin "github.com/acoz-labs/mandalore/plugins/claude-code"
)

type ClaudeOptions struct {
	Options
	ReadOnly    bool   `json:"read_only"`
	RecoverFrom string `json:"recover_from,omitempty"`
}

type ClaudePlan struct {
	ClaudeOptions
	SchemaVersion         int    `json:"schema_version"`
	Harness               string `json:"harness"`
	SignetID              string `json:"signet_id"`
	BinarySHA256          string `json:"binary_sha256"`
	NativeSHA256          string `json:"native_sha256"`
	BindingSHA256         string `json:"binding_sha256"`
	PackageSHA256         string `json:"package_sha256"`
	PackageVersion        string `json:"package_version"`
	SettingsExists        bool   `json:"settings_exists"`
	SettingsSHA256        string `json:"settings_sha256,omitempty"`
	PreviousRoot          string `json:"previous_connection_root,omitempty"`
	PreviousReceiptSHA256 string `json:"previous_receipt_sha256,omitempty"`
	Runtime               string `json:"runtime"`
	Root                  string `json:"connection_root"`
}

func claudePlanKey(p ClaudePlan) string {
	p.Root = ""
	raw, _ := json.Marshal(p)
	return hash(raw)
}

// PrepareClaude observes bytes only. Source/native execution belongs to explicit
// apply. The selected runtime must later prove this same embedded package.
func PrepareClaude(o ClaudeOptions) (ClaudePlan, error) {
	var p ClaudePlan
	var err error
	for _, path := range []*string{&o.StateDir, &o.NativeHome, &o.NativeBinary, &o.Binary, &o.Binding} {
		*path, err = canonical(*path)
		if err != nil {
			return p, err
		}
	}
	for _, path := range []string{o.StateDir, o.NativeHome} {
		st, err := os.Lstat(path)
		if err == nil && !st.IsDir() {
			return p, errors.New("Claude state and native profile must be directories")
		}
		if err != nil && !os.IsNotExist(err) {
			return p, err
		}
	}
	if len(o.Generation) > 64 || strings.IndexFunc(o.Generation, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-')
	}) >= 0 {
		return p, errors.New("invalid Claude connection generation")
	}
	raw, err := readRegular(o.Binding, 32768)
	if err != nil {
		return p, err
	}
	var b binding.Binding
	if strictjson.Decode(raw, &b, 32768) != nil {
		return p, errors.New("invalid Claude signet binding")
	}
	guard := binding.Guard{SHA256: hash(raw), SignetID: b.SignetID}
	s, err := binding.OpenGuarded(o.Binding, "installation", guard)
	if err != nil {
		return p, err
	}
	for _, pair := range [][2]string{{s.Root(), o.StateDir}, {s.Root(), o.NativeHome}, {o.StateDir, o.NativeHome}} {
		if inside(pair[0], pair[1]) || inside(pair[1], pair[0]) {
			return p, errors.New("signet, Claude profile and installation state must not overlap")
		}
	}
	if inside(o.StateDir, o.Binding) || inside(o.NativeHome, o.Binding) {
		return p, errors.New("binding must be outside Claude profile and installation state")
	}
	p = ClaudePlan{ClaudeOptions: o, SchemaVersion: 1, Harness: "claude-code", SignetID: s.ID(), BindingSHA256: guard.SHA256}
	p.BinarySHA256, err = digest(o.Binary)
	if err != nil {
		return ClaudePlan{}, err
	}
	p.NativeSHA256, err = digestLimit(o.NativeBinary, maxNativeBinary)
	if err != nil {
		return ClaudePlan{}, err
	}
	info, err := claudeplugin.Inspect()
	if err != nil {
		return ClaudePlan{}, err
	}
	p.PackageSHA256, p.PackageVersion = info.SHA256, info.Version
	settings, err := inspectClaudeSettings(o.NativeHome)
	if err != nil {
		return ClaudePlan{}, err
	}
	p.SettingsExists, p.SettingsSHA256 = settings.Exists, settings.SHA256
	previous, err := claudeSelectedRegistration(settings, o.StateDir)
	if err != nil {
		return ClaudePlan{}, err
	}
	if o.RecoverFrom != "" {
		root, e := canonical(o.RecoverFrom)
		if e != nil || root != o.RecoverFrom {
			return ClaudePlan{}, errors.New("repair root must be an explicit real directory")
		}
		if previous != "" && previous != root {
			return ClaudePlan{}, errors.New("repair would replace a different native connection")
		}
		previous = root
	}
	if previous != "" {
		receipt, e := ownedClaude(previous, o.StateDir, o.NativeHome, o.RecoverFrom != "")
		if e != nil {
			return ClaudePlan{}, e
		}
		if settings.Cache != "" {
			if err := verifyClaudeCache(settings, receipt.Plan, o.RecoverFrom != ""); err != nil {
				return ClaudePlan{}, err
			}
		}
		// A repair cannot implicitly select a new bank/writer or weaken read-only.
		if o.RecoverFrom != "" && (receipt.Plan.BindingSHA256 != p.BindingSHA256 || receipt.Plan.SignetID != p.SignetID || receipt.Plan.ReadOnly != p.ReadOnly) {
			return ClaudePlan{}, errors.New("repair must preserve the selected binding and read-only mode")
		}
		data, e := readRegular(filepath.Join(previous, "receipt.json"), 65536)
		if e != nil {
			return ClaudePlan{}, e
		}
		p.PreviousRoot, p.PreviousReceiptSHA256 = previous, hash(data)
	}
	p.Runtime = filepath.Join(o.StateDir, "runtimes", "sha256-"+p.BinarySHA256, "mandalore")
	p.Root = filepath.Join(o.StateDir, "claude-code", "connections", claudePlanKey(p))
	for _, path := range []string{p.Runtime, p.Root} {
		resolved, e := canonical(path)
		if e != nil || resolved != path {
			return ClaudePlan{}, errors.New("Claude managed destination is redirected")
		}
	}
	return p, nil
}
