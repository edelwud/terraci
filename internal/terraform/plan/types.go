// Package plan provides terraform plan JSON parsing functionality.
package plan

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
	Address    string // e.g. "module.vpc.aws_vpc.main"
	Type       string // e.g. "aws_vpc"
	Name       string // e.g. "main"
	ModuleAddr string // e.g. "module.vpc"
	Action     string // "create", "update", "delete", "replace", "read", "no-op"
	Attributes []AttrDiff
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
