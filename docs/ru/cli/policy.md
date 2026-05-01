---
title: "terraci policy"
description: "Загрузка и проверка OPA-политик для Terraform планов"
outline: deep
---

# terraci policy

Команды для управления и запуска OPA проверок политик на Terraform планах.

## Подкоманды

### terraci policy pull

Загрузка политик из настроенных источников.

```bash
terraci policy pull
```

Эта команда:
1. Читает источники политик из `.terraci.yaml`
2. Загружает политики в директорию кэша
3. Подготавливает политики для проверки

**Пример вывода:**
```
pulling policies...
  source: path:policies
  source: git:https://github.com/org/policies.git@main
pulled 2 policy sources to .terraci/policies
```

### terraci policy check

Запуск проверки политик на JSON-файлах Terraform планов.

```bash
terraci policy check [флаги]
```

**Флаги:**

| Флаг | Сокр. | Описание |
|------|-------|----------|
| `--module` | `-m` | Проверить конкретный путь модуля |
| `--output` | `-o` | Формат вывода: `text` (по умолчанию) или `json` |

**Примеры:**

```bash
# Проверить все модули с файлами plan.json
terraci policy check

# Проверить конкретный модуль
terraci policy check --module platform/prod/eu-central-1/vpc

# Вывод в формате JSON
terraci policy check --output json
```

**Текстовый вывод:**
```
• policy check results   modules=3
  • module   module=platform/prod/eu-central-1/vpc status=passed
  • module   module=platform/prod/eu-central-1/ec2 status=warned
    • warning   namespace=terraform msg="Инстанс 'web' должен иметь тег Environment"
  • module   module=platform/prod/eu-central-1/s3 status=failed
    • failure   namespace=terraform msg="S3 бакет 'logs' не должен быть публичным"
• summary   modules=3 passed=1 warned=1 failed=1
```

**JSON вывод:**
```json
{
  "total_modules": 3,
  "passed_modules": 1,
  "warned_modules": 1,
  "failed_modules": 1,
  "total_failures": 1,
  "total_warnings": 1,
  "results": [
    {
      "module": "platform/prod/eu-central-1/vpc",
      "failures": [],
      "warnings": [],
      "successes": 5
    },
    {
      "module": "platform/prod/eu-central-1/ec2",
      "failures": [],
      "warnings": [
        {
          "msg": "Инстанс 'web' должен иметь тег Environment",
          "namespace": "terraform"
        }
      ],
      "successes": 3
    },
    {
      "module": "platform/prod/eu-central-1/s3",
      "failures": [
        {
          "msg": "S3 бакет 'logs' не должен быть публичным",
          "namespace": "terraform"
        }
      ],
      "warnings": [],
      "successes": 2
    }
  ]
}
```

## Коды возврата

| Код | Описание |
|-----|----------|
| 0 | Все проверки пройдены (или `on_failure: warn/ignore`) |
| 1 | Найдены нарушения политик (при `on_failure: block`) |
| 2 | Ошибка конфигурации или выполнения |

## Требования

Проверка политик требует наличия файлов `plan.json` в директориях модулей. Создайте их командами:

```bash
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json
```

Или в GitLab CI:

```yaml
plan-module:
  script:
    - terraform init
    - terraform plan -out=plan.tfplan
    - terraform show -json plan.tfplan > plan.json
  artifacts:
    paths:
      - "**/plan.json"
```

## Конфигурация

См. [Проверка политик](/ru/config/policy) для полного описания параметров.

Минимальный пример:

```yaml
extensions:
  policy:
    enabled: true
    sources:
      - path: policies
    namespaces:
      - terraform
    on_failure: block
```

## См. также

- [Проверка политик](/ru/config/policy) - Полное описание конфигурации
- [examples/policy-checks](https://github.com/edelwud/terraci/tree/main/examples/policy-checks) - Примеры политик
