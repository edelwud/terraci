# Конфигурация GitLab CI

Секция `gitlab` управляет генерацией GitLab CI пайплайнов.

## Параметры

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `terraform_binary` | string | `terraform` | Бинарный файл Terraform/OpenTofu |
| `terraform_image` | string/object | `hashicorp/terraform:1.6` | Docker-образ (строка или объект с name/entrypoint) |
| `stages_prefix` | string | `deploy` | Префикс названий стейджей |
| `parallelism` | int | `5` | Макс. параллельных джобов |
| `plan_enabled` | bool | `true` | Генерировать plan-джобы |
| `auto_approve` | bool | `false` | Автоматический apply |
| `cache_enabled` | bool | `false` | Кеширование .terraform |
| `before_script` | []string | `["${TERRAFORM_BINARY} init"]` | Команды перед джобом |
| `after_script` | []string | `[]` | Команды после джоба |
| `tags` | []string | `[]` | Теги раннеров |
| `variables` | map | `{}` | Переменные пайплайна |
| `artifact_paths` | []string | `["*.tfplan"]` | Пути артефактов |
| `id_tokens` | map | `{}` | OIDC токены для облачных провайдеров |
| `rules` | []object | `[]` | Правила выполнения пайплайна |
| `secrets` | map | `{}` | Секреты из внешних хранилищ |

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

Docker-образ для выполнения джобов. Поддерживает как простой строковый формат, так и объектный формат с переопределением entrypoint.

**Строковый формат** (простой):
```yaml
gitlab:
  terraform_image: "hashicorp/terraform:1.6"
```

**Объектный формат** (с entrypoint):
```yaml
gitlab:
  terraform_image:
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

## id_tokens

OIDC токены для аутентификации в облачных провайдерах. Позволяет использовать безопасную аутентификацию без хранения секретов в GitLab.

```yaml
gitlab:
  id_tokens:
    AWS_OIDC_TOKEN:
      aud: "https://gitlab.example.com"
    GCP_OIDC_TOKEN:
      aud: "https://iam.googleapis.com/projects/123456/locations/global/workloadIdentityPools/gitlab-pool/providers/gitlab"
```

Генерируемый результат:

```yaml
default:
  id_tokens:
    AWS_OIDC_TOKEN:
      aud: "https://gitlab.example.com"
```

::: tip AWS OIDC аутентификация
Используйте `id_tokens` с IAM ролями AWS для безопасной аутентификации без хранения ключей:
```yaml
gitlab:
  id_tokens:
    AWS_OIDC_TOKEN:
      aud: "https://gitlab.example.com"
  before_script:
    - >
      export $(printf "AWS_ACCESS_KEY_ID=%s AWS_SECRET_ACCESS_KEY=%s AWS_SESSION_TOKEN=%s"
      $(aws sts assume-role-with-web-identity
      --role-arn ${AWS_ROLE_ARN}
      --role-session-name "GitLabRunner-${CI_PROJECT_ID}-${CI_PIPELINE_ID}"
      --web-identity-token ${AWS_OIDC_TOKEN}
      --duration-seconds 3600
      --query 'Credentials.[AccessKeyId,SecretAccessKey,SessionToken]'
      --output text))
    - ${TERRAFORM_BINARY} init
```
:::

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

## secrets

Секреты из внешних менеджеров секретов (HashiCorp Vault). Секреты инжектируются как переменные окружения или файлы.

**Короткий формат** (рекомендуется):
```yaml
gitlab:
  secrets:
    credentials:
      vault: ci/terraform/gitlab-terraform/credentials@cdp
      file: true
    API_KEY:
      vault: production/api/keys/main@team
```

**Полный формат** (для сложных конфигураций):
```yaml
gitlab:
  secrets:
    AWS_SECRET_ACCESS_KEY:
      vault:
        engine:
          name: kv-v2
          path: secret
        path: aws/credentials
        field: secret_access_key
    DATABASE_PASSWORD:
      vault:
        engine:
          name: kv-v2
          path: secret
        path: production/database
        field: password
      file: true  # Записать в файл вместо переменной окружения
```

Короткий формат `path/to/secret/field@namespace` — стандартный синтаксис GitLab.

::: warning Настройка Vault
Для использования секретов необходима настроенная интеграция GitLab с Vault. См. [документацию GitLab по секретам](https://docs.gitlab.com/ee/ci/secrets/index.html).
:::

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
  cache_enabled: true

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
