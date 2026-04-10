---
title: "Оценка стоимости"
description: "Оценка стоимости AWS: кеш цен, поддерживаемые ресурсы и отображение в MR"
outline: deep
---

# Оценка стоимости

TerraCi умеет оценивать месячную стоимость инфраструктуры на основе Terraform планов, используя данные [AWS Pricing API](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/price-changes.html). Это позволяет видеть финансовое влияние каждого изменения прямо в комментарии к Merge Request.

## Базовая конфигурация

```yaml
plugins:
  cost:
    cache_dir: "~/.terraci/pricing"
    cache_ttl: "24h"
    providers:
      aws:
        enabled: true
```

## Параметры конфигурации

### cache_dir

Директория для кеширования данных о ценах AWS. Кеш хранит индексы цен по сервисам и регионам, чтобы не загружать их повторно при каждом запуске.

```yaml
plugins:
  cost:
    cache_dir: ~/.terraci/pricing  # по умолчанию
```

### cache_ttl

Срок жизни кешированных данных о ценах. После истечения TTL данные будут загружены заново.

```yaml
plugins:
  cost:
    cache_ttl: "24h"  # по умолчанию
```

Поддерживаемые форматы: `1h`, `24h`, `7d` и т.д.

## Как это работает

1. Парсится `plan.json` (результат `terraform show -json`) для каждого модуля
2. Определяются изменения ресурсов (create, update, delete, replace)
3. Каждый тип ресурса сопоставляется с обработчиком из реестра `aws.Registry`
4. Обработчик формирует запрос к кешу цен AWS Bulk Pricing API
5. Рассчитывается часовая и месячная стоимость каждого ресурса
6. Результат агрегируется в стоимость модуля с показателями before/after/diff

Для оценки стоимости необходим `plan.json` (результат `terraform show -json`) в директориях модулей. Он генерируется автоматически при `plan_enabled: true` в конфигурации пайплайна (GitLab или GitHub).

При наличии `state.json` в директории модуля учитываются также неизменяемые ресурсы для полной картины стоимости.

## Поддерживаемые ресурсы AWS

### Вычислительные ресурсы

| Terraform ресурс | Описание |
|---|---|
| `aws_instance` | EC2 инстансы |
| `aws_ebs_volume` | EBS тома |
| `aws_eip` | Elastic IP адреса |
| `aws_nat_gateway` | NAT Gateway |

### Базы данных

| Terraform ресурс | Описание |
|---|---|
| `aws_db_instance` | RDS инстансы |
| `aws_rds_cluster` | RDS кластеры (Aurora) |
| `aws_rds_cluster_instance` | Инстансы кластеров RDS |

### Балансировка нагрузки

| Terraform ресурс | Описание |
|---|---|
| `aws_lb` / `aws_alb` | Application Load Balancer |
| `aws_elb` | Classic Load Balancer |

### Кеширование

| Terraform ресурс | Описание |
|---|---|
| `aws_elasticache_cluster` | ElastiCache кластеры |
| `aws_elasticache_replication_group` | ElastiCache группы репликации |

### Kubernetes

| Terraform ресурс | Описание |
|---|---|
| `aws_eks_cluster` | EKS кластеры |
| `aws_eks_node_group` | EKS группы нод |

### Serverless и очереди

| Terraform ресурс | Описание |
|---|---|
| `aws_lambda_function` | Lambda функции |
| `aws_dynamodb_table` | DynamoDB таблицы |
| `aws_sqs_queue` | SQS очереди |
| `aws_sns_topic` | SNS топики |
| `aws_secretsmanager_secret` | Secrets Manager |

### Хранение и сеть

| Terraform ресурс | Описание |
|---|---|
| `aws_s3_bucket` | S3 бакеты (только хранение) |
| `aws_route53_zone` | Route 53 зоны |

### Мониторинг и безопасность

| Terraform ресурс | Описание |
|---|---|
| `aws_cloudwatch_log_group` | CloudWatch Log Groups |
| `aws_cloudwatch_metric_alarm` | CloudWatch алармы |
| `aws_kms_key` | KMS ключи |

::: tip
Неподдерживаемые типы ресурсов не блокируют оценку -- они просто пропускаются. В отладочном режиме (`-v`) выводится информация о пропущенных ресурсах.
:::

## Интеграция с MR/PR

При включённом плагине cost оценки стоимости отображаются в таблице комментария MR/PR. Для каждого модуля показывается разница месячной стоимости:

```markdown
| Модуль | Plan | Политики | Стоимость |
|--------|------|----------|-----------|
| platform/prod/eu-central-1/vpc | :white_check_mark: | :white_check_mark: | +$124.50/мес |
| platform/prod/eu-central-1/eks | :white_check_mark: | :white_check_mark: | +$1,280/мес |
| platform/prod/eu-central-1/rds | :warning: | :white_check_mark: | -$45.20/мес |
```

## Кеширование цен

TerraCi загружает данные из AWS Bulk Pricing API и кеширует их локально. Это позволяет:

- Избежать лишних запросов к AWS API при повторных запусках
- Ускорить оценку стоимости в CI/CD пайплайнах
- Работать оффлайн с уже загруженным кешем

Кеш организован по сервисам и регионам. При запуске TerraCi автоматически определяет, какие данные необходимы, и загружает только недостающие.

## Полный пример конфигурации

```yaml
plugins:
  cost:
    cache_dir: ~/.terraci/pricing
    cache_ttl: "24h"
    providers:
      aws:
        enabled: true

  # Работает с любым провайдером:
  gitlab:
    plan_enabled: true
    mr:
      comment:
        enabled: true

  # Или с GitHub:
  # github:
  #   plan_enabled: true
  #   pr:
  #     comment:
  #       enabled: true
```

Эта конфигурация включает оценку стоимости с кешированием по умолчанию и отображает результаты в комментариях MR/PR рядом с выводом plan.

## CLI команда

Запуск оценки стоимости из командной строки:

```bash
# Оценить все модули с plan.json
terraci cost

# Оценить один модуль
terraci cost --module platform/prod/eu-central-1/rds

# JSON вывод
terraci cost --output json

# Подробно — стоимость по ресурсам и информация о кеше
terraci cost -v
```

Команда `terraci cost` сканирует `plan.json` файлы, загружает данные о ценах и выводит оценку стоимости по модулям. В выводе показывается расположение кеша и время до его обновления.

В JSON выводе для каждого ресурса есть поле `status`:

- `exact`, если TerraCi смог определить цену на этапе plan
- `usage_estimated`, если TerraCi смог вывести частичную оценку из настроенной capacity
- `usage_unknown`, если без runtime usage данных стоимость определить нельзя
- `unsupported` / `failed`, если ресурс не поддержан или получение цены завершилось ошибкой

> **Примечание:** требуется настройка `plugins.cost` в `.terraci.yaml`.

В CI пайплайнах оценка стоимости выполняется автоматически в рамках `terraci summary`. Используйте `terraci cost` для локальной разработки и проверки.

## Примеры

См. [examples/cost-estimation](https://github.com/edelwud/terraci/tree/main/examples/cost-estimation) — рабочий пример с VPC, EKS и RDS модулями.

## Смотрите также

- [Merge Request](/ru/config/gitlab-mr) — комментарии в MR с результатами plan и стоимостью
- [Генерация пайплайнов](/ru/guide/pipeline-generation) — руководство по генерации CI пайплайнов
