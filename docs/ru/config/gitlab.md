# Конфигурация GitLab CI

Секция `gitlab` управляет генерацией GitLab CI пайплайнов.

## Параметры

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `terraform_binary` | string | `terraform` | Бинарный файл Terraform/OpenTofu |
| `image` | string/object | `hashicorp/terraform:1.6` | Docker-образ (строка или объект с name/entrypoint) |
| `stages_prefix` | string | `deploy` | Префикс названий стейджей |
| `parallelism` | int | `5` | Макс. параллельных джобов |
| `plan_enabled` | bool | `true` | Генерировать plan-джобы |
| `auto_approve` | bool | `false` | Автоматический apply |
| `cache_enabled` | bool | `false` | Кеширование .terraform |
| `init_enabled` | bool | `true` | Авто-инициализация terraform после cd |
| `variables` | map | `{}` | Переменные пайплайна |
| `rules` | []object | `[]` | Правила workflow пайплайна |
| `job_defaults` | object | `null` | Настройки по умолчанию для всех джобов |
| `overwrites` | []object | `[]` | Переопределения для plan/apply джобов |

## terraform_binary

Указывает исполняемый файл для Terraform-команд:

```yaml
gitlab:
  terraform_binary: "terraform"  # Стандартный Terraform
```

Для OpenTofu:
```yaml
gitlab:
  terraform_binary: "tofu"
```

Значение экспортируется как переменная `TERRAFORM_BINARY` и используется в скриптах:
```yaml
before_script:
  - ${TERRAFORM_BINARY} init
script:
  - ${TERRAFORM_BINARY} plan
```

## image

Docker-образ для выполнения джобов (в `default` секции). Поддерживает как простой строковый формат, так и объектный формат с переопределением entrypoint.

**Строковый формат** (простой):
```yaml
gitlab:
  image: "hashicorp/terraform:1.6"
```

**Объектный формат** (с entrypoint):
```yaml
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

::: warning Устаревшее
Поле `terraform_image` устарело. Используйте `image`.
:::

## stages_prefix

Префикс для названий стейджей пайплайна:

```yaml
gitlab:
  stages_prefix: "deploy"
```

Генерирует стейджи:
- `deploy-plan-0`, `deploy-apply-0`
- `deploy-plan-1`, `deploy-apply-1`
- и т.д.

Другие примеры:
```yaml
stages_prefix: "terraform"  # terraform-plan-0, terraform-apply-0
stages_prefix: "infra"      # infra-plan-0, infra-apply-0
```

## parallelism

Максимальное количество параллельных джобов на стейдж:

```yaml
gitlab:
  parallelism: 5
```

При 10 модулях без зависимостей и `parallelism: 5` — выполняются по 5 джобов одновременно.

## plan_enabled

Включает отдельный стейдж для `terraform plan`:

```yaml
gitlab:
  plan_enabled: true
```

С `plan_enabled: true`:
```
deploy-plan-0 → deploy-apply-0 → deploy-plan-1 → deploy-apply-1
```

С `plan_enabled: false`:
```
deploy-apply-0 → deploy-apply-1
```

## auto_approve

Автоматический apply без ручного подтверждения:

```yaml
gitlab:
  auto_approve: false  # Требует ручного подтверждения
```

```yaml
gitlab:
  auto_approve: true   # Автоматический apply
```

::: warning Осторожно
`auto_approve: true` применяет изменения без подтверждения. Используйте только для dev/test окружений.
:::

## cache_enabled

Включает кеширование директории `.terraform` для каждого модуля:

```yaml
gitlab:
  cache_enabled: true
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

## init_enabled

Автоматический запуск `terraform init` после перехода в директорию модуля:

```yaml
gitlab:
  init_enabled: true   # По умолчанию
```

Генерируемый скрипт:
```yaml
script:
  - cd platform/prod/eu-central-1/vpc
  - ${TERRAFORM_BINARY} init      # Добавляется автоматически
  - ${TERRAFORM_BINARY} plan ...
```

## variables

Переменные окружения для пайплайна:

```yaml
gitlab:
  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"
    AWS_DEFAULT_REGION: "eu-central-1"
```

Автоматически добавляемые переменные:
- `TERRAFORM_BINARY` — значение из `terraform_binary`

## rules

Правила workflow для условного запуска пайплайна. Определяют, когда создаются пайплайны.

```yaml
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
gitlab:
  # Terraform/OpenTofu
  terraform_binary: "terraform"
  image: "hashicorp/terraform:1.6"

  # Структура пайплайна
  stages_prefix: "deploy"
  parallelism: 5
  plan_enabled: true
  auto_approve: false
  cache_enabled: true
  init_enabled: true

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
  TERRAFORM_BINARY: "terraform"
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
  - deploy-plan-0
  - deploy-apply-0

plan-platform-prod-eu-central-1-vpc:
  stage: deploy-plan-0
  variables:
    TF_MODULE_PATH: "platform/prod/eu-central-1/vpc"
    TF_SERVICE: "platform"
    TF_ENVIRONMENT: "prod"
    TF_REGION: "eu-central-1"
    TF_MODULE: "vpc"
  script:
    - cd platform/prod/eu-central-1/vpc
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan
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
  stage: deploy-apply-0
  script:
    - cd platform/prod/eu-central-1/vpc
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} apply plan.tfplan
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

## Конфигурации для разных окружений

### Development

```yaml
gitlab:
  image: "hashicorp/terraform:1.6"
  plan_enabled: false
  auto_approve: true

  job_defaults:
    tags:
      - dev
```

### Production

```yaml
gitlab:
  image: "hashicorp/terraform:1.6"
  plan_enabled: true
  auto_approve: false
  parallelism: 3

  job_defaults:
    tags:
      - production
      - secure

  overwrites:
    - type: apply
      rules:
        - if: '$CI_COMMIT_BRANCH == "main"'
          when: manual
```
