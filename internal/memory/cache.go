package memory

import "sync"

// readCache keeps the validated revision graph between reads. It is safe only
// in a process that is the signet's sole writer: every write here takes the
// writer lock, which drops the cache, but a change made by another process is
// not seen until this one writes.
type readCache struct {
	mu      sync.Mutex
	records []Revision
	states  map[string]VisibilityState
	terms   map[string]termIndex // by revision ID
}

// CacheReads makes recall and scope reads reuse the validated graph until the
// next write through this service or its views. Only a sole writer that
// serializes its reads with its writes may call it.
func (s *Service) CacheReads() {
	s.store.reads = &readCache{}
}

func (c *readCache) drop() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.records, c.states, c.terms = nil, nil, nil
	c.mu.Unlock()
}

// index returns the revision's terms, kept until the next write when caching.
func (c *readCache) index(r Revision) termIndex {
	if c == nil {
		return indexTerms(r)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	ix, ok := c.terms[r.ID]
	if !ok {
		if c.terms == nil {
			c.terms = map[string]termIndex{}
		}
		ix = indexTerms(r)
		c.terms[r.ID] = ix
	}
	return ix
}

// graph returns the validated revisions and their visibility states. Callers
// must not modify them.
func (s *Store) graph() ([]Revision, map[string]VisibilityState, error) {
	if c := s.reads; c != nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.states != nil {
			return c.records, c.states, nil
		}
	}
	records, err := s.revisions()
	if err != nil {
		return nil, nil, err
	}
	states, err := s.validateGraphState(records, nil)
	if err != nil {
		return nil, nil, err
	}
	if c := s.reads; c != nil {
		c.records, c.states = records, states
	}
	return records, states, nil
}
