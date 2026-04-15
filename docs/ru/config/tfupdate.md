---
title: "Обновление зависимостей"
description: "Конфигурация плагина tfupdate: проверка, обновление версий и синхронизация lock-файлов Terraform"
outline: deep
---

# Обновление зависимостей

TerraCi умеет проверять доступные обновления для провайдеров и модулей Terraform, используя [Terraform Registry API](https://developer.hashicorp.com/terraform/cloud-docs/api-docs/private-registry). Поддерживается как режим отчёта (только чтение), так и режим записи с обновлением `.tf` файлов и синхронизацией `.terraform.lock.hcl`.

## Базовая конфигурация

```yaml
plugins:
  tfupdate:
    enabled: true
    policy:
      bump: minor
```

## Параметры конфигурации

### enabled

Включение или отключение плагина обновлений.

```yaml
plugins:
  tfupdate:
    enabled: true  # по умолчанию: false
```

Плагин использует политику `EnabledExplicitly` — он должен быть явно включён через `enabled: true`.

### target

Определяет, что проверять: провайдеры, модули или всё.

```yaml
plugins:
  tfupdate:
    target: all  # по умолчанию: all
```

| Значение | Описание |
|----------|----------|
| `all` | Проверять и провайдеры, и модули (по умолчанию) |
| `providers` | Проверять только провайдеры |
| `modules` | Проверять только модули |

### policy.bump

Уровень версионирования для определения «доступных» обновлений.

```yaml
plugins:
  tfupdate:
    policy:
      bump: minor  # обязательный параметр
```

| Значение | Описание |
|----------|----------|
| `patch` | Учитывать только patch-обновления |
| `minor` | Учитывать minor и patch-обновления |
| `major` | Учитывать major, minor и patch-обновления |

### policy.pin

Фиксация обновлённых ограничений до точной версии при записи.

```yaml
plugins:
  tfupdate:
    policy:
      bump: minor
      pin: false  # по умолчанию: false
```

При `true` ограничения типа `~> 5.80` заменяются на `5.80.0` при записи.

### ignore

Список провайдеров или модулей, которые следует исключить из проверки. Значения должны совпадать с полем `source` в блоках `required_providers` или `module`.

```yaml
plugins:
  tfupdate:
    ignore:
      - registry.terraform.io/hashicorp/null
      - registry.terraform.io/hashicorp/random
      - registry.terraform.io/hashicorp/time
```

### timeout

Общий таймаут для запуска. По умолчанию 5 минут для чтения, 20 минут для записи.

```yaml
plugins:
  tfupdate:
    timeout: "15m"
```

### registries

Настройка хостов реестров для поиска провайдеров.

```yaml
plugins:
  tfupdate:
    registries:
      default: registry.terraform.io  # по умолчанию
      providers:
        hashicorp/aws: custom-registry.example.com
```

| Поле | Описание |
|------|----------|
| `default` | Хост реестра по умолчанию |
| `providers` | Переопределение хоста для конкретных провайдеров |

### lock

Настройка синхронизации lock-файлов.

```yaml
plugins:
  tfupdate:
    lock:
      platforms:
        - linux_amd64
        - darwin_arm64
```

| Поле | Описание |
|------|----------|
| `platforms` | Набор платформ для h1-хешей в `.terraform.lock.hcl`. Пусто — все платформы. |

### cache

Настройка кеширования метаданных реестра и артефактов провайдеров.

```yaml
plugins:
  tfupdate:
    cache:
      metadata:
        backend: inmemcache
        ttl: "6h"
        namespace: tfupdate/registry
      artifacts:
        backend: diskblob
        namespace: tfupdate/providers
```

### pipeline

Добавляет джоб `tfupdate-check` в начало сгенерированного CI пайплайна.

```yaml
plugins:
  tfupdate:
    pipeline: false  # по умолчанию: false
```

При `pipeline: true` джоб добавляется с `allow_failure: true`, поэтому доступные обновления не блокируют деплой.

## Полный пример конфигурации

```yaml
plugins:
  tfupdate:
    enabled: true
    target: all
    policy:
      bump: minor
      pin: false
    ignore:
      - registry.terraform.io/hashicorp/null
      - registry.terraform.io/hashicorp/random
    registries:
      default: registry.terraform.io
    lock:
      platforms:
        - linux_amd64
        - darwin_arm64
    cache:
      metadata:
        backend: inmemcache
        ttl: "6h"
      artifacts:
        backend: diskblob
    pipeline: true
    timeout: "15m"
```

## Как это работает

1. TerraCi обнаруживает Terraform-модули в проекте согласно `structure.pattern`
2. Для каждого модуля считываются блоки `required_providers`, `module` и `.terraform.lock.hcl`
3. Планировщик/решатель находит совместимые версии с учётом транзитивных зависимостей провайдеров из модулей
4. Для каждой зависимости выполняется запрос к Terraform Registry API
5. Результаты кешируются для снижения нагрузки на API
6. Сравниваются текущие ограничения версий с доступными обновлениями согласно уровню `bump`
7. При `--write` обновляются `.tf` файлы и синхронизируются `.terraform.lock.hcl`

## Синхронизация lock-файлов

При использовании `--write` TerraCi автоматически обновляет `.terraform.lock.hcl`:

- Записи провайдеров создаются или обновляются с новой версией
- `zh:` хеши собираются из метаданных реестра для всех платформ
- `h1:` хеши вычисляются путём загрузки архивов для настроенных платформ
- Существующие хеши сохраняются и объединяются с новыми

## Интеграция с CI пайплайном

При `pipeline: true` в сгенерированный пайплайн добавляется джоб:

**GitLab CI:**
```yaml
stages:
  - dependency-check
  - deploy-plan-0
  - deploy-apply-0

tfupdate-check:
  stage: dependency-check
  script:
    - terraci tfupdate
  artifacts:
    paths:
      - .terraci/tfupdate-results.json
  allow_failure: true
```

**GitHub Actions:**
```yaml
jobs:
  tfupdate-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: terraci tfupdate
    continue-on-error: true
```

## CLI команда

```bash
# Проверить все зависимости
terraci tfupdate

# Проверить только провайдеры
terraci tfupdate --target providers

# Записать обновлённые версии и синхронизировать lock-файлы
terraci tfupdate --write

# Фиксировать точные версии
terraci tfupdate --write --pin

# Проверить конкретный модуль
terraci tfupdate --module platform/prod/eu-central-1/vpc

# JSON вывод
terraci tfupdate --output json
```

> **Примечание:** требуется `plugins.tfupdate.enabled: true` в `.terraci.yaml`.

## Артефакты

| Файл | Описание |
|------|----------|
| `tfupdate-results.json` | Полные результаты проверки |
| `tfupdate-report.json` | Сводный отчёт для CI |

## Смотрите также

- [terraci tfupdate](/ru/cli/tfupdate) — описание CLI команды и флагов
