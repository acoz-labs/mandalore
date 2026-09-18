package memory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"
)

var ErrVisibilityUpgradeRequired = errors.New("withdrawal and restore require an explicit signet format upgrade")

type VisibilityWrite struct {
	RecordID        string   `json:"record_id"`
	ContentHeads    []string `json:"expected_content_heads"`
	VisibilityHeads []string `json:"expected_visibility_heads"`
	Reason          string   `json:"reason"`
}

type VisibilityReceipt struct {
	SignetID             string           `json:"signet_id"`
	RecordID             string           `json:"record_id"`
	EventID              string           `json:"event_id,omitempty"`
	State                *VisibilityState `json:"visibility,omitempty"`
	DurableLocally       bool             `json:"durable_locally"`
	WriteMayHaveOccurred bool             `json:"write_may_have_occurred"`
	Synchronization      string           `json:"synchronization"`
	Notice               string           `json:"notice"`
}

type VisibilityHistory struct {
	SignetID string                `json:"signet_id"`
	RecordID string                `json:"record_id"`
	State    VisibilityState       `json:"visibility"`
	Events   Page[VisibilityEvent] `json:"events"`
	Notice   string                `json:"notice"`
}

func (s *Service) VisibilityHistory(recordID string, offset, limit int) (VisibilityHistory, error) {
	out := VisibilityHistory{SignetID: s.ID(), RecordID: recordID, Notice: "Visibility decisions preserve content history. Page offsets may shift after writes; inspect all heads before deciding. Previously read context and offline copies cannot be revoked."}
	if !identifier.MatchString(recordID) {
		return out, errors.New("invalid record ID")
	}
	records, err := s.store.revisions()
	if err != nil {
		return out, err
	}
	states, err := s.store.validateGraphState(records, nil)
	if err != nil {
		return out, err
	}
	state, ok := states[recordID]
	if !ok {
		return out, errors.New("record not found")
	}
	state.ContentHeads = append([]string{}, state.ContentHeads...)
	state.VisibilityHeads = append([]string{}, state.VisibilityHeads...)
	out.State = state
	events, err := s.store.visibilityEvents()
	if err != nil {
		return out, err
	}
	// Derive the returned heads from the very events being paged. If concurrent
	// content adds a missing reference, fail validation rather than mix views.
	states, err = ResolveVisibility(records, events, memoizeDeviceValidation(s.store.deviceExists))
	if err != nil {
		return out, err
	}
	state = states[recordID]
	state.ContentHeads = append([]string{}, state.ContentHeads...)
	state.VisibilityHeads = append([]string{}, state.VisibilityHeads...)
	out.State = state
	selected := []VisibilityEvent{}
	for _, e := range events {
		if e.RecordID == recordID {
			selected = append(selected, e)
		}
	}
	out.Events, err = page(selected, offset, limit)
	if err != nil {
		return out, err
	}
	if !fits(out, 32768) {
		return VisibilityHistory{}, errors.New("visibility history exceeds 32 KiB; reduce page size or inspect local evidence")
	}
	return out, nil
}

// ChangeVisibility never checkpoints or synchronizes. Expected heads are an
// explicit compare-and-swap decision, not an instruction to pick a winner.
func (s *Service) ChangeVisibility(ctx context.Context, action string, in VisibilityWrite) (out VisibilityReceipt, err error) {
	return s.changeVisibility(ctx, action, in, writeNewJSON)
}

func (s *Service) changeVisibility(ctx context.Context, action string, in VisibilityWrite, publish func(string, any) error) (out VisibilityReceipt, err error) {
	out = VisibilityReceipt{SignetID: s.ID(), Synchronization: "not-requested", Notice: "No visibility event saved."}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if (action != "withdraw" && action != "restore") || !identifier.MatchString(in.RecordID) || !textWithin(in.Reason, 4096) || len(in.ContentHeads) == 0 || len(in.ContentHeads) > 256 || in.VisibilityHeads == nil || len(in.VisibilityHeads) > 256 {
		return out, errors.New("visibility decision requires record ID, explicit expected heads (at most 256 each), and reason (1–4096 bytes)")
	}
	for _, refs := range [][]string{in.ContentHeads, in.VisibilityHeads} {
		seen := map[string]bool{}
		for _, id := range refs {
			if !identifier.MatchString(id) || seen[id] {
				return out, errors.New("invalid or duplicate expected head")
			}
			seen[id] = true
		}
	}
	out.RecordID = in.RecordID
	if s.store.Signet.Version != 2 {
		return out, ErrVisibilityUpgradeRequired
	}
	err = s.store.WithExclusiveLock(func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		records, err := s.store.revisions()
		if err != nil {
			return err
		}
		states, err := s.store.validateGraphState(records, nil)
		if err != nil {
			return err
		}
		state, ok := states[in.RecordID]
		if !ok || !sameHeads(in.ContentHeads, state.ContentHeads) || !sameHeads(in.VisibilityHeads, state.VisibilityHeads) {
			return ErrStaleHeads
		}
		events, err := s.store.visibilityEvents()
		if err != nil {
			return err
		}
		e := VisibilityEvent{Version: 1, ID: NewID("visibility"), RecordID: in.RecordID, Action: action, Observed: append([]string{}, state.ContentHeads...), Parents: append([]string{}, state.VisibilityHeads...), Reason: in.Reason, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: s.author}
		next, err := ResolveVisibility(records, append(events, e), memoizeDeviceValidation(s.store.deviceExists))
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		dir := filepath.Join(s.Root(), "memory/visibility", in.RecordID)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("invalid visibility directory")
		}
		out.EventID, out.WriteMayHaveOccurred = e.ID, true
		out.Notice = "Visibility publication attempted; inspect this event ID before retrying. Delivery was not requested."
		if err := publish(filepath.Join(dir, e.ID+".json"), e); err != nil {
			return err
		}
		// The event file and leaf directory were synced by writeNewJSON. Flush
		// newly-created ancestry too before claiming the event locally durable.
		for _, parent := range []string{filepath.Dir(dir), filepath.Dir(filepath.Dir(dir))} {
			f, err := os.Open(parent)
			if err != nil {
				return err
			}
			err = f.Sync()
			closeErr := f.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
		}
		result := next[in.RecordID]
		out.State = &result
		out.DurableLocally = true
		out.Notice = "Visibility decision saved locally; no checkpoint or delivery requested. Evidence is preserved, not erased. Previously read context and offline copies cannot be revoked."
		return ctx.Err()
	})
	return out, err
}
