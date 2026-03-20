---
title: "Обзор конфигурации"
description: "Справочник конфигурации TerraCi: формат .terraci.yaml и значения по умолчанию"
outline: deep
---

# Конфигурация

TerraCi настраивается через YAML-файл `.terraci.yaml` в корне проекта.

## Файл конфигурации

TerraCi ищет конфигурацию в следующих местах (по порядку):

1. `.terraci.yaml`
2. `.terraci.yml`
3. `terraci.yaml`
4. `terraci.yml`

Или укажите путь вручную:

```bash
terraci -c /path/to/config.yaml generate
```

## Быстрый старт

Создайте конфигурацию командой:

```bash
terraci init
```

Эта команда запускает интерактивный TUI-мастер, который проведёт вас через выбор провайдера, бинарного файла и настройку структуры директорий. Используйте `terraci init --ci` для неинтерактивного режима.

## Полный пример

```yaml
# CI-провайдер (автоопределяется из переменных окружения, если не задан)
provider: gitlab  # или "github"

# Структура директорий
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true

# Фильтры модулей
exclude:
  - "*/test/*"
  - "*/sandbox/*"
  - "*/.terraform/*"

include: []  # Пустой означает все (после исключений)

# Настройки GitLab CI (не используется при provider: github)
gitlab:
  terraform_binary: "terraform"
  image: "hashicorp/terraform:1.6"
  stages_prefix: "deploy"
  parallelism: 5
  plan_enabled: true
  auto_approve: false
  init_enabled: true

  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"

  # Настройки по умолчанию для всех джобов
  job_defaults:
    tags:
      - terraform
      - docker
    before_script:
      - aws sts get-caller-identity
    artifacts:
      paths:
        - "*.tfplan"
      expire_in: "1 day"

# Настройки GitHub Actions (не используется при provider: gitlab)
# github:
#   terraform_binary: "terraform"
#   runs_on: "ubuntu-latest"
#   plan_enabled: true
#   auto_approve: false
#   init_enabled: true
#   permissions:
#     contents: read
#     pull-requests: write

# Настройки бэкенда
backend:
  type: s3
  bucket: my-terraform-state
  region: us-east-1
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

## Секции конфигурации

| Секция | Описание |
|--------|----------|
| [structure](./structure) | Структура директорий и обнаружение модулей |
| [gitlab](./gitlab) | Настройки GitLab CI пайплайнов |
| [github](./github) | Настройки GitHub Actions пайплайнов |
| [filters](./filters) | Паттерны include/exclude |
| [policy](./policy) | Конфигурация OPA-политик |
| [cost](./cost) | Оценка стоимости AWS-инфраструктуры |
| [gitlab-mr](./gitlab-mr) | Интеграция с Merge Request |

## Значения по умолчанию

Если файл конфигурации не найден, используются эти значения:

```yaml
# provider автоопределяется из переменных окружения CI:
#   GITHUB_ACTIONS → github
#   GITLAB_CI / CI_SERVER_URL → gitlab
#   по умолчанию → gitlab

structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true

gitlab:
  terraform_binary: "terraform"
  image: "hashicorp/terraform:1.6"
  stages_prefix: "deploy"
  parallelism: 5
  plan_enabled: true
  auto_approve: false
  init_enabled: true

backend:
  type: s3
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

## Валидация

Проверьте конфигурацию:

```bash
terraci validate
```

Проверяется:
- Наличие обязательных полей
- Корректность значений глубины
- Парсинг паттерна
- Указание образа

## Переменные окружения

Переменные окружения можно использовать в конфигурации CI:

```yaml
gitlab:
  variables:
    AWS_REGION: "${AWS_REGION}"  # Из окружения CI
```

## YAML-якоря

Используйте YAML-якоря для повторяющихся значений:

```yaml
defaults: &defaults
  tags:
    - terraform
    - docker
  before_script:
    - aws sts get-caller-identity

gitlab:
  image: "hashicorp/terraform:1.6"
  job_defaults:
    <<: *defaults
```

## OpenTofu с минимальными образами

Для минимальных образов OpenTofu с не-shell entrypoint используйте объектный формат:

```yaml
gitlab:
  terraform_binary: "tofu"
  image:
    name: "ghcr.io/opentofu/opentofu:1.9-minimal"
    entrypoint: [""]
```
