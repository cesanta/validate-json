package schema

import (
	"fmt"

	json "github.com/cesanta/ucl"
)

type Validator struct {
	schema json.Value
}

func NewValidator(schema json.Value) *Validator {
	return &Validator{schema: schema}
}

func (v *Validator) Validate(val json.Value) error {
	return v.validateAgainstSchema("#", val, "#", v.schema)
}

func (v *Validator) getSchemaByRef(uri string) (json.Value, error) {
	return nil, fmt.Errorf("schema refs are not supported yet")
}

func isOfType(val json.Value, t string) bool {
	switch val.(type) {
	case json.Array:
		return t == "array"
	case json.Bool:
		return t == "boolean"
	case json.Number:
		// TODO(imax): add proper support for integers.
		return t == "number" || t == "integer"
	case json.Null:
		return t == "null"
	case json.Object:
		return t == "object"
	case json.String:
		return t == "string"
	}
	return false
}

func (v *Validator) validateAgainstSchema(path string, val json.Value, schemaPath string, schema_ json.Value) error {
	schema, ok := schema_.(json.Object)
	if !ok {
		return fmt.Errorf("%q: schema must be an object", schemaPath)
	}

	ref, found := schema.Lookup("$ref")
	if found {
		sref, ok := ref.(json.String)
		if !ok {
			return fmt.Errorf("%q: must be a string", schemaPath+"/$ref")
		}
		s, err := v.getSchemaByRef(sref.Value)
		if err != nil {
			return err
		}
		return v.validateAgainstSchema(path, val, sref.Value, s)
	}

	t, found := schema.Lookup("type")
	if found {
		switch t := t.(type) {
		case json.String:
			if !isOfType(val, t.Value) {
				return fmt.Errorf("%q: must be of type %q", path, t.Value)
			}
		case json.Array:
			match := false
			for i, v := range t.Value {
				t, ok := v.(json.String)
				if !ok {
					return fmt.Errorf("%q: must be a string", schemaPath+fmt.Sprintf("/[%d]", i))
				}
				if isOfType(val, t.Value) {
					match = true
					break
				}
			}
			if !match {
				return fmt.Errorf("%q: must be of one of the types %s", path, t)
			}
		default:
			return fmt.Errorf("%q: must be a string or an array", schemaPath+"/type")
		}
	}

	return nil
}
