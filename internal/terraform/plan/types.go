// Package plan provides terraform plan JSON parsing functionality.
package plan

// Resource change action constants.
const (
	ActionCreate  = "create"
	ActionUpdate  = "update"
	ActionDelete  = "delete"
	ActionReplace = "replace"
	ActionRead    = "read"
	ActionNoOp    = "no-op"
)

// ParsedPlan represents a parsed terraform plan with extracted changes.
type ParsedPlan struct {
	TerraformVersion string
	FormatVersion    string
	ToAdd            int
	ToChange         int
	ToDestroy        int
	ToImport         int
	Resources        []ResourceChange
}

// HasChanges returns true if the plan has any changes.
func (p *ParsedPlan) HasChanges() bool {
	return p.ToAdd > 0 || p.ToChange > 0 || p.ToDestroy > 0
}

// ResourceChange represents a single resource change extracted from the plan.
type ResourceChange struct {
	Address      string         // e.g. "module.vpc.aws_vpc.main"
	Type         string         // e.g. "aws_vpc"
	Name         string         // e.g. "main"
	ModuleAddr   string         // e.g. "module.vpc"
	Action       string         // "create", "update", "delete", "replace", "read", "no-op"
	Attributes   []AttrDiff     // attribute-level diffs (empty for no-op)
	BeforeValues map[string]any // full before-state attributes from plan JSON
	AfterValues  map[string]any // full after-state attributes from plan JSON
}

// AttrDiff represents a single attribute change.
type AttrDiff struct {
	Path      string // e.g. "instance_type", "tags.Name"
	OldValue  string
	NewValue  string
	Sensitive bool
	ForceNew  bool
	Computed  bool
}
