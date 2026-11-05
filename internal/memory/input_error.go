package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ErrInvalidInput marks a write refused because of its input, before anything
// was written. Messages name fields, limits and sizes, never input text.
var ErrInvalidInput = errors.New("invalid input")

type inputError string

func (e inputError) Error() string        { return string(e) }
func (e inputError) Is(target error) bool { return target == ErrInvalidInput }

func invalidInput(format string, args ...any) error {
	return inputError(fmt.Sprintf(format, args...))
}

// checkText applies textWithin and says which part of it failed.
func checkText(field, value string, maxBytes int) error {
	if len(value) > maxBytes {
		return invalidInput("%s is %d bytes; the limit is %d UTF-8 bytes", field, len(value), maxBytes)
	}
	if !textWithin(value, maxBytes) {
		return invalidInput("%s is required: 1–%d UTF-8 bytes, not blank, no NUL", field, maxBytes)
	}
	return nil
}

var revisionSchema = sync.OnceValues(memorySchema)

// checkRevisionInput validates a revision built from caller input on its own,
// so a bad enum or ID is the caller's error rather than a store failure.
func checkRevisionInput(r Revision) error {
	schema, err := revisionSchema()
	if err != nil {
		return err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	var value any
	if err = json.Unmarshal(b, &value); err != nil {
		return err
	}
	err = schema.Validate(value)
	var invalid *jsonschema.ValidationError
	if !errors.As(err, &invalid) {
		return err
	}
	return invalidInput("record field %s is invalid; allowed values are in the tool schema", inputField(invalid))
}

// inputField names the first failing field in the tool's terms.
func inputField(e *jsonschema.ValidationError) string {
	for len(e.Causes) > 0 {
		e = e.Causes[0]
	}
	parts := []string{}
	for _, part := range e.InstanceLocation {
		if _, err := strconv.Atoi(part); err != nil && part != "evidence" {
			parts = append(parts, part)
		}
	}
	field := strings.Join(parts, ".")
	switch field {
	case "":
		return "record"
	case "change_reason":
		return "reason"
	}
	return field
}

// Service write limits, in UTF-8 bytes.
const (
	MaxSummaryBytes        = 256
	MaxBodyBytes           = 8192
	MaxReasonBytes         = 1024
	MaxJournalKindBytes    = 64
	MaxJournalSummaryBytes = 4096
)

func checkJournal(kind, summary string) error {
	if err := checkText("kind", kind, MaxJournalKindBytes); err != nil {
		return err
	}
	return checkText("summary", summary, MaxJournalSummaryBytes)
}
