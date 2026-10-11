// Package validator compiles the JSON Schemas carried by meta-types and
// validates instance attributes against them.
package validator

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// compiled caches compiled schemas by content hash: a schema is recompiled
// only when its text changes, whatever the number of items validated.
var compiled sync.Map // [32]byte → *jsonschema.Schema

// isEmptySchema reports whether the schema imposes no constraint.
func isEmptySchema(schemaJSON json.RawMessage) bool {
	t := bytes.TrimSpace(schemaJSON)
	return len(t) == 0 || bytes.Equal(t, []byte("{}")) || bytes.Equal(t, []byte("null"))
}

// compile returns the compiled schema, from the cache when possible. External
// `$ref` (http, file…) are refused: a schema must be self-contained.
func compile(schemaJSON json.RawMessage) (*jsonschema.Schema, error) {
	key := sha256.Sum256(schemaJSON)
	if v, ok := compiled.Load(key); ok {
		return v.(*jsonschema.Schema), nil
	}
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020
	c.LoadURL = func(url string) (io.ReadCloser, error) {
		return nil, fmt.Errorf("référence externe interdite: %s", url)
	}
	if err := c.AddResource("schema.json", bytes.NewReader(schemaJSON)); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}
	schema, err := c.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}
	compiled.Store(key, schema)
	return schema, nil
}

// CheckSchema verifies that an attributes_schema is acceptable before it is
// stored on a meta-type: valid JSON object, root `"type": "object"`, compilable,
// without external references. An empty schema ({}) is accepted.
func CheckSchema(schemaJSON json.RawMessage) error {
	if isEmptySchema(schemaJSON) {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal(schemaJSON, &obj); err != nil {
		return fmt.Errorf("JSON invalide: %w", err)
	}
	switch t := obj["type"].(type) {
	case nil:
		return errors.New(`"type": "object" est requis à la racine du schéma`)
	case string:
		if t != "object" {
			return fmt.Errorf(`le type racine doit être "object" (reçu %q)`, t)
		}
	default:
		return errors.New(`le type racine doit être exactement "object"`)
	}
	_, err := compile(schemaJSON)
	return err
}

// ValidateAttributes validates data against a JSON Schema.
// Returns nil if valid or no schema. Returns an error describing violations.
func ValidateAttributes(schemaJSON, dataJSON json.RawMessage) error {
	if isEmptySchema(schemaJSON) {
		return nil
	}
	schema, err := compile(schemaJSON)
	if err != nil {
		return err
	}

	var data any
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return fmt.Errorf("invalid JSON data: %w", err)
	}
	return schema.Validate(data)
}

// FormatValidationError converts a JSON Schema validation error into a map of field → message.
func FormatValidationError(err error) map[string]string {
	fields := make(map[string]string)
	var ve *jsonschema.ValidationError
	if errors.As(err, &ve) {
		for _, cause := range ve.Causes {
			path := cause.InstanceLocation
			if path == "" {
				path = "/"
			}
			fields[path] = cause.Message
		}
		if len(fields) == 0 {
			fields["/"] = ve.Message
		}
	} else {
		fields["/"] = err.Error()
	}
	return fields
}
