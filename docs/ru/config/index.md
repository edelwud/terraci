# Конфигурация

TerraCi настраивается через YAML-файл `.terraci.yaml` в корне проекта.

## Быстрый старт

Создайте конфигурацию командой:

```bash
terraci init
```

## Полный пример

```yaml
# Структура директорий
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true

# Фильтры модулей
exclude:
  - "*/test/*"
  - "*/sandbox/*"

include:
  - "platform/*/*/*/*"

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
    after_script:
      - echo "Завершено"
    artifacts:
      paths:
        - "*.tfplan"
      expire_in: "1 day"

# Настройки бэкенда
backend:
  type: s3
  bucket: my-terraform-state
  region: eu-central-1
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

## Секции конфигурации

### [Structure](./structure.md)

Определяет структуру директорий и паттерн обнаружения модулей:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
```

### [GitLab](./gitlab.md)

Настройки генерации GitLab CI пайплайнов:

```yaml
gitlab:
  image: "hashicorp/terraform:1.6"
  plan_enabled: true
```

### [Filters](./filters.md)

Фильтрация модулей по glob-паттернам:

```yaml
exclude:
  - "*/test/*"
include:
  - "platform/*/*/*"
```

## Приоритет конфигурации

TerraCi ищет конфигурацию в следующем порядке:

1. Путь, указанный через `--config`
2. `.terraci.yaml`
3. `.terraci.yml`
4. `terraci.yaml`
5. `terraci.yml`

Если файл не найден, используются значения по умолчанию.

## Валидация

Проверьте конфигурацию:

```bash
terraci validate
```

## Переменные окружения

Переменные окружения можно использовать в конфигурации GitLab CI:

```yaml
gitlab:
  variables:
    AWS_REGION: "${AWS_DEFAULT_REGION}"
    TF_VAR_environment: "${CI_ENVIRONMENT_NAME}"
```

## Типичные конфигурации

### Минимальная

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
```

### С OpenTofu

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

gitlab:
  terraform_binary: "tofu"
  image: "ghcr.io/opentofu/opentofu:1.6"
```

### С OpenTofu Minimal (требует entrypoint)

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

gitlab:
  terraform_binary: "tofu"
  image:
    name: "ghcr.io/opentofu/opentofu:1.9-minimal"
    entrypoint: [""]
```

### Простая структура

```yaml
structure:
  pattern: "{environment}/{region}/{module}"
  min_depth: 3
  max_depth: 3
  allow_submodules: false
```
