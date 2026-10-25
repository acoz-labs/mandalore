package server

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

type SignetStatus struct {
	Slug     string `json:"slug"`
	SignetID string `json:"signet_id"`
	Head     string `json:"head"`
	Backup   Backup `json:"backup"`
	// Remote is checked now: current, ahead, diverged, ahead-or-diverged or
	// unreachable.
	Remote string `json:"remote"`
	// Canon is each configured canon reference's served snapshot: state,
	// commit, age and last refresh error.
	Canon []foundlings.CanonReceipt `json:"canon,omitempty"`
}

// Status reports each signet's head, last push and pending push from the
// server's state, and checks each remote without fetching or writing.
func Status(ctx context.Context, c Config) ([]SignetStatus, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	out := []SignetStatus{}
	for _, sc := range c.Signets {
		g, err := signetsync.Open(sc.Root, sc.SignetID)
		if err != nil {
			return nil, fmt.Errorf("signet %s: cannot open checkout", sc.Slug)
		}
		st := SignetStatus{Slug: sc.Slug, SignetID: sc.SignetID, Backup: Backup{State: "unknown", Remote: "unknown"}}
		if st.Head, err = g.Head(ctx); err != nil {
			return nil, fmt.Errorf("signet %s: cannot read head", sc.Slug)
		}
		if err := readState(filepath.Join(c.StateDir, "signets", sc.Slug, "backup.json"), &st.Backup); err != nil {
			return nil, fmt.Errorf("signet %s: backup state unreadable", sc.Slug)
		}
		check, cancel := context.WithTimeout(ctx, 10*time.Second)
		st.Remote, err = g.RemoteState(check, sc.Remote, st.Backup.Head, false)
		cancel()
		if err != nil {
			st.Remote = "unreachable"
		}
		if len(sc.Canon) > 0 {
			m, err := canonManager(sc)
			if err == nil {
				st.Canon, err = m.CanonStatus(ctx)
			}
			if err != nil {
				return nil, fmt.Errorf("signet %s: canon status unreadable", sc.Slug)
			}
		}
		out = append(out, st)
	}
	return out, nil
}

type CanonRefresh struct {
	Slug  string                    `json:"slug"`
	Canon []foundlings.CanonReceipt `json:"canon"`
}

// RefreshCanon refreshes the configured canon references now, for one signet
// or every signet when slug is empty. A running server serves the new
// snapshots from its next canon read.
func RefreshCanon(ctx context.Context, c Config, slug string) ([]CanonRefresh, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	out := []CanonRefresh{}
	for _, sc := range c.Signets {
		if slug != "" && sc.Slug != slug {
			continue
		}
		m, err := canonManager(sc)
		var receipts []foundlings.CanonReceipt
		if err == nil {
			receipts, err = m.Refresh(ctx)
		}
		if err != nil {
			return nil, fmt.Errorf("signet %s: canon refresh failed", sc.Slug)
		}
		out = append(out, CanonRefresh{Slug: sc.Slug, Canon: receipts})
	}
	if slug != "" && len(out) == 0 {
		return nil, fmt.Errorf("unknown signet %q", slug)
	}
	return out, nil
}

// canonManager reads a signet's canon from a read-only view, as the server's
// refresh does.
func canonManager(sc SignetConfig) (*foundlings.Manager, error) {
	view, err := memory.OpenReadOnlyService(sc.Root)
	if err != nil {
		return nil, err
	}
	return foundlings.NewConfigured(view, foundlingConfig(sc)), nil
}

// Reconcile fast-forwards one signet's checkout to its remote after
// validating it. It refuses divergent history. Restart the server afterwards.
func Reconcile(ctx context.Context, c Config, slug string) (signetsync.Status, error) {
	if err := c.Validate(); err != nil {
		return signetsync.Status{}, err
	}
	for _, sc := range c.Signets {
		if sc.Slug != slug {
			continue
		}
		g, err := signetsync.Open(sc.Root, sc.SignetID)
		if err != nil {
			return signetsync.Status{}, err
		}
		run, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()
		return g.FastForward(run, sc.Remote)
	}
	return signetsync.Status{}, fmt.Errorf("unknown signet %q", slug)
}
