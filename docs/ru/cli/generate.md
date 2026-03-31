---
title: "terraci generate"
description: "Генерация CI пайплайнов с учётом зависимостей и режимом changed-only"
outline: deep
---

# terraci generate

Генерация CI пайплайна (GitLab CI или GitHub Actions) для Terraform-модулей.

## Синтаксис

```bash
terraci generate [flags]
```

## Описание

Команда `generate` анализирует структуру проекта, строит граф зависимостей и генерирует CI пайплайн в формате YAML (GitLab CI или GitHub Actions, в зависимости от настроенного провайдера).

## Флаги

| Флаг | Сокр. | Тип | По умолчанию | Описание |
|------|-------|-----|--------------|----------|
| `--output` | `-o` | string | stdout | Файл для записи пайплайна |
| `--changed-only` | | bool | false | Только изменённые модули |
| `--base-ref` | | string | авто | Базовая ветка для сравнения |
| `--exclude` | `-x` | []string | | Паттерны исключения |
| `--include` | `-i` | []string | | Паттерны включения |
| `--filter` | `-f` | []string | | Фильтр по сегменту (`key=value`, напр. `environment=prod`) |
| `--plan-only` | | bool | false | Генерировать только план-джобы (без apply) |
| `--auto-approve` | | bool | false | Автоматический apply |
| `--no-auto-approve` | | bool | false | Требовать ручного подтверждения |
| `--dry-run` | | bool | false | Просмотр без генерации |

## Примеры

### Базовая генерация

```bash
# Вывод GitLab CI
terraci generate -o .gitlab-ci.yml

# Вывод GitHub Actions
terraci generate -o .github/workflows/terraform.yml

# Вывод в stdout
terraci generate
```

Провайдер автоопределяется из переменных окружения CI (например, `GITLAB_CI` или `GITHUB_ACTIONS`).

### Только изменённые модули

```bash
# Сравнение с main
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml

# Сравнение с конкретным коммитом
terraci generate --changed-only --base-ref abc123 -o .gitlab-ci.yml

# Автоопределение ветки по умолчанию
terraci generate --changed-only -o .gitlab-ci.yml
```

### Фильтрация

```bash
# По окружению
terraci generate --filter environment=production -o .gitlab-ci.yml

# По сервису
terraci generate --filter service=platform -o .gitlab-ci.yml

# По региону
terraci generate --filter region=us-east-1 -o .gitlab-ci.yml

# Комбинирование фильтров (И между разными ключами)
terraci generate --filter service=platform --filter environment=production --filter region=us-east-1 -o .gitlab-ci.yml

# Несколько значений для одного ключа (ИЛИ внутри одного ключа)
terraci generate --filter environment=stage --filter environment=prod -o .gitlab-ci.yml
```

Флаг `--filter` работает с любым именем сегмента, определённым в вашем `structure.pattern`.

### Паттерны include/exclude

```bash
# Исключить тестовые модули
terraci generate --exclude "*/test/*" -o .gitlab-ci.yml

# Несколько исключений
terraci generate -x "*/test/*" -x "*/sandbox/*" -o .gitlab-ci.yml

# Включить конкретный паттерн
terraci generate --include "platform/*/*/*" -o .gitlab-ci.yml
```

### Dry Run

```bash
terraci generate --dry-run
```

Вывод:
```
Dry Run Summary:
  Total modules: 15
  Affected modules: 8
  Stages: 6
  Jobs: 16

Execution Order:
  Level 0: [vpc, iam]
  Level 1: [eks, rds, cache]
  Level 2: [app-backend, app-frontend]
  Level 3: [monitoring]
```

### Интеграция с инструментами

```bash
# Извлечь стейджи
terraci generate | yq '.stages'

# Проверить синтаксис
terraci generate | gitlab-ci-lint

# Сравнение с текущим
terraci generate > new.yml && diff .gitlab-ci.yml new.yml
```

## Структура вывода

### Вывод GitLab CI

```yaml
# Глобальные переменные
variables:
  TERRAFORM_BINARY: "terraform"

# Настройки джобов по умолчанию
default:
  image: hashicorp/terraform:1.6

# Стейджи для каждого уровня выполнения
stages:
  - deploy-plan-0
  - deploy-apply-0
  - deploy-plan-1
  - deploy-apply-1

# План-джобы
plan-service-env-region-module:
  stage: deploy-plan-0
  script:
    - cd service/env/region/module
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan

# Apply-джобы
apply-service-env-region-module:
  stage: deploy-apply-0
  needs:
    - plan-service-env-region-module
  script:
    - cd service/env/region/module
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} apply plan.tfplan
```

### Вывод GitHub Actions

```yaml
name: Terraform
on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read
  pull-requests: write

jobs:
  plan-service-env-region-module:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: hashicorp/setup-terraform@v3
      - name: Terraform Plan
        run: |
          cd service/env/region/module
          terraform init
          terraform plan -out=plan.tfplan
    env:
      TF_MODULE_PATH: service/env/region/module
      TF_SERVICE: service
      TF_ENVIRONMENT: env
      TF_REGION: region
      TF_MODULE: module
```

## Примеры интеграции с CI

### GitLab CI

```yaml
# .gitlab-ci.yml
stages:
  - prepare
  - deploy

generate-pipeline:
  stage: prepare
  script:
    - terraci generate --changed-only --base-ref $CI_MERGE_REQUEST_TARGET_BRANCH_NAME -o pipeline.yml
  artifacts:
    paths:
      - pipeline.yml

trigger-deploy:
  stage: deploy
  trigger:
    include:
      - artifact: pipeline.yml
        job: generate-pipeline
```

### Пайплайны для разных окружений

```bash
# Генерация для каждого окружения отдельно
terraci generate --filter environment=production -o production.yml
terraci generate --filter environment=staging -o staging.yml
```

## Обработка ошибок

| Ошибка | Причина | Решение |
|--------|---------|---------|
| No modules found | Неверная глубина или нет .tf файлов | Проверьте конфигурацию structure |
| Circular dependency | Модули зависят друг от друга циклически | Исправьте ссылки на remote_state |
| Git ref not found | Неверный base-ref | Убедитесь, что ветка/коммит существует |

## Смотрите также

- [Генерация пайплайнов](/ru/guide/pipeline-generation) — руководство по генерации CI пайплайнов
- [Настройка GitLab CI](/ru/config/gitlab) — параметры конфигурации GitLab пайплайнов
- [Настройка GitHub Actions](/ru/config/github) — параметры конфигурации GitHub Actions workflow
