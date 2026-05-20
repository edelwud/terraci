package initwiz

import (
	"errors"
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
)

// InitCategory determines how an InitGroupSpec is rendered in the wizard.
type InitCategory string

const (
	// CategoryProvider groups contain CI-specific infrastructure settings (image, runner).
	CategoryProvider InitCategory = "provider"
	// CategoryPipeline groups contain pipeline behavior settings.
	CategoryPipeline InitCategory = "pipeline"
	// CategoryFeature groups contain optional feature toggles.
	CategoryFeature InitCategory = "feature"
	// CategoryDetail groups contain detail settings for enabled features.
	CategoryDetail InitCategory = "detail"
)

// InitContributor plugins contribute fields and config to the init wizard.
type InitContributor interface {
	plugin.Plugin
	InitGroups() []*InitGroupSpec
	BuildInitConfig(state *StateMap) (*InitContribution, error)
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

// InitContribution holds a validated extension config produced by a plugin's
// init logic.
type InitContribution struct {
	pluginKey string
	config    config.ExtensionValue
}

// NewInitContribution builds a validated init contribution from typed config.
func NewInitContribution(pluginKey string, typedConfig any) (*InitContribution, error) {
	pluginKey = strings.TrimSpace(pluginKey)
	if pluginKey == "" {
		return nil, errors.New("init contribution plugin key is required")
	}
	value, err := config.NewExtensionValue(pluginKey, typedConfig)
	if err != nil {
		return nil, fmt.Errorf("init contribution %q: %w", pluginKey, err)
	}
	return &InitContribution{pluginKey: pluginKey, config: value}, nil
}

// PluginKey returns the extension config key.
func (c *InitContribution) PluginKey() string {
	if c == nil {
		return ""
	}
	return c.pluginKey
}

// ExtensionValue returns a defensive copy of the encoded extension config.
func (c *InitContribution) ExtensionValue() config.ExtensionValue {
	if c == nil {
		return config.ExtensionValue{}
	}
	return c.config.Clone()
}

// DecodeConfig decodes the contribution config into target.
func (c *InitContribution) DecodeConfig(target any) error {
	if c == nil {
		return nil
	}
	return c.config.Decode(target)
}
