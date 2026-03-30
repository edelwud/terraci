package updateengine

import "context"

// RegistryClient queries the Terraform Registry for version information.
type RegistryClient interface {
	ModuleVersions(ctx context.Context, namespace, name, provider string) ([]string, error)
	ProviderVersions(ctx context.Context, namespace, typeName string) ([]string, error)
}
