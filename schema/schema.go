package schema

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var validType = map[string]bool{
	"array":   true,
	"boolean": true,
	"integer": true,
	"null":    true,
	"number":  true,
	"object":  true,
	"string":  true,
}

func ValidateDraft04Schema(b []byte) error {
	var v interface{}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	return validateDraft04Schema("#", v)
}

func validateDraft04Schema(path string, v interface{}) error {
	switch v := v.(type) {
	case map[string]interface{}:
		s, found := v["$ref"]
		if found {
			return validateURI(path+"/$ref", s)
		}
		validators := map[string]func(string, interface{}) error{
			"type":                 validateType,
			"id":                   validateURI,
			"$schema":              validateURI,
			"title":                validateString,
			"description":          validateString,
			"multipleOf":           validateMultipleOf,
			"maximum":              validateNumber,
			"minimum":              validateNumber,
			"exclusiveMaximum":     validateBoolean,
			"exclusiveMinimum":     validateBoolean,
			"minLength":            validatePositiveInteger,
			"maxLength":            validatePositiveInteger,
			"pattern":              validatePattern,
			"additionalItems":      validateBoolOrSchema,
			"items":                validateItems,
			"maxItems":             validatePositiveInteger,
			"minItems":             validatePositiveInteger,
			"uniqueItems":          validateBoolean,
			"maxProperties":        validatePositiveInteger,
			"minProperties":        validatePositiveInteger,
			"required":             validateStringArray,
			"additionalProperties": validateBoolOrSchema,
			"definitions":          validateSchemaCollection,
			"properties":           validateSchemaCollection,
			"patternProperties":    validateSchemaCollection,
			"dependencies":         validateDependencies,
			"enum":                 validateEnum,
			"allOf":                validateSchemaArray,
			"anyOf":                validateSchemaArray,
			"oneOf":                validateSchemaArray,
			"not":                  validateDraft04Schema,
		}
		for prop, validate := range validators {
			val, found := v[prop]
			if !found {
				continue
			}
			err := validate(path+"/"+prop, val)
			if err != nil {
				return err
			}
		}
		_, a := v["exclusiveMaximum"]
		_, b := v["maximum"]
		if a && !b {
			return fmt.Errorf("%q: \"exclusiveMaximum\" requires \"maximum\" to be present")
		}
		_, a = v["exclusiveMinimum"]
		_, b = v["minimum"]
		if a && !b {
			return fmt.Errorf("%q: \"exclusiveMinimum\" requires \"minimum\" to be present")
		}
		return nil
	default:
		return fmt.Errorf("%q has invalid type, it needs to be an object", path)
	}
}

func validateType(path string, v interface{}) error {
	switch v := v.(type) {
	case string:
		if !validType[v] {
			return fmt.Errorf("%q: %q is not a valid type", path, v)
		}
		return nil
	case []interface{}:
		if len(v) < 1 {
			return fmt.Errorf("%q must have at least 1 element", path)
		}
		for _, t := range v {
			s, ok := t.(string)
			if !ok {
				return fmt.Errorf("%q: each element must be a string", path)
			}
			if !validType[s] {
				return fmt.Errorf("%q: %q is not a valid type", path, s)
			}
		}
		// TODO(imax): verify uniqueItems constraint.
		return nil
	default:
		return fmt.Errorf("%q must be a string or an array of strings", path)
	}
}

func isValirURI(s string) error {
	// TODO(imax): implement me.
	return nil
}

func validateURI(path string, v interface{}) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("%q must be a string", path)
	}
	if err := isValirURI(s); err != nil {
		return fmt.Errorf("%q must be a valid URI: %s", path, err)
	}
	return nil
}

func validateString(path string, v interface{}) error {
	_, ok := v.(string)
	if !ok {
		return fmt.Errorf("%q must be a string", path)
	}
	return nil
}

func validateNumber(path string, v interface{}) error {
	_, ok := v.(float64)
	if !ok {
		return fmt.Errorf("%q must be a number", path)
	}
	return nil
}

func validateBoolean(path string, v interface{}) error {
	_, ok := v.(bool)
	if !ok {
		return fmt.Errorf("%q must be a boolean", path)
	}
	return nil
}

func validateMultipleOf(path string, v interface{}) error {
	n, ok := v.(float64)
	if !ok {
		return fmt.Errorf("%q must be a number", path)
	}
	if n <= 0 {
		return fmt.Errorf("%q must be > 0", path)
	}
	return nil
}

func validatePositiveInteger(path string, v interface{}) error {
	n, ok := v.(float64)
	if !ok {
		return fmt.Errorf("%q must be a number", path)
	}
	if n <= 0 {
		return fmt.Errorf("%q must be > 0", path)
	}
	// TODO(imax): check that it's really an integer.
	return nil
}

func validatePattern(path string, v interface{}) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("%q must be a string", path)
	}
	_, err := regexp.Compile(s)
	if err != nil {
		return fmt.Errorf("%q must be a valid regexp: %s", path, err)
	}
	return nil
}

func validateBoolOrSchema(path string, v interface{}) error {
	switch v := v.(type) {
	case bool:
		return nil
	default:
		return validateDraft04Schema(path, v)
	}
}

func validateItems(path string, v interface{}) error {
	switch v := v.(type) {
	case []interface{}:
		return validateSchemaArray(path, v)
	default:
		return validateDraft04Schema(path, v)
	}
}

func validateSchemaArray(path string, v interface{}) error {
	a, ok := v.([]interface{})
	if !ok {
		return fmt.Errorf("%q must be an array", path)
	}
	if len(a) < 1 {
		return fmt.Errorf("%q must have at least 1 element", path)
	}
	for i, v := range a {
		err := validateDraft04Schema(path+fmt.Sprintf("/[%d]", i), v)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateStringArray(path string, v interface{}) error {
	a, ok := v.([]interface{})
	if !ok {
		return fmt.Errorf("%q must be an array", path)
	}
	if len(a) < 1 {
		return fmt.Errorf("%q must have at least 1 element", path)
	}
	for _, t := range a {
		_, ok := t.(string)
		if !ok {
			return fmt.Errorf("%q: each element must be a string", path)
		}
	}
	// TODO(imax): verify uniqueItems constraint.
	return nil
}

func validateSchemaCollection(path string, v interface{}) error {
	m, ok := v.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%q must be an object", path)
	}
	for k, v := range m {
		err := validateDraft04Schema(path+"/"+k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateDependencies(path string, v interface{}) error {
	m, ok := v.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%q must be an object", path)
	}
	for k, v := range m {
		switch v := v.(type) {
		case map[string]interface{}:
			err := validateDraft04Schema(path+"/"+k, v)
			if err != nil {
				return err
			}
		case []interface{}:
			return validateStringArray(path+"/"+k, v)
		default:
			return fmt.Errorf("%q must be an array or an object", path+"/"+k)
		}
	}
	return nil
}

func validateEnum(path string, v interface{}) error {
	a, ok := v.([]interface{})
	if !ok {
		return fmt.Errorf("%q must be an array", path)
	}
	if len(a) < 1 {
		return fmt.Errorf("%q must have at least 1 element", path)
	}
	return nil
}
