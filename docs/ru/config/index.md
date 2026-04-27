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
# Структура директорий
structure:
  pattern: "{service}/{environment}/{region}/{module}"

# Фильтры модулей
exclude:
  - "*/test/*"
  - "*/sandbox/*"
  - "*/.terraform/*"

include: []  # Пустой означает все (после исключений)

# Настройки плагинов
plugins:
  # Настройки GitLab CI
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

  # Настройки GitHub Actions
  # github:
  #   terraform_binary: "terraform"
  #   runs_on: "ubuntu-latest"
  #   plan_enabled: true
  #   auto_approve: false
  #   init_enabled: true
  #   permissions:
  #     contents: read
  #     pull-requests: write
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
| [summary](./summary) | Настройки сводного комментария MR/PR |
| [tfupdate](./tfupdate) | Разрешение зависимостей Terraform и синхронизация lock-файлов |

## Значения по умолчанию

Если файл конфигурации не найден, используются эти значения:

```yaml
# выбор provider:
#   переменная окружения TERRACI_PROVIDER
#   автоопределение CI-окружения
#   единственный активный провайдер

structure:
  pattern: "{service}/{environment}/{region}/{module}"

plugins:
  gitlab:
    terraform_binary: "terraform"
    image: "hashicorp/terraform:1.6"
    stages_prefix: "deploy"
    parallelism: 5
    plan_enabled: true
    auto_approve: false
    init_enabled: true
```

## Валидация

Проверьте конфигурацию:

```bash
terraci validate
```

Проверяется:
- Наличие обязательных полей
- Парсинг паттерна
- Указание образа

## Переменные окружения

Переменные окружения можно использовать в конфигурации CI:

```yaml
plugins:
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

plugins:
  gitlab:
    image: "hashicorp/terraform:1.6"
    job_defaults:
      <<: *defaults
```

## OpenTofu с минимальными образами

Для минимальных образов OpenTofu с не-shell entrypoint используйте объектный формат:

```yaml
plugins:
  gitlab:
    terraform_binary: "tofu"
    image:
      name: "ghcr.io/opentofu/opentofu:1.9-minimal"
      entrypoint: [""]
```
