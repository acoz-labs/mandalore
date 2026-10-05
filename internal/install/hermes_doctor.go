package install

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"

	hermesplugin "github.com/acoz-labs/mandalore/plugins/hermes"
)

type HermesReport struct {
	Healthy    bool        `json:"healthy"`
	Checks     []Check     `json:"checks"`
	Connection *HermesPlan `json:"connection,omitempty"`
	Notice     string      `json:"notice"`
}

func DoctorHermes(ctx context.Context, profile Profile) HermesReport {
	return doctorHermes(ctx, profile, nativeHermes)
}

func doctorHermes(ctx context.Context, profile Profile, run runner) (r HermesReport) {
	r = HermesReport{Healthy: true, Checks: []Check{}, Notice: "Read-only Hermes structure, selected native version and registration checks. No login, repair, synchronization or assertion about active model context."}
	add := func(name string, err error) {
		c := Check{Name: name, Status: "pass", Detail: "Passed"}
		if err != nil {
			c.Status, c.Detail, r.Healthy = "fail", err.Error(), false
		}
		r.Checks = append(r.Checks, c)
	}
	defer func() {
		for _, c := range []Check{
			{Name: "native-login", Detail: "Native authentication and model access were not tested."},
			{Name: "live-tools", Detail: "Test native memory tools in a fresh Hermes session; registration is not proof they loaded."},
			{Name: "remote-freshness", Detail: "No synchronization was requested; local state does not establish remote freshness."},
			{Name: "active-context", Detail: "Existing sessions may retain previous tools or instructions; start fresh or reload after a change."},
		} {
			c.Status = "not-tested"
			r.Checks = append(r.Checks, c)
		}
	}()
	if err := ctx.Err(); err != nil {
		add("cancelled", err)
		return r
	}
	for _, item := range []struct {
		name string
		path *string
	}{{"state-path", &profile.StateDir}, {"native-home", &profile.NativeHome}, {"native-binary", &profile.NativeBinary}} {
		value, err := canonical(*item.path)
		add(item.name, err)
		if err != nil {
			return r
		}
		*item.path = value
	}
	if st, err := os.Stat(profile.NativeHome); err != nil || !st.IsDir() {
		add("native-profile", errors.New("Hermes profile is missing or not a directory; connect explicitly"))
		return r
	}
	s, err := inspectHermesSettings(profile.NativeHome)
	add("native-settings", err)
	if err != nil {
		return r
	}
	root, err := hermesSelectedRegistration(s, profile.StateDir)
	add("native-registration", err)
	if err != nil {
		return r
	}
	if root == "" {
		add("connection", errors.New("no registered Hermes memory connection; use an intact retained receipt to preview recovery"))
		return r
	}
	if !s.Enabled {
		add("native-enabled", errors.New("Mandalore is disabled in this Hermes profile; reconnect explicitly to enable"))
	}
	receipt, err := loadHermesReceipt(root)
	add("ownership-receipt", err)
	if err != nil {
		return r
	}
	p := receipt.Plan
	if p.StateDir != profile.StateDir || p.NativeHome != profile.NativeHome {
		add("ownership", errors.New("Hermes connection belongs to another state directory or profile"))
		return r
	}
	r.Connection = &p
	_, err = ownedHermes(root, profile.StateDir, profile.NativeHome, false)
	add("retained-integrity", err)
	selected := p
	options, digest, locator, err := selectedNative("hermes", p.Options, profile.NativeBinary)
	add("native-executable", err)
	if err != nil {
		return r
	}
	selected.Options, selected.NativeSHA256 = options, digest
	add("binding-and-native-identity", verifyHermesBindingNative(selected))
	add("native-version", hermesNativeVersion(ctx, selected.Options, run))
	add("native-stability", verifyNativeObservation("hermes", p.Options, selected.NativeBinary, digest, locator))
	r.Checks = append(r.Checks, Check{Name: "native-release-acceptance", Status: "not-tested", Detail: "CLI compatibility and registration were inspected. Fresh-session hooks, tools and model access on this native release are not established by this inspection."})

	return r
}

func PrepareHermesRepair(in RepairInput) (HermesPlan, error) {
	root, err := canonical(in.Root)
	if err != nil {
		return HermesPlan{}, err
	}
	r, err := loadHermesReceipt(root)
	if err != nil {
		return HermesPlan{}, errors.New("no intact Hermes ownership receipt; preserve existing files and inspect before reconnecting")
	}
	if _, err := ownedHermes(root, r.Plan.StateDir, r.Plan.NativeHome, true); err != nil {
		return HermesPlan{}, err
	}
	b, err := readRegular(r.Plan.Binding, 32768)
	if err != nil || hash(b) != r.Plan.BindingSHA256 {
		return HermesPlan{}, errors.New("Hermes binding changed; repair will not select another signet or writer")
	}
	info, err := hermesplugin.Inspect()
	if err != nil || info.SHA256 != r.Plan.PackageSHA256 || info.Version != r.Plan.PackageVersion {
		return HermesPlan{}, errors.New("repair requires the retained runtime's package; use that runtime or explicitly preview a connection update")
	}
	o := r.Plan.HermesOptions
	if in.NativeBinary != "" {
		o.NativeBinary = in.NativeBinary
		o.NativeLauncher = ""
	}
	o.Binary = r.Plan.Runtime
	if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
		o.Binary = r.Plan.Binary
		if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
			return HermesPlan{}, errors.New("no intact runtime remains; select a trusted artifact and preview reconnect")
		}
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return HermesPlan{}, err
	}
	o.Generation = "repair-" + hex.EncodeToString(nonce[:])
	o.RecoverFrom = root
	return PrepareHermes(o)
}
