---
title: "GitHub Actions"
description: "Настройка генерации GitHub Actions workflow: раннеры, шаги, джобы, overwrites и permissions"
outline: deep
---

# Конфигурация GitHub Actions

Секция `github` управляет генерацией GitHub Actions workflow. Эта секция используется, когда провайдер — `github`: выбран через `TERRACI_PROVIDER=github`, автоопределён из переменной окружения `GITHUB_ACTIONS` или выведен как единственный активный провайдер. Когда провайдер — `gitlab`, эта секция не используется и вместо неё применяется секция `gitlab`. См. [Конфигурация GitLab CI](/ru/config/gitlab) для эквивалента GitLab.

## Параметры

### terraform_binary

**Тип:** `string`
**По умолчанию:** `"terraform"`

Бинарный файл Terraform/OpenTofu.

```yaml
extensions:
  github:
    terraform_binary: "terraform"  # или "tofu"
```

### runs_on

**Тип:** `string`
**По умолчанию:** `"ubuntu-latest"`

Метка раннера GitHub Actions для джобов.

```yaml
extensions:
  github:
    runs_on: "ubuntu-latest"
    # runs_on: "self-hosted"
```

### container

**Тип:** `object` (опциональный)
**По умолчанию:** нет

Опционально запускать джобы внутри контейнера. Поддерживает строковый и объектный формат.

```yaml
extensions:
  github:
    container:
      name: "hashicorp/terraform:1.6"
      entrypoint: [""]
```

### env

**Тип:** `map[string]string`
**По умолчанию:** `{}`

Переменные окружения на уровне workflow.

```yaml
extensions:
  github:
    env:
      TF_IN_AUTOMATION: "true"
      TF_INPUT: "false"
      AWS_DEFAULT_REGION: "us-east-1"
```

### plan_enabled

**Тип:** `boolean`
**По умолчанию:** `true`

Генерировать отдельные план-джобы.

```yaml
extensions:
  github:
    plan_enabled: true   # plan + apply джобы
    # plan_enabled: false  # только apply
```

### plan_only

**Тип:** `boolean`
**По умолчанию:** `false`

Генерировать только план-джобы без apply-джобов.

```yaml
extensions:
  github:
    plan_only: true
```

### auto_approve

**Тип:** `boolean`
**По умолчанию:** `false`

Автоматический apply без защиты через environment.

```yaml
extensions:
  github:
    auto_approve: false  # Apply использует environment protection
    # auto_approve: true   # Apply выполняется автоматически
```

### init_enabled

**Тип:** `boolean`
**По умолчанию:** `true`

Автоматический запуск `terraform init` перед командами terraform.

```yaml
extensions:
  github:
    init_enabled: true
```

### permissions

**Тип:** `map[string]string`
**По умолчанию:** `{}`

Permissions на уровне workflow. Необходимы для комментариев в PR и аутентификации OIDC.

```yaml
extensions:
  github:
    permissions:
      contents: read
      pull-requests: write
      id-token: write        # Необходимо для OIDC
```

### job_defaults

**Тип:** `object`
**По умолчанию:** `null`

Настройки по умолчанию, применяемые ко всем генерируемым джобам (и plan, и apply). Применяются перед `overwrites`.

Доступные поля:
- `runs_on` — переопределить метку раннера для всех джобов
- `container` — контейнерный образ для всех джобов
- `env` — дополнительные переменные окружения
- `steps_before` — дополнительные шаги перед командами terraform
- `steps_after` — дополнительные шаги после команд terraform

**Пример: Общие шаги настройки для всех джобов**
```yaml
extensions:
  github:
    job_defaults:
      steps_before:
        - uses: actions/checkout@v4
        - uses: hashicorp/setup-terraform@v3
        - name: Configure AWS credentials
          uses: aws-actions/configure-aws-credentials@v4
          with:
            role-to-assume: arn:aws:iam::123456789012:role/terraform
            aws-region: us-east-1
      steps_after:
        - name: Upload logs
          run: echo "Job completed"
```

Каждый шаг в `steps_before` / `steps_after` поддерживает:
- `name` — отображаемое имя шага
- `uses` — ссылка на GitHub Action (например, `actions/checkout@v4`)
- `with` — входные параметры действия в виде пар ключ-значение
- `run` — shell-команда для выполнения
- `env` — переменные окружения на уровне шага

### overwrites

**Тип:** `array`
**По умолчанию:** `[]`

Переопределения на уровне джобов для plan или apply. Применяются после `job_defaults`.

Каждое переопределение содержит:
- `type` — какие джобы переопределять: `plan` или `apply`
- `runs_on` — переопределить метку раннера
- `container` — переопределить контейнерный образ
- `env` — переопределить/добавить переменные окружения
- `steps_before` — переопределить шаги перед командами terraform
- `steps_after` — переопределить шаги после команд terraform

**Пример: Разные раннеры для plan и apply**
```yaml
extensions:
  github:
    overwrites:
      - type: plan
        runs_on: ubuntu-latest

      - type: apply
        runs_on: self-hosted
        env:
          DEPLOY_ENV: "production"
```

**Пример: Дополнительные шаги для apply-джобов**
```yaml
extensions:
  github:
    overwrites:
      - type: apply
        steps_before:
          - uses: actions/checkout@v4
          - uses: hashicorp/setup-terraform@v3
          - name: Approve deployment
            run: echo "Deploying..."
```

### pr

**Тип:** `object`
**По умолчанию:** `null`

Настройки интеграции с Pull Request. Эквивалент секции `mr` в GitLab.

```yaml
extensions:
  github:
    pr:
      comment:
        enabled: true
        on_changes_only: false
```

#### pr.comment

Управление поведением комментариев в PR:

| Поле | Тип | По умолчанию | Описание |
|------|-----|--------------|----------|
| `enabled` | bool | true | Включить комментарии в PR |
| `on_changes_only` | bool | false | Комментировать только при наличии изменений |
| `include_details` | bool | true | Включить полный вывод плана в раскрывающихся секциях |

## Полный пример

```yaml
extensions:
  github:
    # Конфигурация бинарного файла
    terraform_binary: "terraform"
    runs_on: "ubuntu-latest"

    # Настройки workflow
    plan_enabled: true
    auto_approve: false
    init_enabled: true

    # Переменные окружения на уровне workflow
    env:
      TF_IN_AUTOMATION: "true"
      TF_INPUT: "false"

    # Permissions (необходимы для комментариев в PR и OIDC)
    permissions:
      contents: read
      pull-requests: write
      id-token: write

    # Настройки по умолчанию для всех джобов
    job_defaults:
      steps_before:
        - uses: actions/checkout@v4
        - uses: hashicorp/setup-terraform@v3
        - name: Configure AWS credentials
          uses: aws-actions/configure-aws-credentials@v4
          with:
            role-to-assume: arn:aws:iam::123456789012:role/terraform
            aws-region: us-east-1

    # Переопределения для джобов (применяются после job_defaults)
    overwrites:
      - type: apply
        runs_on: self-hosted

    # Интеграция с Pull Request
    pr:
      comment:
        enabled: true
        on_changes_only: false
```

## Переменные джобов

Как и в GitLab, каждый джоб получает переменные окружения, динамически генерируемые из сегментов `structure.pattern`. Для паттерна по умолчанию `{service}/{environment}/{region}/{module}`:

| Переменная | Описание | Пример |
|------------|----------|--------|
| `TF_MODULE_PATH` | Относительный путь к модулю | `platform/prod/us-east-1/vpc` |
| `TF_SERVICE` | Название сервиса | `platform` |
| `TF_ENVIRONMENT` | Окружение | `prod` |
| `TF_REGION` | Регион | `us-east-1` |
| `TF_MODULE` | Название модуля | `vpc` |

Имена переменных формируются путём приведения имени сегмента к верхнему регистру и добавления префикса `TF_`.

## Сравнение с конфигурацией GitLab

| Возможность | GitLab (`gitlab:`) | GitHub (`github:`) |
|-------------|-------------------|-------------------|
| Выбор раннера | `job_defaults.tags` | `runs_on` |
| Контейнерный образ | `image` | `container` (опционально) |
| Команды перед джобом | `job_defaults.before_script` | `job_defaults.steps_before` |
| Команды после джоба | `job_defaults.after_script` | `job_defaults.steps_after` |
| Переменные пайплайна | `variables` | `env` |
| Контроль доступа | `rules` | `permissions` |
| Интеграция MR/PR | секция `mr` | секция `pr` |
| Секреты | `secrets` (Vault) | Через шаги GitHub Action |
| OIDC токены | `id_tokens` | `permissions.id-token: write` |
| Кеширование | `cache_enabled` | Через `actions/cache` в шагах |
| Префикс стейджей | `stages_prefix` | N/A (используются зависимости джобов) |

## Смотрите также

- [Конфигурация GitLab CI](/ru/config/gitlab) — эквивалентная конфигурация для GitLab CI
- [Интеграция с Merge Request](/ru/config/gitlab-mr) — комментарии в MR с результатами plan и политик
- [Генерация пайплайнов](/ru/guide/pipeline-generation) — руководство по генерации CI пайплайнов
