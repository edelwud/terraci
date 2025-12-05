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

### 2. Сопоставление с модулями

Изменённые файлы сопоставляются с родительскими модулями:

| Файл | Модуль |
|------|--------|
| `platform/production/us-east-1/vpc/main.tf` | `platform/production/us-east-1/vpc` |

### 3. Поиск затронутых модулей

TerraCi обходит граф зависимостей для поиска всех зависимых модулей:

```
Изменён: vpc
    ↓
Зависимые: eks, rds, cache
    ↓
Зависимые: app-backend, app-frontend
```

## Опции ссылок

```bash
# Сравнение с main
terraci generate --changed-only --base-ref main

# Сравнение с коммитом
terraci generate --changed-only --base-ref abc123

# Сравнение с тегом
terraci generate --changed-only --base-ref v1.0.0

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

### Комбинирование с фильтрами

```bash
# Только production изменения
terraci generate --changed-only --base-ref main --environment production

# Исключить test модули
terraci generate --changed-only --base-ref main --exclude "*/test/*"
```

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
```
