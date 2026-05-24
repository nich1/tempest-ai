package schema

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ToJSONSchema converts our DSL into a Draft-2020-12 JSON Schema document
// suitable for LLM JSON-mode requests.
//
// File fields are skipped (the LLM sees the file content, not a key).
func (s Schema) ToJSONSchema() ([]byte, error) {
	root := buildObject(s, true)
	return json.Marshal(root)
}

// ToJSONSchemaForOutput returns a JSON Schema where every field is
// emitted (no file type permitted in outputs).
func (s Schema) ToJSONSchemaForOutput() ([]byte, error) {
	root := buildObject(s, false)
	return json.Marshal(root)
}

func buildObject(fields []Field, skipFiles bool) map[string]any {
	props := map[string]any{}
	required := []string{}
	for _, f := range fields {
		if f.Type == TypeFile && skipFiles {
			continue
		}
		props[f.Name] = buildField(f, skipFiles)
		if f.Required && !(f.Type == TypeFile && skipFiles) {
			required = append(required, f.Name)
		}
	}
	out := map[string]any{
		"type":                 "object",
		"properties":           props,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func buildField(f Field, skipFiles bool) map[string]any {
	out := map[string]any{}
	if f.Description != "" {
		out["description"] = f.Description
	}
	switch f.Type {
	case TypeString:
		out["type"] = "string"
	case TypeInteger:
		out["type"] = "integer"
	case TypeNumber:
		out["type"] = "number"
	case TypeBoolean:
		out["type"] = "boolean"
	case TypeFile:
		// Files are sent as base64-encoded data URLs in some flows; we
		// describe it loosely as a string here for completeness.
		out["type"] = "string"
		out["description"] = strings.TrimSpace(f.Description + " (file content)")
	case TypeObject:
		nested := buildObject(f.Fields, skipFiles)
		for k, v := range nested {
			out[k] = v
		}
	case TypeArray:
		if f.Items != nil {
			out["type"] = "array"
			out["items"] = buildField(*f.Items, skipFiles)
		} else {
			out["type"] = "array"
		}
	}
	return out
}

// ToPromptHint emits a short human-readable description of the schema
// so the prompt can include it as a soft instruction (belt-and-suspenders
// alongside JSON mode).
func (s Schema) ToPromptHint(label string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s shape:\n", label)
	for _, f := range s {
		writeFieldHint(&b, f, 1)
	}
	return b.String()
}

func writeFieldHint(b *strings.Builder, f Field, depth int) {
	indent := strings.Repeat("  ", depth)
	req := ""
	if f.Required {
		req = " (required)"
	}
	switch f.Type {
	case TypeObject:
		fmt.Fprintf(b, "%s- %s: object%s\n", indent, f.Name, req)
		for _, c := range f.Fields {
			writeFieldHint(b, c, depth+1)
		}
	case TypeArray:
		if f.Items != nil {
			fmt.Fprintf(b, "%s- %s: array of %s%s\n", indent, f.Name, f.Items.Type, req)
		} else {
			fmt.Fprintf(b, "%s- %s: array%s\n", indent, f.Name, req)
		}
	default:
		fmt.Fprintf(b, "%s- %s: %s%s\n", indent, f.Name, f.Type, req)
	}
}
