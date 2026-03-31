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

// InitGroups returns form groups for the init wizard.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
    return []*initwiz.InitGroupSpec{
        {
            Title:    "Slack Notifications",
            Category: initwiz.CategoryFeature,
            Order:    300,
            Fields: []initwiz.InitField{
                {
                    Key:         "slack.enabled",
                    Title:       "Enable Slack notifications?",
                    Description: "Post plan summaries to a Slack channel",
                    Type:        initwiz.FieldBool,
                    Default:     false,
                },
            },
        },
        {
            Title:    "Slack Settings",
            Category: initwiz.CategoryDetail,
            Order:    300,
            ShowWhen: func(s *initwiz.StateMap) bool {
                return s.Bool("slack.enabled")
            },
            Fields: []initwiz.InitField{
                {
                    Key:         "slack.channel",
                    Title:       "Slack Channel",
                    Description: "Channel to post notifications to",
                    Type:        initwiz.FieldString,
                    Default:     "#terraform-deploys",
                    Placeholder: "#terraform-deploys",
                },
                {
                    Key:     "slack.on_failure",
                    Title:   "Notify on failure",
                    Type:    initwiz.FieldSelect,
                    Default: "always",
                    Options: []initwiz.InitOption{
                        {Label: "Always", Value: "always"},
                        {Label: "Only on failure", Value: "failure"},
                        {Label: "Never", Value: "never"},
                    },
                },
            },
        },
    }
}

// BuildInitConfig constructs the plugin's config from wizard state.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) *initwiz.InitContribution {
    if !state.Bool("slack.enabled") {
        return nil
    }

    return &initwiz.InitContribution{
        PluginKey: "slack",
        Config: map[string]any{
            "enabled":    true,
            "channel":    state.String("slack.channel"),
            "on_failure": state.String("slack.on_failure"),
        },
    }
}
```

## Form Categories

Fields are grouped into categories that determine where they appear:

| Category | Rendering | Use For |
|----------|-----------|---------|
| `CategoryProvider` | Separate group, shown per-provider | CI-specific settings (image, runner) |
| `CategoryPipeline` | Merged into "Pipeline" group | Pipeline behavior (plan_enabled, auto_approve) |
| `CategoryFeature` | Merged into "Features" group | On/off toggles for optional features |
| `CategoryDetail` | Separate group with `ShowWhen` | Detail settings for enabled features |

### Common Pattern: Feature Toggle + Detail

Most plugins use two groups: a feature toggle in `CategoryFeature` and detailed settings in `CategoryDetail` that appear only when the feature is enabled:

```go
// Group 1: Feature toggle (merged with other plugins' toggles)
{
    Category: initwiz.CategoryFeature,
    Fields: []initwiz.InitField{{
        Key:  "myplugin.enabled",
        Type: initwiz.FieldBool,
    }},
}

// Group 2: Detail settings (shown only when enabled)
{
    Category: initwiz.CategoryDetail,
    ShowWhen: func(s *initwiz.StateMap) bool {
        return s.Bool("myplugin.enabled")
    },
    Fields: []initwiz.InitField{...},
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

`StateMap` provides typed access to form values:

```go
state.String("key")     // returns string or ""
state.Bool("key")       // returns bool or false
state.Get("key")        // returns any or nil
state.Provider()        // shorthand for state.String("provider")
state.Binary()          // shorthand for state.String("binary")
```

For form binding, plugins receive `*string` / `*bool` pointers that the TUI mutates directly:

```go
state.StringPtr("key")  // stable *string pointer for huh form
state.BoolPtr("key")    // stable *bool pointer for huh form
```

## Generated Config

When the wizard completes, `BuildInitConfig` results are assembled into `.terraci.yaml`:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

plugins:
  gitlab:
    image: { name: hashicorp/terraform:1.6 }
    plan_enabled: true

  slack:                    # ← your plugin's contribution
    enabled: true
    channel: "#terraform-deploys"
    on_failure: always
```

## See Also

- [CLI Command Plugin](/plugins/command-plugin) — add CLI commands
- [terraci init reference](/cli/init) — init command usage
