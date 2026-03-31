---
title: "Проверка обновлений"
description: "Конфигурация плагина update: проверка и обновление версий провайдеров и модулей Terraform"
outline: deep
---

# Проверка обновлений

TerraCi умеет проверять доступные обновления для провайдеров и модулей Terraform, используя [Terraform Registry API](https://developer.hashicorp.com/terraform/cloud-docs/api-docs/private-registry). Поддерживается как режим отчёта (только чтение), так и режим записи с обновлением `.tf` файлов.

## Базовая конфигурация

```yaml
plugins:
  update:
    enabled: true
```

## Параметры конфигурации

### enabled

Включение или отключение плагина обновлений.

```yaml
plugins:
  update:
    enabled: true  # по умолчанию: false
```

Плагин использует политику `EnabledExplicitly` — он должен быть явно включён через `enabled: true`.

### target

Определяет, что проверять: провайдеры, модули или всё.

```yaml
plugins:
  update:
    target: all  # по умолчанию: all
```

| Значение | Описание |
|----------|----------|
| `all` | Проверять и провайдеры, и модули (по умолчанию) |
| `providers` | Проверять только провайдеры |
| `modules` | Проверять только модули |

### bump

Уровень версионирования для определения «доступных» обновлений.

```yaml
plugins:
  update:
    bump: minor  # по умолчанию: minor
```

| Значение | Описание |
|----------|----------|
| `patch` | Учитывать только patch-обновления |
| `minor` | Учитывать minor и patch-обновления (по умолчанию) |
| `major` | Учитывать major, minor и patch-обновления |

### ignore

Список провайдеров или модулей, которые следует исключить из проверки. Значения должны совпадать с полем `source` в блоках `required_providers` или `module`.

```yaml
plugins:
  update:
    ignore:
      - registry.terraform.io/hashicorp/null
      - registry.terraform.io/hashicorp/random
      - registry.terraform.io/hashicorp/time
```

### pipeline

Добавляет джоб `dependency-update-check` в начало сгенерированного CI пайплайна (фаза до plan).

```yaml
plugins:
  update:
    pipeline: false  # по умолчанию: false
```

При `pipeline: true` джоб добавляется с `allow_failure: true`, поэтому доступные обновления не блокируют деплой — только информируют команду.

## Полный пример конфигурации

```yaml
plugins:
  update:
    enabled: true
    target: all
    bump: minor
    ignore:
      - registry.terraform.io/hashicorp/null
      - registry.terraform.io/hashicorp/random
    pipeline: true
```

## Как это работает

1. TerraCi обнаруживает Terraform-модули в проекте согласно `structure.pattern`
2. Для каждого модуля считываются блоки `required_providers` и `module` из `.tf` файлов
3. Для каждой зависимости выполняется запрос к Terraform Registry API за информацией о последней версии
4. Результаты кешируются в рамках одного запуска для снижения нагрузки на API
5. Сравниваются текущие ограничения версий с доступными обновлениями согласно уровню `bump`
6. Выводится отчёт; при `--write` обновляются соответствующие `.tf` файлы

## Интеграция с CI пайплайном

При `pipeline: true` в сгенерированный пайплайн добавляется джоб перед plan-стадией:

**GitLab CI:**
```yaml
stages:
  - dependency-check
  - deploy-plan-0
  - deploy-apply-0

dependency-update-check:
  stage: dependency-check
  script:
    - terraci update
  artifacts:
    paths:
      - .terraci/update-results.json
  allow_failure: true
```

**GitHub Actions:**
```yaml
jobs:
  dependency-update-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: terraci update
    continue-on-error: true
```

## CLI команда

```bash
# Проверить все зависимости
terraci update

# Проверить только провайдеры
terraci update --target providers

# Записать обновлённые версии в .tf файлы
terraci update --write

# Проверить конкретный модуль
terraci update --module platform/prod/eu-central-1/vpc

# JSON вывод
terraci update --output json
```

> **Примечание:** требуется `plugins.update.enabled: true` в `.terraci.yaml`.

## Смотрите также

- [terraci update](/ru/cli/update) — описание CLI команды и флагов
