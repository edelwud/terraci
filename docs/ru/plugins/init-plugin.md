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
                    Key:     "slack.channel",
                    Title:   "Channel",
                    Type:    initwiz.FieldString,
                    Default: "#terraform-deploys",
                },
                {
                    Key:     "slack.on_failure",
                    Title:   "Notify on failure",
                    Type:    initwiz.FieldSelect,
                    Default: "always",
                    Options: []initwiz.InitOption{
                        {Label: "Always", Value: "always"},
                        {Label: "Only on failure", Value: "failure"},
                    },
                },
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
    if !state.Bool("slack.enabled") {
        return nil, nil
    }
    return initwiz.NewInitContribution("slack", SlackConfig{
        Enabled: true,
        Channel: state.String("slack.channel"),
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

```go
state.String("key")     // строка или ""
state.Bool("key")       // bool или false
state.Get("key")        // any или nil
state.Provider()        // state.String("provider")
state.Binary()          // state.String("binary")
```

## См. также

- [CLI-команда](/ru/plugins/command-plugin) — добавление CLI-команд
- [terraci init](/ru/cli/init) — использование команды init
