package server

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"
)

// ClientInfo describes one installation of an OAuth client, or a pending
// client with none, for `server clients list`. Installations of a client ID
// metadata document share its ID, so each is listed on its own.
type ClientInfo struct {
	ID           string           `json:"id"`
	Installation string           `json:"installation,omitempty"`
	Name         string           `json:"name"`
	Kind         string           `json:"kind"`
	Status       string           `json:"status"` // pending or approved
	Registered   time.Time        `json:"registered"`
	FirstSeen    time.Time        `json:"first_seen,omitzero"`
	LastUsed     time.Time        `json:"last_used,omitzero"`
	RedirectURIs []string         `json:"redirect_uris"`
	Grants       map[string]Scope `json:"grants"`
	Tokens       int              `json:"live_tokens"`
}

func adminStore(c Config) (*oauthStore, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if c.auth() != AuthOAuth {
		return nil, errors.New("OAuth clients exist only with auth oauth")
	}
	return openOAuthStore(c.StateDir, time.Now)
}

// Clients lists approved installations and pending clients, oldest first.
func Clients(c Config) ([]ClientInfo, error) {
	store, err := adminStore(c)
	if err != nil {
		return nil, err
	}
	out := []ClientInfo{}
	now := time.Now()
	err = store.view(func(st *oauthState) error {
		for _, cl := range st.Clients {
			base := ClientInfo{ID: cl.ID, Name: cl.Name, Kind: cl.Kind, Status: "pending", Registered: cl.Registered, RedirectURIs: cl.RedirectURIs, Grants: map[string]Scope{}}
			installations := st.installationsOf(cl.ID)
			if len(installations) == 0 && now.Before(cl.Registered.Add(pendingTTL)) {
				out = append(out, base)
			}
			for _, in := range installations {
				info := base
				info.Installation, info.Status, info.FirstSeen, info.LastUsed, info.Grants = in.ID, "approved", in.FirstSeen, in.LastUsed, maps.Clone(in.Grants)
				for _, t := range st.Tokens {
					if t.Installation == in.ID && !t.Used && now.Before(t.Expires) {
						info.Tokens++
					}
				}
				out = append(out, info)
			}
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		return cmp.Or(out[i].Registered.Compare(out[j].Registered), out[i].FirstSeen.Compare(out[j].FirstSeen), strings.Compare(out[i].Installation, out[j].Installation)) < 0
	})
	return out, err
}

// RevokeClient removes a client, every installation of it, their grants and
// all their tokens, and returns how many installations it had. A running
// server refuses their access tokens on the next request.
func RevokeClient(c Config, id string) (int, error) {
	store, err := adminStore(c)
	if err != nil {
		return 0, err
	}
	var n int
	err = store.update(func(st *oauthState) error {
		if st.Clients[id] == nil {
			return fmt.Errorf("unknown client %q", id)
		}
		n = st.revokeClient(id)
		return nil
	})
	return n, err
}

// RevokeInstallation removes one installation, its grants and its tokens,
// and returns its client. Other installations of the client keep working.
func RevokeInstallation(c Config, id string) (string, error) {
	store, err := adminStore(c)
	if err != nil {
		return "", err
	}
	var client string
	err = store.update(func(st *oauthState) error {
		in := st.Installations[id]
		if in == nil {
			return fmt.Errorf("unknown installation %q", id)
		}
		client = in.Client
		st.revokeInstallations(func(x *installation) bool { return x.ID == id })
		return nil
	})
	return client, err
}

// SetClientScope changes the grant of every installation of a client on one
// signet, or on every signet they hold when slug is empty. Live tokens never
// exceed the new scope.
func SetClientScope(c Config, id, slug string, scope Scope) error {
	return setScope(c, slug, scope, func(st *oauthState) ([]*installation, error) {
		if st.Clients[id] == nil {
			return nil, fmt.Errorf("unknown client %q", id)
		}
		found := st.installationsOf(id)
		if len(found) == 0 {
			return nil, fmt.Errorf("client %q is pending; the owner approves it from the consent page", id)
		}
		return found, nil
	}, fmt.Sprintf("client %q", id))
}

// SetInstallationScope changes one installation's grant, like SetClientScope.
func SetInstallationScope(c Config, id, slug string, scope Scope) error {
	return setScope(c, slug, scope, func(st *oauthState) ([]*installation, error) {
		if in := st.Installations[id]; in != nil {
			return []*installation{in}, nil
		}
		return nil, fmt.Errorf("unknown installation %q", id)
	}, fmt.Sprintf("installation %q", id))
}

func setScope(c Config, slug string, scope Scope, target func(*oauthState) ([]*installation, error), name string) error {
	if !scope.valid() {
		return errors.New("scope must be read or read-write")
	}
	store, err := adminStore(c)
	if err != nil {
		return err
	}
	return store.update(func(st *oauthState) error {
		installations, err := target(st)
		if err != nil {
			return err
		}
		changed := false
		for _, in := range installations {
			for s := range in.Grants {
				if slug == "" || s == slug {
					in.Grants[s], changed = scope, true
				}
			}
		}
		if !changed {
			return fmt.Errorf("%s has no grant on signet %q", name, slug)
		}
		return nil
	})
}
