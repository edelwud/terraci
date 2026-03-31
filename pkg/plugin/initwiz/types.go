package initwiz

import "github.com/edelwud/terraci/pkg/plugin"

// InitCategory determines how an InitGroupSpec is rendered in the wizard.
type InitCategory string

const (
	// CategoryProvider groups contain CI-specific infrastructure settings (image, runner).
	CategoryProvider InitCategory = "provider"
	// CategoryPipeline groups contain pipeline behavior settings (plan_enabled, auto_approve).
	CategoryPipeline InitCategory = "pipeline"
	// CategoryFeature groups contain optional feature toggles (cost, policy, summary).
	CategoryFeature InitCategory = "feature"
	// CategoryDetail groups contain detail settings for enabled features (policy settings).
	CategoryDetail InitCategory = "detail"
)

// InitContributor plugins contribute fields and config to the init wizard.
type InitContributor interface {
	plugin.Plugin
	InitGroups() []*InitGroupSpec
	BuildInitConfig(state *StateMap) *InitContribution
}

// InitGroupSpec describes a group of form fields contributed by a plugin.
type InitGroupSpec struct {
	Title    string
	Category InitCategory
	Order    int
	Fields   []InitField
	ShowWhen func(*StateMap) bool
}

// FieldType identifies the kind of form field in the init wizard.
type FieldType string

const (
	FieldString FieldType = "string"
	FieldBool   FieldType = "bool"
	FieldSelect FieldType = "select"
)

// InitField describes a single form field in the init wizard.
type InitField struct {
	Key         string
	Title       string
	Description string
	Type        FieldType
	Default     any
	Options     []InitOption
	Placeholder string
}

// InitOption represents a selectable option for a field.
type InitOption struct {
	Label string
	Value string
}

// InitContribution holds the config produced by a plugin's init logic.
type InitContribution struct {
	PluginKey string
	Config    map[string]any
}
