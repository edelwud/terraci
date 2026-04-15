package tfupdateengine

import (
	"context"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/lockfile"
	tfregistry "github.com/edelwud/terraci/plugins/tfupdate/internal/registry"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

type providerMetadataAdapter struct {
	registry tfregistry.Client
}

func newProviderMetadataSource(registry tfregistry.Client) lockfile.ProviderMetadataSource {
	if registry == nil {
		return nil
	}
	return providerMetadataAdapter{registry: registry}
}

func (a providerMetadataAdapter) ProviderPlatforms(ctx context.Context, address lockfile.ProviderAddress, version string) ([]string, error) {
	return a.registry.ProviderPlatforms(ctx, sourceProviderAddress(address), version)
}

func (a providerMetadataAdapter) ProviderPackage(ctx context.Context, address lockfile.ProviderAddress, version, platform string) (*registrymeta.ProviderPackage, error) {
	return a.registry.ProviderPackage(ctx, sourceProviderAddress(address), version, platform)
}

func sourceProviderAddress(address lockfile.ProviderAddress) sourceaddr.ProviderAddress {
	return sourceaddr.ProviderAddress{
		Hostname:  address.Hostname,
		Namespace: address.Namespace,
		Type:      address.Type,
	}
}
