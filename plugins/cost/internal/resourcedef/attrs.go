package resourcedef

import (
	"fmt"
	"strconv"
)

// RawAttrs is the only value allowed to carry raw Terraform attribute maps past
// the plan adapter boundary. It defensively copies input and output maps.
type RawAttrs struct {
	values map[string]any
}

// RawAttr is one key/value entry for constructing RawAttrs without exposing maps.
type RawAttr struct {
	Key   string
	Value any
}

// NewRawAttr creates one raw attribute entry.
func NewRawAttr(key string, value any) RawAttr {
	return RawAttr{Key: key, Value: value}
}

// NewRawAttrs creates a defensive raw attribute value.
func NewRawAttrs(values map[string]any) RawAttrs {
	return RawAttrs{values: cloneMap(values)}
}

// NewRawAttrsFromPairs creates raw attributes from explicit key/value pairs.
func NewRawAttrsFromPairs(pairs ...RawAttr) RawAttrs {
	if len(pairs) == 0 {
		return EmptyRawAttrs()
	}
	values := make(map[string]any, len(pairs))
	for _, pair := range pairs {
		if pair.Key == "" {
			continue
		}
		values[pair.Key] = pair.Value
	}
	return RawAttrs{values: cloneMap(values)}
}

// EmptyRawAttrs returns an empty raw attribute value.
func EmptyRawAttrs() RawAttrs {
	return RawAttrs{}
}

// Map returns a defensive map copy for tests and boundary adapters.
func (a RawAttrs) Map() map[string]any {
	return cloneMap(a.values)
}

// IsZero reports whether no attributes are present.
func (a RawAttrs) IsZero() bool {
	return len(a.values) == 0
}

// String returns a string attribute or "" when absent or typed differently.
func (a RawAttrs) String(key string) string {
	if a.values == nil {
		return ""
	}
	if value, ok := a.values[key].(string); ok {
		return value
	}
	return ""
}

// Float returns a numeric attribute as float64 or 0 when absent or typed differently.
func (a RawAttrs) Float(key string) float64 {
	if a.values == nil {
		return 0
	}
	switch value := a.values[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case int32:
		return float64(value)
	case jsonNumber:
		f, err := value.Float64()
		if err == nil {
			return f
		}
	case string:
		f, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return f
		}
	}
	return 0
}

// Int returns a numeric attribute as int or 0 when absent or typed differently.
func (a RawAttrs) Int(key string) int {
	if a.values == nil {
		return 0
	}
	switch value := a.values[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case int32:
		return int(value)
	case float64:
		return int(value)
	case float32:
		return int(value)
	case jsonNumber:
		i, err := value.Int64()
		if err == nil {
			return int(i)
		}
		f, err := value.Float64()
		if err == nil {
			return int(f)
		}
	case string:
		i, err := strconv.Atoi(value)
		if err == nil {
			return i
		}
	}
	return 0
}

// Bool returns a bool attribute or false when absent or typed differently.
func (a RawAttrs) Bool(key string) bool {
	if a.values == nil {
		return false
	}
	switch value := a.values[key].(type) {
	case bool:
		return value
	case string:
		b, err := strconv.ParseBool(value)
		return err == nil && b
	case int:
		return value != 0
	case int64:
		return value != 0
	case int32:
		return value != 0
	case float64:
		return value != 0
	case float32:
		return value != 0
	case jsonNumber:
		i, err := value.Int64()
		if err == nil {
			return i != 0
		}
		f, err := value.Float64()
		if err == nil {
			return f != 0
		}
	}
	return false
}

// StringSlice returns a string slice from []string or []any string values.
func (a RawAttrs) StringSlice(key string) []string {
	if a.values == nil {
		return nil
	}
	switch value := a.values[key].(type) {
	case []string:
		return append([]string(nil), value...)
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// FirstObject returns the first object attribute from list-style Terraform blocks.
func (a RawAttrs) FirstObject(key string) RawAttrs {
	if a.values == nil {
		return EmptyRawAttrs()
	}
	switch value := a.values[key].(type) {
	case []map[string]any:
		if len(value) == 0 {
			return EmptyRawAttrs()
		}
		return NewRawAttrs(value[0])
	case []any:
		if len(value) == 0 {
			return EmptyRawAttrs()
		}
		if m, ok := value[0].(map[string]any); ok {
			return NewRawAttrs(m)
		}
	case map[string]any:
		return NewRawAttrs(value)
	}
	return EmptyRawAttrs()
}

// Attributes is an opaque parsed attribute payload owned by a resource definition.
type Attributes struct {
	value any
}

// NewAttributes wraps a typed parsed attribute payload.
func NewAttributes(value any) Attributes {
	return Attributes{value: value}
}

// AttributesAs returns the parsed attributes as the requested resource-specific type.
func AttributesAs[A any](attrs Attributes) (A, error) {
	value, ok := attrs.value.(A)
	if ok {
		return value, nil
	}
	var zero A
	return zero, fmt.Errorf("resource attributes: unexpected parsed type %T", attrs.value)
}

type jsonNumber interface {
	Float64() (float64, error)
	Int64() (int64, error)
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = cloneValue(value)
	}
	return dst
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []map[string]any:
		out := make([]map[string]any, len(typed))
		for i := range typed {
			out[i] = cloneMap(typed[i])
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = cloneValue(typed[i])
		}
		return out
	case []string:
		return append([]string(nil), typed...)
	default:
		return value
	}
}
