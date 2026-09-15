package install

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"

	piplugin "github.com/acoz-labs/mandalore/plugins/pi"
)

type PiReport struct {
	Healthy    bool    `json:"healthy"`
	Checks     []Check `json:"checks"`
	Connection *PiPlan `json:"connection,omitempty"`
	Notice     string  `json:"notice"`
}

func DoctorPi(ctx context.Context, profile Profile) PiReport { return doctorPi(ctx, profile, nativePi) }

func doctorPi(ctx context.Context, profile Profile, run runner) (r PiReport) {
	r = PiReport{Healthy: true, Checks: []Check{}, Notice: "Read-only Pi structure, selected native version and registration checks. No login, repair, synchronization or assertion about active model context."}
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
			{Name: "live-tools", Detail: "Test native memory tools in a fresh Pi session; registration is not proof they loaded."},
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
		add("native-profile", errors.New("Pi profile is missing or not a directory; connect explicitly"))
		return r
	}
	s, err := inspectPiSettings(profile.NativeHome)
	add("native-settings", err)
	if err != nil {
		return r
	}
	root, err := piSelectedRegistration(s, profile.StateDir)
	add("native-registration", err)
	if err != nil {
		return r
	}
	if root == "" {
		add("connection", errors.New("no registered Pi memory connection; use an intact retained receipt to preview recovery"))
		return r
	}
	receipt, err := loadPiReceipt(root)
	add("ownership-receipt", err)
	if err != nil {
		return r
	}
	p := receipt.Plan
	if p.StateDir != profile.StateDir || p.NativeHome != profile.NativeHome {
		add("ownership", errors.New("Pi connection belongs to another state directory or profile"))
		return r
	}
	r.Connection = &p
	_, err = ownedPi(root, profile.StateDir, profile.NativeHome, false)
	add("retained-integrity", err)
	// A deliberate new native executable can be inspected, but the old connection
	// still needs an explicit reconnect; never call it healthy on version alone.
	selected := p
	selected.NativeBinary = profile.NativeBinary
	add("binding-and-native-identity", verifyPiBindingNative(selected))
	add("native-version", piNativeVersion(ctx, selected.Options, run))
	return r
}

func PreparePiRepair(in RepairInput) (PiPlan, error) {
	root, err := canonical(in.Root)
	if err != nil {
		return PiPlan{}, err
	}
	r, err := loadPiReceipt(root)
	if err != nil {
		return PiPlan{}, errors.New("no intact Pi ownership receipt; preserve existing files and inspect before reconnecting")
	}
	if _, err := ownedPi(root, r.Plan.StateDir, r.Plan.NativeHome, true); err != nil {
		return PiPlan{}, err
	}
	b, err := readRegular(r.Plan.Binding, 32768)
	if err != nil || hash(b) != r.Plan.BindingSHA256 {
		return PiPlan{}, errors.New("Pi binding changed; repair will not select another signet or writer")
	}
	info, err := piplugin.Inspect()
	if err != nil || info.SHA256 != r.Plan.PackageSHA256 || info.Version != r.Plan.PackageVersion {
		return PiPlan{}, errors.New("repair requires the retained runtime's package; use that runtime or explicitly preview a connection update")
	}
	o := r.Plan.PiOptions
	if in.NativeBinary != "" {
		o.NativeBinary = in.NativeBinary
	}
	o.Binary = r.Plan.Runtime
	if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
		o.Binary = r.Plan.Binary
		if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
			return PiPlan{}, errors.New("no intact runtime remains; select a trusted artifact and preview reconnect")
		}
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return PiPlan{}, err
	}
	o.Generation = "repair-" + hex.EncodeToString(nonce[:])
	o.RecoverFrom = root
	return PreparePi(o)
}
