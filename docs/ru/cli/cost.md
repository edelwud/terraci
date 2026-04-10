---
title: terraci cost
description: Оценка стоимости AWS из файлов Terraform plan
---

# terraci cost

Оценка ежемесячной стоимости AWS на основе анализа `plan.json` файлов.

## Использование

```bash
terraci cost [flags]
```

## Флаги

| Флаг | Короткий | Тип | По умолчанию | Описание |
|------|----------|-----|-------------|----------|
| `--module` | `-m` | string | | Оценить стоимость конкретного модуля |
| `--output` | `-o` | string | `text` | Формат вывода: `text`, `json` |

## Как это работает

1. Сканирует рабочую директорию на наличие `plan.json` файлов
2. Определяет регион из пути модуля по настроенному `structure.pattern`
3. Загружает данные о ценах AWS из Bulk Pricing API (кешируются локально)
4. Сопоставляет ресурсы с обработчиками стоимости и рассчитывает ежемесячные оценки
5. Выводит стоимость по модулям с before/after/diff

AWS credentials не требуются — данные о ценах публичны.

## Примеры

```bash
# Оценить все модули
terraci cost

# Один модуль
terraci cost --module platform/prod/eu-central-1/rds

# JSON вывод
terraci cost --output json

# Подробно — стоимость по ресурсам и информация о кеше
terraci cost -v
```

## Вывод

### Текстовый формат (по умолчанию)

```
• cost estimation results
  • module   module=platform/prod/eu-central-1/eks status=🔄 monthly=$35.04
    • cost change   before=$0 after=$35.04 diff=$35.04
  • module   module=platform/prod/eu-central-1/rds status=🔄 monthly=$689.12
    • cost change   before=$0 after=$689.12 diff=$689.12
• total estimated monthly cost
  • monthly   before=$0 after=$762.12 diff=$762.12
```

### JSON формат

```bash
terraci cost --output json
```

```json
{
  "modules": [
    {
      "module_id": "platform/prod/eu-central-1/rds",
      "before_cost": 0,
      "after_cost": 689.12,
      "diff_cost": 689.12,
      "resources": [
        {
          "address": "aws_db_instance.postgres",
          "monthly_cost": 400.77,
          "price_source": "aws-bulk-api"
        },
        {
          "address": "aws_lambda_function.worker",
          "monthly_cost": 12.04,
          "price_source": "usage-based",
          "status": "usage_estimated",
          "status_detail": "usage-based estimate derived from provisioned concurrency"
        },
        {
          "address": "aws_sqs_queue.jobs",
          "monthly_cost": 0,
          "price_source": "usage-based",
          "status": "usage_unknown"
        }
      ]
    }
  ],
  "total_before": 0,
  "total_after": 762.12,
  "total_diff": 762.12,
  "currency": "USD"
}
```

Для каждого ресурса в JSON теперь есть поле `status`:

- `exact` — TerraCi нашел цену на этапе plan
- `usage_estimated` — TerraCi смог вывести частичную оценку из конфигурации
- `usage_unknown` — стоимость по-прежнему неизвестна на этапе plan
- `unsupported` / `failed` — цена не получена; могут присутствовать `failure_kind` и `status_detail`

## Необходимые условия

- `cost.enabled: true` в `.terraci.yaml`
- Файлы `plan.json` в директориях модулей (`terraform show -json plan.tfplan`)

## Кеш цен

Данные о ценах AWS кешируются локально:

- Расположение: `~/.terraci/pricing` (или `cost.cache_dir` в конфиге)
- TTL: 24 часа (или `cost.cache_ttl`)
- Статус кеша показывается в выводе: `expires_in=23h49m` или `status=expired`

## Смотрите также

- [Конфигурация стоимости](/ru/config/cost) — настройка оценки стоимости
- [terraci summary](/ru/cli/summary) — публикует стоимость в комментариях MR/PR
- [examples/cost-estimation](https://github.com/edelwud/terraci/tree/main/examples/cost-estimation) — рабочий пример
