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
| `CIMetadata` | Имя провайдера, ID пайплайна, SHA коммита |
| `GeneratorFactory` | Создание генератора: pipeline IR → YAML |

Опционально:

| Интерфейс | Назначение |
|-----------|-----------|
| `CommentFactory` | Сервис MR/PR комментариев |
| `FlagOverridable` | Поддержка `--plan-only` и `--auto-approve` |

## Работа с Pipeline IR

IR содержит уровни выполнения с модульными джобами и contributed-джобами плагинов:

```go
func (g *generator) Generate(ir *pipeline.IR) (*pipeline.GeneratedPipeline, error) {
    // ir.Levels — уровни параллельного выполнения модулей
    for _, level := range ir.Levels {
        for _, mj := range level.Modules {
            // mj.Module.Path — "platform/prod/eu-central-1/vpc"
            // mj.Plan — *Job (nil если plan отключён)
            // mj.Apply — *Job (nil в plan-only режиме)
        }
    }

    // ir.Jobs — contributed-джобы от плагинов (cost, policy, summary)
    for _, job := range ir.Jobs {
        // job.Name, job.Phase, job.Dependencies, job.Script
    }

    content := renderYAML(ir)
    return &pipeline.GeneratedPipeline{Content: content}, nil
}
```

## Flag Overrides (опционально)

Реализуйте `FlagOverridable` для поддержки `--plan-only` и `--auto-approve`:

```go
func (p *Plugin) SetPlanOnly(v bool) {
    if cfg := p.Config(); cfg != nil {
        cfg.PlanOnly = v
        if v { cfg.PlanEnabled = true }
    }
}

func (p *Plugin) SetAutoApprove(v bool) {
    if cfg := p.Config(); cfg != nil {
        cfg.AutoApprove = v
    }
}
```

## Реализация

```go
package bitbucket

import (
    "os"

    "github.com/edelwud/terraci/pkg/discovery"
    "github.com/edelwud/terraci/pkg/graph"
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
    Image           string `yaml:"image"`
    TerraformBinary string `yaml:"terraform_binary"`
    PlanEnabled     bool   `yaml:"plan_enabled"`
}

// EnvDetector
func (p *Plugin) DetectEnv() bool {
    return os.Getenv("BITBUCKET_PIPELINE_UUID") != ""
}

// CIMetadata
func (p *Plugin) ProviderName() string { return "bitbucket" }
func (p *Plugin) PipelineID() string   { return os.Getenv("BITBUCKET_BUILD_NUMBER") }
func (p *Plugin) CommitSHA() string    { return os.Getenv("BITBUCKET_COMMIT") }

// GeneratorFactory
func (p *Plugin) NewGenerator(
    ctx *plugin.AppContext,
    depGraph *graph.DependencyGraph,
    modules []*discovery.Module,
) pipeline.Generator {
    return &generator{config: p.Config()}
}

type generator struct{ config *Config }

func (g *generator) Generate(ir *pipeline.IR) (*pipeline.GeneratedPipeline, error) {
    content := renderBitbucketYAML(ir, g.config)
    return &pipeline.GeneratedPipeline{Content: content}, nil
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
