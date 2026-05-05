---
title: "Как это работает"
description: "Архитектура: обнаружение модулей, граф зависимостей, топологическая сортировка и генерация пайплайнов"
outline: deep
---

# Как это работает

Это руководство объясняет внутреннюю архитектуру и поток данных TerraCi.

## Обзор

TerraCi обрабатывает ваш Terraform проект в четыре этапа:

```mermaid
flowchart LR
  A["🔍 Обнаружение"] --> B["📄 Парсинг"] --> C["🔗 Граф"] --> D["⚙️ Генерация"]
```

## Этап 1: Обнаружение модулей

TerraCi сканирует структуру директорий для поиска Terraform модулей.

### Как это работает

1. Обход дерева директорий от корня проекта
2. Поиск директорий на настроенной глубине, содержащих `.tf` файлы
3. Парсинг пути по именованным сегментам на основе настроенного паттерна

Паттерн настраивается (например, `{service}/{environment}/{region}/{module}`), и имена сегментов определяют ключи в `components` карте модуля.

### Пример

```
platform/stage/eu-central-1/vpc/main.tf
   │       │         │       │
   │       │         │       └── сегмент "module": vpc
   │       │         └── сегмент "region": eu-central-1
   │       └── сегмент "environment": stage
   └── сегмент "service": platform
```

### ID модуля

Каждый модуль получает уникальный ID: `platform/stage/eu-central-1/vpc`

Этот ID используется для:
- Сопоставления зависимостей
- Именования джобов
- Разрешения пути к state-файлу

## Этап 2: Парсинг HCL

TerraCi парсит `.tf` файлы каждого модуля для извлечения зависимостей.

### Что парсится

1. **Блоки `terraform_remote_state`** - основной источник зависимостей
2. **Блоки `locals`** - разрешение переменных для динамических путей

### Пример Remote State

```hcl
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "my-state"
    key    = "platform/stage/eu-central-1/vpc/terraform.tfstate"
  }
}
```

TerraCi извлекает:
- Тип backend: `s3`
- Путь к state: `platform/stage/eu-central-1/vpc/terraform.tfstate`
- Разрешённый модуль: `platform/stage/eu-central-1/vpc`

### Разрешение путей

TerraCi разрешает переменные в путях state:

```hcl
locals {
  env = "stage"
}

data "terraform_remote_state" "vpc" {
  config = {
    key = "platform/${local.env}/eu-central-1/vpc/terraform.tfstate"
  }
}
```

Становится: `platform/stage/eu-central-1/vpc/terraform.tfstate`

### Обработка for_each

При наличии `for_each` TerraCi раскрывает в несколько зависимостей:

```hcl
data "terraform_remote_state" "services" {
  for_each = toset(["auth", "api", "web"])
  config = {
    key = "platform/stage/eu-central-1/${each.key}/terraform.tfstate"
  }
}
```

Создаёт зависимости от модулей: `auth`, `api`, `web`.

## Этап 3: Построение графа

TerraCi строит направленный ациклический граф (DAG) зависимостей модулей.

### Алгоритм

1. Создание узла для каждого обнаруженного модуля
2. Добавление рёбер от каждого модуля к его зависимостям
3. Обнаружение циклов (ошибка если найдены)
4. Топологическая сортировка алгоритмом Кана

### Топологическая сортировка

Алгоритм Кана гарантирует порядок где зависимости идут первыми:

```mermaid
flowchart TD
  vpc --> eks
  vpc --> rds
  eks --> app
  rds --> app
```

### Уровни выполнения

Модули группируются по уровням для параллельного выполнения:

| Уровень | Модули | Параллельно |
|---------|--------|-------------|
| 0 | vpc | Да (нет зависимостей) |
| 1 | eks, rds | Да (одинаковые зависимости) |
| 2 | app | После уровня 1 |

### Обнаружение циклов

TerraCi обнаруживает циклические зависимости:

```mermaid
flowchart LR
  vpc --> eks --> app --> vpc
  style vpc fill:#fef2f2,stroke:#ef4444,color:#991b1b
  style eks fill:#fef2f2,stroke:#ef4444,color:#991b1b
  style app fill:#fef2f2,stroke:#ef4444,color:#991b1b
  linkStyle default stroke:#ef4444,stroke-width:2px
```

Сообщение об ошибке:
```
Error: circular dependency detected
  vpc -> eks -> app -> vpc
```

## Этап 4: Генерация пайплайна

TerraCi генерирует конфигурацию CI пайплайна из отсортированного графа модулей. Провайдер выбирается через `TERRACI_PROVIDER`, автоматически определяется из окружения (переменная `GITLAB_CI` выбирает GitLab, `GITHUB_ACTIONS` выбирает GitHub Actions) или выводится из единственного активного провайдера.

### Генерация джобов

Для каждого модуля TerraCi генерирует:

1. **Plan джоб** (если `plan_enabled: true`)
   - Выполняет `terraform plan -out=plan.tfplan`
   - Сохраняет план как артефакт

2. **Apply джоб**
   - Зависит от plan джоба (`needs`)
   - Выполняет `terraform apply plan.tfplan`
   - Ручной запуск (если `auto_approve: false`)

### Маппинг стейджей

Уровни выполнения отображаются на GitLab стейджи:

```yaml
stages:
  - deploy-plan-0   # Планы уровня 0
  - deploy-apply-0  # Применение уровня 0
  - deploy-plan-1   # Планы уровня 1
  - deploy-apply-1  # Применение уровня 1
```

### Цепочка зависимостей

```yaml
plan-vpc:
  stage: deploy-plan-0

apply-vpc:
  stage: deploy-apply-0
  needs: [plan-vpc]

plan-eks:
  stage: deploy-plan-1
  needs: [apply-vpc]  # Ждёт применения vpc

apply-eks:
  stage: deploy-apply-1
  needs: [plan-eks]
```

## Диаграмма потока данных

```mermaid
flowchart TD
  A["terraci generate"] --> B
  B["workflow.Run() — scan, filter, parse, graph"] --> C
  C["resolver.CollectContributions(appCtx)"] --> D
  D["pipeline.Build(opts) → *pipeline.IR"] --> E
  E{"Провайдер?"}
  E -->|GitLab| F["gitlab.NewGenerator(ctx, ir)"] --> G[".gitlab-ci.yml"]
  E -->|GitHub| H["github.NewGenerator(ctx, ir)"] --> I["workflow.yml"]
```

Описание каждого этапа:

| Шаг | Функция | Что делает |
|-----|---------|-----------|
| 1 | `workflow.Run()` | Сканирование файловой системы, применение фильтров, парсинг HCL, построение графа зависимостей |
| 2 | `resolver.CollectContributions(appCtx)` | Сбор шагов и отдельных джобов, контрибьютнутых плагинами (cost, policy, summary, tfupdate) |
| 3 | `pipeline.Build(opts)` | Построение провайдер-агностичного IR (`*pipeline.IR{Levels, Jobs}`) — единый вход для исполнения |
| 4 | `provider.NewGenerator(ctx, ir)` + `Generate()` | Привязка IR к провайдеру; преобразование IR в YAML GitLab CI или воркфлоу GitHub Actions |

IR — **единый источник** как для генерации пайплайнов, так и для `terraci local-exec`: провайдеры не обращаются отдельно к графу зависимостей или списку контрибуций — IR уже их в себе содержит.

## Ключевые типы

### Module

Представляет обнаруженный Terraform модуль. Вместо жёстко заданных полей модуль использует `components` карту с ключами из именованных сегментов настроенного паттерна:

```go
type Module struct {
    components map[string]string // {"service": "platform", "environment": "stage", ...}
    segments   []string          // упорядоченные имена сегментов из паттерна

    Path         string  // /abs/path/to/vpc
    RelativePath string  // platform/stage/eu-central-1/vpc
    Parent       *Module
    Children     []*Module
}

func (m *Module) Get(name string) string  // m.Get("service") → "platform"
func (m *Module) ID() string              // возвращает RelativePath
```

Такой дизайн позволяет использовать полностью настраиваемые паттерны. Например, с паттерном `{team}/{env}/{module}` вы используете `m.Get("team")` и `m.Get("env")` вместо фиксированных имён полей.

### RemoteStateRef

Представляет зависимость `terraform_remote_state`:

```go
type RemoteStateRef struct {
    Name         string            // "vpc"
    Backend      string            // "s3"
    Config       map[string]string // bucket, key, region
    WorkspaceDir string            // разрешённый путь модуля
}
```

### DependencyGraph

Управляет связями между модулями:

```go
type DependencyGraph struct {
    nodes map[string]*Module
    edges map[string][]string  // from -> [to, to, ...]
}

func (g *DependencyGraph) AddEdge(from, to *Module)
func (g *DependencyGraph) TopologicalSort() ([]*Module, error)
func (g *DependencyGraph) ExecutionLevels() [][]*Module
func (g *DependencyGraph) DetectCycles() [][]string
```

## Производительность

TerraCi оптимизирован для скорости:

| Размер проекта | Модулей | Время парсинга | Время генерации |
|----------------|---------|----------------|-----------------|
| Маленький | 10 | ~100мс | ~50мс |
| Средний | 50 | ~300мс | ~100мс |
| Большой | 200 | ~1с | ~300мс |

Советы для больших проектов:
- Используйте паттерны `exclude` для пропуска ненужных директорий
- Используйте `--changed-only` для инкрементальных пайплайнов
- Включите кэширование в сгенерированных пайплайнах

## Смотрите также

- [Структура проекта](/ru/guide/project-structure) — требования к структуре директорий
- [Зависимости](/ru/guide/dependencies) — детали обнаружения зависимостей
- [Генерация пайплайнов](/ru/guide/pipeline-generation) — формат сгенерированного вывода
