package generate

import (
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
	"github.com/edelwud/terraci/plugins/gitlab/internal/domain"
)

type (
	Image            = configpkg.Image
	Config           = configpkg.Config
	CacheConfig      = configpkg.CacheConfig
	JobDefaults      = configpkg.JobDefaults
	JobOverwriteType = configpkg.JobOverwriteType
	JobOverwrite     = configpkg.JobOverwrite
	ArtifactsConfig  = configpkg.ArtifactsConfig
	ArtifactReports  = configpkg.ArtifactReports
	CfgSecret        = configpkg.CfgSecret
	CfgVaultSecret   = configpkg.CfgVaultSecret
	IDToken          = configpkg.IDToken
	VaultEngine      = configpkg.VaultEngine

	Pipeline        = domain.Pipeline
	PipelineOptions = domain.PipelineOptions
	NamedJob        = domain.NamedJob
	DefaultConfig   = domain.DefaultConfig
	ImageConfig     = domain.ImageConfig
	Secret          = domain.Secret
	VaultSecret     = domain.VaultSecret
	Job             = domain.Job
	JobOptions      = domain.JobOptions
	Cache           = domain.Cache
	JobNeed         = domain.JobNeed
	Rule            = domain.Rule
	Artifacts       = domain.Artifacts
	Reports         = domain.Reports
	Workflow        = domain.Workflow
)

var (
	NewPipeline = domain.NewPipeline
	NewJob      = domain.NewJob
)
