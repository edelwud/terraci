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
	key           string
	title         string
	description   string
	typ           FieldType
	stringKey     StateKey[string]
	boolKey       StateKey[bool]
	stringDefault string
	boolDefault   bool
	options       []InitOption
	placeholder   string
}

// InitOption represents a selectable option for a field.
type InitOption struct {
	Label string
	Value string
}

// StringFieldOptions describes a string input field.
type StringFieldOptions struct {
	Key         StateKey[string]
	Title       string
	Description string
	Default     string
	Placeholder string
}

// BoolFieldOptions describes a boolean confirmation field.
type BoolFieldOptions struct {
	Key         StateKey[bool]
	Title       string
	Description string
	Default     bool
}

// SelectFieldOptions describes a string select field.
type SelectFieldOptions struct {
	Key         StateKey[string]
	Title       string
	Description string
	Default     string
	Options     []InitOption
}

// NewStringField constructs a string input field.
func NewStringField(opts StringFieldOptions) InitField {
	validateFieldOptions(opts.Key.Name(), opts.Title, FieldString, nil)
	return InitField{
		key:           opts.Key.Name(),
		title:         opts.Title,
		description:   opts.Description,
		typ:           FieldString,
		stringKey:     opts.Key,
		stringDefault: opts.Default,
		placeholder:   opts.Placeholder,
	}
}

// NewBoolField constructs a boolean confirmation field.
func NewBoolField(opts BoolFieldOptions) InitField {
	validateFieldOptions(opts.Key.Name(), opts.Title, FieldBool, nil)
	return InitField{
		key:         opts.Key.Name(),
		title:       opts.Title,
		description: opts.Description,
		typ:         FieldBool,
		boolKey:     opts.Key,
		boolDefault: opts.Default,
	}
}

// NewSelectField constructs a string select field.
func NewSelectField(opts SelectFieldOptions) InitField {
	validateFieldOptions(opts.Key.Name(), opts.Title, FieldSelect, opts.Options)
	return InitField{
		key:           opts.Key.Name(),
		title:         opts.Title,
		description:   opts.Description,
		typ:           FieldSelect,
		stringKey:     opts.Key,
		stringDefault: opts.Default,
		options:       cloneOptions(opts.Options),
	}
}

// Key returns the raw state key name for display and deterministic de-duping.
func (f InitField) Key() string { return f.key }

// Title returns the field title.
func (f InitField) Title() string { return f.title }

// Description returns the field description.
func (f InitField) Description() string { return f.description }

// Type returns the field type.
func (f InitField) Type() FieldType { return f.typ }

// Options returns defensive select options.
func (f InitField) Options() []InitOption { return cloneOptions(f.options) }

// Placeholder returns the string input placeholder.
func (f InitField) Placeholder() string { return f.placeholder }

// StringKey returns the typed string key for string and select fields.
func (f InitField) StringKey() StateKey[string] { return f.stringKey }

// BoolKey returns the typed bool key for bool fields.
func (f InitField) BoolKey() StateKey[bool] { return f.boolKey }

// ApplyDefault writes the field's default value when the key is not present.
func (f InitField) ApplyDefault(state *StateMap) {
	switch f.typ {
	case FieldBool:
		if _, ok := f.boolKey.Lookup(state); !ok {
			f.boolKey.Set(state, f.boolDefault)
		}
	case FieldString, FieldSelect:
		if _, ok := f.stringKey.Lookup(state); !ok {
			f.stringKey.Set(state, f.stringDefault)
		}
	}
}

// Clone returns a defensive field copy.
func (f InitField) Clone() InitField {
	f.options = cloneOptions(f.options)
	return f
}

func cloneOptions(options []InitOption) []InitOption {
	if len(options) == 0 {
		return nil
	}
	return append([]InitOption(nil), options...)
}

func validateFieldOptions(key, title string, typ FieldType, options []InitOption) {
	if strings.TrimSpace(key) == "" {
		panic("init field key is required")
	}
	if strings.TrimSpace(title) == "" {
		panic("init field title is required")
	}
	if typ == FieldSelect && len(options) == 0 {
		panic("init select field options are required")
	}
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
