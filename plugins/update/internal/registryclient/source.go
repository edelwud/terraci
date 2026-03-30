package registryclient

import (
	"fmt"
	"strings"
)

// ParseModuleSource parses a registry module source like "hashicorp/consul/aws".
func ParseModuleSource(source string) (namespace, name, provider string, err error) {
	parts := strings.Split(source, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid registry module source %q: expected namespace/name/provider", source)
	}
	return parts[0], parts[1], parts[2], nil
}

// ParseProviderSource parses a provider source like "hashicorp/aws".
func ParseProviderSource(source string) (namespace, typeName string, err error) {
	parts := strings.Split(source, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid provider source %q: expected namespace/type", source)
	}
	return parts[0], parts[1], nil
}

// IsRegistrySource returns true if the source looks like a Terraform registry reference.
func IsRegistrySource(source string) bool {
	if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") {
		return false
	}
	if strings.Contains(source, "://") || strings.Contains(source, "::") {
		return false
	}
	return len(strings.Split(source, "/")) == 3
}
