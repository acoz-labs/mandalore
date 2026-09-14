package install

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	"github.com/acoz-labs/mandalore/internal/binding"
)

type Profile struct {
	StateDir     string `json:"state_dir"`
	NativeHome   string `json:"native_home"`
	NativeBinary string `json:"native_binary"`
}

type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}
type Report struct {
	Healthy    bool    `json:"healthy"`
	Checks     []Check `json:"checks"`
	Connection *Plan   `json:"connection,omitempty"`
	Notice     string  `json:"notice"`
}

func Doctor(ctx context.Context, profile Profile) Report { return doctor(ctx, profile, native) }

func doctor(ctx context.Context, profile Profile, run runner) (r Report) {
	r = Report{Healthy: true, Checks: []Check{}, Notice: "Read-only structural inspection and native inventory. Untested boundaries are not implied healthy; no repair or synchronization is performed."}
	defer func() {
		for _, item := range []Check{
			{Name: "native-login", Detail: "Native account authentication was not exercised."},
			{Name: "hook-trust", Detail: "Review the exact hooks through the native agent UI."},
			{Name: "live-mcp", Detail: "Check MCP startup in a fresh native session."},
			{Name: "remote-freshness", Detail: "Local receipts do not prove network freshness; explicitly synchronize when allowed."},
			{Name: "active-context", Detail: "Existing sessions may retain earlier instructions or bindings; start a fresh session after a change."},
		} {
			item.Status = "not-tested"
			r.Checks = append(r.Checks, item)
		}
	}()
	add := func(name string, err error) {
		c := Check{Name: name, Status: "pass", Detail: "Passed"}
		if err != nil {
			c.Status, c.Detail, r.Healthy = "fail", err.Error(), false
		}
		r.Checks = append(r.Checks, c)
	}
	for _, item := range []struct {
		name string
		path *string
	}{{"state-path", &profile.StateDir}, {"native-home", &profile.NativeHome}, {"native-binary", &profile.NativeBinary}} {
		path, err := canonical(*item.path)
		add(item.name, err)
		if err != nil {
			return r
		}
		*item.path = path
	}
	// Do not invoke a native CLI against a missing profile, which could create it.
	if st, err := os.Stat(profile.NativeHome); err != nil || !st.IsDir() {
		add("native-profile", errors.New("native profile is missing or not a directory; connect explicitly"))
		return r
	}
	o := Options{StateDir: profile.StateDir, NativeHome: profile.NativeHome, NativeBinary: profile.NativeBinary}
	// Native inventory runs in an existing directory; it does not need a binding.
	o.Binding = filepath.Join(profile.NativeHome, "unused-binding-path")
	ms, ps, err := inventory(ctx, o, run)
	add("native-inventory", err)
	if err != nil {
		return r
	}
	var selected *marketplace
	for i := range ms {
		if ms[i].Name != "mandalore" {
			continue
		}
		if selected != nil {
			add("connection", errors.New("ambiguous Mandalore marketplace registration"))
			return r
		}
		selected = &ms[i]
	}
	if selected == nil {
		add("connection", errors.New("no native Mandalore registration; connect or use a retained receipt to preview recovery"))
		return r
	}
	if selected.Source.Type != "local" || selected.Source.Path != selected.Root || filepath.Dir(selected.Root) != filepath.Join(profile.StateDir, "connections") {
		add("connection", errors.New("Mandalore registration belongs to an unmanaged source; preserve it"))
		return r
	}
	receipt, err := loadReceipt(selected.Root)
	add("connection-receipt", err)
	if err != nil {
		return r
	}
	p := receipt.Plan
	if p.StateDir != profile.StateDir || p.NativeHome != profile.NativeHome {
		add("connection-ownership", errors.New("receipt belongs to another installation or native profile"))
		return r
	}
	r.Connection = &p
	add("bundle-integrity", verifyTree(p.Root, receipt.Files, false))
	add("cache-integrity", verifyCache(p, false))
	d, err := digest(p.Runtime)
	if err == nil && d != p.BinarySHA256 {
		err = errors.New("pinned runtime bytes changed")
	}
	add("runtime-integrity", err)
	d, err = digestLimit(profile.NativeBinary, maxNativeBinary)
	if err == nil && d != p.NativeSHA256 {
		err = errors.New("selected native executable changed; verify compatibility and preview reconnect")
	}
	add("native-executable-identity", err)
	s, err := binding.Open(p.Binding, "doctor")
	if err == nil && s.ID() != p.SignetID {
		err = errors.New("binding points to another signet")
	}
	add("signet-binding", err)
	b, err := readRegular(p.Binding, 32768)
	if err == nil && hash(b) != p.BindingSHA256 {
		err = errors.New("binding configuration changed; explicitly reconnect instead of silently repairing")
	}
	add("binding-integrity", err)
	for _, item := range []struct{ key, want string }{{"MANDALORE_BIN", p.Runtime}, {"MANDALORE_BINDING", p.Binding}} {
		if value := os.Getenv(item.key); value != "" {
			resolved, err := canonical(value)
			if err != nil || resolved != item.want {
				add(item.key, errors.New(item.key+" overrides the managed default in this process; clear it or deliberately select that connection"))
			}
		}
	}
	count := 0
	for _, v := range ps {
		if v.ID == p.PluginID && v.Enabled && v.Version == p.Version && v.Source.Type == "local" && v.Source.Path == p.Root {
			count++
		}
		if v.Enabled && (v.Name == "my-friday-memory" || v.Name == "my-friday" || v.Name == "mandalore" && v.ID != p.PluginID) {
			add("duplicate-memory-integration", errors.New("another active memory integration requires explicit ownership resolution"))
		}
	}
	if count != 1 {
		add("native-plugin", errors.New("selected plugin is missing, disabled, duplicated or stale"))
	} else {
		add("native-plugin", nil)
	}
	return r
}

type RepairInput struct {
	Root         string `json:"connection_root"`
	NativeBinary string `json:"native_binary,omitempty"`
}

// PrepareRepair never repairs in place. It requires intact ownership evidence
// and preserves all existing generations, including missing-only generations.
func PrepareRepair(in RepairInput) (Plan, error) {
	root, err := canonical(in.Root)
	if err != nil {
		return Plan{}, err
	}
	r, err := loadReceipt(root)
	if err != nil {
		return Plan{}, errors.New("no intact ownership receipt; preserve the source and explicitly reconnect after inspection")
	}
	if _, err := owned(root, r.Plan, true); err != nil {
		return Plan{}, err
	}
	b, err := readRegular(r.Plan.Binding, 32768)
	if err != nil || hash(b) != r.Plan.BindingSHA256 {
		return Plan{}, errors.New("binding changed or is missing; repair will not select a different signet or writer")
	}
	o := r.Plan.Options
	if in.NativeBinary != "" {
		o.NativeBinary = in.NativeBinary
	}
	// Prefer the pinned copy, not an old source path which may have disappeared.
	o.Binary = r.Plan.Runtime
	if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
		o.Binary = r.Plan.Binary
		if d, err := digest(o.Binary); err != nil || d != r.Plan.BinarySHA256 {
			return Plan{}, errors.New("no intact runtime copy remains; select a trusted artifact and preview reconnect")
		}
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return Plan{}, err
	}
	o.Generation = "repair-" + hex.EncodeToString(nonce[:])
	return Prepare(o)
}
