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
	fileIndex, err := tffile.BuildIndex(mod.Path)
	if err != nil {
		fileIndex = nil
	}

	return &moduleScanContext{
		module:    mod,
		parsed:    parsed,
		lockIndex: buildLockIndex(parsed.LockedProviders),
		fileIndex: fileIndex,
	}
}
