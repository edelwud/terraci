---
title: Система плагинов
description: "Встроенные и кастомные плагины: активация, конфигурация и расширяемость через xterraci"
outline: deep
---

# Система плагинов

TerraCi использует систему плагинов на этапе компиляции по аналогии с `database/sql` в Go. Плагины регистрируются через `init()` и blank-импорты при сборке.

## Встроенные плагины

| Плагин | Назначение | Нужна конфигурация |
|--------|-----------|-------------------|
| `git` | Определение изменённых модулей через `git diff` | Нет |
| `gitlab` | Генерация GitLab CI пайплайнов и MR-комментарии | Да (наличие секции активирует) |
| `github` | Генерация GitHub Actions воркфлоу и PR-комментарии | Да (наличие секции активирует) |
| `summary` | Публикация сводных комментариев в MR/PR | Нет (включён по умолчанию) |
| `cost` | Оценка стоимости облачных ресурсов (AWS) | Да (`providers.aws.enabled: true`) |
| `policy` | Проверка политик через встроенный OPA | Да (`enabled: true`) |
| `tfupdate` | Разрешение зависимостей Terraform и синхронизация lock-файлов | Да (`enabled: true`) |

## Политики активации

Каждый плагин имеет политику активации, определяющую когда он участвует в текущем запуске.

### Всегда активен

Плагин **git** не требует конфигурации. Он обеспечивает определение изменённых модулей для режима `--changed-only` и доступен всегда.

### Активируется при наличии конфига

Плагины **gitlab** и **github** активируются при наличии соответствующей секции в `extensions:`. Удаление секции отключает плагин:

```yaml
extensions:
  gitlab:      # наличие этой секции включает плагин
    image: { name: hashicorp/terraform:1.6 }
```

### Активен по умолчанию

Плагин **summary** включён по умолчанию. Он публикует сводку планов в MR/PR. Можно отключить явно:

```yaml
extensions:
  summary:
    enabled: false   # отключить
```

### Требует явного включения

Плагины **cost**, **policy** и **tfupdate** требуют явной активации:

```yaml
extensions:
  cost:
    providers:
      aws:
        enabled: true

  policy:
    enabled: true
    sources:
      - path: policies

  tfupdate:
    enabled: true
    policy:
      bump: minor
```

## Определение CI-провайдера

TerraCi автоматически определяет активный CI-провайдер:

1. **`TERRACI_PROVIDER`** -- явное указание:
   ```bash
   TERRACI_PROVIDER=gitlab terraci generate -o pipeline.yml
   ```
2. **Переменные окружения** -- `GITLAB_CI=true` выбирает GitLab, `GITHUB_ACTIONS=true` выбирает GitHub
3. **Единственный активный провайдер** -- если активен только один CI-провайдер, он используется автоматически

Если настроено несколько провайдеров и окружение не определено, TerraCi возвращает ошибку с рекомендацией задать `TERRACI_PROVIDER`.

## Возможности плагинов

Плагины реализуют один или несколько интерфейсов-возможностей. Фреймворк обнаруживает их через type assertion:

| Возможность | Назначение | Плагины |
|------------|---------|---------|
| `CommandProvider` | CLI-субкоманды (`terraci cost`, `terraci local-exec` и т.д.) | cost, policy, summary, tfupdate, localexec |
| `PipelineContributor` | Добавление шагов/джобов в pipeline IR | cost, policy, summary, tfupdate |
| `InitContributor` | Поля формы для `terraci init` | gitlab, github, cost, policy, summary, tfupdate |
| `PipelineGeneratorFactory` | Создание генератора пайплайна (`NewGenerator(ctx, *pipeline.IR)`) | gitlab, github |
| `CommentServiceFactory` | Создание сервиса MR/PR комментариев | gitlab, github |
| `EnvDetector` | Определение CI-окружения по переменным среды | gitlab, github |
| `CIInfoProvider` | Имя провайдера, ID пайплайна, SHA коммита | gitlab, github |
| `ChangeDetectionProvider` | Определение изменённых модулей через VCS | git |
| `RuntimeProvider` | Ленивая инициализация тяжёлых зависимостей | cost, policy, tfupdate |
| `Preflightable` | Дешёвая валидация при старте | gitlab, github, git, cost, policy, tfupdate |
| `VersionProvider` | Информация о версии для `terraci version` | policy |
| `KVCacheProvider` | KV-кэш бэкенд по имени | inmemcache |
| `BlobStoreProvider` | Бэкенд blob/object store (`NewBlobStore(ctx, appCtx, opts)`) | diskblob |
| `FlagOverridable` | Прямые CLI-override-ы (`--plan-only`, `--auto-approve`) | gitlab, github |

Один плагин может реализовывать несколько возможностей. Например, `cost` реализует `CommandProvider` (команда `terraci cost`), `PipelineContributor` (шаг оценки стоимости в пайплайне), `InitContributor` (переключатель в мастере init), `RuntimeProvider` (ленивая инициализация estimator) и `Preflightable` (валидация конфига).

## Жизненный цикл плагина

Каждый плагин проходит одинаковый жизненный цикл:

```
1. Register    -- init() регистрирует плагин через registry.RegisterFactory()
2. Configure   -- фреймворк декодирует секцию extensions.<key> из YAML
3. Preflight   -- дешёвая валидация (определение окружения, проверка конфига)
4. Freeze      -- AppContext замораживается, мутации запрещены
5. Execute     -- команды лениво создают RuntimeProvider рантаймы по необходимости
```

**Preflight** выполняется для всех включённых плагинов до любой команды. Он должен быть быстрым и без побочных эффектов — никаких сетевых вызовов и тяжёлого state. Тяжёлая работа (API-клиенты, кеши, estimators) принадлежит `RuntimeProvider`, который создаёт рантайм лениво, когда команда реально его требует.

## Кастомные плагины

### Сборка через xterraci

`xterraci` создаёт кастомный бинарник TerraCi с дополнительными или исключёнными плагинами:

```bash
# Добавить внешний плагин
xterraci build --with github.com/your-org/terraci-plugin-slack

# Закрепить конкретную версию
xterraci build --with github.com/your-org/terraci-plugin-slack@v1.2.0

# Использовать локальный плагин во время разработки
xterraci build --with github.com/your-org/plugin=../my-plugin

# Убрать ненужные встроенные плагины
xterraci build --without cost --without policy

# Комбинировать
xterraci build \
  --with github.com/your-org/terraci-plugin-slack \
  --without cost \
  --output ./build/terraci-custom
```

### Как это работает

`xterraci build`:
1. Создаёт временный Go-модуль
2. Генерирует `main.go` с blank-импортами выбранных плагинов
3. Выполняет `go get` для каждого внешнего плагина
4. Собирает бинарник через `go build`

Результат идентичен стандартному `terraci`, но с другим набором скомпилированных плагинов.

### Список встроенных плагинов

```bash
xterraci list-plugins
```

### Написание плагина

Минимальный внешний плагин:

**1. Регистрация** -- функция `init()`, вызывающая `registry.RegisterFactory()`:

```go
package myplugin

import (
    "github.com/edelwud/terraci/pkg/plugin"
    "github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
    registry.RegisterFactory(func() plugin.Plugin {
        return &Plugin{
            BasePlugin: plugin.BasePlugin[*Config]{
                PluginName: "myplugin",
                PluginDesc: "My custom plugin",
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
    Enabled bool   `yaml:"enabled"`
    APIKey  string `yaml:"api_key"`
}
```

**2. Возможности** -- реализуйте нужные интерфейсы:

```go
// CommandProvider -- добавляет команду `terraci myplugin`
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
    return []*cobra.Command{{
        Use:   "myplugin",
        Short: "Run my custom plugin",
        RunE: func(cmd *cobra.Command, _ []string) error {
            // логика плагина
            return nil
        },
    }}
}
```

**3. Go-модуль** -- опубликуйте как Go-модуль с `go.mod`, зависящим от `github.com/edelwud/terraci`.

### Конвенция файлов плагина

Для крупных плагинов используйте конвенцию «один файл на возможность»:

| Файл | Содержимое |
|------|-----------|
| `plugin.go` | `init()`, struct плагина, BasePlugin embedding |
| `lifecycle.go` | Реализация `Preflightable` |
| `commands.go` | `CommandProvider` с cobra-командами |
| `runtime.go` | `RuntimeProvider` для ленивого тяжёлого state |
| `usecases.go` | Оркестрация команд над типизированным рантаймом |
| `pipeline.go` | `PipelineContributor` — шаги/джобы |
| `init_wizard.go` | `InitContributor` — поля формы |
| `output.go` | Хелперы рендеринга CLI |
| `report.go` | Сборка CI-отчётов |

### Рабочий пример

Смотрите [examples/external-plugin](https://github.com/edelwud/terraci/tree/main/examples/external-plugin) для полного рабочего примера плагина, добавляющего команду `terraci hello`.

## Справочник конфигурации

| Плагин | Страница конфигурации |
|--------|----------------------|
| GitLab CI | [config/gitlab](/ru/config/gitlab) |
| GitLab MR | [config/gitlab-mr](/ru/config/gitlab-mr) |
| GitHub Actions | [config/github](/ru/config/github) |
| Оценка стоимости | [config/cost](/ru/config/cost) |
| Проверка политик | [config/policy](/ru/config/policy) |
| Обновление зависимостей | [config/tfupdate](/ru/config/tfupdate) |
