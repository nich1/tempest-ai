// Package schema defines the input/output schema DSL the API accepts on
// POST /jobs, validates user-supplied inputs against it, and converts it
// to a JSON Schema doc the LLM can consume in JSON-mode.
//
// DSL example (top level is a list of fields):
//
//   [
//     {"name": "company",  "type": "string",  "required": true},
//     {"name": "tags",     "type": "array",   "items": {"type": "string"}, "required": false},
//     {"name": "address",  "type": "object",  "required": true,
//       "fields": [
//         {"name": "city",    "type": "string", "required": true},
//         {"name": "country", "type": "string", "required": true}
//       ]
//     },
//     {"name": "report",   "type": "file", "required": false}  // input only, max 1
//   ]
package schema

import (
	"encoding/json"
	"fmt"
)

// Type enumerates the supported field types.
type Type string

const (
	TypeString  Type = "string"
	TypeInteger Type = "integer"
	TypeNumber  Type = "number"
	TypeBoolean Type = "boolean"
	TypeFile    Type = "file"
	TypeObject  Type = "object"
	TypeArray   Type = "array"
)

// Valid reports whether t is a known type.
func (t Type) Valid() bool {
	switch t {
	case TypeString, TypeInteger, TypeNumber, TypeBoolean, TypeFile, TypeObject, TypeArray:
		return true
	}
	return false
}

// Field is one entry in the schema. Only the fields relevant to the
// declared Type should be populated.
type Field struct {
	Name        string  `json:"name"`
	Type        Type    `json:"type"`
	Required    bool    `json:"required"`
	Description string  `json:"description,omitempty"`
	Fields      []Field `json:"fields,omitempty"` // for type=object
	Items       *Field  `json:"items,omitempty"`  // for type=array
}

// Schema is a list of top-level fields representing an object.
type Schema []Field

// Parse decodes a raw schema payload into a Schema.
func Parse(raw json.RawMessage) (Schema, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("schema is empty")
	}
	var s Schema
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}
	return s, nil
}

// Kind labels which side of a job a schema is for.
type Kind int

const (
	KindInput  Kind = 0
	KindOutput Kind = 1
)

// Validate checks the schema is structurally well-formed.
//
// For input schemas, file is allowed at the top level at most once.
// For output schemas, file is forbidden anywhere.
func (s Schema) Validate(kind Kind) error {
	if len(s) == 0 {
		return fmt.Errorf("schema must declare at least one field")
	}
	seen := make(map[string]bool, len(s))
	fileCount := 0
	for _, f := range s {
		if f.Name == "" {
			return fmt.Errorf("field has empty name")
		}
		if seen[f.Name] {
			return fmt.Errorf("duplicate field name: %q", f.Name)
		}
		seen[f.Name] = true
		if f.Type == TypeFile {
			if kind == KindOutput {
				return fmt.Errorf("field %q: file is not a valid output type", f.Name)
			}
			fileCount++
		}
		if err := validateFieldShape(f, kind, true); err != nil {
			return err
		}
	}
	if fileCount > 1 {
		return fmt.Errorf("at most one file field is allowed per schema")
	}
	return nil
}

// validateFieldShape walks a field and its nested children. topLevel
// constrains where file is allowed.
func validateFieldShape(f Field, kind Kind, topLevel bool) error {
	if !f.Type.Valid() {
		return fmt.Errorf("field %q: unknown type %q", f.Name, f.Type)
	}
	switch f.Type {
	case TypeFile:
		if !topLevel {
			return fmt.Errorf("field %q: file is only allowed at the top level of an input schema", f.Name)
		}
		if kind == KindOutput {
			return fmt.Errorf("field %q: file is not a valid output type", f.Name)
		}
	case TypeObject:
		if len(f.Fields) == 0 {
			return fmt.Errorf("field %q: object must declare nested fields", f.Name)
		}
		seen := make(map[string]bool, len(f.Fields))
		for _, child := range f.Fields {
			if seen[child.Name] {
				return fmt.Errorf("field %q: duplicate nested field %q", f.Name, child.Name)
			}
			seen[child.Name] = true
			if err := validateFieldShape(child, kind, false); err != nil {
				return err
			}
		}
	case TypeArray:
		if f.Items == nil {
			return fmt.Errorf("field %q: array must declare items", f.Name)
		}
		// Array items don't have names; we synthesize one for error messages.
		item := *f.Items
		if item.Name == "" {
			item.Name = f.Name + ".items"
		}
		if err := validateFieldShape(item, kind, false); err != nil {
			return err
		}
	}
	return nil
}
