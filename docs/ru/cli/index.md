# CLI-команды

TerraCi предоставляет набор команд для работы с Terraform-проектами.

## Установка

```bash
# Из исходников
go install github.com/edelwud/terraci@latest

# Docker
docker run --rm -v $(pwd):/workspace ghcr.io/edelwud/terraci generate
```

## Глобальные флаги

| Флаг | Сокращение | По умолчанию | Описание |
|------|------------|--------------|----------|
| `--config` | `-c` | `.terraci.yaml` | Путь к файлу конфигурации |
| `--dir` | `-d` | `.` | Рабочая директория |
| `--verbose` | `-v` | `false` | Подробный вывод |
| `--help` | `-h` | | Показать справку |

## Команды

### [generate](./generate.md)

Генерация GitLab CI пайплайна:

```bash
terraci generate -o .gitlab-ci.yml
```

### [validate](./validate.md)

Валидация проекта и конфигурации:

```bash
terraci validate
```

### [graph](./graph.md)

Визуализация графа зависимостей:

```bash
terraci graph --format dot -o deps.dot
```

### [init](./init.md)

Инициализация конфигурации:

```bash
terraci init
```

### [summary](./summary.md)

Публикация результатов plan в MR:

```bash
terraci summary
```

### [policy](./policy.md)

Проверка Terraform планов на соответствие OPA политикам:

```bash
# Загрузить политики из источников
terraci policy pull

# Проверить все модули
terraci policy check

# Проверить конкретный модуль
terraci policy check --module platform/prod/eu-central-1/vpc
```

## Примеры использования

### Базовый workflow

```bash
# 1. Инициализация конфигурации
terraci init

# 2. Настройка .terraci.yaml под проект

# 3. Валидация
terraci validate

# 4. Просмотр зависимостей
terraci graph --format levels

# 5. Генерация пайплайна
terraci generate --dry-run
terraci generate -o .gitlab-ci.yml
```

### Работа с изменёнными модулями

```bash
# Генерация только для изменённых модулей
terraci generate --changed-only --base-ref main

# Просмотр затронутых модулей
terraci graph --changed-only --format levels
```

### Фильтрация модулей

```bash
# Только production
terraci generate --environment prod

# Исключить тестовые модули
terraci generate --exclude "*/test/*"

# Конкретный модуль и зависимые
terraci generate --module platform/prod/eu-central-1/vpc
```

### Docker

```bash
# Генерация пайплайна
docker run --rm \
  -v $(pwd):/workspace \
  -w /workspace \
  ghcr.io/edelwud/terraci generate -o .gitlab-ci.yml

# Валидация
docker run --rm \
  -v $(pwd):/workspace \
  -w /workspace \
  ghcr.io/edelwud/terraci validate
```

## Коды возврата

| Код | Описание |
|-----|----------|
| `0` | Успешное выполнение |
| `1` | Ошибка выполнения |
| `2` | Ошибка конфигурации |
| `3` | Модули не найдены |

## Переменные окружения

| Переменная | Описание |
|------------|----------|
| `TERRACI_CONFIG` | Путь к конфигурации (альтернатива `--config`) |
| `TERRACI_DIR` | Рабочая директория (альтернатива `--dir`) |
| `TERRACI_VERBOSE` | Включить verbose-режим (`true`/`false`) |

## Автодополнение

### Bash

```bash
terraci completion bash > /etc/bash_completion.d/terraci
```

### Zsh

```bash
terraci completion zsh > "${fpath[1]}/_terraci"
```

### Fish

```bash
terraci completion fish > ~/.config/fish/completions/terraci.fish
```

## Справка

```bash
# Общая справка
terraci --help

# Справка по команде
terraci generate --help
terraci graph --help
```
