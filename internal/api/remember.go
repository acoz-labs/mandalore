package api

import (
	"context"

	"github.com/acoz-labs/mandalore/internal/memory"
)

const correctionGuidance = " For corrections, keep the original record_id (record-...) and set supersedes to its current revision IDs (revision-...), not record IDs; both fields are required together. Preserve the original kind and scope. Omit both identity fields only for genuinely new knowledge. Obtain IDs from recall/history. After validation errors, inspect memory_history and the input schema; do not guess IDs, create duplicate/test records, or withdraw the original as a correction workaround."

func remember(_ context.Context, s *memory.Service, in memory.Write) (Receipt, error) {
	r, err := s.Remember(in)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{SignetID: s.ID(), ID: r.ID, RecordID: r.RecordID, DurableLocally: true, Synchronization: "not-requested"}, nil
}

func appendJournal(_ context.Context, s *memory.Service, in JournalWrite) (Receipt, error) {
	r, err := s.AppendJournal(in.Kind, in.Summary)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{SignetID: s.ID(), ID: r.ID, DurableLocally: true, Synchronization: "not-requested"}, nil
}
