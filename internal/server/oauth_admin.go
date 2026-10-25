package server

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// ClientInfo describes one OAuth client for `server clients list`.
type ClientInfo struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Kind         string           `json:"kind"`
	Status       string           `json:"status"` // pending or approved
	Registered   time.Time        `json:"registered"`
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

// Clients lists approved and pending clients, oldest first.
func Clients(c Config) ([]ClientInfo, error) {
	store, err := adminStore(c)
	if err != nil {
		return nil, err
	}
	out := []ClientInfo{}
	now := time.Now()
	err = store.view(func(st *oauthState) error {
		for _, cl := range st.Clients {
			if len(cl.Grants) == 0 && !now.Before(cl.Registered.Add(pendingTTL)) {
				continue
			}
			info := ClientInfo{ID: cl.ID, Name: cl.Name, Kind: cl.Kind, Status: "pending", Registered: cl.Registered, RedirectURIs: cl.RedirectURIs, Grants: map[string]Scope{}}
			for slug, scope := range cl.Grants {
				info.Grants[slug], info.Status = scope, "approved"
			}
			for _, t := range st.Tokens {
				if t.Client == cl.ID && !t.Used && now.Before(t.Expires) {
					info.Tokens++
				}
			}
			out = append(out, info)
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Registered.Before(out[j].Registered) })
	return out, err
}

// RevokeClient removes a client, its grants and all its tokens. A running
// server refuses its access token on the next request.
func RevokeClient(c Config, id string) error {
	store, err := adminStore(c)
	if err != nil {
		return err
	}
	return store.update(func(st *oauthState) error {
		if st.Clients[id] == nil {
			return fmt.Errorf("unknown client %q", id)
		}
		st.revokeClient(id)
		delete(st.Clients, id)
		return nil
	})
}

// SetClientScope changes a client's grant on one signet, or on every signet
// it holds when slug is empty. Live tokens never exceed the new scope.
func SetClientScope(c Config, id, slug string, scope Scope) error {
	if !scope.valid() {
		return errors.New("scope must be read or read-write")
	}
	store, err := adminStore(c)
	if err != nil {
		return err
	}
	return store.update(func(st *oauthState) error {
		cl := st.Clients[id]
		if cl == nil {
			return fmt.Errorf("unknown client %q", id)
		}
		if slug != "" {
			if _, ok := cl.Grants[slug]; !ok {
				return fmt.Errorf("client %q has no grant on signet %q", id, slug)
			}
			cl.Grants[slug] = scope
			return nil
		}
		if len(cl.Grants) == 0 {
			return fmt.Errorf("client %q is pending; the owner approves it from the consent page", id)
		}
		for s := range cl.Grants {
			cl.Grants[s] = scope
		}
		return nil
	})
}
