---
title: "Поддержка OpenTofu"
description: "Настройка TerraCi для OpenTofu: переключение бинарника, миграция и совместимость"
outline: deep
---

# Поддержка OpenTofu

TerraCi полноценно поддерживает [OpenTofu](https://opentofu.org/), open-source форк Terraform.

## Конфигурация

Переключитесь на OpenTofu, задав бинарник в верхнеуровневой секции `execution:`:

```yaml
execution:
  binary: tofu

extensions:
  # Для GitLab CI
  gitlab:
    image: "ghcr.io/opentofu/opentofu:1.6"

  # Для GitHub Actions (бинарник на стороне провайдера не задаётся; образ задают шаги воркфлоу)
  github:
    runs_on: ubuntu-latest
```

## Как это работает

При установке `execution.binary: tofu`, TerraCi:

1. Устанавливает `TERRAFORM_BINARY=tofu` в переменных пайплайна
2. Использует `${TERRAFORM_BINARY}` во всех генерируемых скриптах
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
  # ...

apply-platform-prod-vpc:
  script:
    - cd platform/prod/us-east-1/vpc
    - ${TERRAFORM_BINARY} apply plan.tfplan
  # ...
```

## Официальные образы OpenTofu

Используйте официальные Docker-образы OpenTofu:

| Образ | Описание |
|-------|----------|
| `ghcr.io/opentofu/opentofu:latest` | Последняя стабильная |
| `ghcr.io/opentofu/opentofu:1.6` | Версия 1.6.x |
| `ghcr.io/opentofu/opentofu:1.6.0` | Конкретная версия |

## Смешанные окружения

Если у вас есть модули и на Terraform, и на OpenTofu, можно переопределять настройки на уровне отдельных джобов:

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

## Руководство по миграции

### С Terraform на OpenTofu

1. Обновите `.terraci.yaml`:
   ```yaml
   execution:
     binary: tofu

   extensions:
     # GitLab CI
     gitlab:
       image: "ghcr.io/opentofu/opentofu:1.6"

     # GitHub Actions
     github:
       runs_on: ubuntu-latest
   ```

2. Перегенерируйте пайплайны:
   ```bash
   terraci generate -o .gitlab-ci.yml
   ```

3. Протестируйте с dry-run:
   ```bash
   terraci generate --dry-run
   ```

4. Закоммитьте и запушьте

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

Если бинарник имеет нестандартное имя или путь, укажите его в `execution.binary`. TerraCi экспортирует значение в переменную `TERRAFORM_BINARY` и использует её везде, где генерирует команду Terraform:

```yaml
execution:
  binary: "/usr/local/bin/tofu-1.6"
```

## Переменные окружения

`TERRAFORM_BINARY` экспортируется автоматически из `execution.binary`. Другие переменные окружения, специфичные для OpenTofu, добавляйте через `execution.env` (на весь воркфлоу) или на уровне провайдера:

```yaml
execution:
  binary: tofu
  env:
    TF_CLI_CONFIG_FILE: "/etc/tofu/config.tfrc"
    TOFU_LOG: "INFO"

extensions:
  # GitLab
  gitlab:
    variables:
      TF_CLI_CONFIG_FILE: "/etc/tofu/config.tfrc"

  # GitHub Actions
  github:
    env:
      TF_CLI_CONFIG_FILE: "/etc/tofu/config.tfrc"
```

## Следующие шаги

- [Быстрый старт](/ru/guide/getting-started) — установка TerraCi и настройка первого проекта
- [Настройка GitLab CI](/ru/config/gitlab) — полный справочник опций пайплайна GitLab
