package planner

import (
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/versionkit"
)

type moduleChoice struct {
	version      versionkit.Version
	providerDeps []registrymeta.ModuleProviderDep
}

type modulePlan struct {
	call          *parser.ModuleCall
	update        domain.ModuleVersionUpdate
	address       sourceaddr.ModuleAddress
	current       versionkit.Version
	latest        versionkit.Version
	choices       []moduleChoice
	updateChoices int
}

type providerPlan struct {
	required        *parser.RequiredProvider
	update          domain.ProviderVersionUpdate
	address         sourceaddr.ProviderAddress
	current         versionkit.Version
	hasCurrent      bool
	latest          versionkit.Version
	versions        []versionkit.Version
	baseConstraints []string
	locked          *parser.LockedProvider
}

type selectedModule struct {
	plan   *modulePlan
	choice moduleChoice
}
