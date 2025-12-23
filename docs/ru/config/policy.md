# Проверка политик

TerraCi интегрирует [Open Policy Agent (OPA)](https://www.openpolicyagent.org/) для применения правил соответствия к Terraform планам. Политики пишутся на [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/), декларативном языке политик OPA.

## Базовая конфигурация

```yaml
policy:
  enabled: true
  sources:
    - path: policies
  namespaces:
    - terraform
  on_failure: block
```

## Параметры конфигурации

### enabled

Включение или отключение проверки политик глобально.

```yaml
policy:
  enabled: true  # по умолчанию: false
```

### sources

Список источников политик. TerraCi поддерживает три типа источников:

#### Локальный путь

```yaml
policy:
  sources:
    - path: policies           # Относительно корня проекта
    - path: /absolute/path     # Абсолютный путь
```

#### Git репозиторий

```yaml
policy:
  sources:
    - git: https://github.com/org/terraform-policies.git
      ref: main                # Ветка, тег или SHA коммита
```

#### OCI реестр

```yaml
policy:
  sources:
    - oci: oci://ghcr.io/org/policies:v1.0
```

### namespaces

Пространства имён Rego пакетов для проверки. TerraCi ищет правила `deny` и `warn` в этих пространствах имён.

```yaml
policy:
  namespaces:
    - terraform              # Проверяет data.terraform.deny, data.terraform.warn
    - terraform.aws          # Проверяет data.terraform.aws.deny и т.д.
    - terraform.security
```

По умолчанию: `["terraform"]`

### on_failure

Действие при провале проверки политик (сработали правила deny):

| Значение | Описание |
|----------|----------|
| `block` | Завершить пайплайн с ошибкой (код возврата 1) |
| `warn` | Вывести предупреждения, но продолжить (код возврата 0) |
| `ignore` | Молча игнорировать нарушения |

```yaml
policy:
  on_failure: block  # по умолчанию
```

### on_warning

Действие при наличии предупреждений (сработали правила warn):

```yaml
policy:
  on_warning: warn  # по умолчанию
```

### show_in_comment

Включить результаты проверки политик в комментарий MR.

```yaml
policy:
  show_in_comment: true  # по умолчанию: true
```

### cache_dir

Директория для кэширования загруженных политик.

```yaml
policy:
  cache_dir: .terraci/policies  # по умолчанию
```

### overwrites

Переопределение настроек политик для конкретных модулей с использованием glob-паттернов:

```yaml
policy:
  enabled: true
  on_failure: block

  overwrites:
    # Разрешить sandbox-деплои только с предупреждениями
    - match: "*/sandbox/*"
      on_failure: warn

    # Пропустить проверку политик для legacy модулей
    - match: "legacy/*"
      enabled: false

    # Другие namespaces для определённых модулей
    - match: "platform/prod/*"
      namespaces:
        - terraform
        - terraform.production
```

## Написание политик

Политики должны использовать синтаксис OPA v1 Rego с ключевыми словами `contains` и `if`.

### Правила Deny

Правила deny блокируют деплой при срабатывании:

```rego
package terraform

import rego.v1

deny contains msg if {
    resource := input.resource_changes[_]
    resource.type == "aws_s3_bucket"
    resource.change.after.acl == "public-read"
    msg := sprintf("S3 бакет '%s' не должен быть публичным", [resource.name])
}
```

### Правила Warn

Правила warn генерируют предупреждения без блокировки:

```rego
package terraform

import rego.v1

warn contains msg if {
    resource := input.resource_changes[_]
    resource.type == "aws_instance"
    not resource.change.after.tags.Environment
    msg := sprintf("Инстанс '%s' должен иметь тег Environment", [resource.name])
}
```

### Структура Input

Политики получают JSON плана Terraform в качестве input. Основные поля:

```json
{
  "format_version": "1.1",
  "resource_changes": [
    {
      "type": "aws_s3_bucket",
      "name": "example",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {
          "bucket": "my-bucket",
          "acl": "private"
        }
      }
    }
  ],
  "planned_values": { ... },
  "configuration": { ... }
}
```

См. [формат JSON вывода Terraform](https://developer.hashicorp.com/terraform/internals/json-format) для полной схемы.

## Генерируемый пайплайн

При включении проверки политик TerraCi добавляет стадию `policy-check`:

```yaml
stages:
  - deploy-plan-0
  - deploy-plan-1
  - policy-check    # После всех планов
  - deploy-apply-0
  - deploy-apply-1
  - summary

policy-check:
  stage: policy-check
  script:
    - terraci policy pull
    - terraci policy check
  needs:
    - job: plan-vpc
      optional: true
    - job: plan-eks
      optional: true
  artifacts:
    paths:
      - .terraci/policy-results.json
    when: always
```

## Команды

### Загрузка политик

Загрузка политик из настроенных источников:

```bash
terraci policy pull
```

### Проверка политик

Запуск проверки политик на Terraform планах:

```bash
# Проверить все модули с plan.json
terraci policy check

# Проверить конкретный модуль
terraci policy check --module platform/prod/eu-central-1/vpc

# Вывод в формате JSON
terraci policy check --output json
```

## Интеграция с MR

Результаты проверки политик включаются в комментарий MR:

```markdown
### ❌ Проверка политик

**3** модуля проверено: ✅ **1** пройдено | ⚠️ **1** с предупреждениями | ❌ **1** с ошибками

<details>
<summary>❌ Ошибки (1)</summary>

**platform/prod/eu-central-1/vpc:**
- `terraform`: S3 бакет 'logs' не должен быть публичным

</details>
```

## Примеры

См. [examples/policy-checks](https://github.com/edelwud/terraci/tree/main/examples/policy-checks) для:

- Полной конфигурации `.terraci.yaml`
- Примеров Rego политик для AWS ресурсов
- Настройки GitLab CI пайплайна
