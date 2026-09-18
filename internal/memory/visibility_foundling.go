package memory

import "errors"

var ErrVisibilityWithheld = errors.New("foundling promotion is withheld; inspect and explicitly resolve the existing record visibility before promoting this origin")

func visibilityWithheld(state string) bool {
	return state != "visible" && state != "content-conflict"
}

// Called by the foundling verification callback while putSourced holds the writer
// lock and before either source or revision is published. Registration IDs and
// pins are intentionally not the identity: relinking unchanged source bytes at
// the same portable source/path must not evade a withdrawal. This is an exact
// origin check, not semantic duplicate detection or a ban on manual learning.
func (s *Store) checkFoundlingVisibility(recordID string, origin *ExternalOrigin) error {
	if s.Signet.Version == 1 {
		return nil
	}
	records, err := s.revisions()
	if err != nil {
		return err
	}
	states, err := s.validateGraphState(records, nil)
	if err != nil {
		return err
	}
	if recordID != "" && visibilityWithheld(states[recordID].State) {
		return ErrVisibilityWithheld
	}
	seenSources := map[string]bool{}
	for _, r := range records {
		if !visibilityWithheld(states[r.RecordID].State) {
			continue
		}
		for _, id := range r.Evidence.SourceRefs {
			if seenSources[id] {
				continue
			}
			seenSources[id] = true
			var source Source
			if err := s.readSource(id, &source); err != nil {
				return err
			}
			o := source.ExternalOrigin
			if o != nil && o.SourceIdentity == origin.SourceIdentity && o.RelativeLocator == origin.RelativeLocator && o.ContentSHA256 == origin.ContentSHA256 {
				return ErrVisibilityWithheld
			}
		}
	}
	return nil
}
