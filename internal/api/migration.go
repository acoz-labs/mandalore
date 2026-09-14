package api

import (
	"context"
	"errors"
	"io/fs"
	"os"

	"github.com/acoz-labs/mandalore/internal/migration"
)

type migrationFailure struct {
	err    error
	result *migration.Result
}

func (e *migrationFailure) Error() string { return e.err.Error() }
func (e *migrationFailure) Unwrap() error { return e.err }

func migrationError(err error, result *migration.Result) error {
	if err == nil {
		return nil
	}
	var pathErr *fs.PathError
	var linkErr *os.LinkError
	if errors.As(err, &pathErr) || errors.As(err, &linkErr) {
		err = errors.New("filesystem operation failed; inspect the explicit source, output and retained staging paths before retrying")
	}
	return &migrationFailure{err: err, result: result}
}

var migrations = []Operation{
	connectionOperation("migration_preflight", "Read-only bounded validation of an explicit My Friday memory-only bank, output path and optional legacy binding/native inventory. No predecessor execution, signet activation, Git copying or data writes. Native listing may produce native-owned logs.", true, func(ctx context.Context, in migration.Options) (migration.Plan, error) {
		p, err := migration.Preflight(ctx, in)
		return p, migrationError(err, nil)
	}),
	connectionOperation("migration_apply", "Convert an explicitly reviewed migration plan into a new retained bundle. Requires writers_stopped acknowledgement; preserves original source bytes and authorship. Does not bind, activate, initialize Git or synchronize. Inspect partial publication receipts before retrying.", false, func(ctx context.Context, in migration.ApplyInput) (migration.Result, error) {
		r, err := migration.Apply(ctx, in)
		return r, migrationError(err, &r)
	}),
}
