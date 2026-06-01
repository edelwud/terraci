---
title: "Init Wizard Plugin"
description: "Add configuration fields to the terraci init interactive wizard"
outline: deep
---

# Init Wizard Plugin

Add your plugin's configuration fields to the `terraci init` interactive TUI wizard. Users can configure your plugin through a guided form instead of editing YAML manually.

## Use Cases

- **Plugin settings** — let users enable/configure your plugin during setup
- **Team defaults** — provide curated presets for your organization
- **Feature toggles** — add on/off switches for optional features

## How It Works

The init wizard collects form groups from all plugins and renders them in a TUI:

```
┌─────────────────────────────────────────────┐
│  TerraCi Setup                              │
│                                             │
│  Basics                                     │
│    Provider: [GitLab ▾]                     │
│    Binary:   [terraform ▾]                  │
│    Pattern:  {service}/{env}/{region}/{mod}  │
│                                             │
│  Features                                   │
│    ☑ Enable plan summaries?                 │
│    ☐ Enable cost estimation?                │
│    ☑ Enable Slack notifications?  ← yours   │
│                                             │
│  Slack Settings              ← your detail  │
│    Channel: #terraform-deploys              │
│    Webhook: https://hooks.slack.com/...     │
│                                             │
└─────────────────────────────────────────────┘
```

## Implementation

Implement `InitContributor` from `pkg/plugin/initwiz`:

```go
import "github.com/edelwud/terraci/pkg/plugin/initwiz"

var (
    slackEnabledKey   = initwiz.MustStateKey[bool]("slack.enabled")
    slackChannelKey   = initwiz.MustStateKey[string]("slack.channel")
    slackOnFailureKey = initwiz.MustStateKey[string]("slack.on_failure")
)

// InitGroups returns form groups for the init wizard.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
    return []*initwiz.InitGroupSpec{
        {
            Title:    "Slack Notifications",
            Category: initwiz.CategoryFeature,
            Order:    300,
            Fields: []initwiz.InitField{
                initwiz.NewBoolField(initwiz.BoolFieldOptions{
                    Key:         slackEnabledKey,
                    Title:       "Enable Slack notifications?",
                    Description: "Post plan summaries to a Slack channel",
                    Default:     false,
                }),
            },
        },
        {
            Title:    "Slack Settings",
            Category: initwiz.CategoryDetail,
            Order:    300,
            ShowWhen: func(s *initwiz.StateMap) bool {
                return slackEnabledKey.Get(s)
            },
            Fields: []initwiz.InitField{
                initwiz.NewStringField(initwiz.StringFieldOptions{
                    Key:         slackChannelKey,
                    Title:       "Slack Channel",
                    Description: "Channel to post notifications to",
                    Default:     "#terraform-deploys",
                    Placeholder: "#terraform-deploys",
                }),
                initwiz.NewSelectField(initwiz.SelectFieldOptions{
                    Key:     slackOnFailureKey,
                    Title:   "Notify on failure",
                    Default: "always",
                    Options: []initwiz.InitOption{
                        {Label: "Always", Value: "always"},
                        {Label: "Only on failure", Value: "failure"},
                        {Label: "Never", Value: "never"},
                    },
                }),
            },
        },
    }
}

type SlackConfig struct {
    Enabled   bool   `yaml:"enabled,omitempty"`
    Channel   string `yaml:"channel,omitempty"`
    OnFailure string `yaml:"on_failure,omitempty"`
}

func (c SlackConfig) Clone() SlackConfig { return c }

// BuildInitConfig constructs the plugin's typed config from wizard state.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) (*initwiz.InitContribution, error) {
    if !slackEnabledKey.Get(state) {
        return nil, nil
    }

    return initwiz.NewInitContribution("slack", SlackConfig{
        Enabled:   true,
        Channel:   slackChannelKey.Get(state),
        OnFailure: slackOnFailureKey.Get(state),
    })
}
```

The contribution contract is typed end to end: plugin code returns a typed
config struct, `initwiz.NewInitContribution` encodes it into a validated
extension value, and `cmd/terraci/internal/initflow` assembles the final file.
Skip optional config by returning `nil, nil`; return an error when the wizard
state cannot produce a valid config. The command package only renders the TUI,
preview, and output file.

## Form Categories

Fields are grouped into categories that determine where they appear:

| Category | Rendering | Use For |
|----------|-----------|---------|
| `CategoryProvider` | Separate group, shown per-provider | CI-specific settings (image, runner) |
| `CategoryFeature` | Merged into "Features" group | On/off toggles for optional features |
| `CategoryDetail` | Separate group with `ShowWhen` | Detail settings for enabled features |

### Common Pattern: Feature Toggle + Detail

Most plugins use two groups: a feature toggle in `CategoryFeature` and detailed settings in `CategoryDetail` that appear only when the feature is enabled:

```go
var myPluginEnabledKey = initwiz.MustStateKey[bool]("myplugin.enabled")

// Group 1: Feature toggle (merged with other plugins' toggles)
{
    Category: initwiz.CategoryFeature,
    Fields: []initwiz.InitField{
        initwiz.NewBoolField(initwiz.BoolFieldOptions{
            Key:   myPluginEnabledKey,
            Title: "Enable my plugin?",
        }),
    },
}

// Group 2: Detail settings (shown only when enabled)
{
    Category: initwiz.CategoryDetail,
    ShowWhen: func(s *initwiz.StateMap) bool {
        return myPluginEnabledKey.Get(s)
    },
    Fields: []initwiz.InitField{
        // additional initwiz.NewStringField/NewSelectField entries
    },
}
```

## Field Types

| Type | Widget | Value Type |
|------|--------|------------|
| `FieldString` | Text input | `string` |
| `FieldBool` | Confirm toggle | `bool` |
| `FieldSelect` | Select dropdown | `string` |

## Ordering

`Order` controls group position within its category. Lower values appear first. Built-in plugins use:

| Plugin | Order |
|--------|-------|
| CI providers | 100 |
| summary | 199 |
| cost | 200 |
| policy | 201 |
| update | 202 |

Use `300+` for custom plugins to appear after built-ins.

## StateMap

`StateMap` is mutable form state, but plugin authors access it only through
typed `StateKey[T]` values. Define keys once at package scope:

```go
var channelKey = initwiz.MustStateKey[string]("slack.channel")
var enabledKey = initwiz.MustStateKey[bool]("slack.enabled")

channel := channelKey.Get(state)
enabled, explicitlySet := enabledKey.Lookup(state)
enabledKey.Set(state, true)
```

The TUI layer binds stable pointers through those same keys; plugins normally do
not need this unless they build their own UI:

```go
channelPtr := channelKey.Bind(state)
enabledPtr := enabledKey.Bind(state)
```

## Generated Config

When the wizard completes, initflow calls typed `BuildInitConfig`
contributions and assembles `.terraci.yaml`:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

extensions:
  gitlab:
    image: { name: hashicorp/terraform:1.6 }

  slack:                    # ← your plugin's contribution
    enabled: true
    channel: "#terraform-deploys"
    on_failure: always
```

## See Also

- [CLI Command Plugin](/plugins/command-plugin) — add CLI commands
- [terraci init reference](/cli/init) — init command usage
