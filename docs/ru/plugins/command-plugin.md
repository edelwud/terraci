---
title: "Плагин CLI-команды"
description: "Добавление кастомных субкоманд terraci: Slack-уведомления, Jira-тикеты, аудит-отчёты"
outline: deep
---

# Плагин CLI-команды

Самый распространённый тип плагина. Добавляет новую субкоманду `terraci <command>`, которую можно запускать из терминала.

::: tip Интеграция с CI-пайплайном
`CommandProvider` сам по себе добавляет только CLI-команду. Чтобы команда автоматически выполнялась как шаг в генерируемых CI-пайплайнах, необходимо также реализовать [`PipelineContributor`](/ru/plugins/pipeline-plugin). Этот интерфейс внедряет вашу команду в pipeline IR, и TerraCi генерирует соответствующий джоб/шаг в выходном YAML.
:::

## Сценарии использования

- **Slack/Teams уведомления** — отправка сводки планов в канал
- **Jira/Linear тикеты** — создание задач из изменений плана
- **Аудит-отчёты** — генерация отчётов compliance из данных плана
- **Дополнительные провайдеры стоимости** — расширение cost estimation за пределы AWS
- **Гейты деплоя** — проверка внешних систем согласования перед apply

## Минимальный пример

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

func (p *Plugin) CommandSpecs() ([]plugin.CommandSpec, error) {
    cmd, err := plugin.NewCommandSpec(plugin.CommandSpecOptions{
        Use:   "slack",
        Short: "Post plan summary to Slack",
        RunE: func(cmd *cobra.Command, _ []string) error {
            cmdCtx, current, err := plugin.CommandPlugin[*Plugin](cmd, "slack")
            if err != nil {
                return err
            }
            if err := plugin.RequireEnabled(current, "slack plugin is not enabled"); err != nil {
                return err
            }
            cfg := current.Config()
            appCtx := cmdCtx.AppContext()
            _ = appCtx // используйте AppContext для reports/resolvers/runtime paths, если нужно
            fmt.Printf("Posting to %s\n", cfg.Channel)
            return nil
        },
    })
    if err != nil {
        return nil, err
    }
    return []plugin.CommandSpec{cmd}, nil
}
```

## Конфигурация

```yaml
extensions:
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/T.../B.../xxx"
    channel: "#terraform-deploys"
```

## Ключевые паттерны

### Флаги команды

```go
cmd.Flags().StringVar(&channel, "channel", "", "Slack channel")
cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without posting")
```

### Доступ к модулям

```go
project, err := workflow.PlanProject(ctx, workflow.ProjectRequest{
    WorkDir: appCtx.WorkDir(),
    Config:  appCtx.Config(),
})
if err != nil {
    return err
}
modules := project.Workflow.Filtered.All()
```

### Ленивая инициализация тяжёлых зависимостей

```go
func (p *Plugin) runtime(_ context.Context, _ *plugin.AppContext) (*slackRuntime, error) {
    return &slackRuntime{client: slack.New(p.Config().WebhookURL)}, nil
}
```

### Preflight-валидация

```go
func (p *Plugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
    if p.Config().WebhookURL == "" {
        return fmt.Errorf("slack: webhook_url is required")
    }
    return nil
}
```

## Добавление в CI-пайплайн

Чтобы ваша команда выполнялась как шаг в генерируемых пайплайнах, реализуйте `PipelineContributor` вместе с `CommandProvider`:

```go
import (
    "fmt"

    "github.com/edelwud/terraci/pkg/pipeline"
    "github.com/edelwud/terraci/pkg/plugin"
)

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

Добавьте переключатель `pipeline` в конфиг:

```yaml
extensions:
  slack:
    enabled: true
    channel: "#terraform-deploys"
    pipeline: true   # внедрить в CI-пайплайн
```

```go
type Config struct {
    Enabled  bool   `yaml:"enabled"`
    Channel  string `yaml:"channel"`
    Pipeline bool   `yaml:"pipeline"`
}
```

Это генерирует отдельный джоб `slack-notify`, запускающийся после завершения всех plan-джобов. Без `PipelineContributor` пользователям пришлось бы вручную добавлять шаг в конфигурацию пайплайна.

## Структура проекта

```
terraci-plugin-slack/
├── go.mod
├── plugin.go       # init(), Plugin, Config
├── commands.go     # CommandProvider
├── runtime.go      # Plugin-local lazy runtime builder (опционально)
├── lifecycle.go    # Preflightable (опционально)
└── README.md
```

## Сборка и тестирование

```bash
xterraci build \
  --with github.com/your-org/terraci-plugin-slack=./terraci-plugin-slack \
  --output ./build/terraci

./build/terraci slack --channel #test --dry-run
```

## См. также

- [Шаг пайплайна](/ru/plugins/pipeline-plugin) — внедрение шагов в CI-пайплайны
- [Рабочий пример](https://github.com/edelwud/terraci/tree/main/examples/external-plugin)
