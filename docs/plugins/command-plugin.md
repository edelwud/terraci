---
title: "CLI Command Plugin"
description: "Add custom terraci subcommands: Slack notifications, Jira tickets, audit reports, and any integration"
outline: deep
---

# CLI Command Plugin

The most common plugin type. Adds a new `terraci <command>` subcommand that users can run from the terminal.

::: tip CI Pipeline Integration
A `CommandProvider` alone only adds a CLI command. To have your command run automatically as a step in generated CI pipelines, you must also implement [`PipelineContributor`](/plugins/pipeline-plugin). This injects your command into the pipeline IR, and TerraCi generates the corresponding job/step in the output YAML.
:::

## Use Cases

- **Slack/Teams notifications** — post plan summaries to a channel
- **Jira/Linear tickets** — create issues from plan changes
- **Audit reports** — generate compliance reports from plan data
- **Custom cost providers** — extend cost estimation beyond AWS
- **Deployment gates** — check external approval systems before apply

## Minimal Example

```go
package slack

import (
    "fmt"

    "github.com/spf13/cobra"

    "github.com/edelwud/terraci/pkg/plugin"
    "github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
    registry.Register(&Plugin{
        BasePlugin: plugin.BasePlugin[*Config]{
            PluginName: "slack",
            PluginDesc: "Post plan summaries to Slack",
            EnableMode: plugin.EnabledExplicitly,
            DefaultCfg: func() *Config { return &Config{} },
            IsEnabledFn: func(cfg *Config) bool {
                return cfg != nil && cfg.Enabled
            },
        },
    })
}

type Plugin struct {
    plugin.BasePlugin[*Config]
}

type Config struct {
    Enabled    bool   `yaml:"enabled"`
    WebhookURL string `yaml:"webhook_url"`
    Channel    string `yaml:"channel"`
}

// Commands implements plugin.CommandProvider.
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
    var channel string

    cmd := &cobra.Command{
        Use:   "slack",
        Short: "Post plan summary to Slack",
        RunE: func(cmd *cobra.Command, _ []string) error {
            cfg := p.Config()
            if channel == "" {
                channel = cfg.Channel
            }
            fmt.Printf("Posting to %s via %s\n", channel, cfg.WebhookURL)
            // your Slack API logic here
            return nil
        },
    }

    cmd.Flags().StringVar(&channel, "channel", "", "Slack channel (overrides config)")

    return []*cobra.Command{cmd}
}
```

## Configuration

Users add your plugin to `.terraci.yaml`:

```yaml
plugins:
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/T.../B.../xxx"
    channel: "#terraform-deploys"
```

## Adding Flags

Use cobra's flag system. Flags are automatically shown in `terraci slack --help`:

```go
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
    var (
        channel string
        dryRun  bool
        format  string
    )

    cmd := &cobra.Command{
        Use:   "slack",
        Short: "Post plan summary to Slack",
        RunE: func(cmd *cobra.Command, _ []string) error {
            // use channel, dryRun, format
            return nil
        },
    }

    cmd.Flags().StringVar(&channel, "channel", "", "Slack channel")
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without posting")
    cmd.Flags().StringVar(&format, "output", "text", "Output format: text, json")

    return []*cobra.Command{cmd}
}
```

## Multiple Subcommands

Return multiple commands to add a command group:

```go
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
    return []*cobra.Command{
        {
            Use:   "notify send",
            Short: "Send notification",
            RunE:  func(cmd *cobra.Command, _ []string) error { /* ... */ return nil },
        },
        {
            Use:   "notify status",
            Short: "Check notification delivery",
            RunE:  func(cmd *cobra.Command, _ []string) error { /* ... */ return nil },
        },
    }
}
```

## Accessing Module Data

Use `discovery.Scanner` to find Terraform modules:

```go
func runMyCommand(ctx context.Context, appCtx *plugin.AppContext) error {
    cfg := appCtx.Config()
    segments, err := config.ParsePattern(cfg.Structure.Pattern)
    if err != nil {
        return err
    }

    scanner := discovery.NewScanner(appCtx.WorkDir(), segments)
    modules, err := scanner.Scan(ctx)
    if err != nil {
        return err
    }

    for _, m := range modules {
        fmt.Printf("Module: %s (%d .tf files)\n", m.Path, len(m.Files))
    }
    return nil
}
```

## Reading Plan Results

For plugins that process `terraform plan` output:

```go
func readPlanResults(appCtx *plugin.AppContext) error {
    collection, err := discovery.ScanPlanResults(appCtx.ServiceDir())
    if err != nil {
        return err
    }

    for _, plan := range collection.Plans {
        fmt.Printf("Module: %s, Changes: %d add, %d change, %d destroy\n",
            plan.ModulePath, plan.Add, plan.Change, plan.Destroy)
    }
    return nil
}
```

## Heavy Initialization with RuntimeProvider

If your command needs expensive setup (API clients, caches), use the lazy runtime pattern:

```go
type slackRuntime struct {
    client *slack.Client
}

func (p *Plugin) Runtime(_ context.Context, _ *plugin.AppContext) (any, error) {
    cfg := p.Config()
    client := slack.New(cfg.WebhookURL)
    return &slackRuntime{client: client}, nil
}

func (p *Plugin) runtime(ctx context.Context, appCtx *plugin.AppContext) (*slackRuntime, error) {
    return plugin.BuildRuntime[*slackRuntime](ctx, p, appCtx)
}
```

The runtime is only constructed when the command actually runs — not at startup.

## Preflight Validation

Add cheap checks that run before any command:

```go
func (p *Plugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
    cfg := p.Config()
    if cfg.WebhookURL == "" {
        return fmt.Errorf("slack: webhook_url is required")
    }
    return nil
}
```

## Adding to CI Pipelines

To run your command as a step in generated pipelines, implement `PipelineContributor` alongside `CommandProvider`:

```go
import "github.com/edelwud/terraci/pkg/pipeline"

// PipelineContribution injects `terraci slack` into the PostPlan phase.
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
    cfg := p.Config()
    if cfg == nil || !cfg.Pipeline {
        return nil
    }

    return &pipeline.Contribution{
        Jobs: []pipeline.ContributedJob{
            {
                Name:          "slack-notify",
                Phase:         pipeline.PhasePostPlan,
                DependsOnPlan: true,
                Commands:      []string{"terraci slack --channel " + cfg.Channel},
                AllowFailure:  true,
            },
        },
    }
}
```

Add a `pipeline` toggle to your config so users can opt in:

```yaml
plugins:
  slack:
    enabled: true
    channel: "#terraform-deploys"
    pipeline: true   # inject into CI pipeline
```

```go
type Config struct {
    Enabled  bool   `yaml:"enabled"`
    Channel  string `yaml:"channel"`
    Pipeline bool   `yaml:"pipeline"`  // opt-in for pipeline integration
}
```

This generates a standalone `slack-notify` job that runs after all plan jobs complete. Without `PipelineContributor`, users would have to manually add the step to their pipeline config.

## Full Project Layout

```
terraci-plugin-slack/
├── go.mod
├── go.sum
├── plugin.go       # init(), Plugin, Config
├── commands.go     # CommandProvider
├── runtime.go      # RuntimeProvider (optional)
├── lifecycle.go    # Preflightable (optional)
└── README.md
```

## Build and Test

```bash
# Build with your plugin
xterraci build \
  --with github.com/your-org/terraci-plugin-slack=./terraci-plugin-slack \
  --output ./build/terraci

# Test
./build/terraci slack --channel #test --dry-run
```

## See Also

- [Pipeline Step Plugin](/plugins/pipeline-plugin) — inject steps into CI pipelines
- [Working Example](https://github.com/edelwud/terraci/tree/main/examples/external-plugin)
