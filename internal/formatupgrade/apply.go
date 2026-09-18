package formatupgrade

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

type ApplyRequest struct {
	Plan           Plan `json:"plan"`
	StoppedWriters bool `json:"stopped_writers"`
}

type Receipt struct {
	SignetID          string `json:"signet_id"`
	Phase             string `json:"phase"`
	StagingDirectory  string `json:"staging_directory,omitempty"`
	EvidenceID        string `json:"evidence_id,omitempty"`
	EvidencePublished bool   `json:"evidence_published"`
	Activated         bool   `json:"activated"`
	DurableLocally    bool   `json:"durable_locally"`
	Checkpointed      bool   `json:"checkpointed"`
	Delivered         bool   `json:"delivered"`
	Notice            string `json:"notice"`
}

type prepared struct {
	Plan             Plan                 `json:"plan"`
	Record           memory.UpgradeRecord `json:"record"`
	OriginalManifest string               `json:"original_manifest"`
	Manifest         string               `json:"manifest"`
}

var ErrApply = errors.New("upgrade stopped; inspect the receipt and retained preparation before explicit recovery; no rollback or synchronization was performed")

func Apply(ctx context.Context, in ApplyRequest) (Receipt, error)   { return apply(ctx, in, false, nil) }
func Recover(ctx context.Context, in ApplyRequest) (Receipt, error) { return apply(ctx, in, true, nil) }

func encoded(value any) ([]byte, error) {
	b, err := json.MarshalIndent(value, "", "  ")
	return append(b, '\n'), err
}

func apply(ctx context.Context, in ApplyRequest, resume bool, fault func(string) error) (out Receipt, err error) {
	out.Phase = "validate"
	defer func() {
		if err != nil {
			out.Notice = ErrApply.Error()
			if ctx.Err() != nil {
				err = ctx.Err()
			} else {
				err = ErrApply
			}
		}
	}()
	step := func(phase string) error {
		out.Phase = phase
		if err := ctx.Err(); err != nil {
			return err
		}
		if fault != nil {
			return fault(phase)
		}
		return nil
	}
	p := in.Plan
	if err = ctx.Err(); err != nil {
		return
	}
	if !in.StoppedWriters || !reflect.DeepEqual(p, planFor(p.BindingPath, p.BindingSHA256, p.Root, p.Source)) {
		return out, ErrApply
	}
	name, b, pin, err := readBinding(p.BindingPath)
	if err != nil || name != p.BindingPath || pin != p.BindingSHA256 || b.SignetID != p.Source.SignetID {
		return out, ErrApply
	}
	guard := binding.Guard{SHA256: pin, SignetID: b.SignetID}
	service, err := binding.OpenGuarded(name, "format-upgrade", guard)
	if err != nil {
		return
	}
	canonical, err := filepath.EvalSymlinks(service.Root())
	if err != nil || canonical != p.Root {
		return out, ErrApply
	}
	store, err := memory.Open(p.Root)
	if err != nil {
		return
	}
	out.SignetID = store.Signet.ID
	err = store.WithFormatUpgradeLock(func() error {
		root, err := os.OpenRoot(p.Root)
		if err != nil {
			return err
		}
		defer root.Close()
		info, err := root.Stat(".")
		if err != nil {
			return err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || fmt.Sprintf("%d:%d", st.Dev, st.Ino) != p.Source.RootIdentity {
			return ErrApply
		}
		_, stageErr := root.Lstat(stage)
		if stageErr == nil {
			out.StagingDirectory = filepath.Join(p.Root, stage)
		}
		if stageErr == nil && !resume {
			return ErrApply
		}
		if stageErr != nil && !errors.Is(stageErr, os.ErrNotExist) {
			return stageErr
		}
		if errors.Is(stageErr, os.ErrNotExist) {
			if resume {
				return ErrApply
			}
			fresh, err := Preview(ctx, Request{BindingPath: p.BindingPath})
			if err != nil || !reflect.DeepEqual(fresh, p) {
				return ErrApply
			}
			out.StagingDirectory = filepath.Join(p.Root, stage)
			if err := ensureDirectory(root, stage); err != nil {
				return err
			}
			if err := step("stage-created"); err != nil {
				return err
			}
		}
		stageInfo, err := root.Lstat(stage)
		if err != nil || !stageInfo.IsDir() || stageInfo.Mode().Perm() != 0700 {
			return ErrApply
		}
		var state prepared
		data, stateErr := readFile(root, stage+"/prepared.json", preparedLimit)
		if stateErr == nil {
			if err := strictjson.Decode(data, &state, preparedLimit); err != nil {
				return err
			}
		} else if errors.Is(stateErr, os.ErrNotExist) {
			fresh, err := Preview(ctx, Request{BindingPath: p.BindingPath})
			if err != nil || !reflect.DeepEqual(fresh, p) {
				return ErrApply
			}
			entries, err := root.Open(stage)
			if err != nil {
				return err
			}
			names, err := entries.Readdirnames(-1)
			entries.Close()
			if err != nil {
				return err
			}
			for _, name := range names {
				if !strings.HasPrefix(name, ".prepare-") {
					return ErrApply
				}
			}
			original, err := readFile(root, "signet.json", 4<<20)
			if err != nil {
				return err
			}
			var manifest memory.Signet
			if err := strictjson.Decode(original, &manifest, 4<<20); err != nil {
				return err
			}
			manifest.Version = 2
			next, err := encoded(manifest)
			if err != nil {
				return err
			}
			u := memory.UpgradeRecord{Version: 1, ID: memory.NewID("upgrade"), SignetID: b.SignetID, From: 1, To: 2, OriginalManifestSHA256: p.Source.ManifestSHA256, BaseHead: p.Source.Head, PortableSHA256: p.Source.PortableSHA256, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: memory.Authorship{DeviceID: b.DeviceID, Actor: b.Actor, Harness: "format-upgrade"}}
			state = prepared{Plan: p, Record: u, OriginalManifest: string(original), Manifest: string(next)}
			data, err := encoded(state)
			if err != nil {
				return err
			}
			if err := publish(root, stage+"/prepared.json", data); err != nil {
				return err
			}
			if err := syncPath(root, stage); err != nil {
				return err
			}
		} else {
			return stateErr
		}
		if !reflect.DeepEqual(state.Plan, p) || state.Record.Authorship != (memory.Authorship{DeviceID: b.DeviceID, Actor: b.Actor, Harness: "format-upgrade"}) {
			return ErrApply
		}
		out.EvidenceID = state.Record.ID
		transition := signetsync.UpgradeTransition{Source: p.Source, OriginalManifest: []byte(state.OriginalManifest), Manifest: []byte(state.Manifest), Record: state.Record}
		inspect := func() (signetsync.UpgradeTransitionState, error) {
			if _, err := binding.OpenGuarded(p.BindingPath, "format-upgrade", guard); err != nil {
				return signetsync.UpgradeTransitionState{}, err
			}
			sy, err := signetsync.Open(p.Root, p.Source.SignetID)
			if err != nil {
				return signetsync.UpgradeTransitionState{}, err
			}
			return sy.InspectUpgradeTransition(ctx, transition)
		}
		observed, err := inspect()
		out.EvidencePublished, out.Activated = observed.Published, observed.Activated
		if err != nil {
			return err
		}
		if err := step("prepared"); err != nil {
			return err
		}
		observed, err = inspect()
		out.EvidencePublished, out.Activated = observed.Published, observed.Activated
		if err != nil {
			return err
		}
		receiptPath := "provenance/upgrades/" + state.Record.ID + ".json"
		if !observed.Published {
			if err := ensureDirectory(root, "provenance/upgrades"); err != nil {
				return err
			}
			b, err := encoded(state.Record)
			if err != nil {
				return err
			}
			if err := publish(root, receiptPath, b); err != nil {
				return err
			}
			out.EvidencePublished = true
			if err := step("receipt-linked"); err != nil {
				return err
			}
		}
		if err := syncPath(root, receiptPath); err != nil {
			return err
		}
		if err := syncPath(root, "provenance/upgrades"); err != nil {
			return err
		}
		if err := step("receipt-durable"); err != nil {
			return err
		}
		if _, err := inspect(); err != nil {
			return err
		}
		if !observed.Activated {
			tmp := stage + "/.prepare-" + memory.NewID("manifest")
			if err := privateFile(root, tmp, []byte(state.Manifest)); err != nil {
				return err
			}
			current, err := readFile(root, "signet.json", 4<<20)
			if err != nil || !bytes.Equal(current, []byte(state.OriginalManifest)) {
				return ErrApply
			}
			if err := root.Rename(tmp, "signet.json"); err != nil {
				return err
			}
			out.Activated = true
			if err := step("manifest-replaced"); err != nil {
				return err
			}
		}
		if err := syncPath(root, "signet.json"); err != nil {
			return err
		}
		if err := syncPath(root, "."); err != nil {
			return err
		}
		if _, err := inspect(); err != nil {
			return err
		}
		if err := step("activated"); err != nil {
			return err
		}
		out.DurableLocally, out.Phase = true, "complete"
		out.Notice = "Format2 activated locally; retained recovery state preserved. Reconnect clients. No Git checkpoint or remote delivery was performed."
		return nil
	})
	return
}
