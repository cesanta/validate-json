package schema

import (
	json "github.com/cesanta/ucl"
)

func equal(a json.Value, b json.Value) bool {
	switch x := a.(type) {
	case *json.Array:
		b, ok := b.(*json.Array)
		if !ok {
			return false
		}
		if len(x.Value) != len(b.Value) {
			return false
		}
		for i, item := range x.Value {
			if !equal(item, b.Value[i]) {
				return false
			}
		}
		return true
	case *json.Bool:
		b, ok := b.(*json.Bool)
		if !ok {
			return false
		}
		return x.Value == b.Value
	case json.Number:
		b, ok := b.(json.Number)
		if !ok {
			return false
		}
		return x.Value == b.Value // XXX: comparing floating point numbers.
	case *json.Null:
		_, ok := b.(*json.Null)
		if !ok {
			return false
		}
		return true
	case *json.Object:
		b, ok := b.(*json.Object)
		if !ok {
			return false
		}
		if len(x.Value) != len(b.Value) {
			return false
		}
		for i, item := range x.Value {
			if !equal(item, b.Find(i.Value)) {
				return false
			}
		}
		return true
	case *json.String:
		b, ok := b.(*json.String)
		if !ok {
			return false
		}
		return x.Value == b.Value
	default:
		return false
	}
}
