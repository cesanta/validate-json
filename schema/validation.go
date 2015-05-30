package schema

import (
	"fmt"
	"regexp"
	"strings"

	json "github.com/cesanta/ucl"
)

type Validator struct {
	schema json.Value
	loader *Loader
}

func NewValidator(schema json.Value, loader *Loader) *Validator {
	return &Validator{schema: schema, loader: loader}
}

func (v *Validator) Validate(val json.Value) error {
	return v.validateAgainstSchema("#", val, "#", v.schema)
}

func (v *Validator) getSchemaByRef(uri string) (json.Value, json.Value, error) {
	if strings.HasPrefix(uri, "#") {
		r, err := resolveRef(v.schema, uri[1:])
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve ref %q: %s", uri, err)
		}
		return r, v.schema, nil
	}
	if v.loader == nil {
		return nil, nil, fmt.Errorf("need to have a loader (passed to NewValidator) to resolve remote refs")
	}
	if i := strings.Index(uri, "#"); i > 0 {
		uri, ref := uri[:i], uri[i+1:]
		s, err := v.loader.Get(uri)
		if err != nil {
			return nil, nil, err
		}
		r, err := resolveRef(s, ref)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve ref %q within %q: %s", ref, uri, err)
		}
		return r, s, nil
	}
	r, err := v.loader.Get(uri)
	return r, r, err
}

func isOfType(val json.Value, t string) bool {
	switch val.(type) {
	case *json.Array:
		return t == "array"
	case *json.Bool:
		return t == "boolean"
	case json.Number:
		// TODO(imax): add proper support for integers.
		return t == "number" || t == "integer"
	case *json.Null:
		return t == "null"
	case *json.Object:
		return t == "object"
	case *json.String:
		return t == "string"
	}
	return false
}

func (v *Validator) validateAgainstSchema(path string, val json.Value, schemaPath string, schema_ json.Value) error {
	schema, ok := schema_.(*json.Object)
	if !ok {
		return fmt.Errorf("%q: schema must be an object", schemaPath)
	}

	ref, found := schema.Lookup("$ref")
	if found {
		sref, ok := ref.(*json.String)
		if !ok {
			return fmt.Errorf("%q: must be a string", schemaPath+"/$ref")
		}
		s, whole, err := v.getSchemaByRef(sref.Value)
		if err != nil {
			return err
		}
		return NewValidator(whole, v.loader).validateAgainstSchema(path, val, sref.Value, s)
	}

	t, found := schema.Lookup("type")
	if found {
		switch t := t.(type) {
		case *json.String:
			if !isOfType(val, t.Value) {
				return fmt.Errorf("%q: must be of type %q", path, t.Value)
			}
		case *json.Array:
			match := false
			for i, v := range t.Value {
				t, ok := v.(*json.String)
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

	i, found := schema.Lookup("allOf")
	if found {
		a, ok := i.(*json.Array)
		if !ok {
			return fmt.Errorf("%q must be an array", schemaPath+"/allOf")
		}
		if len(a.Value) < 1 {
			return fmt.Errorf("%q must have at least 1 element", schemaPath+"/allOf")
		}
		for i, s := range a.Value {
			err := ValidateDraft04Schema(s)
			if err != nil {
				return fmt.Errorf("%q must be a valid schema: %s", fmt.Sprintf("%s/allOf/[%d]", schemaPath, i), err)
			}
			err = v.validateAgainstSchema(path, val, fmt.Sprintf("%s/allOf/[%d]", schemaPath, i), s)
			if err != nil {
				return err
			}
		}
	}

	i, found = schema.Lookup("anyOf")
	if found {
		a, ok := i.(*json.Array)
		if !ok {
			return fmt.Errorf("%q must be an array", schemaPath+"/anyOf")
		}
		if len(a.Value) < 1 {
			return fmt.Errorf("%q must have at least 1 element", schemaPath+"/anyOf")
		}
		errs := []string{}
		for i, s := range a.Value {
			err := ValidateDraft04Schema(s)
			if err != nil {
				return fmt.Errorf("%q must be a valid schema: %s", fmt.Sprintf("%s/anyOf/[%d]", schemaPath, i), err)
			}
			err = v.validateAgainstSchema(path, val, fmt.Sprintf("%s/anyOf/[%d]", schemaPath, i), s)
			if err != nil {
				errs = append(errs, err.Error())
			}
		}
		if len(errs) == len(a.Value) {
			return fmt.Errorf("%q must be valid against at least one of the schemas in %q, but it is not:\n%s", path, schemaPath+"/anyOf", strings.Join(errs, "\n"))
		}
	}

	i, found = schema.Lookup("oneOf")
	if found {
		a, ok := i.(*json.Array)
		if !ok {
			return fmt.Errorf("%q must be an array", schemaPath+"/oneOf")
		}
		if len(a.Value) < 1 {
			return fmt.Errorf("%q must have at least 1 element", schemaPath+"/oneOf")
		}
		errs := make([]string, len(a.Value))
		valid := []int{}
		for i, s := range a.Value {
			err := ValidateDraft04Schema(s)
			if err != nil {
				return fmt.Errorf("%q must be a valid schema: %s", fmt.Sprintf("%s/oneOf/[%d]", schemaPath, i), err)
			}
			err = v.validateAgainstSchema(path, val, fmt.Sprintf("%s/oneOf/[%d]", schemaPath, i), s)
			if err != nil {
				errs[i] = err.Error()
			} else {
				valid = append(valid, i)
			}
		}
		if len(valid) == 0 {
			return fmt.Errorf("%q must be valid against against one of the schemas in %q, but it is not:\n%s", path, schemaPath+"/oneOf", strings.Join(errs, "\n"))
		}
		if len(valid) > 1 {
			ss := []string{}
			for _, vv := range valid {
				ss = append(ss, fmt.Sprintf("%s/oneOf/[%d]", schemaPath, vv))
			}
			return fmt.Errorf("%q must be valid against exactly one of the schemas in %q, but it is valid against %s", path, schemaPath+"/oneOf", strings.Join(ss, " and "))
		}
	}

	if not, found := schema.Lookup("not"); found {
		err := ValidateDraft04Schema(not)
		if err != nil {
			return fmt.Errorf("%q must be a valid schema: %s", schemaPath+"/not", err)
		}
		err = v.validateAgainstSchema(path, val, schemaPath+"/not", not)
		if err == nil {
			return fmt.Errorf("%q must not be valid against %q, but it is", path, schemaPath+"/not")
		}
	}

	i, found = schema.Lookup("enum")
	if found {
		enum, ok := i.(*json.Array)
		if !ok {
			return fmt.Errorf("%q must be an array", schemaPath+"/enum")
		}
		if len(enum.Value) < 1 {
			return fmt.Errorf("%q must have at least one element", schemaPath+"/enum")
		}
		valid := false
		for _, item := range enum.Value {
			if equal(item, val) {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("%q must be one of %q", path, enum)
		}
	}

	switch val := val.(type) {
	case *json.String:
		v, found := schema.Lookup("minLength")
		if found {
			minLen, ok := v.(json.Number)
			if !ok {
				return fmt.Errorf("%q must be a number", schemaPath+"/minLength")
			}
			if len(val.Value) < int(minLen.Value) {
				return fmt.Errorf("%q must have at least %d characters", path, int(minLen.Value))
			}
		}
		v, found = schema.Lookup("maxLength")
		if found {
			maxLen, ok := v.(json.Number)
			if !ok {
				return fmt.Errorf("%q must be a number", schemaPath+"/maxLength")
			}
			if len(val.Value) > int(maxLen.Value) {
				return fmt.Errorf("%q must have at most %d characters", path, int(maxLen.Value))
			}
		}
		v, found = schema.Lookup("pattern")
		if found {
			pattern, ok := v.(*json.String)
			if !ok {
				return fmt.Errorf("%q must be a string", schemaPath+"/pattern")
			}
			re, err := regexp.Compile(pattern.Value)
			if err != nil {
				return fmt.Errorf("%q must be a valid regexp: %s", schemaPath+"/pattern", err)
			}
			if !re.MatchString(val.Value) {
				return fmt.Errorf("%q must match regexp %q", path, pattern.Value)
			}
		}
	case *json.Array:
		i, found := schema.Lookup("items")
		// If "items" is not present it is assumed to be an empty object, which
		// means that any item is valid and "additionalItems" is ignored.
		if found {
			switch items := i.(type) {
			case *json.Object:
				err := ValidateDraft04Schema(items)
				if err != nil {
					return fmt.Errorf("%q must be a valid schema: %s", schemaPath+"/items")
				}
				for i, item := range val.Value {
					err := v.validateAgainstSchema(fmt.Sprintf("%s/[%d]", path, i), item, schemaPath+"/items", items)
					if err != nil {
						return err
					}
				}
			case *json.Array:
				for i, item := range items.Value {
					err := ValidateDraft04Schema(item)
					if err != nil {
						return fmt.Errorf("%q must be a valid schema: %s", fmt.Sprintf("%s/[%d]", schemaPath, i), err)
					}
				}
				for i := 0; i < len(items.Value) && i < len(val.Value); i++ {
					err := v.validateAgainstSchema(fmt.Sprintf("%s/[%d]", path, i), val.Value[i],
						fmt.Sprintf("%s/[%d]", schemaPath, i), items.Value[i])
					if err != nil {
						return err
					}
				}
				ai, found := schema.Lookup("additionalItems")
				if found {
					switch ai := ai.(type) {
					case *json.Bool:
						if ai.Value == false && len(items.Value) < len(val.Value) {
							return fmt.Errorf("%q must have not more than %d items", path, len(items.Value))
						}
					case *json.Object:
						err := ValidateDraft04Schema(ai)
						if err != nil {
							return fmt.Errorf("%q must be a valid schema: %s", schemaPath+"/additionalItems", err)
						}
						for i := len(items.Value); i < len(val.Value); i++ {
							err := v.validateAgainstSchema(fmt.Sprintf("%s/[%d]", path, i), val.Value[i],
								schemaPath+"/additionalItems", ai)
							if err != nil {
								return err
							}
						}
					default:
						return fmt.Errorf("%q must be an array or a boolean", schemaPath+"/additionalItems")
					}
				}
			default:
				return fmt.Errorf("%q must be an array or an object", schemaPath+"/items")
			}
		}
		i, found = schema.Lookup("maxItems")
		if found {
			maxItems, ok := i.(json.Number)
			if !ok {
				return fmt.Errorf("%q must be a number", schemaPath+"/maxItems")
			}
			if len(val.Value) > int(maxItems.Value) {
				return fmt.Errorf("%q must have at most %d items", path, int(maxItems.Value))
			}
		}
		i, found = schema.Lookup("minItems")
		if found {
			minItems, ok := i.(json.Number)
			if !ok {
				return fmt.Errorf("%q must be a number", schemaPath+"/minItems")
			}
			if len(val.Value) < int(minItems.Value) {
				return fmt.Errorf("%q must have at least %d items", path, int(minItems.Value))
			}
		}
		i, found = schema.Lookup("uniqueItems")
		if found {
			u, ok := i.(*json.Bool)
			if !ok {
				return fmt.Errorf("%q must be a boolean", schemaPath+"/uniqueItems")
			}
			if u.Value {
				for i := range val.Value {
					for j := i + 1; j < len(val.Value); j++ {
						if equal(val.Value[i], val.Value[j]) {
							return fmt.Errorf("%q: all items must be unique, but item %d is equal to item %d", path, i, j)
						}
					}
				}
			}
		}
	case *json.Object:
		i, found := schema.Lookup("maxProperties")
		if found {
			maxProps, ok := i.(json.Number)
			if !ok {
				return fmt.Errorf("%q must be a number", schemaPath+"/maxProperties")
			}
			if len(val.Value) > int(maxProps.Value) {
				return fmt.Errorf("%q must have at most %d properties", path, int(maxProps.Value))
			}
		}
		i, found = schema.Lookup("minProperties")
		if found {
			minProps, ok := i.(json.Number)
			if !ok {
				return fmt.Errorf("%q must be a number", schemaPath+"/minProperties")
			}
			if len(val.Value) < int(minProps.Value) {
				return fmt.Errorf("%q must have at least %d properties", path, int(minProps.Value))
			}
		}
		i, found = schema.Lookup("required")
		if found {
			req, ok := i.(*json.Array)
			if !ok || len(req.Value) < 1 {
				return fmt.Errorf("%q must be an array with at least one element", schemaPath+"/required")
			}
			for i, p := range req.Value {
				prop, ok := p.(*json.String)
				if !ok {
					return fmt.Errorf("%q must be a string", fmt.Sprintf("%s/required/[%d]", schemaPath, i))
				}
				_, found := val.Lookup(prop.Value)
				if !found {
					return fmt.Errorf("%q must have property %q", path, prop.Value)
				}
			}
		}
		type schemaWithPath struct {
			schema json.Value
			path   string
		}
		validateWith := map[string][]schemaWithPath{}
		for k := range val.Value {
			validateWith[k.Value] = nil
		}
		i, found = schema.Lookup("properties")
		if found {
			props, ok := i.(*json.Object)
			if !ok {
				return fmt.Errorf("%q must be an object", schemaPath+"/properties")
			}
			for k, v := range props.Value {
				err := ValidateDraft04Schema(v)
				if err != nil {
					return fmt.Errorf("%q must be a valid schema: %s", schemaPath+"/properties/"+k.Value)
				}
				_, found = validateWith[k.Value]
				if found {
					validateWith[k.Value] = []schemaWithPath{{v, schemaPath + "/properties/" + k.Value}}
				}
			}
		}
		i, found = schema.Lookup("patternProperties")
		if found {
			pprops, ok := i.(*json.Object)
			if !ok {
				return fmt.Errorf("%q must be an object", schemaPath+"/patternProperties")
			}
			for k, v := range pprops.Value {
				err := ValidateDraft04Schema(v)
				if err != nil {
					return fmt.Errorf("%q must be a valid schema: %s", schemaPath+"/patternProperties/"+k.Value)
				}
				re, err := regexp.Compile(k.Value)
				if err != nil {
					return fmt.Errorf("%q: %q is not a valid regexp: %s", schemaPath+"/patternProperties", k.Value, err)
				}
				for p := range validateWith {
					if re.MatchString(p) {
						validateWith[p] = append(validateWith[p], schemaWithPath{v, schemaPath + "/patternProperties/" + k.Value})
					}
				}
			}
		}
		i, found = schema.Lookup("additionalProperties")
		if found {
			switch ap := i.(type) {
			case *json.Object:
				err := ValidateDraft04Schema(ap)
				if err != nil {
					return fmt.Errorf("%q must be a valid schema: %s", schemaPath+"/additionalProperties")
				}
				for k, v := range validateWith {
					if len(v) == 0 {
						validateWith[k] = []schemaWithPath{{ap, schemaPath + "/additionalProperties"}}
					}
				}
			case *json.Bool:
				if ap.Value == false {
					for k, v := range validateWith {
						if len(v) == 0 {
							return fmt.Errorf("%q is not in %q, is not matched by anything in %q and %q is set to false",
								path+"/"+k, schemaPath+"/properties", schemaPath+"/patternProperties", schemaPath+"/additionalProperties")
						}
					}
				}
			default:
				return fmt.Errorf("%q must be an object or a boolean", schemaPath+"/additionalProperties")
			}
		}
		for prop, schemas := range validateWith {
			for _, s := range schemas {
				err := v.validateAgainstSchema(path+"/"+prop, val.Find(prop), s.path, s.schema)
				if err != nil {
					return err
				}
			}
		}
		i, found = schema.Lookup("dependencies")
		if found {
			deps, ok := i.(*json.Object)
			if !ok {
				return fmt.Errorf("%q must be an object", schemaPath+"/dependencies")
			}
			for prop, propDeps := range deps.Value {
				if _, found = val.Lookup(prop.Value); !found {
					continue
				}
				switch deps := propDeps.(type) {
				case *json.Array:
					if len(deps.Value) < 1 {
						return fmt.Errorf("%q must have at least one element", schemaPath+"/dependencies/"+prop.Value)
					}
					for i, item := range deps.Value {
						req, ok := item.(*json.String)
						if !ok {
							return fmt.Errorf("%q must be a string", fmt.Sprintf("%s/dependencies/%s/[%d]", schemaPath, prop.Value, i))
						}
						if _, found = val.Lookup(req.Value); !found {
							return fmt.Errorf("%q: %q requires %q to be also present", path, prop.Value, req.Value)
						}
					}
				case *json.Object:
					err := ValidateDraft04Schema(deps)
					if err != nil {
						return fmt.Errorf("%q must be a valid schema: %s", schemaPath+"/dependencies/"+prop.Value, err)
					}
					err = v.validateAgainstSchema(path, val, schemaPath+"/dependencies/"+prop.Value, deps)
					if err != nil {
						return err
					}
				default:
					return fmt.Errorf("%q must be an array or an object", schemaPath+"/dependencies/"+prop.Value)
				}
			}
		}
	case json.Number:
		i, found := schema.Lookup("multipleOf")
		if found {
			div, ok := i.(json.Number)
			if !ok || div.Value <= 0 {
				return fmt.Errorf("%q must be a number and greater than 0", schemaPath+"/multipleOf")
			}
			// TODO(imax): find a nice way to handle this for floating point numbers.
			if val.Value/div.Value != float64(int(val.Value/div.Value)) {
				return fmt.Errorf("%q must be a multiple of %g", path, div.Value)
			}
		}
		i, found = schema.Lookup("maximum")
		if found {
			max, ok := i.(json.Number)
			if !ok {
				return fmt.Errorf("%q must be a number", schemaPath+"/maximum")
			}
			exclude := false
			i, found := schema.Lookup("exclusiveMaximum")
			if found {
				e, ok := i.(*json.Bool)
				if !ok {
					return fmt.Errorf("%q must be a boolean", schemaPath+"/exclusiveMaximum")
				}
				exclude = e.Value
			}
			if exclude {
				if val.Value >= max.Value {
					return fmt.Errorf("%q must be less than %g", path, max.Value)
				}
			} else {
				if val.Value > max.Value {
					return fmt.Errorf("%q must be less then or equal to %g", path, max.Value)
				}
			}
		}
		i, found = schema.Lookup("minimum")
		if found {
			min, ok := i.(json.Number)
			if !ok {
				return fmt.Errorf("%q must be a number", schemaPath+"/minimum")
			}
			exclude := false
			i, found := schema.Lookup("exclusiveMinimum")
			if found {
				e, ok := i.(*json.Bool)
				if !ok {
					return fmt.Errorf("%q must be a boolean", schemaPath+"/exclusiveMinimum")
				}
				exclude = e.Value
			}
			if exclude {
				if val.Value <= min.Value {
					return fmt.Errorf("%q must be greater than %g", path, min.Value)
				}
			} else {
				if val.Value < min.Value {
					return fmt.Errorf("%q must be greater then or equal to %g", path, min.Value)
				}
			}
		}
	}

	return nil
}
