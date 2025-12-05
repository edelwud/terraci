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
  terraform_image: "hashicorp/terraform:1.6"
  stages_prefix: "deploy"
  parallelism: 5
  plan_enabled: true
  auto_approve: false
  before_script:
    - ${TERRAFORM_BINARY} init
  after_script:
    - echo "Завершено"
  tags:
    - terraform
    - docker
  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"
  artifact_paths:
    - "*.tfplan"

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
  terraform_image: "hashicorp/terraform:1.6"
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
  terraform_image: "ghcr.io/opentofu/opentofu:1.6"
```

### Простая структура

```yaml
structure:
  pattern: "{environment}/{region}/{module}"
  min_depth: 3
  max_depth: 3
  allow_submodules: false
```
