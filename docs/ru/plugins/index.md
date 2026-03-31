---
title: Плагины
description: "Расширение TerraCi кастомными плагинами: CLI-команды, шаги пайплайна, CI-провайдеры и формы настройки"
outline: deep
---

# Плагины

TerraCi построен как plugin-first система. Каждая функция — генерация пайплайнов, оценка стоимости, проверка политик, MR-комментарии — это плагин. Вы можете добавлять свои плагины для интеграции с любыми инструментами и сервисами.

## Что могут плагины?

| Тип плагина | Что добавляет | Примеры |
|-------------|--------------|---------|
| [CLI-команда](/ru/plugins/command-plugin) | Новая субкоманда `terraci <command>` | Slack-уведомления, отчёты, аудит |
| [Шаг пайплайна](/ru/plugins/pipeline-plugin) | Джобы/шаги в генерируемые CI-пайплайны | Сканирование безопасности, гейты согласования |
| [CI-провайдер](/ru/plugins/provider-plugin) | Поддержка новой CI-системы | Bitbucket Pipelines, Jenkins, CircleCI |
| [Поле в мастере](/ru/plugins/init-plugin) | Поля конфигурации в `terraci init` TUI | Настройки плагина, корпоративные дефолты |

## Быстрый старт

Соберите кастомный TerraCi с вашим плагином за 3 шага:

```bash
# 1. Напишите плагин (см. гайды ниже)

# 2. Соберите бинарник
xterraci build --with github.com/your-org/terraci-plugin-slack

# 3. Используйте
./terraci slack --channel #deploys
```

## Архитектура

```
                    ┌──────────────────────────┐
                    │      Ядро TerraCi         │
                    │                           │
                    │  discovery → parser →     │
                    │  graph → pipeline IR      │
                    └──────────┬───────────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
        ┌─────┴─────┐   ┌─────┴─────┐   ┌─────┴─────┐
        │ Встроенные │   │ Встроенные │   │   Ваш     │
        │  плагины   │   │  плагины   │   │  плагин   │
        │            │   │            │   │           │
        │  gitlab    │   │  cost      │   │  slack    │
        │  github    │   │  policy    │   │  jira     │
        │  git       │   │  summary   │   │  vault    │
        │            │   │  update    │   │  ...      │
        └────────────┘   └────────────┘   └───────────┘
```

Плагины компилируются в бинарник. Нет рантайм-загрузки — нулевой overhead и полная типобезопасность.

## Гайды

### [CLI-команда](/ru/plugins/command-plugin)
Добавьте новую `terraci <command>`. Самый распространённый тип — идеален для уведомлений, отчётов, интеграций.

### [Шаг пайплайна](/ru/plugins/pipeline-plugin)
Внедрите кастомные джобы или шаги в генерируемые CI-пайплайны. Для сканирования безопасности, гейтов согласования или post-deploy хуков.

### [CI-провайдер](/ru/plugins/provider-plugin)
Добавьте поддержку новой CI-системы. Реализуйте генерацию пайплайна, определение окружения и MR/PR-комментарии.

### [Поле в мастере](/ru/plugins/init-plugin)
Добавьте поля конфигурации в интерактивный мастер `terraci init`. Пользователи настроят ваш плагин через TUI-форму.

## Встроенные плагины

| Плагин | Возможности | Конфигурация |
|--------|------------|-------------|
| **git** | ChangeDetection, Preflight | Всегда активен |
| **gitlab** | Generator, EnvDetector, Comments, Preflight, Init | [config/gitlab](/ru/config/gitlab) |
| **github** | Generator, EnvDetector, Comments, Preflight, Init | [config/github](/ru/config/github) |
| **summary** | Command, Pipeline, Init | Включён по умолчанию |
| **cost** | Command, Pipeline, Runtime, Preflight, Init | [config/cost](/ru/config/cost) |
| **policy** | Command, Pipeline, Runtime, Preflight, Version, Init | [config/policy](/ru/config/policy) |
| **update** | Command, Runtime, Preflight, Init | [config/update](/ru/config/update) |

## Основы

### Регистрация

Каждый плагин регистрируется в `init()`:

```go
package myplugin

import (
    "github.com/edelwud/terraci/pkg/plugin"
    "github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
    registry.Register(&Plugin{
        BasePlugin: plugin.BasePlugin[*Config]{
            PluginName: "myplugin",
            PluginDesc: "Что делает мой плагин",
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
    Enabled bool `yaml:"enabled"`
}
```

Пользователи настраивают плагин в `.terraci.yaml`:

```yaml
plugins:
  myplugin:
    enabled: true
```

### Политики активации

| Политика | Поведение | Используют |
|----------|----------|-----------|
| `EnabledAlways` | Всегда активен, конфиг не нужен | git |
| `EnabledWhenConfigured` | Активен при наличии секции конфига | gitlab, github |
| `EnabledByDefault` | Активен пока не `enabled: false` | summary |
| `EnabledExplicitly` | Требует явного opt-in | cost, policy, update |

### Жизненный цикл

```
Register → Configure → Preflight → Freeze → Execute
```

### AppContext

Каждая возможность получает `*plugin.AppContext`:

```go
ctx.WorkDir()    // корневая директория проекта
ctx.ServiceDir() // абсолютный путь к .terraci
ctx.Config()     // полная конфигурация TerraCi (defensive copy)
ctx.Version()    // строка версии TerraCi
ctx.Reports()    // реестр отчётов для обмена между плагинами
```

### Сборка

```bash
# Из опубликованного модуля
xterraci build --with github.com/your-org/terraci-plugin-slack

# Из локальной директории при разработке
xterraci build --with github.com/your-org/plugin=./my-plugin

# Исключить ненужные встроенные плагины
xterraci build --without cost --without policy
```

## См. также

- [examples/external-plugin](https://github.com/edelwud/terraci/tree/main/examples/external-plugin) — рабочий пример
- [Система плагинов](/ru/guide/plugins) — архитектурный обзор
