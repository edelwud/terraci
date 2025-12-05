# Конфигурация GitLab CI

Секция `gitlab` управляет генерацией GitLab CI пайплайнов.

## Параметры

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `terraform_binary` | string | `terraform` | Бинарный файл Terraform/OpenTofu |
| `terraform_image` | string | `hashicorp/terraform:1.6` | Docker-образ |
| `stages_prefix` | string | `deploy` | Префикс названий стейджей |
| `parallelism` | int | `5` | Макс. параллельных джобов |
| `plan_enabled` | bool | `true` | Генерировать plan-джобы |
| `auto_approve` | bool | `false` | Автоматический apply |
| `before_script` | []string | `["${TERRAFORM_BINARY} init"]` | Команды перед джобом |
| `after_script` | []string | `[]` | Команды после джоба |
| `tags` | []string | `[]` | Теги раннеров |
| `variables` | map | `{}` | Переменные пайплайна |
| `artifact_paths` | []string | `["*.tfplan"]` | Пути артефактов |

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

## terraform_image

Docker-образ для выполнения джобов:

```yaml
gitlab:
  terraform_image: "hashicorp/terraform:1.6"
```

Примеры образов:
- `hashicorp/terraform:1.6` — официальный Terraform
- `hashicorp/terraform:latest` — последняя версия
- `ghcr.io/opentofu/opentofu:1.6` — OpenTofu
- `registry.example.com/terraform:custom` — кастомный образ

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

## before_script / after_script

Команды до и после основного скрипта:

```yaml
gitlab:
  before_script:
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} validate
  after_script:
    - echo "Джоб завершен"
```

## tags

Теги для выбора GitLab Runner:

```yaml
gitlab:
  tags:
    - terraform
    - docker
    - aws
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

## artifact_paths

Пути для сохранения артефактов:

```yaml
gitlab:
  artifact_paths:
    - "*.tfplan"
    - "*.json"
```

## Полный пример

```yaml
gitlab:
  # Terraform/OpenTofu
  terraform_binary: "terraform"
  terraform_image: "hashicorp/terraform:1.6"

  # Стейджи
  stages_prefix: "deploy"
  parallelism: 5

  # Поведение
  plan_enabled: true
  auto_approve: false

  # Скрипты
  before_script:
    - ${TERRAFORM_BINARY} init -backend-config="bucket=${TF_STATE_BUCKET}"
    - ${TERRAFORM_BINARY} validate
  after_script:
    - ${TERRAFORM_BINARY} output -json > outputs.json

  # Раннеры
  tags:
    - terraform
    - docker
    - production

  # Переменные
  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"
    TF_CLI_ARGS_plan: "-parallelism=30"
    TF_CLI_ARGS_apply: "-parallelism=30"

  # Артефакты
  artifact_paths:
    - "*.tfplan"
    - "outputs.json"
```

## Генерируемая структура джоба

```yaml
plan-platform-prod-eu-central-1-vpc:
  stage: deploy-plan-0
  image: hashicorp/terraform:1.6
  variables:
    TF_MODULE_PATH: "platform/prod/eu-central-1/vpc"
    TF_SERVICE: "platform"
    TF_ENVIRONMENT: "prod"
    TF_REGION: "eu-central-1"
    TF_MODULE: "vpc"
  before_script:
    - ${TERRAFORM_BINARY} init
  script:
    - cd platform/prod/eu-central-1/vpc
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan
  artifacts:
    paths:
      - platform/prod/eu-central-1/vpc/plan.tfplan
    expire_in: 1 day
  tags:
    - terraform
    - docker
  resource_group: platform/prod/eu-central-1/vpc
```

## Конфигурации для разных окружений

### Development

```yaml
gitlab:
  terraform_image: "hashicorp/terraform:1.6"
  plan_enabled: false
  auto_approve: true
  tags:
    - dev
```

### Production

```yaml
gitlab:
  terraform_image: "hashicorp/terraform:1.6"
  plan_enabled: true
  auto_approve: false
  parallelism: 3
  tags:
    - production
    - secure
```
