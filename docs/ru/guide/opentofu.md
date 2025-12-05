# Поддержка OpenTofu

TerraCi полноценно поддерживает [OpenTofu](https://opentofu.org/), open-source форк Terraform.

## Конфигурация

Переключитесь на OpenTofu, обновив `.terraci.yaml`:

```yaml
gitlab:
  terraform_binary: "tofu"
  terraform_image: "ghcr.io/opentofu/opentofu:1.6"
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

1. Обновите `.terraci.yaml`:
   ```yaml
   gitlab:
     terraform_binary: "tofu"
     terraform_image: "ghcr.io/opentofu/opentofu:1.6"
   ```

2. Перегенерируйте пайплайны:
   ```bash
   terraci generate -o .gitlab-ci.yml
   ```

3. Протестируйте с dry-run:
   ```bash
   terraci generate --dry-run
   ```

## Совместимость состояний

OpenTofu совместим с файлами состояния Terraform. Можно:

1. Сохранить существующие state-файлы Terraform
2. Мигрировать на OpenTofu без изменения state
3. Использовать те же S3/GCS бэкенды

Разрешение зависимостей TerraCi работает идентично для обоих инструментов.
