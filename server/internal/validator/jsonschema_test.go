package validator

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCheckSchema(t *testing.T) {
	cases := []struct {
		name   string
		schema string
		wantOK bool
		errHas string
	}{
		{"empty", `{}`, true, ""},
		{"blank", ``, true, ""},
		{"valid object", `{"type":"object","required":["title"],"properties":{"title":{"type":"string"}}}`, true, ""},
		{"missing type", `{"properties":{"a":{"type":"string"}}}`, false, `"type": "object"`},
		{"wrong type", `{"type":"objet"}`, false, `"object"`},
		{"array type", `{"type":["object","null"]}`, false, "exactly"},
		{"not an object", `[1,2]`, false, "invalid JSON"},
		{"broken json", `{"type":"object",`, false, "invalid JSON"},
		{"bad keyword value", `{"type":"object","properties":{"a":{"type":"strng"}}}`, false, "invalid schema"},
		{"external ref", `{"type":"object","properties":{"a":{"$ref":"http://example.com/x.json"}}}`, false, "external reference"},
		{"file ref", `{"type":"object","properties":{"a":{"$ref":"file:///etc/passwd"}}}`, false, "external reference"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckSchema(json.RawMessage(tc.schema))
			if tc.wantOK && err != nil {
				t.Fatalf("expected OK, got %v", err)
			}
			if !tc.wantOK {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tc.errHas) {
					t.Errorf("error %q should contain %q", err, tc.errHas)
				}
			}
		})
	}
}

func TestValidateAttributes(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","required":["title"],"properties":{"title":{"type":"string","minLength":1},"year":{"type":"integer","minimum":1900}}}`)

	if err := ValidateAttributes(schema, json.RawMessage(`{"title":"Breaking Bad","year":2008}`)); err != nil {
		t.Fatalf("valid data rejected: %v", err)
	}
	if err := ValidateAttributes(json.RawMessage(`{}`), json.RawMessage(`{"anything":1}`)); err != nil {
		t.Fatalf("empty schema must accept everything: %v", err)
	}

	err := ValidateAttributes(schema, json.RawMessage(`{"year":1800}`))
	if err == nil {
		t.Fatal("expected validation error")
	}
	fields := FormatValidationError(err)
	if _, ok := fields["/"]; !ok {
		t.Errorf("expected a root error for the missing title, got %v", fields)
	}
	if _, ok := fields["/year"]; !ok {
		t.Errorf("expected an error on /year, got %v", fields)
	}

	if err := ValidateAttributes(schema, json.RawMessage(`not json`)); err == nil {
		t.Error("expected error for invalid data JSON")
	}
}

func TestCompileCache(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"n":{"type":"integer"}}}`)
	first, err := compile(schema)
	if err != nil {
		t.Fatal(err)
	}
	second, err := compile(json.RawMessage(string(schema))) // same content, other buffer
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("expected the cached *Schema to be reused")
	}
}
