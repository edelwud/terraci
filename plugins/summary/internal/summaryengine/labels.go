package summaryengine

import (
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	tfplan "github.com/edelwud/terraci/internal/terraform/plan"
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/diagnostic"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/planresults"
)

var labelPlaceholderRE = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)

// LabelTemplate is a configured summary label expression.
type LabelTemplate string

// LabelRequest contains the data needed to resolve summary labels.
type LabelRequest struct {
	WorkDir     string
	Segments    []string
	PlanResults *ci.PlanResultCollection
	Templates   []string
	Parser      PlanParser
}

// LabelResult is the deterministic output of label resolution.
type LabelResult struct {
	Labels      []string
	Diagnostics diagnostic.List
}

// PlanParser parses a terraform plan JSON file for resource-level labels.
type PlanParser interface {
	ParsePlan(path string) (*tfplan.ParsedPlan, error)
}

type defaultPlanParser struct{}

func (defaultPlanParser) ParsePlan(path string) (*tfplan.ParsedPlan, error) {
	return tfplan.ParseJSON(path)
}

// ResolveLabels resolves static, module-level, and resource-level label templates.
func ResolveLabels(req LabelRequest) LabelResult {
	if len(req.Templates) == 0 {
		return LabelResult{}
	}
	parser := req.Parser
	if parser == nil {
		parser = defaultPlanParser{}
	}

	builder := newLabelBuilder()
	resourcePlans := newResourcePlanCache(parser, req.WorkDir)
	for _, raw := range req.Templates {
		tmpl := LabelTemplate(raw)
		placeholders := tmpl.Placeholders()
		switch {
		case len(placeholders) == 0:
			builder.addStatic(raw)
		case tmpl.HasResourcePlaceholders():
			resolveResourceLabels(builder, resourcePlans, req, tmpl, placeholders)
		default:
			resolveModuleLabels(builder, req, tmpl, placeholders)
		}
	}

	return LabelResult{
		Labels:      builder.labels(),
		Diagnostics: builder.diagnostics,
	}
}

// Placeholders returns the unique placeholders used by the template.
func (t LabelTemplate) Placeholders() []string {
	matches := labelPlaceholderRE.FindAllStringSubmatch(string(t), -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	placeholders := make([]string, 0, len(matches))
	for _, match := range matches {
		name := match[1]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		placeholders = append(placeholders, name)
	}
	sort.Strings(placeholders)
	return placeholders
}

// HasResourcePlaceholders reports whether this template needs plan.json resources.
func (t LabelTemplate) HasResourcePlaceholders() bool {
	for _, placeholder := range t.Placeholders() {
		if strings.HasPrefix(placeholder, "resource_") {
			return true
		}
	}
	return false
}

type labelBuilder struct {
	set         map[string]struct{}
	diagnostics diagnostic.List
}

func newLabelBuilder() *labelBuilder {
	return &labelBuilder{set: make(map[string]struct{})}
}

func (b *labelBuilder) addStatic(label string) {
	label = strings.TrimSpace(label)
	if label == "" {
		b.warn("summary label is empty and was skipped")
		return
	}
	b.set[label] = struct{}{}
}

func (b *labelBuilder) addRendered(tmpl LabelTemplate, values map[string]string, placeholders []string, context string) {
	label, unresolved := renderLabelTemplate(tmpl, values, placeholders)
	if len(unresolved) > 0 {
		b.warn(fmt.Sprintf("summary label %q skipped for %s: unresolved placeholders %s", tmpl, context, strings.Join(unresolved, ", ")))
		return
	}
	label = strings.TrimSpace(label)
	if label == "" {
		b.warn(fmt.Sprintf("summary label %q resolved to an empty value for %s and was skipped", tmpl, context))
		return
	}
	b.set[label] = struct{}{}
}

func (b *labelBuilder) warn(msg string) {
	b.diagnostics = b.diagnostics.Append(diagnostic.Warning(msg, diagnostic.WithSource("summary labels")))
}

func (b *labelBuilder) labels() []string {
	labels := make([]string, 0, len(b.set))
	for label := range b.set {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels
}

func resolveModuleLabels(builder *labelBuilder, req LabelRequest, tmpl LabelTemplate, placeholders []string) {
	plans := changedOrFailedPlans(labelPlans(req.PlanResults))
	for i := range plans {
		values := moduleLabelValues(plans[i], req.Segments)
		context := plans[i].ModulePath()
		if context == "" {
			context = plans[i].ModuleID()
		}
		builder.addRendered(tmpl, values, placeholders, context)
	}
}

func resolveResourceLabels(builder *labelBuilder, plans *resourcePlanCache, req LabelRequest, tmpl LabelTemplate, placeholders []string) {
	changed := changedPlans(labelPlans(req.PlanResults))
	for i := range changed {
		resources, err := plans.changedResources(changed[i])
		if err != nil {
			builder.warn(fmt.Sprintf("summary resource labels skipped for %s: %v", changed[i].ModulePath(), err))
			continue
		}
		moduleValues := moduleLabelValues(changed[i], req.Segments)
		for _, resource := range resources {
			values := copyStringMap(moduleValues)
			values["resource_address"] = resource.Address
			values["resource_type"] = resource.Type
			values["resource_name"] = resource.Name
			values["resource_action"] = resource.Action
			builder.addRendered(tmpl, values, placeholders, fmt.Sprintf("%s:%s", changed[i].ModulePath(), resource.Address))
		}
	}
}

func labelPlans(collection *ci.PlanResultCollection) []ci.PlanResult {
	if collection == nil {
		return nil
	}
	return collection.Results()
}

func renderLabelTemplate(tmpl LabelTemplate, values map[string]string, placeholders []string) (rendered string, unresolved []string) {
	unresolved = make([]string, 0)
	rendered = string(tmpl)
	for _, placeholder := range placeholders {
		value, ok := values[placeholder]
		if !ok || strings.TrimSpace(value) == "" {
			unresolved = append(unresolved, "{"+placeholder+"}")
			continue
		}
		rendered = strings.ReplaceAll(rendered, "{"+placeholder+"}", value)
	}
	return rendered, unresolved
}

func moduleLabelValues(result ci.PlanResult, segments []string) map[string]string {
	components := result.Components()
	values := make(map[string]string, len(components)+3)
	values["module_id"] = result.ModuleID()
	values["module_path"] = result.ModulePath()
	values["status"] = string(result.Status())

	if len(components) == 0 && result.ModulePath() != "" {
		components = planresults.ParseModulePathComponents(result.ModulePath(), segments)
	}
	maps.Copy(values, components)
	return values
}

func changedOrFailedPlans(plans []ci.PlanResult) []ci.PlanResult {
	out := make([]ci.PlanResult, 0, len(plans))
	for i := range plans {
		if plans[i].Status() == ci.PlanStatusChanges || plans[i].Status() == ci.PlanStatusFailed {
			out = append(out, plans[i])
		}
	}
	return out
}

func changedPlans(plans []ci.PlanResult) []ci.PlanResult {
	out := make([]ci.PlanResult, 0, len(plans))
	for i := range plans {
		if plans[i].Status() == ci.PlanStatusChanges {
			out = append(out, plans[i])
		}
	}
	return out
}

func changedResources(resources []tfplan.ResourceChange) []tfplan.ResourceChange {
	out := make([]tfplan.ResourceChange, 0, len(resources))
	for _, resource := range resources {
		if resource.Action == "" || resource.Action == tfplan.ActionNoOp {
			continue
		}
		out = append(out, resource)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Address == out[j].Address {
			return out[i].Action < out[j].Action
		}
		return out[i].Address < out[j].Address
	})
	return out
}

type resourcePlanCache struct {
	parser    PlanParser
	workDir   string
	resources map[string][]tfplan.ResourceChange
	errs      map[string]error
}

func newResourcePlanCache(parser PlanParser, workDir string) *resourcePlanCache {
	return &resourcePlanCache{
		parser:    parser,
		workDir:   workDir,
		resources: make(map[string][]tfplan.ResourceChange),
		errs:      make(map[string]error),
	}
}

func (c *resourcePlanCache) changedResources(result ci.PlanResult) ([]tfplan.ResourceChange, error) {
	key := result.ModulePath()
	if resources, ok := c.resources[key]; ok {
		return resources, nil
	}
	if err, ok := c.errs[key]; ok {
		return nil, err
	}
	parsed, err := c.parser.ParsePlan(planJSONPath(c.workDir, result.ModulePath()))
	if err != nil {
		c.errs[key] = err
		return nil, err
	}
	resources := changedResources(parsed.Resources)
	c.resources[key] = resources
	return resources, nil
}

func planJSONPath(workDir, modulePath string) string {
	return filepath.Join(workDir, filepath.FromSlash(modulePath), pipeline.PlanJSONFilename)
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	maps.Copy(out, in)
	return out
}
