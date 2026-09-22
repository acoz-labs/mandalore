package install

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	piplugin "github.com/acoz-labs/mandalore/plugins/pi"
)

type PiOptions struct {
	Options
	ReadOnly    bool   `json:"read_only"`
	RecoverFrom string `json:"recover_from,omitempty"`
}

type PiPlan struct {
	PiOptions
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

func piPlanKey(p PiPlan) string {
	p.Root = ""
	raw, _ := json.Marshal(p)
	return hash(raw)
}

// PreparePi observes bytes only. Source/native execution belongs to explicit
// apply. The selected runtime must later prove this same embedded package.
func PreparePi(o PiOptions) (PiPlan, error) {
	var p PiPlan
	if o.SessionTransportVersion != 0 && o.SessionTransportVersion != 1 || o.ReadOnly && o.SessionTransportVersion != 0 {
		return p, errors.New("session transport requires an explicitly enabled writable connection")
	}
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
			return p, errors.New("Pi state and native profile must be directories")
		}
		if err != nil && !os.IsNotExist(err) {
			return p, err
		}
	}
	if len(o.Generation) > 64 || strings.IndexFunc(o.Generation, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-')
	}) >= 0 {
		return p, errors.New("invalid Pi connection generation")
	}
	raw, err := readRegular(o.Binding, 32768)
	if err != nil {
		return p, err
	}
	var b binding.Binding
	if strictjson.Decode(raw, &b, 32768) != nil {
		return p, errors.New("invalid Pi signet binding")
	}
	guard := binding.Guard{SHA256: hash(raw), SignetID: b.SignetID}
	s, err := binding.OpenGuarded(o.Binding, "installation", guard)
	if err != nil {
		return p, err
	}
	for _, pair := range [][2]string{{s.Root(), o.StateDir}, {s.Root(), o.NativeHome}, {o.StateDir, o.NativeHome}} {
		if inside(pair[0], pair[1]) || inside(pair[1], pair[0]) {
			return p, errors.New("signet, Pi profile and installation state must not overlap")
		}
	}
	if inside(o.StateDir, o.Binding) || inside(o.NativeHome, o.Binding) {
		return p, errors.New("binding must be outside Pi profile and installation state")
	}
	p = PiPlan{PiOptions: o, SchemaVersion: 1, Harness: "pi", SignetID: s.ID(), BindingSHA256: guard.SHA256}
	p.BinarySHA256, err = digest(o.Binary)
	if err != nil {
		return PiPlan{}, err
	}
	p.NativeSHA256, err = digestLimit(o.NativeBinary, maxNativeBinary)
	if err != nil {
		return PiPlan{}, err
	}
	info, err := piplugin.Inspect()
	if err != nil {
		return PiPlan{}, err
	}
	p.PackageSHA256, p.PackageVersion = info.SHA256, info.Version
	settings, err := inspectPiSettings(o.NativeHome)
	if err != nil {
		return PiPlan{}, err
	}
	p.SettingsExists, p.SettingsSHA256 = settings.Exists, settings.SHA256
	previous, err := piSelectedRegistration(settings, o.StateDir)
	if err != nil {
		return PiPlan{}, err
	}
	if o.RecoverFrom != "" {
		root, e := canonical(o.RecoverFrom)
		if e != nil || root != o.RecoverFrom {
			return PiPlan{}, errors.New("repair root must be an explicit real directory")
		}
		if previous != "" && previous != root {
			return PiPlan{}, errors.New("repair would replace a different native connection")
		}
		previous = root
	}
	if previous != "" {
		receipt, e := ownedPi(previous, o.StateDir, o.NativeHome, o.RecoverFrom != "")
		if e != nil {
			return PiPlan{}, e
		}
		// A repair cannot implicitly select a new bank/writer or weaken read-only.
		if o.RecoverFrom != "" && (receipt.Plan.BindingSHA256 != p.BindingSHA256 || receipt.Plan.SignetID != p.SignetID || receipt.Plan.ReadOnly != p.ReadOnly || receipt.Plan.SessionTransportVersion != p.SessionTransportVersion) {
			return PiPlan{}, errors.New("repair must preserve the selected binding and read-only mode")
		}
		data, e := readRegular(filepath.Join(previous, "receipt.json"), 65536)
		if e != nil {
			return PiPlan{}, e
		}
		p.PreviousRoot, p.PreviousReceiptSHA256 = previous, hash(data)
	}
	p.Runtime = filepath.Join(o.StateDir, "runtimes", "sha256-"+p.BinarySHA256, "mandalore")
	p.Root = filepath.Join(o.StateDir, "pi", "connections", piPlanKey(p))
	for _, path := range []string{p.Runtime, p.Root} {
		resolved, e := canonical(path)
		if e != nil || resolved != path {
			return PiPlan{}, errors.New("Pi managed destination is redirected")
		}
	}
	return p, nil
}

func piSelectedRegistration(s piSettings, state string) (string, error) {
	selected := ""
	for _, item := range s.Packages {
		if item.Path == "" {
			name := strings.TrimPrefix(strings.TrimSpace(item.Source), "npm:")
			name, _, _ = strings.Cut(name, "@")
			if name == "mandalore" || name == "my-friday" || name == "my-friday-memory" {
				return "", errors.New("unmanaged remote memory package; resolve it explicitly")
			}
			continue
		}
		root := filepath.Dir(item.Path)
		managed := filepath.Base(item.Path) == "package" && filepath.Dir(root) == filepath.Join(state, "pi", "connections")
		raw, err := readRegular(filepath.Join(item.Path, "package.json"), 32768)
		name := ""
		if err == nil {
			var manifest map[string]any
			if strictjson.Decode(raw, &manifest, 32768) != nil {
				return "", errors.New("invalid local Pi package manifest; inspect before connecting")
			}
			name, _ = manifest["name"].(string)
		} else if !os.IsNotExist(err) {
			return "", err
		}
		if !managed && name != "mandalore" && name != "my-friday" && name != "my-friday-memory" {
			continue
		}
		if selected != "" || !managed || item.Filtered {
			return "", errors.New("foreign, filtered or ambiguous Pi memory registration; preserve it for explicit review")
		}
		if strings.IndexFunc(root, unicode.IsControl) >= 0 {
			return "", errors.New("invalid Pi registration path")
		}
		selected = root
	}
	return selected, nil
}
