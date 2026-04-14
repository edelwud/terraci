package sourceaddr

import (
	"fmt"
	"strings"
)

const DefaultProviderRegistryHostname = "registry.terraform.io"

// ProviderAddress is a normalized provider source address.
type ProviderAddress struct {
	Hostname  string
	Namespace string
	Type      string
}

// ModuleAddress is a normalized registry module source address.
type ModuleAddress struct {
	Hostname  string
	Namespace string
	Name      string
	Provider  string
	Subdir    string
}

// ParseRegistryModuleSource parses a registry module source like
// "hashicorp/consul/aws" or "app.terraform.io/hashicorp/consul/aws".
func ParseRegistryModuleSource(source string) (ModuleAddress, error) {
	base, subdir := splitModuleSubdir(source)
	parts := strings.Split(base, "/")
	switch len(parts) {
	case 3:
		return ModuleAddress{
			Hostname:  DefaultProviderRegistryHostname,
			Namespace: parts[0],
			Name:      parts[1],
			Provider:  parts[2],
			Subdir:    subdir,
		}, nil
	case 4:
		return ModuleAddress{
			Hostname:  parts[0],
			Namespace: parts[1],
			Name:      parts[2],
			Provider:  parts[3],
			Subdir:    subdir,
		}, nil
	default:
		return ModuleAddress{}, fmt.Errorf("invalid registry module source %q", source)
	}
}

// WithHostname returns a copy of the module address using the provided hostname when non-empty.
func (a ModuleAddress) WithHostname(host string) ModuleAddress {
	if host != "" {
		a.Hostname = host
	}
	if a.Hostname == "" {
		a.Hostname = DefaultProviderRegistryHostname
	}
	return a
}

func (a ModuleAddress) Source() string {
	base := ""
	if a.Hostname == "" || a.Hostname == DefaultProviderRegistryHostname {
		base = strings.Join([]string{a.Namespace, a.Name, a.Provider}, "/")
	} else {
		base = strings.Join([]string{a.Hostname, a.Namespace, a.Name, a.Provider}, "/")
	}
	if a.Subdir == "" {
		return base
	}
	return base + "//" + a.Subdir
}

// ParseProviderSource parses a provider source like "hashicorp/aws".
func ParseProviderSource(source string) (namespace, typeName string, err error) {
	address, err := ParseProviderAddress(source)
	if err != nil {
		return "", "", err
	}
	return address.Namespace, address.Type, nil
}

// ParseProviderAddress parses a short or fully-qualified provider source.
func ParseProviderAddress(source string) (ProviderAddress, error) {
	parts := strings.Split(source, "/")
	switch len(parts) {
	case 2:
		return ProviderAddress{
			Hostname:  DefaultProviderRegistryHostname,
			Namespace: parts[0],
			Type:      parts[1],
		}, nil
	case 3:
		return ProviderAddress{
			Hostname:  parts[0],
			Namespace: parts[1],
			Type:      parts[2],
		}, nil
	default:
		return ProviderAddress{}, fmt.Errorf(
			"invalid provider source %q: expected namespace/type or hostname/namespace/type",
			source,
		)
	}
}

func (a ProviderAddress) ShortSource() string {
	return a.Namespace + "/" + a.Type
}

// Source returns the fully-qualified source form.
func (a ProviderAddress) Source() string {
	return a.Hostname + "/" + a.Namespace + "/" + a.Type
}

// WithHostname returns a copy of the provider address using the provided hostname when non-empty.
func (a ProviderAddress) WithHostname(host string) ProviderAddress {
	if host != "" {
		a.Hostname = host
	}
	if a.Hostname == "" {
		a.Hostname = DefaultProviderRegistryHostname
	}
	return a
}

// IsRegistryModuleSource returns true if the source looks like a Terraform registry module reference.
func IsRegistryModuleSource(source string) bool {
	if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") {
		return false
	}
	if strings.Contains(source, "://") || strings.Contains(source, "::") {
		return false
	}
	base, _ := splitModuleSubdir(source)
	parts := strings.Split(base, "/")
	return len(parts) == 3 || len(parts) == 4
}

func splitModuleSubdir(source string) (base, subdir string) {
	parts := strings.SplitN(source, "//", 2)
	base = parts[0]
	if len(parts) == 2 {
		subdir = parts[1]
	}
	return base, subdir
}
