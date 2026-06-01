---
title: "Плагин мастера настройки"
description: "Добавление полей конфигурации в интерактивный мастер terraci init"
outline: deep
---

# Плагин мастера настройки

Добавляет поля конфигурации вашего плагина в интерактивный TUI-мастер `terraci init`. Пользователи настраивают плагин через форму вместо ручного редактирования YAML.

## Реализация

Реализуйте `InitContributor` из `pkg/plugin/initwiz`:

```go
import "github.com/edelwud/terraci/pkg/plugin/initwiz"

var (
    slackEnabledKey   = initwiz.MustStateKey[bool]("slack.enabled")
    slackChannelKey   = initwiz.MustStateKey[string]("slack.channel")
    slackOnFailureKey = initwiz.MustStateKey[string]("slack.on_failure")
)

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
                    Key:     slackChannelKey,
                    Title:   "Channel",
                    Default: "#terraform-deploys",
                }),
                initwiz.NewSelectField(initwiz.SelectFieldOptions{
                    Key:     slackOnFailureKey,
                    Title:   "Notify on failure",
                    Default: "always",
                    Options: []initwiz.InitOption{
                        {Label: "Always", Value: "always"},
                        {Label: "Only on failure", Value: "failure"},
                    },
                }),
            },
        },
    }
}

type SlackConfig struct {
    Enabled bool   `yaml:"enabled,omitempty"`
    Channel string `yaml:"channel,omitempty"`
}

func (c SlackConfig) Clone() SlackConfig { return c }

func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) (*initwiz.InitContribution, error) {
    if !slackEnabledKey.Get(state) {
        return nil, nil
    }
    return initwiz.NewInitContribution("slack", SlackConfig{
        Enabled: true,
        Channel: slackChannelKey.Get(state),
    })
}
```

Контракт типизирован от начала до конца: плагин собирает typed config struct,
`initwiz.NewInitContribution` кодирует его в валидированное extension value, а
`cmd/terraci/internal/initflow` собирает итоговый файл. Командный слой только
рендерит TUI, preview и записывает файл. Чтобы пропустить опциональную секцию,
верните `nil, nil`; если состояние wizard невалидно, верните ошибку.

## Категории форм

| Категория | Отображение | Для чего |
|-----------|------------|---------|
| `CategoryProvider` | Отдельная группа с ShowWhen | CI-настройки (image, runner) |
| `CategoryPipeline` | Объединяется в группу "Pipeline" | Поведение пайплайна (plan_enabled) |
| `CategoryFeature` | Объединяется в группу "Features" | Переключатели фич |
| `CategoryDetail` | Отдельная группа с ShowWhen | Детальные настройки включённых фич |

## Типы полей

| Тип | Виджет | Значение |
|-----|--------|---------|
| `FieldString` | Текстовый ввод | `string` |
| `FieldBool` | Переключатель | `bool` |
| `FieldSelect` | Выпадающий список | `string` |

## Порядок

`Order` управляет позицией группы. Встроенные плагины: 100-202. Используйте `300+` для кастомных.

## StateMap

`StateMap` остается mutable состоянием формы, но плагин работает с ним через
typed `StateKey[T]`, объявленные один раз на уровне пакета:

```go
var channelKey = initwiz.MustStateKey[string]("slack.channel")
var enabledKey = initwiz.MustStateKey[bool]("slack.enabled")

channel := channelKey.Get(state)
enabled, explicitlySet := enabledKey.Lookup(state)
enabledKey.Set(state, true)
```

## См. также

- [CLI-команда](/ru/plugins/command-plugin) — добавление CLI-команд
- [terraci init](/ru/cli/init) — использование команды init
