---
title: "Проверка политик"
description: "Конфигурация OPA-политик: источники, пространства имён, правила, overwrites и интеграция с MR/PR"
outline: deep
---

# Проверка политик

TerraCi интегрирует [Open Policy Agent (OPA)](https://www.openpolicyagent.org/) для применения правил соответствия к Terraform планам. Политики пишутся на [Rego v1](https://www.openpolicyagent.org/docs/latest/policy-language/).

## Базовая конфигурация

```yaml
policy:
  enabled: true
  sources:
    - path: terraform           # имя директории = имя Rego package
  namespaces:
    - terraform
  on_failure: block
```

## Параметры конфигурации

### enabled

Включение/отключение проверки политик глобально.

```yaml
policy:
  enabled: true  # по умолчанию: false
```

### sources

Список источников политик. Каждый источник — директория с `.rego` файлами. Имя директории должно совпадать с `package` декларацией внутри файлов.

#### Локальный путь

```yaml
policy:
  sources:
    - path: terraform           # package terraform → data.terraform.deny/warn
    - path: compliance          # package compliance → data.compliance.deny/warn
```

#### Git репозиторий

```yaml
policy:
  sources:
    - git: https://github.com/org/terraform-policies.git
      ref: main
```

#### OCI реестр

```yaml
policy:
  sources:
    - oci: oci://ghcr.io/org/policies:v1.0
```

### namespaces

Rego пакеты для проверки. TerraCi запрашивает `data.<namespace>.deny` и `data.<namespace>.warn` для каждого namespace.

```yaml
policy:
  namespaces:
    - terraform              # data.terraform.deny, data.terraform.warn
    - compliance             # data.compliance.deny, data.compliance.warn
```

По умолчанию: `["terraform"]`

Несколько namespace позволяют разделять ответственность — правила безопасности в `terraform`, контроль расходов в `compliance` и т.д.

### on_failure

Действие при срабатывании `deny` правил:

| Значение | Описание |
|----------|----------|
| `block` | Завершить пайплайн с ошибкой (код возврата 1) — **по умолчанию** |
| `warn` | Переклассифицировать нарушения в предупреждения (код возврата 0) |
| `ignore` | Молча игнорировать |

### on_warning

Действие при срабатывании `warn` правил:

```yaml
policy:
  on_warning: warn  # по умолчанию
```

### show_in_comment

Включить результаты в комментарий MR/PR.

```yaml
policy:
  show_in_comment: true  # по умолчанию: true
```

### cache_dir

Директория для кэширования загруженных политик (git/OCI источники).

```yaml
policy:
  cache_dir: .terraci/policies  # по умолчанию
```

### overwrites

Переопределение настроек для конкретных модулей через `**` glob-паттерны:

```yaml
policy:
  enabled: true
  on_failure: block

  overwrites:
    # Sandbox: переклассифицировать ошибки в предупреждения
    - match: "**/sandbox/**"
      on_failure: warn

    # Legacy: полностью отключить проверки
    - match: "legacy/**"
      enabled: false

    # Production: добавить compliance namespace
    - match: "**/prod/**"
      namespaces:
        - terraform
        - compliance
```

#### Glob-паттерны

| Паттерн | Совпадает | НЕ совпадает |
|---------|-----------|-------------|
| `**/sandbox/**` | `platform/sandbox/eu-central-1/test` | `platform/stage/eu-central-1/app` |
| `legacy/**` | `legacy/old/eu-central-1/db` | `platform/legacy/module` |
| `**/prod/**` | `platform/prod/eu-central-1/vpc` | `platform/stage/eu-central-1/vpc` |

- `**` — любое количество сегментов пути (включая ноль)
- `*` — один сегмент пути

#### Поведение overwrites

- **`on_failure: warn`** — deny нарушения переклассифицируются в предупреждения (отображаются, но не блокируют)
- **`enabled: false`** — модуль полностью пропускается, без evaluation
- **`namespaces: [...]`** — заменяет список namespace для совпадающих модулей
- Несколько overwrites могут совпасть — применяются по порядку

## Написание политик

Политики используют OPA v1 Rego синтаксис с `import rego.v1`.

### Правила Deny (блокировка деплоя)

```rego
package terraform

import rego.v1

# METADATA
# description: Запрет публичных S3 бакетов
# entrypoint: true
deny contains msg if {
    some resource in input.resource_changes
    resource.type == "aws_s3_bucket"
    not "delete" in resource.change.actions
    resource.change.after.acl == "public-read"
    msg := sprintf("S3 bucket '%s' не должен быть публичным", [resource.name])
}
```

### Правила Warn (предупреждение)

```rego
warn contains msg if {
    some resource in input.resource_changes
    resource.type == "aws_s3_bucket"
    not "delete" in resource.change.actions
    not has_versioning(resource)
    msg := sprintf("S3 bucket '%s' должен иметь versioning", [resource.name])
}

has_versioning(resource) if {
    some v in resource.change.after.versioning
    v.enabled == true
}
```

### Ключевые паттерны Rego

| Паттерн | Назначение |
|---------|-----------|
| `some resource in input.resource_changes` | Итерация ресурсов (не `[_]`) |
| `"create" in resource.change.actions` | Проверка членства |
| `not "delete" in resource.change.actions` | Отрицание членства |
| `resource.type in taggable_types` | Проверка значения в списке |
| `deny contains msg if { ... }` | Блокирующее правило |
| `warn contains msg if { ... }` | Предупреждающее правило |

::: tip Линтинг
Проверяйте политики с помощью [Regal](https://docs.styra.com/regal): `regal lint terraform/ compliance/`
:::

### Несколько namespace

Разделяйте политики по ответственности — каждая директория = Rego package:

```
terraform/          → package terraform    (безопасность)
  tags.rego
  s3.rego
compliance/         → package compliance   (контроль расходов)
  cost.rego
```

```yaml
policy:
  sources:
    - path: terraform
    - path: compliance
  namespaces:
    - terraform
    - compliance
```

### Структура Input

Политики получают JSON плана Terraform (`terraform show -json plan.tfplan`):

```json
{
  "format_version": "1.2",
  "resource_changes": [
    {
      "type": "aws_s3_bucket",
      "name": "example",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {
          "bucket": "my-bucket",
          "acl": "private",
          "tags": { "Environment": "stage" }
        }
      }
    }
  ]
}
```

## Генерируемый пайплайн

При включении политик TerraCi добавляет стадию `policy-check` между plan и apply:

**GitLab CI:**
```yaml
stages:
  - deploy-plan-0
  - policy-check
  - deploy-apply-0

policy-check:
  stage: policy-check
  script:
    - terraci policy pull
    - terraci policy check
  needs: [plan-vpc, plan-eks]
```

**GitHub Actions:**
```yaml
jobs:
  policy-check:
    needs: [plan-vpc, plan-eks]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/download-artifact@v4
      - run: terraci policy pull && terraci policy check
```

## Команды

```bash
terraci policy pull                                    # Загрузить политики
terraci policy check                                   # Проверить все модули
terraci policy check --module platform/prod/.../vpc    # Один модуль
terraci policy check --output json                     # JSON вывод
terraci policy check -v                                # Подробный вывод
```

## Интеграция с MR/PR

Результаты включаются в комментарий MR/PR:

```markdown
### ❌ Проверка политик

**5** модулей: ✅ **2** ок | ⚠️ **1** предупреждения | ❌ **2** ошибки

<details>
<summary>❌ Ошибки (2)</summary>

**platform/stage/eu-central-1/bad:**
- `terraform`: S3 bucket 'public' не должен быть публичным
- `compliance`: Instance 'web' использует дорогой тип 'p4d.24xlarge'

</details>
```

## Примеры

См. [examples/policy-checks](https://github.com/edelwud/terraci/tree/main/examples/policy-checks) — рабочий пример с:

- Двумя namespace (`terraform` + `compliance`)
- Пятью модулями (pass, warn, fail, skip)
- Overwrites для sandbox и legacy
- Rego-политиками, проходящими Regal lint

## Смотрите также

- [Policy CLI](/ru/cli/policy) — команды pull и check
