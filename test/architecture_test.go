package test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const moduleImportPath = "github.com/edelwud/terraci"

func TestArchitecture_ImportBoundaries(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "pkg", "plugins") {
		imports := fileImports(t, filepath.Join(root, rel))
		for _, imp := range imports {
			if isProductionFile(rel) && strings.HasPrefix(rel, "pkg/") {
				if strings.HasPrefix(imp, moduleImportPath+"/plugins/") {
					violations = append(violations, rel+" imports plugin implementation package "+imp)
				}
				if !strings.HasPrefix(rel, "pkg/plugin/") && importsPluginSDK(imp) {
					violations = append(violations, rel+" imports plugin SDK package "+imp)
				}
			}

			if strings.HasPrefix(rel, "plugins/") {
				if reason, ok := siblingPluginImportViolation(rel, imp); ok {
					violations = append(violations, reason)
				}
			}
		}
	}

	if len(violations) > 0 {
		t.Fatalf("architecture import boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}

func TestArchitecture_PipelineContributionBuilders(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "pkg", "plugins", "test", "examples") {
		if strings.HasPrefix(rel, "pkg/pipeline/") {
			continue
		}
		file := parseGoFile(t, filepath.Join(root, rel), parser.ParseComments)
		aliases := importAliases(file, moduleImportPath+"/pkg/pipeline")
		if len(aliases) == 0 {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			lit, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			selector, ok := lit.Type.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if !ok || !aliases[ident.Name] {
				return true
			}
			switch selector.Sel.Name {
			case "Contribution", "ContributedJob":
				violations = append(violations, rel+" manually constructs pipeline."+selector.Sel.Name+"; use pipeline.NewPluginCommandJob/NewContributedJob and pipeline.NewContribution")
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Fatalf("pipeline contribution constructor violations:\n%s", strings.Join(violations, "\n"))
	}
}

func TestArchitecture_ConfigSnapshotAndDocs(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "pkg", "plugins", "cmd", "test", "examples") {
		file := parseGoFile(t, filepath.Join(root, rel), 0)
		ast.Inspect(file, func(node ast.Node) bool {
			if isProductionFile(rel) {
				if call, ok := node.(*ast.CallExpr); ok {
					if selector, ok := callSelector(call); ok && selector.Sel.Name == "MutableCopy" {
						violations = append(violations, rel+" calls MutableCopy in production code; consume config.Snapshot accessors or isolate compatibility adapters")
					}
				}
			}
			assign, ok := node.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for _, lhs := range assign.Lhs {
				if containsConfigCall(lhs) {
					violations = append(violations, rel+" assigns through Config(); build a new config.Config and create a new AppContext")
				}
			}
			return true
		})
	}

	stalePatterns := []string{
		"FlagOverridable",
		"shared *config.Config",
		"ctx.Config() (`*config.Config`",
		"BuildConfig(pattern",
		"BuildInitConfig(state *initwiz.StateMap) *initwiz.InitContribution",
		"BuildInitConfig(state *initwiz.StateMap) *InitContribution",
		"&initwiz.InitContribution",
		"Config: map[string]any",
		"map[string]map[string]any",
		"return &pipeline.Contribution",
		"Jobs: []pipeline.ContributedJob",
		"PipelineContribution(ctx *plugin.AppContext) *pipeline.Contribution",
		"PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution",
		"PipelineContributionEnabled(_ *plugin.AppContext) bool",
		"CollectContributions(ctx *plugin.AppContext) []*pipeline.Contribution",
		"resolver.CollectContributions",
		"registry.ByCapabilityFrom",
		"ctx.Resolver()",
		"app.Plugins",
		"app.Config",
		`Annotations["skipConfig"]`,
		"workflow.Run()",
		"workflow.ResolveTargets()",
		"workflow.OptionsFromConfig()",
		"pipeline.NewIR",
		"pipeline.NewCommandJob",
		"pipeline.JobName",
		"discovery.NewScanner(appCtx.WorkDir()",
		"Use `discovery.Scanner`",
		"legacy pointer-shaped",
		"pipeline.BuildOptions",
		"localexec/internal/planner",
		"Evaluate(ctx context.Context, input any",
		"Metadata  map[string]any",
		"policyinput.Envelope{",
	}
	for _, rel := range textFiles(t, root, "AGENTS.md", "docs", "examples", "pkg/plugin/doc.go") {
		if allowUnder(rel, "docs/.vitepress/dist/") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		for _, pattern := range stalePatterns {
			banTextPattern(&violations, rel, text, pattern, "stale mutable-config/manual-contribution reference")
		}
	}

	if len(violations) > 0 {
		t.Fatalf("config snapshot/documentation violations:\n%s", strings.Join(violations, "\n"))
	}
}

func TestArchitecture_InitExtensionContracts(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "pkg", "plugins", "cmd") {
		if !isProductionFile(rel) || !isInitWizardProductionFile(rel) {
			continue
		}
		file := parseGoFile(t, filepath.Join(root, rel), 0)
		initwizAliases := importAliases(file, moduleImportPath+"/pkg/plugin/initwiz")
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.CallExpr:
				if isBuildConfigCall(n) {
					violations = append(violations, rel+" calls config.BuildConfig; use config.NewExtensionValue/NewExtensionSet and config.Build")
				}
			case *ast.CompositeLit:
				if isMapStringAny(n.Type) || isMapStringMapStringAny(n.Type) {
					violations = append(violations, rel+" constructs untyped init extension config map; use a typed config struct or typed map values")
					return true
				}
				if isInitContributionLiteral(n.Type, initwizAliases) {
					violations = append(violations, rel+" manually constructs initwiz.InitContribution; use initwiz.NewInitContribution")
				}
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Fatalf("init extension contract violations:\n%s", strings.Join(violations, "\n"))
	}
}

func TestArchitecture_InitFlowBoundaries(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "cmd/terraci/cmd") {
		if !isProductionFile(rel) || !isInitCommandFile(rel) {
			continue
		}
		file := parseGoFile(t, filepath.Join(root, rel), 0)
		configAliases := importAliases(file, moduleImportPath+"/pkg/config")
		registryAliases := importAliases(file, moduleImportPath+"/pkg/plugin/registry")

		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if selector, ok := callSelector(call); ok {
				switch {
				case selectorCallMatches(selector, registryAliases, "ByCapabilityFrom"):
					violations = append(violations, rel+" discovers init contributors directly; use cmd/terraci/internal/initflow")
				case selectorCallMatches(selector, configAliases, "Build"),
					selectorCallMatches(selector, configAliases, "NewExtensionSet"):
					violations = append(violations, rel+" builds init config directly; delegate to initflow.BuildConfig")
				case selector.Sel.Name == "BuildInitConfig":
					violations = append(violations, rel+" calls plugin BuildInitConfig directly; delegate to initflow.BuildConfig")
				}
				return true
			}
			if ident, ok := call.Fun.(*ast.Ident); ok {
				switch ident.Name {
				case "BuildInitConfig", "initStateDefaults", "buildConfigFromState":
					violations = append(violations, rel+" uses command-level init orchestration helper "+ident.Name+"; delegate to initflow")
				}
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Fatalf("init flow boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}

func TestArchitecture_CommandRunFlowBoundaries(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "cmd", "pkg", "plugins", "test", "examples") {
		if !isProductionFile(rel) {
			continue
		}
		file := parseGoFile(t, filepath.Join(root, rel), 0)
		registryAliases := importAliases(file, moduleImportPath+"/pkg/plugin/registry")
		configAliases := importAliases(file, moduleImportPath+"/pkg/config")
		workflowAliases := importAliases(file, moduleImportPath+"/pkg/workflow")
		pipelineAliases := importAliases(file, moduleImportPath+"/pkg/pipeline")

		ast.Inspect(file, func(node ast.Node) bool {
			if strings.HasPrefix(rel, "cmd/terraci/cmd/") {
				switch n := node.(type) {
				case *ast.SelectorExpr:
					if ident, ok := n.X.(*ast.Ident); ok && ident.Name == "app" {
						switch n.Sel.Name {
						case "Config", "Plugins":
							violations = append(violations, rel+" reads mutable app."+n.Sel.Name+"; use runflow.Prepared from command context")
						}
					}
					if n.Sel.Name == "Annotations" {
						violations = append(violations, rel+" touches cobra annotations directly; use runflow.MarkCommand/PolicyFromCommand")
					}
				case *ast.KeyValueExpr:
					if ident, ok := n.Key.(*ast.Ident); ok && ident.Name == "Annotations" {
						violations = append(violations, rel+" sets cobra annotations directly; use runflow.MarkCommand")
					}
				}
			}

			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if strings.HasPrefix(rel, "cmd/terraci/cmd/") {
				if ident, identOK := call.Fun.(*ast.Ident); identOK {
					switch ident.Name {
					case "workflowOptions",
						"resolveGenerateTargets",
						"newPipelineGenerator",
						"formatGraph",
						"formatList",
						"formatLevels",
						"printStats",
						"computeLibraryModulesSummary":
						violations = append(violations, rel+" calls command-local orchestration helper "+ident.Name+"; delegate to cmd/terraci/internal/*flow")
					}
				}
			}
			selector, ok := callSelector(call)
			if !ok {
				return true
			}

			if selectorCallMatches(selector, registryAliases, "ByCapabilityFrom") && !isRegistryPackageFile(rel) {
				violations = append(violations, rel+" uses raw registry.ByCapabilityFrom; use a typed registry view or resolver method")
			}

			if strings.HasPrefix(rel, "cmd/terraci/cmd/") {
				switch {
				case selectorCallMatches(selector, configAliases, "Load"),
					selectorCallMatches(selector, configAliases, "LoadOrDefault"):
					violations = append(violations, rel+" loads command config directly; delegate command setup to runflow.Prepare")
				case selectorCallMatches(selector, workflowAliases, "Run"),
					selectorCallMatches(selector, workflowAliases, "ResolveTargets"):
					violations = append(violations, rel+" runs workflow orchestration directly; delegate to cmd/terraci/internal/projectflow")
				case selectorCallMatches(selector, pipelineAliases, "Build"):
					violations = append(violations, rel+" builds pipeline IR directly; delegate to cmd/terraci/internal/generateflow")
				case selector.Sel.Name == "DecodeAndSet",
					selector.Sel.Name == "Preflight",
					selector.Sel.Name == "PreflightsForStartup",
					selector.Sel.Name == "CollectContributions",
					selector.Sel.Name == "ConfigLoaders",
					selector.Sel.Name == "CommandProviders",
					selector.Sel.Name == "VersionProviders",
					selector.Sel.Name == "RuntimeProviders":
					violations = append(violations, rel+" runs command lifecycle or capability discovery directly; delegate to runflow/schemaflow/versionflow")
				}
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Fatalf("command run flow boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}

func TestArchitecture_WorkflowProjectPlanningBoundaries(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "cmd", "pkg", "plugins", "examples") {
		if !isProductionFile(rel) || strings.HasPrefix(rel, "pkg/workflow/") {
			continue
		}
		file := parseGoFile(t, filepath.Join(root, rel), 0)
		workflowAliases := importAliases(file, moduleImportPath+"/pkg/workflow")
		if len(workflowAliases) == 0 {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := callSelector(call)
			if !ok {
				return true
			}
			switch {
			case selectorCallMatches(selector, workflowAliases, "Run"):
				violations = append(violations, rel+" calls workflow.Run directly; use workflow.PlanProject")
			case selectorCallMatches(selector, workflowAliases, "ResolveTargets"):
				violations = append(violations, rel+" calls workflow.ResolveTargets directly; use workflow.PlanProject")
			case selectorCallMatches(selector, workflowAliases, "OptionsFromConfig"):
				violations = append(violations, rel+" calls workflow.OptionsFromConfig directly; use workflow.PlanProject")
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Fatalf("workflow project planning boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}

func TestArchitecture_PipelineIRValueBoundaries(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "cmd", "pkg", "plugins", "test", "examples") {
		if allowUnder(rel, "pkg/pipeline/") {
			continue
		}
		file := parseGoFile(t, filepath.Join(root, rel), 0)
		pipelineAliases := importAliases(file, moduleImportPath+"/pkg/pipeline")
		if len(pipelineAliases) == 0 {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.CallExpr:
				selector, ok := callSelector(typed)
				if !ok {
					return true
				}
				switch {
				case selectorCallMatches(selector, pipelineAliases, "Build"):
					violations = append(violations, rel+" calls pipeline.Build directly; use pipeline.BuildProjectIR")
				case selectorCallMatches(selector, pipelineAliases, "NewIR"):
					violations = append(violations, rel+" calls pipeline.NewIR directly; use pipeline.BuildProjectIR or pkg/pipeline/pipelinetest")
				case selectorCallMatches(selector, pipelineAliases, "NewCommandJob"):
					violations = append(violations, rel+" calls pipeline.NewCommandJob directly; use pipeline.BuildProjectIR or pkg/pipeline/pipelinetest")
				case selectorCallMatches(selector, pipelineAliases, "JobName"):
					violations = append(violations, rel+" calls pipeline.JobName directly; use IR.JobForModule, IR.FindJob, or pkg/pipeline/pipelinetest")
				}
			case *ast.CompositeLit:
				selector, ok := typed.Type.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				ident, ok := selector.X.(*ast.Ident)
				if !ok || !pipelineAliases[ident.Name] {
					return true
				}
				switch selector.Sel.Name {
				case "BuildOptions":
					violations = append(violations, rel+" manually constructs pipeline.BuildOptions; use pipeline.ProjectIRRequest")
				case "IR", "Job", "Operation", "TerraformOperation", "JobGroup":
					violations = append(violations, rel+" manually constructs pipeline."+selector.Sel.Name+"; use pipeline.BuildProjectIR or pkg/pipeline/pipelinetest")
				}
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Fatalf("pipeline IR value boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}

func TestArchitecture_CIRenderValueBoundaries(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "cmd", "pkg", "plugins", "test", "examples") {
		if allowUnder(rel, "pkg/ci/") || allowUnder(rel, "pkg/ci/citest/") {
			continue
		}
		file := parseGoFile(t, filepath.Join(root, rel), 0)
		ciAliases := importAliases(file, moduleImportPath+"/pkg/ci")
		if len(ciAliases) == 0 {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			lit, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			selector, ok := lit.Type.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if !ok || !ciAliases[ident.Name] {
				return true
			}
			switch selector.Sel.Name {
			case "RenderBlock", "RenderTable", "RenderValue":
				violations = append(violations, rel+" manually constructs ci."+selector.Sel.Name+"; use typed ci render constructors")
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Fatalf("ci render value boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}

func TestArchitecture_CIRenderDocs(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	stalePatterns := []string{
		"RenderTextBlock",
		"RenderListBlock",
		"RenderTableBlock",
		"RenderDetailsBlock",
		`"payload": {"blocks"`,
		`"payload":{"blocks"`,
	}
	for _, rel := range textFiles(t, root, "AGENTS.md", "docs", "examples", "pkg/plugin/doc.go") {
		if allowUnder(rel, "docs/.vitepress/dist/") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		for _, pattern := range stalePatterns {
			banTextPattern(&violations, rel, text, pattern, "stale unversioned render-payload reference")
		}
	}

	if len(violations) > 0 {
		t.Fatalf("ci render documentation violations:\n%s", strings.Join(violations, "\n"))
	}
}

func TestArchitecture_ProviderOutputValueBoundaries(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "plugins/gitlab", "plugins/github") {
		if !isProductionFile(rel) || isProviderDomainPackageFile(rel) {
			continue
		}
		file := parseGoFile(t, filepath.Join(root, rel), 0)
		gitlabDomainAliases := importAliases(file, moduleImportPath+"/plugins/gitlab/internal/domain")
		githubDomainAliases := importAliases(file, moduleImportPath+"/plugins/github/internal/domain")

		ast.Inspect(file, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.CompositeLit:
				if isProviderOutputLiteral(typed.Type, gitlabDomainAliases, githubDomainAliases) {
					violations = append(violations, rel+" manually constructs provider output domain value; use provider output builders")
				}
			case *ast.AssignStmt:
				for _, lhs := range typed.Lhs {
					if isProviderJobsIndex(lhs) {
						violations = append(violations, rel+" mutates provider document jobs map directly; use provider document builders")
					}
				}
			}
			return true
		})
	}

	stalePatterns := []string{
		"Pipeline{Stages, Jobs}",
		"Workflow{Jobs, Steps}",
		"domain.Pipeline{",
		"domain.Workflow{",
		"domain.Job{",
		"domain.Step{",
		".Jobs[",
	}
	for _, rel := range textFiles(t, root, "AGENTS.md", "docs", "examples", "pkg/plugin/doc.go") {
		if allowUnder(rel, "docs/.vitepress/dist/") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		for _, pattern := range stalePatterns {
			banTextPattern(&violations, rel, text, pattern, "stale mutable provider-output reference")
		}
	}

	if len(violations) > 0 {
		t.Fatalf("provider output value boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}

func TestArchitecture_CostAttributeBoundaries(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "plugins/cost/internal") {
		if !isProductionFile(rel) {
			continue
		}
		file := parseGoFile(t, filepath.Join(root, rel), 0)
		resourcespecAliases := importAliases(file, moduleImportPath+"/plugins/cost/internal/resourcespec")

		ast.Inspect(file, func(node ast.Node) bool {
			if len(resourcespecAliases) > 0 {
				if selector, ok := node.(*ast.SelectorExpr); ok {
					ident, identOK := selector.X.(*ast.Ident)
					if identOK && resourcespecAliases[ident.Name] {
						switch selector.Sel.Name {
						case "ResourceSpec", "LookupSpec", "DescribeSpec", "StandardPricingSpec",
							"FixedPricingSpec", "UsagePricingSpec", "SubresourceSpec", "Compile", "MustCompile":
							violations = append(violations, rel+" uses removed map-based resourcespec."+selector.Sel.Name+"; use resourcespec.TypedSpec")
						}
					}
				}
			}

			if !isCostAttrContractPackage(rel) {
				return true
			}
			if rel == "plugins/cost/internal/resourcedef/attrs.go" ||
				rel == "plugins/cost/internal/engine/terraform_adapter.go" {
				return true
			}
			if isMapStringAnyNode(node) {
				violations = append(violations, rel+" uses map[string]any in cost attribute contract code; use resourcedef.RawAttrs or typed attributes")
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Fatalf("cost attribute boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}

func TestArchitecture_PolicyOPABoundaries(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "plugins/policy") {
		if !isProductionFile(rel) {
			continue
		}
		file := parseGoFile(t, filepath.Join(root, rel), 0)
		inputAliases := importAliases(file, moduleImportPath+"/plugins/policy/internal/input")

		ast.Inspect(file, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.CompositeLit:
				if isPolicyEnvelopeLiteral(typed.Type, inputAliases) {
					violations = append(violations, rel+" manually constructs policyinput.Envelope; use policyinput.NewEnvelope or policyinput.Build")
				}
				if !isPolicyRawBoundaryFile(rel) && isMapStringAny(typed.Type) {
					violations = append(violations, rel+" constructs raw policy map[string]any outside OPA/JSON boundary")
				}
			case *ast.MapType:
				if !isPolicyRawBoundaryFile(rel) && isStringIdent(typed.Key) && isAnyIdent(typed.Value) {
					violations = append(violations, rel+" uses raw policy map[string]any outside OPA/JSON boundary")
				}
			case *ast.Field:
				if !isPolicyRawBoundaryFile(rel) && len(typed.Names) == 1 && typed.Names[0].Name == "Metadata" && isMapStringAny(typed.Type) {
					violations = append(violations, rel+" exposes Finding.Metadata as map[string]any; use policyengine.FindingMetadata")
				}
				for _, name := range typed.Names {
					if name.Name == "input" && isAnyIdent(typed.Type) {
						violations = append(violations, rel+" accepts untyped policy input; use policyinput.Envelope")
					}
				}
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Fatalf("policy OPA boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}

func repoRoot(tb testing.TB) string {
	tb.Helper()

	dir, err := os.Getwd()
	if err != nil {
		tb.Fatalf("Getwd() error = %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatal("repository root with go.mod not found")
		}
		dir = parent
	}
}

func isPolicyRawBoundaryFile(rel string) bool {
	return rel == "plugins/policy/internal/input/envelope.go" ||
		rel == "plugins/policy/internal/engine/engine.go" ||
		rel == "plugins/policy/internal/result.go"
}

func isPolicyEnvelopeLiteral(expr ast.Expr, aliases map[string]bool) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Envelope" {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && aliases[ident.Name]
}

func isProviderDomainPackageFile(rel string) bool {
	return strings.HasPrefix(rel, "plugins/gitlab/internal/domain/") ||
		strings.HasPrefix(rel, "plugins/github/internal/domain/")
}

func isProviderOutputLiteral(expr ast.Expr, aliasSets ...map[string]bool) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch selector.Sel.Name {
	case "Pipeline", "Workflow", "Job", "Step":
	default:
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	for _, aliases := range aliasSets {
		if aliases[ident.Name] {
			return true
		}
	}
	return false
}

func isProviderJobsIndex(expr ast.Expr) bool {
	index, ok := expr.(*ast.IndexExpr)
	if !ok {
		return false
	}
	selector, ok := index.X.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "Jobs"
}

func isCostAttrContractPackage(rel string) bool {
	return strings.HasPrefix(rel, "plugins/cost/internal/resourcedef/") ||
		strings.HasPrefix(rel, "plugins/cost/internal/resourcespec/") ||
		strings.HasPrefix(rel, "plugins/cost/internal/runtime/") ||
		strings.HasPrefix(rel, "plugins/cost/internal/engine/") ||
		strings.HasPrefix(rel, "plugins/cost/internal/cloud/aws/") ||
		strings.HasPrefix(rel, "plugins/cost/internal/cloud/awskit/")
}

func isMapStringAnyNode(node ast.Node) bool {
	switch typed := node.(type) {
	case *ast.MapType:
		return isStringIdent(typed.Key) && isAnyIdent(typed.Value)
	case *ast.CompositeLit:
		return isMapStringAny(typed.Type)
	default:
		return false
	}
}

func goFiles(tb testing.TB, root string, roots ...string) []string {
	tb.Helper()

	var files []string
	for _, scanRoot := range roots {
		err := filepath.WalkDir(filepath.Join(root, scanRoot), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				switch d.Name() {
				case ".git", "build", "dist", "testdata", "vendor":
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			tb.Fatalf("walk %s: %v", scanRoot, err)
		}
	}
	return files
}

func fileImports(tb testing.TB, path string) []string {
	tb.Helper()

	file := parseGoFile(tb, path, parser.ImportsOnly)

	imports := make([]string, 0, len(file.Imports))
	for _, spec := range file.Imports {
		importPath := importPath(tb, spec)
		if importPath != "" {
			imports = append(imports, importPath)
		}
	}
	return imports
}

func parseGoFile(tb testing.TB, path string, mode parser.Mode) *ast.File {
	tb.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, mode)
	if err != nil {
		tb.Fatalf("parse %s: %v", path, err)
	}
	return file
}

func importAliases(file *ast.File, target string) map[string]bool {
	aliases := make(map[string]bool)
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != target {
			continue
		}
		switch {
		case spec.Name == nil:
			aliases[filepath.Base(path)] = true
		case spec.Name.Name != "." && spec.Name.Name != "_":
			aliases[spec.Name.Name] = true
		}
	}
	return aliases
}

func containsConfigCall(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if found {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		found = selector.Sel.Name == "Config"
		return !found
	})
	return found
}

func isInitWizardProductionFile(rel string) bool {
	return strings.HasSuffix(rel, "/init_wizard.go") ||
		rel == "cmd/terraci/cmd/init.go" ||
		rel == "cmd/terraci/cmd/init_tui.go" ||
		strings.HasPrefix(rel, "pkg/plugin/initwiz/")
}

func isBuildConfigCall(call *ast.CallExpr) bool {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name == "BuildConfig"
	case *ast.SelectorExpr:
		ident, ok := fun.X.(*ast.Ident)
		return ok && ident.Name == "config" && fun.Sel.Name == "BuildConfig"
	default:
		return false
	}
}

func isInitCommandFile(rel string) bool {
	return rel == "cmd/terraci/cmd/init.go" || rel == "cmd/terraci/cmd/init_tui.go"
}

func isRegistryPackageFile(rel string) bool {
	return strings.HasPrefix(rel, "pkg/plugin/registry/")
}

func callSelector(call *ast.CallExpr) (*ast.SelectorExpr, bool) {
	fun := call.Fun
	switch typed := fun.(type) {
	case *ast.IndexExpr:
		fun = typed.X
	case *ast.IndexListExpr:
		fun = typed.X
	}
	selector, ok := fun.(*ast.SelectorExpr)
	return selector, ok
}

func selectorCallMatches(selector *ast.SelectorExpr, aliases map[string]bool, name string) bool {
	if selector == nil || selector.Sel.Name != name {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && aliases[ident.Name]
}

func banTextPattern(violations *[]string, rel, text, pattern, reason string) {
	if strings.Contains(text, pattern) {
		*violations = append(*violations, rel+" contains "+reason+" "+strconv.Quote(pattern))
	}
}

func allowUnder(rel string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	return false
}

func isMapStringMapStringAny(expr ast.Expr) bool {
	mapType, ok := expr.(*ast.MapType)
	if !ok || !isStringIdent(mapType.Key) {
		return false
	}
	return isMapStringAny(mapType.Value)
}

func isMapStringAny(expr ast.Expr) bool {
	mapType, ok := expr.(*ast.MapType)
	if !ok {
		return false
	}
	return isStringIdent(mapType.Key) && isAnyIdent(mapType.Value)
}

func isStringIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "string"
}

func isAnyIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "any"
}

func isInitContributionLiteral(expr ast.Expr, initwizAliases map[string]bool) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "InitContribution" {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && initwizAliases[ident.Name]
}

func importPath(tb testing.TB, spec *ast.ImportSpec) string {
	tb.Helper()
	path, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		tb.Fatalf("unquote import path %s: %v", spec.Path.Value, err)
	}
	return path
}

func isProductionFile(rel string) bool {
	return !strings.HasSuffix(rel, "_test.go")
}

func textFiles(tb testing.TB, root string, roots ...string) []string {
	tb.Helper()

	var files []string
	for _, scanRoot := range roots {
		path := filepath.Join(root, scanRoot)
		err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				switch d.Name() {
				case ".git", "build", "dist", "vendor", "node_modules":
					return filepath.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			switch filepath.Ext(path) {
			case ".md", ".go":
				files = append(files, rel)
			}
			if filepath.Base(path) == "AGENTS.md" {
				files = append(files, rel)
			}
			return nil
		})
		if err != nil {
			tb.Fatalf("walk %s: %v", scanRoot, err)
		}
	}
	return files
}

func importsPluginSDK(imp string) bool {
	return imp == moduleImportPath+"/pkg/plugin" || strings.HasPrefix(imp, moduleImportPath+"/pkg/plugin/")
}

func siblingPluginImportViolation(rel, imp string) (string, bool) {
	if !strings.HasPrefix(imp, moduleImportPath+"/plugins/") {
		return "", false
	}

	parts := strings.Split(rel, "/")
	if len(parts) < 2 {
		return "", false
	}
	pluginName := parts[1]
	if pluginName == "internal" {
		return "", false
	}

	importRest := strings.TrimPrefix(imp, moduleImportPath+"/plugins/")
	importedPlugin := strings.Split(importRest, "/")[0]
	if importedPlugin == pluginName || importedPlugin == "internal" {
		return "", false
	}

	scope := "production"
	if strings.HasSuffix(rel, "_test.go") {
		scope = "test"
	}
	return rel + " has " + scope + " import of sibling plugin package " + imp, true
}
