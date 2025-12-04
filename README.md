# TerraCi

CLI-инструмент для анализа Terraform-проектов и автоматической генерации GitLab CI пайплайнов с учётом зависимостей между модулями.

## Возможности

- Автоматическое обнаружение Terraform-модулей по структуре директорий
- Извлечение зависимостей из `terraform_remote_state` (включая `for_each`)
- Построение графа зависимостей с топологической сортировкой
- Генерация GitLab CI пайплайнов с правильным порядком выполнения
- Фильтрация модулей по glob-паттернам
- Git-интеграция: генерация пайплайнов только для изменённых модулей

## Установка

```bash
# Из исходников
go install github.com/terraci/terraci/cmd/terraci@latest

# Или сборка локально
make build
```

## Быстрый старт

```bash
# Инициализация конфигурации
terraci init

# Валидация структуры проекта
terraci validate

# Генерация пайплайна
terraci generate -o .gitlab-ci.yml

# Только для изменённых модулей
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml
```

## Структура проекта

TerraCi ожидает следующую структуру директорий:

```
project/
├── service/
│   └── environment/
│       └── region/
│           ├── module/           # depth 4
│           │   └── main.tf
│           └── module/
│               └── submodule/    # depth 5 (опционально)
│                   └── main.tf
```

Пример:
```
infrastructure/
├── cdp/
│   ├── stage/
│   │   └── eu-central-1/
│   │       ├── vpc/
│   │       ├── eks/
│   │       └── ec2/
│   │           └── rabbitmq/    # submodule
│   └── prod/
│       └── eu-central-1/
│           └── vpc/
```

## Команды

| Команда | Описание |
|---------|----------|
| `terraci generate` | Генерация GitLab CI пайплайна |
| `terraci validate` | Валидация структуры и зависимостей |
| `terraci graph` | Визуализация графа зависимостей |
| `terraci init` | Создание конфигурационного файла |
| `terraci version` | Информация о версии |

## Конфигурация

Файл `.terraci.yaml`:

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
```

## Примеры

### Граф зависимостей в формате DOT

```bash
terraci graph --format dot -o deps.dot
dot -Tpng deps.dot -o deps.png
```

### Фильтрация по окружению

```bash
terraci generate --environment prod -o prod-pipeline.yml
```

### Исключение модулей

```bash
terraci generate --exclude "*/sandbox/*" --exclude "*/test/*"
```

## Лицензия

MIT
