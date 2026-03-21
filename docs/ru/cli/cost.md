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
