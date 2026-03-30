package checker

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/plugins/update/internal/tffile"
)

type moduleScanContext struct {
	module    *discovery.Module
	parsed    *parser.ParsedModule
	lockIndex map[string]*parser.LockedProvider
	fileIndex *tffile.Index
}

func (s *checkSession) newModuleScanContext(
	mod *discovery.Module,
	parsed *parser.ParsedModule,
) *moduleScanContext {
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

func (c *moduleScanContext) ensureFileIndex() *tffile.Index {
	if c.fileIndex == nil {
		c.fileIndex = tffile.BuildIndexFromParsedFiles(c.parsed.Files)
	}

	return c.fileIndex
}
