package tfupdateengine

import (
	"context"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
)

// RegistryClient queries the Terraform Registry for version information.
type RegistryClient interface {
	ModuleVersions(ctx context.Context, hostname, namespace, name, provider string) ([]string, error)
	ModuleProviderDeps(ctx context.Context, hostname, namespace, name, provider, version string) ([]registrymeta.ModuleProviderDep, error)
	ProviderVersions(ctx context.Context, hostname, namespace, typeName string) ([]string, error)
	ProviderPlatforms(ctx context.Context, hostname, namespace, typeName, version string) ([]string, error)
	ProviderPackage(ctx context.Context, hostname, namespace, typeName, version, platform string) (*registrymeta.ProviderPackage, error)
}
