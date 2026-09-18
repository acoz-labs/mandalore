package memory

import (
	"errors"
	"time"
)

// VisibilityEvent records an explicit visibility decision independently of the
// content supersession graph. It preserves evidence; it is not a deletion event.
// Version is the event schema version, not the enclosing signet format version.
type VisibilityEvent struct {
	Version    int        `json:"schema_version"`
	ID         string     `json:"id"`
	RecordID   string     `json:"record_id"`
	Action     string     `json:"action"`
	Observed   []string   `json:"observed_content_heads"`
	Parents    []string   `json:"parent_visibility_heads"`
	Reason     string     `json:"reason"`
	RecordedAt string     `json:"recorded_at"`
	Authorship Authorship `json:"authorship"`
}

// Metadata validation alone does not establish graph validity or authorize a
// write. Callers must validate all typed causal edges and compare expected heads.
func validateVisibilityEvent(e VisibilityEvent, device func(string) error) error {
	invalid := errors.New("invalid visibility event metadata")
	if e.Version != 1 || !identifier.MatchString(e.ID) || !identifier.MatchString(e.RecordID) || (e.Action != "withdraw" && e.Action != "restore") || !textWithin(e.Reason, 4096) || len(e.Observed) == 0 || e.Parents == nil {
		return invalid
	}
	if _, err := time.Parse(time.RFC3339Nano, e.RecordedAt); err != nil {
		return invalid
	}
	seen := map[string]bool{}
	for _, refs := range [][]string{e.Observed, e.Parents} {
		for _, id := range refs {
			if !identifier.MatchString(id) || id == e.ID || seen[id] {
				return invalid
			}
			seen[id] = true
		}
	}
	return validateAuthorship(e.Authorship, device)
}
