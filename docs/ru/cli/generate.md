# terraci generate

Генерация GitLab CI пайплайна для Terraform-модулей.

## Синтаксис

```bash
terraci generate [flags]
```

## Описание

Команда `generate` анализирует структуру проекта, строит граф зависимостей и генерирует GitLab CI пайплайн с правильным порядком выполнения модулей.

## Флаги

| Флаг | Тип | По умолчанию | Описание |
|------|-----|--------------|----------|
| `-o, --output` | string | stdout | Файл для записи пайплайна |
| `--dry-run` | bool | false | Показать что будет сгенерировано |
| `--changed-only` | bool | false | Только изменённые модули |
| `--base-ref` | string | `main` | Базовая ветка для сравнения |
| `--module` | string | | Конкретный модуль |
| `--include` | []string | | Паттерны включения |
| `--exclude` | []string | | Паттерны исключения |
| `--environment` | string | | Фильтр по окружению |

## Примеры

### Базовая генерация

```bash
# Вывод в stdout
terraci generate

# Сохранение в файл
terraci generate -o .gitlab-ci.yml
```

### Dry Run

Просмотр информации без генерации YAML:

```bash
terraci generate --dry-run
```

Вывод:
```
Modules discovered: 15
Modules to process: 15
Execution levels: 4
Total jobs: 30

Level 0 (parallel):
  - platform/prod/eu-central-1/vpc
  - platform/prod/eu-west-1/vpc

Level 1:
  - platform/prod/eu-central-1/eks
  - platform/prod/eu-west-1/eks
  ...
```

### Только изменённые модули

```bash
# Сравнение с main
terraci generate --changed-only

# Сравнение с конкретной веткой
terraci generate --changed-only --base-ref develop

# Сравнение с тегом
terraci generate --changed-only --base-ref v1.0.0
```

При `--changed-only` TerraCi:
1. Определяет изменённые файлы через `git diff`
2. Находит затронутые модули
3. Добавляет зависимые модули (downstream)
4. Генерирует пайплайн только для них

### Конкретный модуль

```bash
# Модуль и его зависимые
terraci generate --module platform/prod/eu-central-1/vpc
```

### Фильтрация

```bash
# Только production
terraci generate --environment prod

# Исключить тестовые модули
terraci generate --exclude "*/test/*"

# Только конкретный сервис
terraci generate --include "billing/*/*/*"

# Комбинирование
terraci generate \
  --include "platform/*/*/*" \
  --exclude "*/dev/*" \
  -o .gitlab-ci.yml
```

### Разные конфигурации

```bash
# Production пайплайн
terraci generate \
  -c production.terraci.yaml \
  -o .gitlab-ci-prod.yml

# Staging пайплайн
terraci generate \
  -c staging.terraci.yaml \
  -o .gitlab-ci-stage.yml
```

## Сгенерированный пайплайн

### Структура

```yaml
stages:
  - deploy-plan-0
  - deploy-apply-0
  - deploy-plan-1
  - deploy-apply-1

variables:
  TF_IN_AUTOMATION: "true"
  TERRAFORM_BINARY: "terraform"

default:
  image: hashicorp/terraform:1.6
  before_script:
    - ${TERRAFORM_BINARY} init
  tags:
    - terraform

# План для VPC (уровень 0)
plan-platform-prod-eu-central-1-vpc:
  stage: deploy-plan-0
  script:
    - cd platform/prod/eu-central-1/vpc
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan
  artifacts:
    paths:
      - platform/prod/eu-central-1/vpc/plan.tfplan

# Apply для VPC
apply-platform-prod-eu-central-1-vpc:
  stage: deploy-apply-0
  needs:
    - job: plan-platform-prod-eu-central-1-vpc
  script:
    - cd platform/prod/eu-central-1/vpc
    - ${TERRAFORM_BINARY} apply plan.tfplan
  when: manual

# План для EKS (уровень 1, зависит от VPC)
plan-platform-prod-eu-central-1-eks:
  stage: deploy-plan-1
  needs:
    - job: apply-platform-prod-eu-central-1-vpc
  script:
    - cd platform/prod/eu-central-1/eks
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan
```

### Переменные джобов

Каждый джоб получает переменные:

| Переменная | Описание |
|------------|----------|
| `TF_MODULE_PATH` | Относительный путь к модулю |
| `TF_SERVICE` | Название сервиса |
| `TF_ENVIRONMENT` | Окружение |
| `TF_REGION` | Регион |
| `TF_MODULE` | Название модуля |

### Resource Groups

Джобы используют `resource_group` для предотвращения параллельного apply одного модуля:

```yaml
apply-platform-prod-eu-central-1-vpc:
  resource_group: platform/prod/eu-central-1/vpc
```

## Интеграция с CI

### GitLab CI

```yaml
# .gitlab-ci.yml
include:
  - local: .generated-pipeline.yml

generate-pipeline:
  stage: prepare
  script:
    - terraci generate -o .generated-pipeline.yml
  artifacts:
    paths:
      - .generated-pipeline.yml
```

### Динамический пайплайн

```yaml
generate:
  stage: prepare
  script:
    - terraci generate --changed-only -o generated.yml
  artifacts:
    paths:
      - generated.yml

trigger:
  stage: deploy
  trigger:
    include:
      - artifact: generated.yml
        job: generate
```

## Диагностика

### Модули не найдены

```bash
# Проверить обнаружение
terraci validate --verbose

# Проверить структуру
terraci graph --format levels
```

### Неверный порядок

```bash
# Проверить зависимости
terraci graph --module <module-id> --dependents
terraci graph --module <module-id> --dependencies
```

### Циклические зависимости

```bash
# Показывает ошибку при наличии циклов
terraci validate
```
