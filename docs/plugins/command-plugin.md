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
    registry.RegisterFactory(func() plugin.Plugin {
        return &Plugin{
            BasePlugin: plugin.BasePlugin[*Config]{
                PluginName: "slack",
                PluginDesc: "Post plan summaries to Slack",
                EnableMode: plugin.EnabledExplicitly,
                DefaultCfg: func() *Config { return &Config{} },
                IsEnabledFn: func(cfg *Config) bool {
                    return cfg != nil && cfg.Enabled
                },
            },
        }
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

func (c *Config) Clone() *Config {
    if c == nil {
        return nil
    }
    out := *c
    return &out
}

// Commands implements plugin.CommandProvider.
func (p *Plugin) Commands() []*cobra.Command {
    var channel string

    cmd := &cobra.Command{
        Use:   "slack",
        Short: "Post plan summary to Slack",
        RunE: func(cmd *cobra.Command, _ []string) error {
            _, current, err := plugin.CommandPlugin[*Plugin](cmd, "slack")
            if err != nil {
                return err
            }
            if err := plugin.RequireEnabled(current, "slack plugin is not enabled"); err != nil {
                return err
            }
            cfg := current.Config()
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
extensions:
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/T.../B.../xxx"
    channel: "#terraform-deploys"
```

## Adding Flags

Use cobra's flag system. Flags are automatically shown in `terraci slack --help`:

```go
func (p *Plugin) Commands() []*cobra.Command {
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
func (p *Plugin) Commands() []*cobra.Command {
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

Use the canonical project planner to find Terraform modules with the active
config, filters, library handling, and parser behavior:

```go
func runMyCommand(ctx context.Context, appCtx *plugin.AppContext) error {
    project, err := workflow.PlanProject(ctx, workflow.ProjectRequest{
        WorkDir: appCtx.WorkDir(),
        Config:  appCtx.Config(),
    })
    if err != nil {
        return err
    }

    modules := project.Workflow.Filtered.All()
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
    collection, err := planresults.Scan(appCtx.ServiceDir(), nil)
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

## Heavy Initialization with a Plugin-Local Runtime

If your command needs expensive setup (API clients, caches), use the lazy runtime pattern:

```go
type slackRuntime struct {
    client *slack.Client
}

func (p *Plugin) runtime(_ context.Context, _ *plugin.AppContext) (*slackRuntime, error) {
    cfg := p.Config()
    client := slack.New(cfg.WebhookURL)
    return &slackRuntime{client: client}, nil
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

To run your command as a standalone generated pipeline job, implement `PipelineContributor` alongside `CommandProvider`:

```go
import (
    "fmt"

    "github.com/edelwud/terraci/pkg/pipeline"
    "github.com/edelwud/terraci/pkg/plugin"
)

// PipelineContribution adds a DAG job that runs after plan JSON is available.
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) (*pipeline.Contribution, error) {
    cfg := p.Config()
    if cfg == nil {
        return nil, fmt.Errorf("slack config is required")
    }

    job, err := pipeline.NewPluginCommandJob(pipeline.PluginCommandJobOptions{
        Name: "slack-notify",
        Consumes: []pipeline.ResourceRequest{
            pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
        },
        Commands:     []string{"terraci slack --channel " + cfg.Channel},
        AllowFailure: true,
    })
    if err != nil {
        return nil, err
    }
    contribution, err := pipeline.NewContribution(job)
    if err != nil {
        return nil, err
    }
    return contribution, nil
}

func (p *Plugin) PipelineContributionEnabled(_ *plugin.AppContext) (bool, error) {
    cfg := p.Config()
    return cfg != nil && cfg.Pipeline, nil
}
```

Add a `pipeline` toggle to your config so users can opt in:

```yaml
extensions:
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
├── runtime.go      # Plugin-local lazy runtime builder (optional)
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

- [Pipeline Job Plugin](/plugins/pipeline-plugin) — add DAG jobs to CI pipelines
- [Working Example](https://github.com/edelwud/terraci/tree/main/examples/external-plugin)
