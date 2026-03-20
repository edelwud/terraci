---
title: "Поддержка OpenTofu"
description: "Настройка TerraCi для OpenTofu: переключение бинарника, миграция и совместимость"
outline: deep
---

# Поддержка OpenTofu

TerraCi полноценно поддерживает [OpenTofu](https://opentofu.org/), open-source форк Terraform.

## Конфигурация

Переключитесь на OpenTofu, обновив `.terraci.yaml`:

```yaml
# Для GitLab CI
gitlab:
  terraform_binary: "tofu"
  image: "ghcr.io/opentofu/opentofu:1.6"

# Для GitHub Actions
github:
  terraform_binary: "tofu"
```

## Как это работает

При установке `terraform_binary: "tofu"`, TerraCi:

1. Устанавливает `TERRAFORM_BINARY=tofu` в переменных пайплайна
2. Использует `${TERRAFORM_BINARY}` во всех скриптах
3. Генерирует команды `tofu init`, `tofu plan`, `tofu apply`

## Сгенерированный пайплайн

```yaml
variables:
  TERRAFORM_BINARY: "tofu"

default:
  image: ghcr.io/opentofu/opentofu:1.6
  before_script:
    - ${TERRAFORM_BINARY} init

plan-platform-prod-vpc:
  script:
    - cd platform/prod/us-east-1/vpc
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan
```

## Официальные образы OpenTofu

| Образ | Описание |
|-------|----------|
| `ghcr.io/opentofu/opentofu:latest` | Последняя стабильная |
| `ghcr.io/opentofu/opentofu:1.6` | Версия 1.6.x |
| `ghcr.io/opentofu/opentofu:1.6.0` | Конкретная версия |

## Миграция с Terraform

1. Обновите `.terraci.yaml` (для вашего провайдера):
   ```yaml
   # GitLab CI
   gitlab:
     terraform_binary: "tofu"
     image: "ghcr.io/opentofu/opentofu:1.6"

   # GitHub Actions
   github:
     terraform_binary: "tofu"
   ```

2. Перегенерируйте пайплайны:
   ```bash
   terraci generate -o .gitlab-ci.yml
   ```

3. Протестируйте с dry-run:
   ```bash
   terraci generate --dry-run
   ```

## Смешанные окружения

Если у вас есть модули и на Terraform, и на OpenTofu, можно переопределить настройки для каждого джоба:

```yaml
# В вашем шаблоне пайплайна
.tofu-job:
  variables:
    TERRAFORM_BINARY: "tofu"
  image: ghcr.io/opentofu/opentofu:1.6

.terraform-job:
  variables:
    TERRAFORM_BINARY: "terraform"
  image: hashicorp/terraform:1.6
```

Затем расширяйте сгенерированные джобы по необходимости.

## Совместимость состояний

OpenTofu совместим с файлами состояния Terraform. Можно:

1. Сохранить существующие state-файлы Terraform
2. Мигрировать на OpenTofu без изменения state
3. Использовать те же S3/GCS бэкенды

Разрешение зависимостей TerraCi работает идентично для обоих инструментов.

### Постепенная миграция

Мигрируйте модуль за модулем, используя переопределение джобов:

```yaml
# Переопределение конкретных джобов для OpenTofu
apply-platform-prod-vpc:
  extends: .tofu-job
```

## Совместимость версий

TerraCi работает с:

| Инструмент | Поддерживаемые версии |
|------------|----------------------|
| Terraform | 0.12+ |
| OpenTofu | 1.0+ |

Парсинг HCL совместим с обоими инструментами.

## Пользовательский путь к бинарнику

Если бинарник имеет нестандартное имя или путь:

```yaml
# GitLab
gitlab:
  terraform_binary: "/usr/local/bin/tofu-1.6"
  before_script:
    - ${TERRAFORM_BINARY} init

# GitHub Actions
github:
  terraform_binary: "/usr/local/bin/tofu-1.6"
```

## Переменные окружения

Настройте переменные окружения, специфичные для OpenTofu:

```yaml
# GitLab
gitlab:
  variables:
    TERRAFORM_BINARY: "tofu"
    TF_CLI_CONFIG_FILE: "/etc/tofu/config.tfrc"
    TOFU_LOG: "INFO"

# GitHub Actions
github:
  variables:
    TERRAFORM_BINARY: "tofu"
    TF_CLI_CONFIG_FILE: "/etc/tofu/config.tfrc"
    TOFU_LOG: "INFO"
```

## Следующие шаги

- [Быстрый старт](/ru/guide/getting-started) — установка и первые шаги с TerraCi
- [Настройка GitLab CI](/ru/config/gitlab) — все опции конфигурации пайплайна
