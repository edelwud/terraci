---
title: "Плагин шагов пайплайна"
description: "Внедрение кастомных джобов и шагов в генерируемые CI-пайплайны"
outline: deep
---

# Плагин шагов пайплайна

Внедряйте кастомные джобы или шаги в генерируемый CI-пайплайн. Вклад вашего плагина объединяется с pipeline IR и появляется в финальном GitLab CI / GitHub Actions YAML.

## Сценарии использования

- **Сканирование безопасности** — tfsec, checkov, Snyk после plan
- **Гейты согласования** — внешнее подтверждение перед apply
- **Post-deploy хуки** — smoke-тесты после apply
- **Уведомления** — сообщения на определённых фазах
- **Итоговые отчёты** — агрегация результатов после завершения всех джобов

## Фазы пайплайна

```
PrePlan → Plan → PostPlan → PreApply → Apply → PostApply → Finalize
```

| Фаза | Константа | Когда | Типичное использование |
|------|-----------|-------|----------------------|
| Pre-Plan | `pipeline.PhasePrePlan` | До `terraform plan` | Настройка, авторизация |
| Post-Plan | `pipeline.PhasePostPlan` | После `terraform plan` | Сканирование, проверки, стоимость |
| Pre-Apply | `pipeline.PhasePreApply` | До `terraform apply` | Проверки согласования |
| Post-Apply | `pipeline.PhasePostApply` | После `terraform apply` | Smoke-тесты, уведомления |
| Finalize | `pipeline.PhaseFinalize` | После всех остальных джобов | Summary-комментарии, итоговые отчёты |

::: tip Фаза Finalize
Джобы в `PhaseFinalize` автоматически зависят от всех остальных contributed-джобов. Они всегда выполняются последними. Встроенный плагин `summary` использует эту фазу для публикации MR/PR-комментариев после завершения всех plan и policy джобов.
:::

## Добавление шагов

Шаги внедряются в каждый plan/apply джоб. Каждый шаг содержит одну команду:

```go
import "github.com/edelwud/terraci/pkg/pipeline"

func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
    cfg := p.Config()
    if cfg == nil || !cfg.Pipeline {
        return nil
    }

    return &pipeline.Contribution{
        Steps: []pipeline.Step{
            {
                Phase:   pipeline.PhasePostPlan,
                Name:    "tfsec",
                Command: "tfsec --format json --out tfsec-report.json .",
            },
        },
    }
}
```

### Фильтрация шагов по фазе

Фреймворк фильтрует шаги при формировании джобов:
- **Plan-джобы** получают шаги `PhasePrePlan` + `PhasePostPlan`
- **Apply-джобы** получают шаги `PhasePreApply` + `PhasePostApply`
- Шаги `PhaseFinalize` **не внедряются** в модульные джобы — используйте contributed-джобы

## Добавление джобов

Отдельные джобы выполняются как самостоятельные CI-джобы:

```go
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
    return &pipeline.Contribution{
        Jobs: []pipeline.ContributedJob{
            {
                Name:          "security-scan",
                Phase:         pipeline.PhasePostPlan,
                DependsOnPlan: true,
                Commands: []string{
                    "checkov -d . --output json > checkov-report.json",
                },
            },
        },
    }
}
```

### Поля ContributedJob

| Поле | Тип | Описание |
|------|-----|---------|
| `Name` | `string` | Имя джоба в генерируемом пайплайне |
| `Phase` | `Phase` | Определяет имя стадии (`Phase.String()`) |
| `Commands` | `[]string` | Shell-команды для выполнения |
| `DependsOnPlan` | `bool` | Если `true` — зависит от всех plan-джобов |
| `ArtifactPaths` | `[]string` | Пути для сбора CI-артефактов |
| `AllowFailure` | `bool` | Если `true` — провал джоба не фейлит пайплайн |

### Finalize-джобы

Для джобов, которые должны выполняться после всего остального:

```go
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
    return &pipeline.Contribution{
        Jobs: []pipeline.ContributedJob{
            {
                Name:          "my-summary",
                Phase:         pipeline.PhaseFinalize,
                DependsOnPlan: true,
                Commands:      []string{"terraci my-summary"},
            },
        },
    }
}
```

`PhaseFinalize`-джобы автоматически зависят от всех других contributed-джобов (cost, policy и т.д.). Указывать эти зависимости вручную не нужно.

## Результат в генерируемом YAML

**GitLab CI:**
```yaml
plan-vpc:
  stage: deploy-plan-0
  script:
    - cd platform/prod/eu-central-1/vpc
    - terraform init
    - terraform plan -out=plan.tfplan
    - tfsec --format json --out tfsec-report.json .  # ← PostPlan шаг

security-scan:
  stage: post-plan
  needs: [plan-vpc, plan-eks, plan-rds]               # ← DependsOnPlan: true
  script:
    - checkov -d . --output json > checkov-report.json
```

## Полный пример

```go
package security

import (
    "github.com/edelwud/terraci/pkg/pipeline"
    "github.com/edelwud/terraci/pkg/plugin"
    "github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
    registry.RegisterFactory(func() plugin.Plugin {
        return &Plugin{
            BasePlugin: plugin.BasePlugin[*Config]{
                PluginName:  "security",
                PluginDesc:  "Security scanning for Terraform plans",
                EnableMode:  plugin.EnabledExplicitly,
                DefaultCfg:  func() *Config { return &Config{} },
                IsEnabledFn: func(cfg *Config) bool { return cfg != nil && cfg.Enabled },
            },
        }
    })
}

type Plugin struct{ plugin.BasePlugin[*Config] }

type Config struct {
    Enabled  bool   `yaml:"enabled"`
    Pipeline bool   `yaml:"pipeline"`
    Tool     string `yaml:"tool"`
}

func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
    cfg := p.Config()
    if cfg == nil || !cfg.Pipeline {
        return nil
    }
    tool := cfg.Tool
    if tool == "" {
        tool = "tfsec"
    }
    return &pipeline.Contribution{
        Steps: []pipeline.Step{{
            Phase:   pipeline.PhasePostPlan,
            Command: tool + " --format json .",
        }},
    }
}
```

## См. также

- [CLI-команда](/ru/plugins/command-plugin) — добавление CLI-команд
- [Генерация пайплайнов](/ru/guide/pipeline-generation) — как работает pipeline IR
