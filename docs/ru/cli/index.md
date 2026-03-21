---
title: "CLI справочник"
description: "Интерфейс командной строки TerraCi: все команды, глобальные опции и коды выхода"
outline: deep
---

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

| Команда | Описание |
|---------|----------|
| [generate](./generate.md) | Генерация CI пайплайна (GitLab CI или GitHub Actions) |
| [validate](./validate.md) | Валидация структуры проекта |
| [graph](./graph.md) | Граф зависимостей (DOT, PlantUML, list, levels) |
| [init](./init.md) | Инициализация конфигурации (интерактивный TUI-мастер) |
| [cost](./cost.md) | Оценка стоимости AWS из файлов плана |
| [summary](./summary.md) | Публикация результатов plan в MR/PR |
| [policy](./policy.md) | Загрузка и проверка OPA-политик |
| `version` | Информация о версии |

## Примеры использования

```bash
# Генерация пайплайна (GitLab CI)
terraci generate -o .gitlab-ci.yml

# Генерация пайплайна (GitHub Actions)
terraci generate -o .github/workflows/terraform.yml

# Валидация с подробным выводом
terraci validate -v

# Использование другого конфига
terraci -c custom.yaml generate

# Работа в другой директории
terraci -d /path/to/project validate

# Фильтрация модулей по сегменту
terraci generate --filter environment=production --filter region=us-east-1
```

### Работа с изменёнными модулями

```bash
# Генерация только для изменённых модулей
terraci generate --changed-only --base-ref main

# Просмотр затронутых модулей
terraci graph --changed-only --format levels
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
