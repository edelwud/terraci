# terraci init

Инициализация конфигурационного файла TerraCi.

## Синтаксис

```bash
terraci init [flags]
```

## Описание

Команда `init` создаёт файл `.terraci.yaml` с разумными значениями по умолчанию.

## Флаги

| Флаг | Тип | По умолчанию | Описание |
|------|-----|--------------|----------|
| `--force` | bool | false | Перезаписать существующий файл |

## Примеры

### Базовая инициализация

```bash
terraci init
```

Создаёт `.terraci.yaml`:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true

exclude:
  - "*/test/*"
  - "*/sandbox/*"

gitlab:
  terraform_binary: "terraform"
  terraform_image: "hashicorp/terraform:1.6"
  stages_prefix: "deploy"
  parallelism: 5
  plan_enabled: true
  auto_approve: false
  before_script:
    - ${TERRAFORM_BINARY} init
  tags:
    - terraform
    - docker
  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"
  artifact_paths:
    - "*.tfplan"

backend:
  type: s3
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

### Перезапись существующего

```bash
terraci init --force
```

Перезаписывает `.terraci.yaml` без запроса подтверждения.

### Инициализация в другой директории

```bash
terraci -d /path/to/project init
```

Создаёт конфигурацию в указанной директории.

## Что создаётся

Команда создаёт:
- `.terraci.yaml` в текущей (или указанной) директории

Команда НЕ изменяет:
- Существующие Terraform-файлы
- Конфигурацию GitLab CI
- Другие файлы проекта

## После инициализации

1. **Проверьте конфигурацию**
   ```bash
   cat .terraci.yaml
   ```

2. **Настройте под свой проект**
   - Измените паттерн под вашу структуру
   - Обновите Docker-образ
   - Добавьте исключения
   - Настройте backend

3. **Валидация**
   ```bash
   terraci validate
   ```

4. **Сгенерируйте первый пайплайн**
   ```bash
   terraci generate --dry-run
   terraci generate -o .gitlab-ci.yml
   ```

## Шаблоны конфигураций

### OpenTofu

После init измените для OpenTofu:

```yaml
gitlab:
  terraform_binary: "tofu"
  terraform_image: "ghcr.io/opentofu/opentofu:1.6"
```

### GCS Backend

```yaml
backend:
  type: gcs
  bucket: my-terraform-state
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

### Простая структура

```yaml
structure:
  pattern: "{environment}/{region}/{module}"
  min_depth: 3
  max_depth: 3
  allow_submodules: false
```

### Production-ready

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true

exclude:
  - "*/test/*"
  - "*/sandbox/*"
  - "*/dev/*"

gitlab:
  terraform_binary: "terraform"
  terraform_image: "hashicorp/terraform:1.6"
  stages_prefix: "deploy"
  parallelism: 3
  plan_enabled: true
  auto_approve: false
  before_script:
    - ${TERRAFORM_BINARY} init -backend-config="bucket=${TF_STATE_BUCKET}"
  tags:
    - terraform
    - production
  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"
    TF_CLI_ARGS_plan: "-parallelism=30"
    TF_CLI_ARGS_apply: "-parallelism=30"
  artifact_paths:
    - "*.tfplan"

backend:
  type: s3
  bucket: company-terraform-state
  region: eu-central-1
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

## Устранение проблем

### Файл уже существует

```
Error: .terraci.yaml already exists. Use --force to overwrite.
```

Решение: Используйте `--force` или вручную отредактируйте существующий файл.

### Нет прав на запись

```
Error: permission denied: .terraci.yaml
```

Решение: Проверьте права на директорию и файл.

## Рекомендации

1. **Начните с дефолтов** — init создаёт разумные значения
2. **Проверьте структуру** — убедитесь, что паттерн соответствует вашему проекту
3. **Добавьте в git** — `.terraci.yaml` должен быть в репозитории
4. **Настройте CI сначала** — проверьте `--dry-run` перед деплоем
