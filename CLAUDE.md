# TerraCi - AI Assistant Guide

CLI-инструмент для анализа Terraform-проектов, построения графа зависимостей и генерации GitLab CI пайплайнов.

## Структура проекта

```
terraci/
├── cmd/terraci/
│   ├── main.go                 # Точка входа
│   └── cmd/                    # Cobra-команды
│       ├── root.go             # Корневая команда, глобальные флаги
│       ├── generate.go         # Генерация пайплайна
│       ├── validate.go         # Валидация проекта
│       ├── graph.go            # Визуализация графа
│       ├── init.go             # Инициализация конфига
│       └── version.go          # Версия
├── internal/
│   ├── discovery/              # Обнаружение модулей
│   │   └── module.go           # Scanner, Module, ModuleIndex
│   ├── parser/                 # Парсинг HCL
│   │   ├── hcl.go              # Parser, ParsedModule, RemoteStateRef
│   │   └── dependency.go       # DependencyExtractor
│   ├── graph/                  # Граф зависимостей
│   │   └── dependency.go       # DependencyGraph, TopologicalSort
│   ├── pipeline/gitlab/        # Генерация GitLab CI
│   │   └── generator.go        # Generator, Pipeline, Job
│   ├── filter/                 # Фильтрация модулей
│   │   └── glob.go             # GlobFilter, CompositeFilter
│   └── git/                    # Git-интеграция
│       └── diff.go             # Client, ChangedModulesDetector
├── pkg/config/                 # Публичный пакет конфигурации
│   └── config.go               # Config, Load(), Validate()
├── Makefile
├── go.mod
└── .terraci.example.yaml
```

## Ключевые типы

### discovery.Module
```go
type Module struct {
    Service      string    // cdp
    Environment  string    // stage, prod
    Region       string    // eu-central-1
    Module       string    // vpc, eks
    Submodule    string    // опционально: rabbitmq (для ec2/rabbitmq)
    Path         string    // абсолютный путь
    RelativePath string    // относительный путь
    Parent       *Module   // ссылка на родительский модуль
    Children     []*Module // дочерние сабмодули
}

func (m *Module) ID() string      // service/env/region/module[/submodule]
func (m *Module) Name() string    // module или module/submodule
func (m *Module) IsSubmodule() bool
```

### discovery.Scanner
```go
type Scanner struct {
    RootDir  string
    MinDepth int  // default: 4
    MaxDepth int  // default: 5
}

func (s *Scanner) Scan() ([]*Module, error)
```

Паттерн директорий: `service/environment/region/module[/submodule]`

### parser.RemoteStateRef
```go
type RemoteStateRef struct {
    Name         string            // имя data-блока
    Backend      string            // s3, gcs, etc.
    Config       map[string]string // конфиг бэкенда
    ForEach      bool              // есть ли for_each
    WorkspaceDir string            // резолвленный путь
}
```

### graph.DependencyGraph
```go
func (g *DependencyGraph) AddNode(module *discovery.Module)
func (g *DependencyGraph) AddEdge(from, to *discovery.Module)
func (g *DependencyGraph) TopologicalSort() ([]*discovery.Module, error)
func (g *DependencyGraph) ExecutionLevels() [][]*discovery.Module
func (g *DependencyGraph) DetectCycles() [][]string
func (g *DependencyGraph) ToDOT() string
```

## CLI команды

```bash
# Генерация пайплайна
terraci generate -o .gitlab-ci.yml
terraci generate --changed-only --base-ref main
terraci generate --exclude "*/test/*" --environment prod

# Валидация
terraci validate

# Граф зависимостей
terraci graph --format dot -o deps.dot
terraci graph --format levels
terraci graph --module cdp/stage/eu-central-1/vpc --dependents

# Инициализация
terraci init
```

## Глобальные флаги

- `-c, --config` — путь к конфигу (по умолчанию ищет `.terraci.yaml`)
- `-d, --dir` — рабочая директория
- `-v, --verbose` — подробный вывод

## Конфигурация (.terraci.yaml)

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true

exclude:
  - "*/test/*"
  - "*/sandbox/*"

gitlab:
  terraform_image: "hashicorp/terraform:1.6"
  parallelism: 5
  plan_enabled: true
  auto_approve: false
  tags: [terraform, docker]
```

## Сборка и тесты

```bash
make build      # Сборка бинарника
make test       # Запуск тестов
make lint       # Линтинг
make install    # Установка в $GOPATH/bin
```

## Поток данных

1. `Scanner.Scan()` — обнаружение модулей в директориях
2. `ModuleIndex` — индексация для быстрого поиска
3. `Parser.ParseModule()` — парсинг HCL, извлечение locals и remote_state
4. `DependencyExtractor` — определение зависимостей между модулями
5. `DependencyGraph` — построение DAG, топологическая сортировка
6. `Generator.Generate()` — генерация GitLab CI YAML

## Алгоритмы

- **Топологическая сортировка**: алгоритм Кана для упорядочивания модулей
- **Детекция циклов**: DFS для поиска циклических зависимостей
- **Execution Levels**: группировка модулей для параллельного выполнения
- **Path Resolution**: интерполяция переменных в путях state-файлов

## Зависимости

- `github.com/spf13/cobra` — CLI-фреймворк
- `github.com/hashicorp/hcl/v2` — парсинг HCL
- `github.com/zclconf/go-cty` — типы CTY для HCL
- `gopkg.in/yaml.v3` — YAML

## Известные особенности

- Модули могут существовать на глубине 4 (базовые) и 5 (сабмодули) одновременно
- `for_each` в remote_state разворачивается в множественные зависимости
- Фильтры поддерживают `**` для произвольной глубины пути
