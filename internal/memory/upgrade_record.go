package memory

import (
	"errors"
	"regexp"
	"time"
)

// UpgradeRecord is append-only evidence of a specific local format transition.
// It is not permission to upgrade another clone, a visibility decision, or proof
// of Git delivery. Transition code must independently compare the pinned blobs.
type UpgradeRecord struct {
	Version                int        `json:"schema_version"`
	ID                     string     `json:"id"`
	SignetID               string     `json:"signet_id"`
	From                   int        `json:"from_version"`
	To                     int        `json:"to_version"`
	OriginalManifestSHA256 string     `json:"original_manifest_sha256"`
	BaseHead               string     `json:"base_head"`
	PortableSHA256         string     `json:"portable_sha256"`
	RecordedAt             string     `json:"recorded_at"`
	Authorship             Authorship `json:"authorship"`
}

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)
var gitObjectHex = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)
var ErrUpgradePending = errors.New("format upgrade is prepared but not active; inspect explicit upgrade recovery before further writes")

// ValidateUpgradeRecords checks the closed receipt inventory, not its claimed
// historical digests. Snapshot consumers and sync must additionally enforce exact
// old-evidence preservation against their pinned source/base when transitioning.
func ValidateUpgradeRecords(s Signet, records []UpgradeRecord, device func(string) error) error {
	if s.Version != 1 && s.Version != 2 {
		return errors.New("unsupported signet format")
	}
	if s.Version == 2 && len(records) == 0 {
		return errors.New("upgraded signet requires transition evidence")
	}
	seen := map[string]bool{}
	for _, r := range records {
		if r.Version != 1 || !identifier.MatchString(r.ID) || r.SignetID != s.ID || r.From != 1 || r.To != 2 || !sha256Hex.MatchString(r.OriginalManifestSHA256) || !sha256Hex.MatchString(r.PortableSHA256) || !gitObjectHex.MatchString(r.BaseHead) || seen[r.ID] {
			return errors.New("invalid format transition evidence")
		}
		if _, err := time.Parse(time.RFC3339Nano, r.RecordedAt); err != nil {
			return errors.New("invalid format transition time")
		}
		if err := validateAuthorship(r.Authorship, device); err != nil {
			return err
		}
		seen[r.ID] = true
	}
	if s.Version == 1 && len(records) > 0 {
		return ErrUpgradePending
	}
	return nil
}
