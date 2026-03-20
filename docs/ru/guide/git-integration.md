---
title: "Git интеграция"
description: "Режим changed-only: определение изменённых модулей через git diff и генерация целевых пайплайнов"
outline: deep
---

# Git интеграция

TerraCi интегрируется с Git для генерации пайплайнов только для изменённых модулей и их зависимых.

## Режим changed-only

Включите режим флагом `--changed-only`:

```bash
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml
```

## Как это работает

### 1. Определение изменённых файлов

TerraCi выполняет `git diff`:

```bash
git diff --name-only main...HEAD
```

Вывод:
```
platform/production/us-east-1/vpc/main.tf
platform/production/us-east-1/vpc/variables.tf
shared/modules/vpc/main.tf
```

### 2. Сопоставление с модулями

Изменённые файлы сопоставляются с родительскими модулями:

| Файл | Модуль |
|------|--------|
| `platform/production/us-east-1/vpc/main.tf` | `platform/production/us-east-1/vpc` |
| `platform/production/us-east-1/vpc/variables.tf` | `platform/production/us-east-1/vpc` |

Файлы вне директорий модулей (например, `shared/modules/`) игнорируются.

### 3. Поиск затронутых модулей

TerraCi обходит граф зависимостей для поиска всех зависимых модулей:

```
Изменён: vpc
    ↓
Зависимые: eks, rds, cache
    ↓
Зависимые: app-backend, app-frontend
```

### 4. Генерация пайплайна

Пайплайн генерируется только для затронутых модулей с сохранением правильного порядка зависимостей.

## Опции ссылок

### Базовая ссылка

Укажите базовую ветку или коммит:

```bash
# Сравнение с main
terraci generate --changed-only --base-ref main

# Сравнение с коммитом
terraci generate --changed-only --base-ref abc123

# Сравнение с тегом
terraci generate --changed-only --base-ref v1.0.0
```

### Автоопределение ветки по умолчанию

Если `--base-ref` не указан, TerraCi пытается автоматически определить основную ветку:

```bash
terraci generate --changed-only  # Автоматически определяет main/master
```

### Сравнение коммитов

Сравнение между конкретными коммитами:

```bash
# Последние 5 коммитов
terraci generate --changed-only --base-ref HEAD~5
```

## Примеры использования

### Пайплайн для Pull Request

```yaml
generate-pipeline:
  stage: prepare
  script:
    - terraci generate --changed-only --base-ref $CI_MERGE_REQUEST_TARGET_BRANCH_NAME -o generated.yml
  artifacts:
    paths:
      - generated.yml

deploy:
  stage: deploy
  trigger:
    include:
      - artifact: generated.yml
        job: generate-pipeline
```

### Деплой на feature-ветках

Деплой только изменённой инфраструктуры на feature-ветках:

```yaml
deploy-feature:
  script:
    - terraci generate --changed-only --base-ref main -o pipeline.yml
    - gitlab-runner exec shell < pipeline.yml
  rules:
    - if: $CI_COMMIT_BRANCH != "main"
```

### Полный деплой по расписанию

Запуск полного деплоя по расписанию, в остальных случаях — только для изменений:

```yaml
generate:
  script:
    - |
      if [ "$CI_PIPELINE_SOURCE" = "schedule" ]; then
        terraci generate -o pipeline.yml
      else
        terraci generate --changed-only --base-ref main -o pipeline.yml
      fi
```

### Комбинирование с фильтрами

```bash
# Только production изменения
terraci generate --changed-only --base-ref main --filter environment=production

# Исключить test модули
terraci generate --changed-only --base-ref main --exclude "*/test/*"

# Несколько фильтров
terraci generate --changed-only --base-ref main --filter environment=prod --filter region=us-east-1
```

::: tip Синтаксис фильтров
Флаг `--filter key=value` заменяет устаревшие флаги `--service`, `--environment` и `--region`. Ключ должен совпадать с именем сегмента из настроенного паттерна.
:::

## Просмотр изменённых модулей

```bash
terraci generate --changed-only --base-ref main --dry-run
```

Вывод:
```
Changed files:
  - platform/production/us-east-1/vpc/main.tf

Changed modules:
  - platform/production/us-east-1/vpc

Affected modules (including dependents):
  - platform/production/us-east-1/vpc
  - platform/production/us-east-1/eks
  - platform/production/us-east-1/rds
  - platform/production/us-east-1/app
```

## Устранение неполадок

### Изменения не обнаружены

Если ни один модуль не определён как изменённый:

1. Убедитесь, что базовая ссылка существует:
   ```bash
   git rev-parse main
   ```

2. Проверьте git diff вручную:
   ```bash
   git diff --name-only main...HEAD
   ```

3. Убедитесь, что изменённые файлы находятся в директориях модулей

### Слишком много затронутых модулей

Если затронуто больше модулей, чем ожидалось:

1. Проверьте граф зависимостей:
   ```bash
   terraci graph --format list
   ```

2. Убедитесь, что ссылки remote state корректны

3. Оцените, является ли цепочка зависимостей намеренной

### Незакоммиченные изменения

TerraCi учитывает только закоммиченные изменения. Чтобы включить незакоммиченные:

```bash
# Сначала закоммитьте
git add -A
git commit -m "WIP"
terraci generate --changed-only --base-ref main
```

Или используйте сравнение с веткой по умолчанию, которое включает изменения рабочей директории:

```bash
terraci generate --changed-only  # Включает изменения рабочей директории
```

## Следующие шаги

- [Сабмодули](/ru/guide/submodules) — вложенные модули на глубине 5
- [Поддержка OpenTofu](/ru/guide/opentofu) — переключение на OpenTofu
