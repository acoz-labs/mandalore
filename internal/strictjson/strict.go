// Package strictjson validates bounded object inputs without echoing their values.
package strictjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"sync"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"
)

var ErrInvalid = errors.New("invalid JSON input: use the documented object schema and size limit")

type definition struct {
	schema   *jsonschema.Schema
	resolved *jsonschema.Resolved
}

var definitions sync.Map

func schemaFor(out any) (definition, error) {
	t := reflect.TypeOf(out)
	if t == nil || t.Kind() != reflect.Pointer {
		return definition{}, ErrInvalid
	}
	t = t.Elem()
	if cached, ok := definitions.Load(t); ok {
		return cached.(definition), nil
	}
	s, err := jsonschema.ForType(t, nil)
	if err != nil {
		return definition{}, ErrInvalid
	}
	resolved, err := s.Resolve(nil)
	if err != nil {
		return definition{}, ErrInvalid
	}
	d := definition{s, resolved}
	definitions.Store(t, d)
	return d, nil
}

// Schema is shared by discovery and input validation. Treat the returned value
// as immutable; callers must not mutate the cached schema.
func Schema(out any) (*jsonschema.Schema, error) {
	d, err := schemaFor(out)
	return d.schema, err
}

func value(d *json.Decoder, depth int) error {
	if depth > 64 {
		return ErrInvalid
	}
	token, err := d.Token()
	if err != nil {
		return ErrInvalid
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		keys := map[string]bool{}
		for d.More() {
			token, err := d.Token()
			if err != nil {
				return ErrInvalid
			}
			key, ok := token.(string)
			if !ok || keys[key] {
				return ErrInvalid
			}
			keys[key] = true
			if err := value(d, depth+1); err != nil {
				return err
			}
		}
		token, err = d.Token()
		if err != nil || token != json.Delim('}') {
			return ErrInvalid
		}
	case '[':
		for d.More() {
			if err := value(d, depth+1); err != nil {
				return err
			}
		}
		token, err = d.Token()
		if err != nil || token != json.Delim(']') {
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	return nil
}

func Decode(data []byte, out any, maxBytes int) error {
	if len(data) > maxBytes || !utf8.Valid(data) {
		return ErrInvalid
	}
	raw := bytes.TrimSpace(data)
	if len(raw) == 0 || raw[0] != '{' {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := value(d, 0); err != nil {
		return ErrInvalid
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrInvalid
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return ErrInvalid
	}
	schema, err := schemaFor(out)
	if err != nil {
		return ErrInvalid
	}
	if err := schema.resolved.Validate(generic); err != nil {
		return ErrInvalid
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return ErrInvalid
	}
	return nil
}
