package schema

import (
	"encoding/json"
	"fmt"
)

// ValidateInputs checks that the supplied raw inputs payload conforms to
// the schema. A file field is permitted to be missing from inputs because
// its value is referenced indirectly via file_blob_key on the job row.
func (s Schema) ValidateInputs(raw json.RawMessage) error {
	if len(raw) == 0 {
		return fmt.Errorf("inputs are empty")
	}
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("inputs are not a JSON object: %w", err)
	}
	return s.validateObject(v, "")
}

func (s Schema) validateObject(obj map[string]any, path string) error {
	declared := make(map[string]Field, len(s))
	for _, f := range s {
		declared[f.Name] = f
	}
	for _, f := range s {
		val, present := obj[f.Name]
		if !present {
			if f.Required && f.Type != TypeFile {
				return fmt.Errorf("missing required field: %s", joinPath(path, f.Name))
			}
			continue
		}
		if err := validateValue(val, f, joinPath(path, f.Name)); err != nil {
			return err
		}
	}
	return nil
}

func validateValue(val any, f Field, path string) error {
	if val == nil {
		if f.Required {
			return fmt.Errorf("field %s is required and cannot be null", path)
		}
		return nil
	}
	switch f.Type {
	case TypeString:
		if _, ok := val.(string); !ok {
			return fmt.Errorf("field %s: expected string, got %T", path, val)
		}
	case TypeInteger:
		num, ok := val.(float64)
		if !ok {
			return fmt.Errorf("field %s: expected integer, got %T", path, val)
		}
		if num != float64(int64(num)) {
			return fmt.Errorf("field %s: expected integer, got float", path)
		}
	case TypeNumber:
		if _, ok := val.(float64); !ok {
			return fmt.Errorf("field %s: expected number, got %T", path, val)
		}
	case TypeBoolean:
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("field %s: expected boolean, got %T", path, val)
		}
	case TypeFile:
		// Inputs may carry the file blob key as a string under this name,
		// but it's not required - the API references it via the dedicated
		// file_blob_key column. We allow either a string or absence.
		if _, ok := val.(string); !ok {
			return fmt.Errorf("field %s: expected string blob key, got %T", path, val)
		}
	case TypeObject:
		obj, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("field %s: expected object, got %T", path, val)
		}
		nested := Schema(f.Fields)
		if err := nested.validateObject(obj, path); err != nil {
			return err
		}
	case TypeArray:
		arr, ok := val.([]any)
		if !ok {
			return fmt.Errorf("field %s: expected array, got %T", path, val)
		}
		if f.Items == nil {
			return fmt.Errorf("field %s: array schema missing items", path)
		}
		for i, item := range arr {
			if err := validateValue(item, *f.Items, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("field %s: unknown type %q", path, f.Type)
	}
	return nil
}

func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}
