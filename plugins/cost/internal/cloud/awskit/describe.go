package awskit

import (
	"fmt"
	"strconv"
)

// DescribeBuilder helps handlers build human-readable description maps
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

// Bool adds a field when the value is true.
func (d DescribeBuilder) Bool(key string, value bool) DescribeBuilder {
	if value {
		d[key] = "true"
	}
	return d
}

// Int adds a field when the value is non-zero.
func (d DescribeBuilder) Int(key string, value int) DescribeBuilder {
	if value != 0 {
		d[key] = strconv.Itoa(value)
	}
	return d
}

// Float adds a field when the value is greater than zero.
func (d DescribeBuilder) Float(key string, value float64, format string) DescribeBuilder {
	if value > 0 {
		d[key] = fmt.Sprintf(format, value)
	}
	return d
}

// Map returns the underlying description map.
func (d DescribeBuilder) Map() map[string]string {
	return map[string]string(d)
}
