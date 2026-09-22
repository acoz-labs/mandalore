package install

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"

	claudeplugin "github.com/acoz-labs/mandalore/plugins/claude-code"
)

type ClaudeReport struct {
	Healthy    bool        `json:"healthy"`
	Checks     []Check     `json:"checks"`
	Connection *ClaudePlan `json:"connection,omitempty"`
	Notice     string      `json:"notice"`
}

func DoctorClaude(ctx context.Context, profile Profile) ClaudeReport {
	return doctorClaude(ctx, profile, nativeClaude)
}

func doctorClaude(ctx context.Context, profile Profile, run runner) (r ClaudeReport) {
	r = ClaudeReport{Healthy: true, Checks: []Check{}, Notice: "Read-only Claude structure, selected native version and registration checks. No login, repair, synchronization or assertion about active model context."}
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
			{Name: "live-tools", Detail: "Test native memory tools in a fresh Claude session; registration is not proof they loaded."},
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
		add("native-profile", errors.New("Claude profile is missing or not a directory; connect explicitly"))
		return r
	}
	s, err := inspectClaudeSettings(profile.NativeHome)
	add("native-settings", err)
	if err != nil {
		return r
	}
	root, err := claudeSelectedRegistration(s, profile.StateDir)
	add("native-registration", err)
	if err != nil {
		return r
	}
	if root == "" {
		add("connection", errors.New("no registered Claude memory connection; use an intact retained receipt to preview recovery"))
		return r
	}
	receipt, err := loadClaudeReceipt(root)
	add("ownership-receipt", err)
	if err != nil {
		return r
	}
	p := receipt.Plan
	if p.StateDir != profile.StateDir || p.NativeHome != profile.NativeHome {
		add("ownership", errors.New("Claude connection belongs to another state directory or profile"))
		return r
	}
	r.Connection = &p
	_, err = ownedClaude(root, profile.StateDir, profile.NativeHome, false)
	add("retained-integrity", err)
	add("native-cache", verifyClaudeCache(s, p, false))
	// A deliberate new native executable can be inspected, but the old connection
	// still needs an explicit reconnect; never call it healthy on version alone.
	selected := p
	selected.NativeBinary = profile.NativeBinary
	add("binding-and-native-identity", verifyClaudeBindingNative(selected))
	add("native-version", claudeNativeVersion(ctx, selected.Options, run))
	return r
}

func PrepareClaudeRepair(in RepairInput) (ClaudePlan, error) {
	root, err := canonical(in.Root)
	if err != nil {
		return ClaudePlan{}, err
	}
	r, err := loadClaudeReceipt(root)
	if err != nil {
		return ClaudePlan{}, errors.New("no intact Claude ownership receipt; preserve existing files and inspect before reconnecting")
	}
	if _, err := ownedClaude(root, r.Plan.StateDir, r.Plan.NativeHome, true); err != nil {
		return ClaudePlan{}, err
	}
	b, err := readRegular(r.Plan.Binding, 32768)
	if err != nil || hash(b) != r.Plan.BindingSHA256 {
		return ClaudePlan{}, errors.New("Claude binding changed; repair will not select another signet or writer")
	}
	info, err := claudeplugin.Inspect()
	if err != nil || info.SHA256 != r.Plan.PackageSHA256 || info.Version != r.Plan.PackageVersion {
		return ClaudePlan{}, errors.New("repair requires the retained runtime's package; use that runtime or explicitly preview a connection update")
	}
	o := r.Plan.ClaudeOptions
	if in.NativeBinary != "" {
		o.NativeBinary = in.NativeBinary
	}
	o.Binary = r.Plan.Runtime
	if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
		o.Binary = r.Plan.Binary
		if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
			return ClaudePlan{}, errors.New("no intact runtime remains; select a trusted artifact and preview reconnect")
		}
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return ClaudePlan{}, err
	}
	o.Generation = "repair-" + hex.EncodeToString(nonce[:])
	o.RecoverFrom = root
	return PrepareClaude(o)
}
