package registry

import (
	"context"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

// Client queries Terraform-compatible registries for module/provider metadata.
type Client interface {
	ModuleVersions(ctx context.Context, address sourceaddr.ModuleAddress) ([]string, error)
	ModuleProviderDeps(ctx context.Context, address sourceaddr.ModuleAddress, version string) ([]registrymeta.ModuleProviderDep, error)
	ProviderVersions(ctx context.Context, address sourceaddr.ProviderAddress) ([]string, error)
	ProviderPlatforms(ctx context.Context, address sourceaddr.ProviderAddress, version string) ([]string, error)
	ProviderPackage(ctx context.Context, address sourceaddr.ProviderAddress, version, platform string) (*registrymeta.ProviderPackage, error)
}
