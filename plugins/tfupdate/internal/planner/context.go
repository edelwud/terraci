package planner

import (
	"context"
	"sort"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/tffile"
)

const skipReasonIgnored = "ignored by config"

type Solver struct {
	ctx      context.Context
	config   *tfupdateengine.UpdateConfig
	registry tfupdateengine.RegistryClient
}

type moduleScanContext struct {
	module    *discovery.Module
	parsed    *parser.ParsedModule
	lockIndex map[string]*parser.LockedProvider
	fileIndex *tffile.Index
}

func New(ctx context.Context, config *tfupdateengine.UpdateConfig, registry tfupdateengine.RegistryClient) *Solver {
	return &Solver{
		ctx:      ctx,
		config:   config,
		registry: registry,
	}
}

func newModuleScanContext(mod *discovery.Module, parsed *parser.ParsedModule) *moduleScanContext {
	return &moduleScanContext{
		module:    mod,
		parsed:    parsed,
		lockIndex: buildLockIndex(parsed.LockedProviders),
	}
}

func (c *moduleScanContext) findModuleFile(callName string) string {
	return c.ensureFileIndex().FindModuleBlockFile(callName)
}

func (c *moduleScanContext) findProviderFile(providerName string) string {
	return c.ensureFileIndex().FindProviderBlockFile(providerName)
}

func (c *moduleScanContext) primaryTerraformFile() string {
	if len(c.parsed.Files) == 0 {
		return ""
	}

	files := make([]string, 0, len(c.parsed.Files))
	for path := range c.parsed.Files {
		files = append(files, path)
	}
	sort.Strings(files)
	return files[0]
}

func (c *moduleScanContext) ensureFileIndex() *tffile.Index {
	if c.fileIndex == nil {
		c.fileIndex = tffile.BuildIndexFromParsedFiles(c.parsed.Files)
	}
	return c.fileIndex
}

func buildLockIndex(locked []*parser.LockedProvider) map[string]*parser.LockedProvider {
	idx := make(map[string]*parser.LockedProvider, len(locked))
	for _, lp := range locked {
		idx[stripRegistryPrefix(lp.Source)] = lp
	}
	return idx
}

func stripRegistryPrefix(source string) string {
	for _, prefix := range []string{"registry.terraform.io/", "registry.opentofu.org/"} {
		if strings.HasPrefix(source, prefix) {
			return source[len(prefix):]
		}
	}
	return source
}
