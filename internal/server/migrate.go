package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/cutover"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

// LostDevice is an enrolled 1.x device the operator cannot preflight.
type LostDevice struct {
	ID     string `json:"device_id"`
	Reason string `json:"reason"`
}

type MigrateInit struct {
	ConfigPath, Remote, Slug, Root string
	Preflights                     []string // receipt files from `server migrate preflight`
	Lost                           []LostDevice
}

// DeviceEvidence is why init accepted an enrolled device.
type DeviceEvidence struct {
	DeviceID  string `json:"device_id"`
	Evidence  string `json:"evidence"` // preflight or lost
	Head      string `json:"head,omitempty"`
	CheckedAt string `json:"checked_at,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// Migration is init's record in the server's state, never in the signet.
type Migration struct {
	Version        int              `json:"schema_version"`
	Slug           string           `json:"slug"`
	SignetID       string           `json:"signet_id"`
	InitializedAt  string           `json:"initialized_at"`
	ClonedHead     string           `json:"cloned_head"` // the remote head init checked receipts against
	ServerDeviceID string           `json:"server_device_id"`
	Devices        []DeviceEvidence `json:"devices"`
	Pushed         bool             `json:"pushed"`
	PushedHead     string           `json:"pushed_head,omitempty"` // the enrollment commit, once pushed
	Notice         string           `json:"notice,omitempty"`
}

// Init clones the remote fresh, validates it, enrolls the server's device and
// adds the signet to the server configuration. It refuses unless every
// enrolled device has a clean preflight receipt or is marked lost.
func Init(ctx context.Context, in MigrateInit) (Migration, error) {
	out := Migration{Version: 1, Slug: in.Slug, Devices: []DeviceEvidence{}}
	c, err := readConfig(in.ConfigPath)
	if err != nil {
		return out, err
	}
	for _, sc := range c.Signets {
		if sc.Slug == in.Slug {
			return out, fmt.Errorf("signet %s is already configured", in.Slug)
		}
	}
	// Check the configuration and the new entry before touching anything.
	probe := c
	probe.Signets = append(slices.Clone(c.Signets), SignetConfig{Slug: in.Slug, Root: in.Root, SignetID: "pending", DeviceID: "pending", Remote: in.Remote})
	if err := probe.Validate(); err != nil {
		return out, err
	}
	if _, err := os.Lstat(in.Root); !errors.Is(err, os.ErrNotExist) {
		return out, errors.New("root already exists; init makes a fresh clone")
	}
	// The configuration and state are written last; catch an unwritable one
	// before anything is pushed.
	state := filepath.Join(c.StateDir, "signets", in.Slug)
	if err := os.MkdirAll(state, 0700); err != nil {
		return out, errors.New("cannot create the signet's state directory")
	}
	// Each probe has its own name, so a temp file an interrupted write left
	// behind cannot fail it.
	for _, path := range []string{in.ConfigPath, filepath.Join(state, "migration.json")} {
		f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".probe-*")
		if err != nil {
			return out, fmt.Errorf("cannot write beside %s", filepath.Base(path))
		}
		f.Close()
		os.Remove(f.Name())
	}
	receipts := map[string]cutover.Receipt{}
	for _, path := range in.Preflights {
		r, err := readReceipt(path)
		if err != nil {
			return out, fmt.Errorf("preflight receipt %s: %w", filepath.Base(path), err)
		}
		if r.DeviceID == "" {
			return out, fmt.Errorf("preflight receipt %s names no device: its clone has no binding; mark that device lost instead", filepath.Base(path))
		}
		if _, dup := receipts[r.DeviceID]; dup {
			return out, fmt.Errorf("two preflight receipts for device %s", r.DeviceID)
		}
		receipts[r.DeviceID] = r
	}
	lost := map[string]string{}
	for _, l := range in.Lost {
		if strings.TrimSpace(l.Reason) == "" || len(l.Reason) > 1024 {
			return out, fmt.Errorf("device %s needs --reason (1–1024 bytes)", l.ID)
		}
		if _, dup := lost[l.ID]; dup {
			return out, fmt.Errorf("device %s is marked lost twice", l.ID)
		}
		if _, both := receipts[l.ID]; both {
			return out, fmt.Errorf("device %s has a preflight receipt and is marked lost; choose one", l.ID)
		}
		lost[l.ID] = l.Reason
	}
	if err := os.MkdirAll(filepath.Dir(in.Root), 0700); err != nil {
		return out, errors.New("cannot create the root's parent directory")
	}
	work, err := os.MkdirTemp(filepath.Dir(in.Root), ".mandalore-init-")
	if err != nil {
		return out, errors.New("cannot create a working directory beside root")
	}
	// The working directory and the clone in it belong to this call alone.
	defer os.RemoveAll(work)
	checkout := filepath.Join(work, "signet")
	if err := signetsync.Clone(ctx, in.Remote, checkout); err != nil {
		return out, errors.New("cannot clone main from the remote")
	}
	store, err := memory.Open(checkout)
	if err != nil {
		return out, errors.New("the remote is not a signet")
	}
	if err := store.Validate(); err != nil {
		return out, fmt.Errorf("the remote signet fails validation: %w", err)
	}
	if store.Signet.Version < 2 {
		return out, errors.New("the remote signet is format 1; upgrade it to format 2 first (docs/migration-2.0.md runbook step 1, or A new signet in docs/server.md)")
	}
	out.SignetID = store.Signet.ID
	g, err := signetsync.Open(checkout, store.Signet.ID)
	if err != nil {
		return out, err
	}
	if out.ClonedHead, err = g.Head(ctx); err != nil {
		return out, err
	}
	enrolled, err := store.DeviceIDs()
	if err != nil {
		return out, errors.New("enrolled devices are unreadable")
	}
	missing := []string{}
	for _, id := range enrolled {
		if reason, ok := lost[id]; ok {
			out.Devices = append(out.Devices, DeviceEvidence{DeviceID: id, Evidence: "lost", Reason: reason})
			delete(lost, id)
			continue
		}
		r, ok := receipts[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		delete(receipts, id)
		if r.SignetID != store.Signet.ID {
			return out, fmt.Errorf("device %s's preflight is for another signet", id)
		}
		if !r.Clean {
			return out, fmt.Errorf("device %s is blocked: %s", id, strings.Join(r.Blockers, "; "))
		}
		if ok, err := g.Contains(ctx, r.Head); err != nil || !ok {
			return out, fmt.Errorf("device %s's preflight head is not on the remote; sync that clone and preflight again", id)
		}
		out.Devices = append(out.Devices, DeviceEvidence{DeviceID: id, Evidence: "preflight", Head: r.Head, CheckedAt: r.CheckedAt})
	}
	for id := range receipts {
		return out, fmt.Errorf("preflight receipt names device %s, which this signet has not enrolled", id)
	}
	for id := range lost {
		return out, fmt.Errorf("device %s is marked lost but this signet has not enrolled it", id)
	}
	if len(missing) > 0 {
		return out, fmt.Errorf("no clean preflight for enrolled device(s) %s; pass each one's receipt with --preflight, or --device-lost ID --reason TEXT", strings.Join(missing, ", "))
	}
	service, err := memory.OpenReadOnlyService(checkout)
	if err != nil {
		return out, err
	}
	out.ServerDeviceID, err = service.EnrollClientDevice(fmt.Sprintf("Mandalore server (%s), enrolled %s", in.Slug, time.Now().UTC().Format("2006-01-02")))
	if err != nil {
		return out, err
	}
	if err := os.Rename(checkout, in.Root); err != nil {
		return out, errors.New("cannot move the new checkout to root")
	}
	// From here the checkout is at root and the configuration changes last;
	// a failure says what is already written.
	pushState := "not pushed"
	late := func(what string) error {
		return &cutover.PartialError{Err: errors.New(what), Written: fmt.Sprintf("the checkout at %s with server device %s enrolled (%s); the configuration is unchanged (see Recovery in docs/migration-2.0.md)", in.Root, out.ServerDeviceID, pushState)}
	}
	if g, err = signetsync.Open(in.Root, out.SignetID); err != nil {
		return out, late("cannot reopen the checkout")
	}
	if _, err := g.Checkpoint(ctx); err != nil {
		return out, late("cannot commit the server device enrollment")
	}
	out.InitializedAt = time.Now().UTC().Format(time.RFC3339)
	push, cancel := context.WithTimeout(ctx, time.Minute)
	out.PushedHead, err = g.Push(push, in.Remote)
	cancel()
	out.Pushed = err == nil
	pushState = "pushed to the remote as " + short(out.PushedHead)
	if !out.Pushed {
		pushState = "push failed, so the remote may or may not have it"
		out.Notice = "The enrollment commit is not pushed yet; the server pushes it when it starts. Verify after that."
	}
	if err := writeState(filepath.Join(state, "migration.json"), out); err != nil {
		return out, late("cannot record the migration in server state")
	}
	c.Signets = append(c.Signets, SignetConfig{Slug: in.Slug, Root: in.Root, SignetID: out.SignetID, DeviceID: out.ServerDeviceID, Remote: in.Remote})
	if err := c.Validate(); err != nil {
		return out, late(err.Error())
	}
	if err := writeConfig(in.ConfigPath, c); err != nil {
		return out, late("cannot write the server configuration")
	}
	return out, nil
}

// writeConfig is init's last write; tests make it fail.
var writeConfig = func(path string, c Config) error { return writeState(path, c) }

// readConfig decodes a configuration strictly but, unlike LoadConfig, allows
// it to have no signets yet.
func readConfig(path string) (Config, error) {
	var c Config
	raw, err := readBounded(path, maxConfigBytes)
	if err != nil {
		return c, errors.New("server configuration unavailable")
	}
	if err := strictjson.Decode(raw, &c, maxConfigBytes); err != nil {
		return c, errors.New("invalid server configuration: unknown, duplicate or mistyped field")
	}
	return c, nil
}

func readReceipt(path string) (cutover.Receipt, error) {
	var envelope struct {
		ProtocolVersion int             `json:"protocol_version"`
		OK              bool            `json:"ok"`
		Result          cutover.Receipt `json:"result"`
	}
	raw, err := readBounded(path, 65536)
	if err != nil {
		return envelope.Result, errors.New("unreadable")
	}
	if err := strictjson.Decode(raw, &envelope, 65536); err != nil || !envelope.OK {
		return envelope.Result, errors.New("not the JSON output of a successful preflight")
	}
	if r := envelope.Result; r.Version != 1 || r.Kind != cutover.ReceiptKind {
		return r, errors.New("not a preflight receipt")
	}
	return envelope.Result, nil
}

func readBounded(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err == nil && int64(len(raw)) > limit {
		err = errors.New("file too large")
	}
	return raw, err
}

type Verification struct {
	Slug       string `json:"slug"`
	Head       string `json:"head"`
	RemoteHead string `json:"remote_head"`
	CanaryID   string `json:"canary_record_id"`
	Notice     string `json:"notice"`
}

// Verify proves a migrated signet: its head equals the remote, a read works,
// and a canary written then withdrawn in a task scope pushes cleanly. It
// writes as the server, so it refuses while the server is listening.
func Verify(ctx context.Context, c Config, slug string) (Verification, error) {
	out := Verification{Slug: slug}
	if err := c.Validate(); err != nil {
		return out, err
	}
	i := slices.IndexFunc(c.Signets, func(sc SignetConfig) bool { return sc.Slug == slug })
	if i < 0 {
		return out, fmt.Errorf("unknown signet %q", slug)
	}
	sc := c.Signets[i]
	if conn, err := net.DialTimeout("tcp", c.Listen, time.Second); err == nil {
		conn.Close()
		return out, errors.New("the server is listening; stop it first, since verify writes as the server")
	}
	g, err := signetsync.Open(sc.Root, sc.SignetID)
	if err != nil {
		return out, errors.New("cannot open the signet checkout")
	}
	run, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	if out.Head, err = g.Head(run); err != nil {
		return out, err
	}
	if out.RemoteHead, err = g.RemoteHead(run, sc.Remote); err != nil {
		return out, errors.New("remote unreachable")
	}
	if out.Head == "" || out.Head != out.RemoteHead {
		return out, fmt.Errorf("server head %s differs from remote %s", short(out.Head), short(out.RemoteHead))
	}
	m, err := memory.OpenService(sc.Root, memory.Authorship{DeviceID: sc.DeviceID, Actor: c.Owner.Name, Harness: "mandalore-server"})
	if err != nil {
		return out, errors.New("cannot open the signet as the server device")
	}
	if _, err := m.Recall("", nil, 5, 8192); err != nil {
		return out, fmt.Errorf("synthetic read failed: %w", err)
	}
	scope := &memory.Scope{Kind: "task", ID: "migration-verify"}
	visible := func(id string) (bool, error) {
		p, err := m.Recall("migration verification canary", scope, 50, 32768)
		return slices.ContainsFunc(p.Current, func(h memory.Hit) bool { return h.RecordID == id }), err
	}
	// From the canary write on, a failure says what is already written.
	stage := "a canary write was attempted and may be partly saved"
	late := func(what string) error {
		return &cutover.PartialError{Err: errors.New(what), Written: stage + " in " + sc.Root + " (see Recovery in docs/migration-2.0.md)"}
	}
	canary, err := m.Remember(memory.Write{Kind: "fact", Summary: "Migration verification canary", Scope: scope, Basis: "observation",
		Body:   "Written by mandalore server migrate verify at " + time.Now().UTC().Format(time.RFC3339) + " and withdrawn at once.",
		Reason: "Prove the 2.0 server writes and pushes cleanly after the cutover."})
	if err != nil {
		return out, late("canary write failed: " + err.Error())
	}
	out.CanaryID = canary.RecordID
	stage = "canary " + canary.RecordID + ", visible and not withdrawn, uncommitted"
	if ok, err := visible(canary.RecordID); err != nil || !ok {
		return out, late("the canary is not visible after its write")
	}
	if _, err := m.ChangeVisibility(run, "withdraw", memory.VisibilityWrite{RecordID: canary.RecordID, ContentHeads: []string{canary.ID}, VisibilityHeads: []string{}, Reason: "Migration verification canary"}); err != nil {
		return out, late("canary withdraw failed: " + err.Error())
	}
	stage = "canary " + canary.RecordID + ", withdrawn, uncommitted"
	if ok, err := visible(canary.RecordID); err != nil || ok {
		return out, late("the canary is still visible after its withdrawal")
	}
	if _, err := g.Checkpoint(run); err != nil {
		return out, late("cannot commit the canary")
	}
	stage = "canary " + canary.RecordID + ", withdrawn and committed, not confirmed pushed"
	pushed, err := g.Push(run, sc.Remote)
	if err != nil {
		return out, late("the canary did not push cleanly")
	}
	stage = "canary " + canary.RecordID + ", withdrawn, committed and pushed as " + short(pushed)
	if out.RemoteHead, err = g.RemoteHead(run, sc.Remote); err != nil || out.RemoteHead != pushed {
		return out, late("the remote does not show the pushed canary")
	}
	out.Head = pushed
	out.Notice = "Verified: head equals the remote, and a withdrawn canary pushed cleanly. Start the server."
	return out, nil
}

func short(head string) string {
	if len(head) > 12 {
		return head[:12]
	}
	if head == "" {
		return "(none)"
	}
	return head
}
