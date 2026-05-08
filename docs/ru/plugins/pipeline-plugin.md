---
title: "Pipeline Job Plugin"
description: "Добавление standalone DAG-джобов в CI пайплайны: policy, cost, summary, update checks"
outline: deep
---

# Pipeline Job Plugin

Pipeline-плагины добавляют standalone DAG-джобы в provider-agnostic IR.
Джобы объявляют, какие typed resources они читают и пишут; TerraCi сам выводит
зависимости, восстановление артефактов и необходимость детального plan.

## Сценарии

- **Policy checks** — читают `plan.json`, публикуют policy report
- **Cost estimation** — читают `plan.json`, публикуют cost report
- **Dependency updates** — публикуют result/report артефакты
- **Summary** — читает `plan.json` и optional plugin reports, затем публикует MR/PR комментарий

## Как Это Работает

Фаз и injected plan/apply steps больше нет. Пайплайн строится как DAG:

1. Core создает Terraform plan/apply jobs для target modules.
2. Плагины добавляют standalone jobs.
3. Каждая job объявляет `Consumes` и `Produces`.
4. `pipeline.Build` резолвит concrete resources, artifacts и dependencies.
5. Providers только рендерят готовый IR.

## Пример Job

```go
func (p *Plugin) PipelineContribution(ctx *plugin.AppContext) *pipeline.Contribution {
    serviceDir := ctx.Config().ServiceDir

    return &pipeline.Contribution{
        Jobs: []pipeline.ContributedJob{
            {
                Name:     "security-scan",
                Commands: []string{"terraci security check"},
                Consumes: []pipeline.ResourceRequest{
                    pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
                },
                Produces: []pipeline.ResourceSpec{
                    pipeline.PluginResource(
                        pipeline.ResourceKindPluginReport,
                        "security",
                        pipeline.WorkspacePath(serviceDir, ci.ReportFilename("security")),
                    ),
                },
            },
        },
    }
}
```

## Поля ContributedJob

| Поле | Тип | Описание |
|------|-----|----------|
| `Name` | `string` | Имя job в generated pipeline |
| `Commands` | `[]string` | Shell-команды |
| `Dependencies` | `[]JobDependency` | Явные control dependencies, если нужны |
| `Consumes` | `[]ResourceRequest` | Typed resources, которые надо восстановить |
| `Produces` | `[]ResourceSpec` | Typed resources, которые job публикует |
| `AllowFailure` | `bool` | Ошибка job не валит pipeline |

## Resources

```go
pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON)
pipeline.ModulePlanResource(pipeline.ResourceKindPlanText, "platform/prod/eu-central-1/vpc")

pipeline.AllPluginResources(pipeline.ResourceKindPluginReport, true)
pipeline.PluginProducerResource(pipeline.ResourceKindPluginResult, "policy", false)
```

Запрос `PlanJSON` или `PlanText` автоматически включает detailed plan только
для подходящих модулей.

## Summary Job

Summary не требует специальной фазы. Она оказывается в конце DAG, потому что
читает ресурсы, которые производят предыдущие jobs:

```go
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
    return &pipeline.Contribution{
        Jobs: []pipeline.ContributedJob{
            {
                Name:     "terraci-summary",
                Commands: []string{"terraci summary"},
                Consumes: []pipeline.ResourceRequest{
                    pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
                    pipeline.AllPluginResources(pipeline.ResourceKindPluginReport, true),
                },
            },
        },
    }
}
```

## Activation

Registry вызывает `PipelineContribution` только для enabled plugin configs. Если
есть отдельный `pipeline` toggle, реализуйте `plugin.PipelineContributionGate`:

```go
func (p *Plugin) PipelineContributionEnabled(_ *plugin.AppContext) bool {
    cfg := p.Config()
    return cfg != nil && cfg.Pipeline
}
```

## См. Также

- [CI Provider Plugin](/ru/plugins/provider-plugin) — рендер IR для нового CI
- [Генерация Pipeline](/ru/guide/pipeline-generation) — как строятся generated pipelines
