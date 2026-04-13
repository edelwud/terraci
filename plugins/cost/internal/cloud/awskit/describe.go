package awskit

import (
	"fmt"
	"strconv"
)

// DescribeBuilder helps resource definitions build human-readable description maps
// without repeating nil/zero-value guards inline.
type DescribeBuilder map[string]string

// NewDescribeBuilder creates an empty description builder.
func NewDescribeBuilder() DescribeBuilder {
	return make(DescribeBuilder)
}

// String adds a field when the value is non-empty.
func (d DescribeBuilder) String(key, value string) DescribeBuilder {
	if value != "" {
		d[key] = value
	}
	return d
}

// StringIf adds a string field when condition is true and the value is non-empty.
func (d DescribeBuilder) StringIf(condition bool, key, value string) DescribeBuilder {
	if !condition {
		return d
	}
	return d.String(key, value)
}

// Bool adds a field when the value is true.
func (d DescribeBuilder) Bool(key string, value bool) DescribeBuilder {
	if value {
		d[key] = "true"
	}
	return d
}

// BoolIf adds a boolean field when condition and value are both true.
func (d DescribeBuilder) BoolIf(condition bool, key string, value bool) DescribeBuilder {
	if !condition {
		return d
	}
	return d.Bool(key, value)
}

// Int adds a field when the value is non-zero.
func (d DescribeBuilder) Int(key string, value int) DescribeBuilder {
	if value != 0 {
		d[key] = strconv.Itoa(value)
	}
	return d
}

// IntIf adds an integer field when condition is true and the value is non-zero.
func (d DescribeBuilder) IntIf(condition bool, key string, value int) DescribeBuilder {
	if !condition {
		return d
	}
	return d.Int(key, value)
}

// Float adds a field when the value is greater than zero.
func (d DescribeBuilder) Float(key string, value float64, format string) DescribeBuilder {
	if value > 0 {
		d[key] = fmt.Sprintf(format, value)
	}
	return d
}

// FloatIf adds a float field when condition is true and the value is greater than zero.
func (d DescribeBuilder) FloatIf(condition bool, key string, value float64, format string) DescribeBuilder {
	if !condition {
		return d
	}
	return d.Float(key, value, format)
}

// Map returns the underlying description map.
func (d DescribeBuilder) Map() map[string]string {
	return d
}
