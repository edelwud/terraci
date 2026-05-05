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
# Опциональная сервисная директория (по умолчанию .terraci) для рантайм-артефактов и отчётов
service_dir: .terraci

# Структура директорий
structure:
  pattern: "{service}/{environment}/{region}/{module}"

# Фильтры модулей
exclude:
  - "*/test/*"
  - "*/sandbox/*"
  - "*/.terraform/*"

include: []  # Пустой означает все (после исключений)

# Общие настройки выполнения Terraform/OpenTofu
execution:
  binary: terraform        # или "tofu"
  init_enabled: true       # автоматически вызывать terraform init
  plan_enabled: true       # генерировать plan-джобы
  plan_mode: standard      # "standard" или "detailed"
  parallelism: 4           # размер пула воркеров для local-exec

# Настройки расширений
extensions:
  # Настройки GitLab CI
  gitlab:
    image:
      name: hashicorp/terraform:1.6
    stages_prefix: "deploy"
    parallelism: 5
    auto_approve: false

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
  #   runs_on: "ubuntu-latest"
  #   auto_approve: false
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
service_dir: .terraci

structure:
  pattern: "{service}/{environment}/{region}/{module}"

execution:
  binary: terraform
  init_enabled: true
  plan_enabled: true
  plan_mode: standard
  parallelism: 4
```

Выбор провайдера в рантайме:

1. Переменная `TERRACI_PROVIDER` — явный override
2. Автоопределение CI (`GITLAB_CI`, `GITHUB_ACTIONS`)
3. Единственный сконфигурированный провайдер

Дефолты плагинов GitLab/GitHub (если они активированы) приходят из struct-тегов — см. [`gitlab`](./gitlab) и [`github`](./github).

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
extensions:
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

extensions:
  gitlab:
    image: "hashicorp/terraform:1.6"
    job_defaults:
      <<: *defaults
```

## OpenTofu с минимальными образами

Для минимальных образов OpenTofu с не-shell entrypoint используйте объектный формат и переключите бинарь выполнения:

```yaml
execution:
  binary: tofu

extensions:
  gitlab:
    image:
      name: "ghcr.io/opentofu/opentofu:1.9-minimal"
      entrypoint: [""]
```
