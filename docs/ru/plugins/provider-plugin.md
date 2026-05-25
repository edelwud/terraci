---
title: "Плагин CI-провайдера"
description: "Поддержка новых CI-систем: Bitbucket Pipelines, Jenkins, CircleCI, Azure DevOps"
outline: deep
---

# Плагин CI-провайдера

Добавляет поддержку новой CI-системы помимо GitLab CI и GitHub Actions. Самый сложный тип плагина — требует реализации нескольких интерфейсов для генерации YAML, определения окружения и публикации MR/PR-комментариев.

## Сценарии

- **Bitbucket Pipelines** — генерация `bitbucket-pipelines.yml`
- **Jenkins** — генерация `Jenkinsfile` с параллельными стадиями
- **CircleCI** — генерация `.circleci/config.yml`
- **Azure DevOps** — генерация `azure-pipelines.yml`

## Обязательные интерфейсы

| Интерфейс | Назначение |
|-----------|-----------|
| `EnvDetector` | Определение CI-окружения |
| `CIInfoProvider` | Имя провайдера, ID пайплайна, SHA коммита |
| `PipelineGeneratorFactory` | Декларация требований к IR и создание IR-bound генератора |

Опционально:

| Интерфейс | Назначение |
|-----------|-----------|
| `CommentServiceFactory` | Сервис MR/PR комментариев |

## Работа с Pipeline IR

Ядро сначала спрашивает у провайдера `PipelineRequirements(ctx)`, затем строит
IR один раз через `pipeline.BuildProjectIR(req)` и передаёт его в фабрику.
Генератор только рендерит immutable IR через getters. IR уже содержит модули,
contributions и зависимости.

```go
func (p *Plugin) PipelineRequirements(ctx *plugin.AppContext) pipeline.BuildRequirements {
    return pipeline.RequirementsForResources(
        pipeline.AllPlanResources(pipeline.ResourceKindPlanText),
        pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
    )
}

func (p *Plugin) NewGenerator(ctx *plugin.AppContext, ir *pipeline.IR) pipeline.Generator {
    return &generator{config: p.Config(), ir: ir}
}

func (g *generator) Generate() (pipeline.GeneratedPipeline, error) {
    // IR — это flat DAG из pipeline.Job value objects.
    for _, job := range g.ir.Jobs() {
        // job.Kind() — plan, apply или command
        // job.Module() — module metadata для plan/apply
        // job.Dependencies() — обязательные control edges
        // job.InputArtifacts() — артефакты для восстановления из producer jobs
        // job.Operation() — typed payload; для shell-driven CI используйте cishell.RenderOperation
    }

    return renderYAML(g.ir), nil
}
```

## Реализация

```go
package bitbucket

import (
    "os"

    "github.com/edelwud/terraci/pkg/pipeline"
    "github.com/edelwud/terraci/pkg/plugin"
    "github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
    registry.RegisterFactory(func() plugin.Plugin {
        return &Plugin{
            BasePlugin: plugin.BasePlugin[*Config]{
                PluginName: "bitbucket",
                PluginDesc: "Bitbucket Pipelines generation",
                EnableMode: plugin.EnabledWhenConfigured,
                DefaultCfg: func() *Config { return &Config{} },
            },
        }
    })
}

type Plugin struct{ plugin.BasePlugin[*Config] }

type Config struct {
    Image    string `yaml:"image"`
    PlanOnly bool   `yaml:"plan_only"`
}

func (c *Config) Clone() *Config {
    if c == nil {
        return nil
    }
    out := *c
    return &out
}

// EnvDetector
func (p *Plugin) DetectEnv() bool {
    return os.Getenv("BITBUCKET_PIPELINE_UUID") != ""
}

// CIInfoProvider
func (p *Plugin) ProviderName() string { return "bitbucket" }
func (p *Plugin) PipelineID() string   { return os.Getenv("BITBUCKET_BUILD_NUMBER") }
func (p *Plugin) CommitSHA() string    { return os.Getenv("BITBUCKET_COMMIT") }

// PipelineGeneratorFactory
func (p *Plugin) PipelineRequirements(ctx *plugin.AppContext) pipeline.BuildRequirements {
    return pipeline.BuildRequirements{}
}

func (p *Plugin) NewGenerator(ctx *plugin.AppContext, ir *pipeline.IR) pipeline.Generator {
    return &generator{config: p.Config(), ir: ir}
}

type generator struct {
    config *Config
    ir     *pipeline.IR
}

func (g *generator) Generate() (pipeline.GeneratedPipeline, error) {
    return renderBitbucketYAML(g.ir, g.config), nil
}

func (g *generator) DryRun() (*pipeline.DryRunResult, error) {
    return g.ir.DryRun(countModules(g.ir)), nil
}
```

## Резолвинг провайдера

TerraCi определяет активный CI-провайдер в порядке:

1. **`TERRACI_PROVIDER`** — явное указание
2. **Автоопределение** — `DetectEnv()` возвращает `true`
3. **Единственный активный провайдер** — активен только один провайдер

Ваш провайдер обнаруживается автоматически. Изменения в core не нужны.

## См. также

- [Шаг пайплайна](/ru/plugins/pipeline-plugin) — внедрение шагов без полного провайдера
- [Генерация пайплайнов](/ru/guide/pipeline-generation) — как работает IR
