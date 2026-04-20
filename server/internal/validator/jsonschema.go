package validator

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// ValidateAttributes validates data against a JSON Schema.
// Returns nil if valid or no schema. Returns an error describing violations.
func ValidateAttributes(schemaJSON, dataJSON json.RawMessage) error {
	if len(schemaJSON) == 0 || string(schemaJSON) == "{}" || string(schemaJSON) == "null" {
		return nil
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", bytes.NewReader(schemaJSON)); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}
	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return fmt.Errorf("invalid JSON data: %w", err)
	}

	if err := schema.Validate(data); err != nil {
		return err
	}
	return nil
}

// FormatValidationError converts a JSON Schema validation error into a map of field → message.
func FormatValidationError(err error) map[string]string {
	fields := make(map[string]string)
	if ve, ok := err.(*jsonschema.ValidationError); ok {
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
