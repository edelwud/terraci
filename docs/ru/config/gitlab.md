---
title: "GitLab CI"
description: "Настройка генерации пайплайнов: образ, стадии, джобы, overwrites, секреты и OIDC"
outline: deep
---

# Конфигурация GitLab CI

Секция `gitlab` управляет генерацией GitLab CI пайплайнов. Эта секция используется только когда провайдер — `gitlab` (автоопределяется из переменных окружения). При использовании GitHub Actions применяется секция `github`. См. [Конфигурация GitHub Actions](/ru/config/github) для эквивалента GitHub.

## Параметры

::: info Настройки выполнения
:::

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `image` | string/object | `hashicorp/terraform:1.6` | Docker-образ (строка или объект с name/entrypoint) |
| `stages_prefix` | string | `deploy` | Префикс названий стейджей |
| `variables` | map | `{}` | Переменные пайплайна |
| `rules` | []object | `[]` | Правила workflow пайплайна |
| `job_defaults` | object | `null` | Настройки по умолчанию для всех джобов |
| `overwrites` | []object | `[]` | Переопределения для plan/apply джобов |

## image

Docker-образ для выполнения джобов (в `default` секции). Поддерживает как простой строковый формат, так и объектный формат с переопределением entrypoint.

**Строковый формат** (простой):
```yaml
extensions:
  gitlab:
    image: "hashicorp/terraform:1.6"
```

**Объектный формат** (с entrypoint):
```yaml
extensions:
  gitlab:
    image:
      name: "ghcr.io/opentofu/opentofu:1.9-minimal"
      entrypoint: [""]
```

Примеры образов:
- `hashicorp/terraform:1.6` — официальный Terraform
- `hashicorp/terraform:latest` — последняя версия
- `ghcr.io/opentofu/opentofu:1.6` — OpenTofu
- `registry.example.com/terraform:custom` — кастомный образ

::: tip OpenTofu Minimal
Минимальные образы OpenTofu (например, `opentofu:1.9-minimal`) имеют не-shell entrypoint. Используйте объектный формат с `entrypoint: [""]` для совместимости с GitLab CI.
:::


## stages_prefix

Префикс для названий стейджей пайплайна:

```yaml
extensions:
  gitlab:
    stages_prefix: "deploy"
```

Генерирует стейджи:
- `deploy-0`, `deploy-1`
- `deploy-2`, `deploy-3`
- и т.д.

Другие примеры:
```yaml
stages_prefix: "terraform"  # terraform-0, terraform-1
stages_prefix: "infra"      # infra-0, infra-1
```


Генерирует только plan-джобы, без apply. Полезно для read-only пайплайнов на ветках/MR-ах.

```yaml
extensions:
  gitlab:
```

CLI-флаг `--plan-only` команды `terraci generate` переопределяет это значение.

::: tip Plan-стейдж в целом
:::

## Ручной apply

Поведение apply-джобов настраивается через `job_defaults` или `overwrites`.
Например, сделать apply ручным:

```yaml
extensions:
  gitlab:
    overwrites:
      - type: apply
        when: manual
```


Включает кеширование директории `.terraform` для каждого модуля:

```yaml
extensions:
  gitlab:
```

При включенном кешировании каждый джоб получает конфигурацию кеша:

```yaml
plan-platform-prod-eu-central-1-vpc:
  cache:
    key: platform-prod-eu-central-1-vpc
    paths:
      - platform/prod/eu-central-1/vpc/.terraform/
```

Ключ кеша формируется из пути к модулю, где слеши заменяются на дефисы.

::: tip Преимущества
- Ускорение `terraform init` — провайдеры загружаются из кеша
- Экономия трафика — модули и провайдеры не скачиваются повторно
- Работает на уровне отдельных модулей — независимое кеширование
:::

## cache


```yaml
extensions:
  gitlab:
    cache:
      enabled: true
      key: "terraform-{service}-{environment}-{module}"
      policy: pull-push
      paths:
        - "{module_path}/.terraform/"
        - "{module_path}/.terraform.lock.hcl"
```

Поддерживаемые плейсхолдеры для `cache.key` и `cache.paths`:

- `{module_path}`
- `{service}`
- `{environment}`
- `{region}`
- `{module}`

Если `cache.paths` не указан, TerraCi сохраняет текущее поведение по умолчанию:

```yaml
cache:
  paths:
    - "{module_path}/.terraform/"
```

## variables

Переменные окружения для пайплайна:

```yaml
extensions:
  gitlab:
    variables:
      TF_IN_AUTOMATION: "true"
      TF_INPUT: "false"
      AWS_DEFAULT_REGION: "eu-central-1"
```

TerraCi не добавляет скрытые provider-level переменные для Terraform binary:
выбранный `execution.binary` записывается в Terraform operations внутри IR, а
GitLab generator рендерит команды напрямую из этих operations.

## rules

Правила workflow для условного запуска пайплайна. Определяют, когда создаются пайплайны.

```yaml
extensions:
  gitlab:
    rules:
      - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
        when: always
      - if: '$CI_COMMIT_BRANCH == "main"'
        when: always
      - if: '$CI_COMMIT_TAG'
        when: never
      - when: never
```

Генерируемый результат:

```yaml
workflow:
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
      when: always
    - if: '$CI_COMMIT_BRANCH == "main"'
      when: always
    - when: never
```

Каждое правило может содержать:
- `if` — условное выражение
- `when` — когда выполнять: `always`, `never`, `on_success`, `manual`, `delayed`
- `changes` — паттерны файлов, при изменении которых срабатывает правило

## job_defaults

Настройки по умолчанию, применяемые ко всем генерируемым джобам (и plan, и apply). Применяются перед `overwrites`, поэтому overwrites могут переопределять job_defaults.

Доступные поля:
- `image` — Docker-образ для всех джобов
- `id_tokens` — OIDC токены для всех джобов
- `secrets` — секреты для всех джобов
- `before_script` — команды перед каждым джобом
- `after_script` — команды после каждого джоба
- `artifacts` — конфигурация артефактов
- `tags` — теги раннеров
- `rules` — правила на уровне джоба
- `variables` — дополнительные переменные

**Пример: Общие настройки для всех джобов**
```yaml
extensions:
  gitlab:
    job_defaults:
      tags:
        - terraform
        - docker
      rules:
        - if: '$CI_COMMIT_BRANCH == "main"'
          when: on_success
      variables:
        CUSTOM_VAR: "value"
```

## overwrites

Переопределения на уровне джобов для plan или apply. Позволяет настраивать разные типы джобов с разными параметрами. Применяются после `job_defaults`.

Каждое переопределение содержит:
- `type` — какие джобы переопределять: `plan` или `apply`
- `image` — переопределить Docker-образ
- `id_tokens` — переопределить OIDC токены
- `secrets` — переопределить секреты
- `before_script` — переопределить before_script
- `after_script` — переопределить after_script
- `artifacts` — переопределить конфигурацию артефактов
- `tags` — переопределить теги раннеров
- `rules` — установить правила на уровне джоба
- `variables` — переопределить/добавить переменные

**Пример: Разные образы для plan и apply**
```yaml
extensions:
  gitlab:
    image: "hashicorp/terraform:1.6"

    overwrites:
      - type: plan
        image: "custom/terraform-plan:1.6"
        tags:
          - plan-runner

      - type: apply
        image: "custom/terraform-apply:1.6"
        tags:
          - apply-runner
          - production
```

**Пример: Добавление правил для apply-джобов**
```yaml
extensions:
  gitlab:
    overwrites:
      - type: apply
        rules:
          - if: '$CI_COMMIT_BRANCH == "main"'
            when: manual
          - when: never
```

**Пример: Разные секреты для разных типов джобов**
```yaml
extensions:
  gitlab:
    job_defaults:
      secrets:
        COMMON_SECRET:
          vault: common/secret@namespace

    overwrites:
      - type: apply
        secrets:
          DEPLOY_KEY:
            vault: deploy/key@namespace
            file: true
```

**Пример: job_defaults с overwrites**
```yaml
extensions:
  gitlab:
    # Общие настройки для всех джобов
    job_defaults:
      tags:
        - terraform
      rules:
        - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
          when: on_success

    # Переопределение только для apply-джобов
    overwrites:
      - type: apply
        tags:
          - terraform
          - production
        rules:
          - if: '$CI_COMMIT_BRANCH == "main"'
            when: manual
```

## Полный пример

```yaml
execution:
  binary: terraform
  init_enabled: true

extensions:
  gitlab:
    image: "hashicorp/terraform:1.6"

    # Структура пайплайна
    stages_prefix: "deploy"

    # Переменные пайплайна
    variables:
      TF_IN_AUTOMATION: "true"
      TF_INPUT: "false"

    # Правила workflow
    rules:
      - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
        when: always
      - if: '$CI_COMMIT_BRANCH == "main"'
        when: always

    # Настройки по умолчанию для всех джобов
    job_defaults:
      tags:
        - terraform
        - docker
      before_script:
        - aws sts get-caller-identity
      after_script:
        - echo "Джоб завершен"
      id_tokens:
        AWS_OIDC_TOKEN:
          aud: "https://gitlab.example.com"
      secrets:
        CREDENTIALS:
          vault: ci/terraform/credentials@namespace
      rules:
        - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
          when: on_success

    # Переопределения для джобов (применяются после job_defaults)
    overwrites:
      - type: apply
        tags:
          - production
          - secure
        rules:
          - if: '$CI_COMMIT_BRANCH == "main"'
            when: manual
```

## Генерируемая структура

```yaml
variables:
  TF_IN_AUTOMATION: "true"
  TF_INPUT: "false"

default:
  image: hashicorp/terraform:1.6

workflow:
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
      when: always
    - if: '$CI_COMMIT_BRANCH == "main"'
      when: always

stages:
  - deploy-0
  - deploy-1

plan-platform-prod-eu-central-1-vpc:
  stage: deploy-0
  variables:
    TF_MODULE_PATH: "platform/prod/eu-central-1/vpc"
    TF_SERVICE: "platform"
    TF_ENVIRONMENT: "prod"
    TF_REGION: "eu-central-1"
    TF_MODULE: "vpc"
  script:
    - cd platform/prod/eu-central-1/vpc
    - terraform init
    - terraform plan -out=plan.tfplan
  tags:
    - terraform
    - docker
  before_script:
    - aws sts get-caller-identity
  after_script:
    - echo "Джоб завершен"
  id_tokens:
    AWS_OIDC_TOKEN:
      aud: "https://gitlab.example.com"
  secrets:
    CREDENTIALS:
      vault: ci/terraform/credentials@namespace
  artifacts:
    paths:
      - platform/prod/eu-central-1/vpc/plan.tfplan
    expire_in: 1 day
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
      when: on_success
  cache:
    key: platform-prod-eu-central-1-vpc
    paths:
      - platform/prod/eu-central-1/vpc/.terraform/
  resource_group: platform/prod/eu-central-1/vpc

apply-platform-prod-eu-central-1-vpc:
  stage: deploy-1
  script:
    - cd platform/prod/eu-central-1/vpc
    - terraform init
    - terraform apply plan.tfplan
  needs:
    - plan-platform-prod-eu-central-1-vpc
  tags:
    - production
    - secure
  before_script:
    - aws sts get-caller-identity
  after_script:
    - echo "Джоб завершен"
  id_tokens:
    AWS_OIDC_TOKEN:
      aud: "https://gitlab.example.com"
  secrets:
    CREDENTIALS:
      vault: ci/terraform/credentials@namespace
  rules:
    - if: '$CI_COMMIT_BRANCH == "main"'
      when: manual
  resource_group: platform/prod/eu-central-1/vpc
```

## Переменные джобов

Каждый джоб получает переменные окружения, которые динамически генерируются из имён сегментов, определённых в `structure.pattern`. Для паттерна по умолчанию `{service}/{environment}/{region}/{module}` переменные следующие:

| Переменная | Описание | Пример |
|------------|----------|--------|
| `TF_MODULE_PATH` | Относительный путь к модулю | `platform/prod/us-east-1/vpc` |
| `TF_SERVICE` | Название сервиса | `platform` |
| `TF_ENVIRONMENT` | Окружение | `prod` |
| `TF_REGION` | Регион | `us-east-1` |
| `TF_MODULE` | Название модуля | `vpc` |

Если вы используете пользовательский паттерн, например `{team}/{stack}/{datacenter}/{component}`, переменные будут `TF_TEAM`, `TF_STACK`, `TF_DATACENTER` и `TF_COMPONENT`. Имена переменных всегда формируются путём приведения имени сегмента к верхнему регистру и добавления префикса `TF_`.

## Конфигурации для разных окружений

### Development

```yaml
execution:

extensions:
  gitlab:
    image: "hashicorp/terraform:1.6"

    job_defaults:
      tags:
        - dev
```

### Production

```yaml
execution:

extensions:
  gitlab:
    image: "hashicorp/terraform:1.6"

    job_defaults:
      when: manual
      tags:
        - production
        - secure

    overwrites:
      - type: apply
        rules:
          - if: '$CI_COMMIT_BRANCH == "main"'
            when: manual
```

## Смотрите также

- [Конфигурация summary](/ru/config/summary) — комментарии MR/PR с результатами plan и отчётами плагинов
- [Конфигурация GitHub Actions](/ru/config/github) — эквивалентная конфигурация для GitHub Actions
- [Генерация пайплайнов](/ru/guide/pipeline-generation) — руководство по генерации CI пайплайнов
