---
title: "Структура проекта"
description: "Паттерны директорий и обнаружение модулей для Terraform-монорепозиториев"
outline: deep
---

# Структура проекта

TerraCi обнаруживает Terraform-модули на основе структуры директорий. На этой странице описаны поддерживаемые структуры и их настройка.

## Паттерн по умолчанию

Паттерн по умолчанию:

```
{service}/{environment}/{region}/{module}
```

Это соответствует 4-уровневой структуре директорий:

```
infrastructure/
├── platform/              # service
│   ├── production/        # environment
│   │   └── us-east-1/     # region
│   │       ├── vpc/       # module (глубина 4)
│   │       ├── eks/       # module (глубина 4)
│   │       └── rds/       # module (глубина 4)
│   └── staging/
│       └── us-east-1/
│           └── vpc/
└── analytics/
    └── production/
        └── eu-west-1/
            └── redshift/
```

## Идентификация модулей

Каждый модуль идентифицируется по относительному пути, который также является его ID. Сегменты пути отображаются на именованные компоненты в соответствии с настроенным паттерном:

| Путь | `Get("service")` | `Get("environment")` | `Get("region")` | `Get("module")` |
|------|---------|-------------|--------|--------|
| `platform/production/us-east-1/vpc` | platform | production | us-east-1 | vpc |
| `analytics/production/eu-west-1/redshift` | analytics | production | eu-west-1 | redshift |

ID модуля — это его относительный путь: `platform/production/us-east-1/vpc`

Имена сегментов полностью настраиваемы. С паттерном `{team}/{env}/{module}` компонентами будут `team`, `env` и `module`.

## Сабмодули (глубина 5)

TerraCi поддерживает вложенные сабмодули на глубине 5:

```
infrastructure/
└── platform/
    └── production/
        └── us-east-1/
            └── ec2/                    # родительский модуль (глубина 4)
                ├── main.tf             # файлы родительского модуля
                ├── rabbitmq/           # сабмодуль (глубина 5)
                │   └── main.tf
                └── redis/              # сабмодуль (глубина 5)
                    └── main.tf
```

В этом случае:
- `platform/production/us-east-1/ec2` — базовый модуль
- `platform/production/us-east-1/ec2/rabbitmq` — сабмодуль
- `platform/production/us-east-1/ec2/redis` — сабмодуль

::: tip
Родительский модуль и сабмодули могут существовать одновременно. TerraCi обнаруживает все директории с `.tf` файлами на глубине паттерна и глубже.
:::

## Конфигурация

Настройте структуру в `.terraci.yaml`:

```yaml
structure:
  # Паттерн директорий
  pattern: "{service}/{environment}/{region}/{module}"
```

### Пользовательские паттерны

Можно настроить паттерн для разных структур:

**3-уровневая структура:**
```yaml
structure:
  pattern: "{environment}/{region}/{module}"
```

**5-уровневая структура:**
```yaml
structure:
  pattern: "{org}/{service}/{environment}/{region}/{module}"
```

## Что является модулем?

Директория считается Terraform-модулем, если:

1. Она находится на глубине, определённой паттерном (количество сегментов), или глубже (сабмодули)
2. Содержит хотя бы один `.tf` файл

TerraCi игнорирует:
- Скрытые директории (начинающиеся с `.`)
- Директории без `.tf` файлов

## Примеры

### Мультиоблачная структура

```
infrastructure/
├── aws/
│   └── production/
│       └── us-east-1/
│           └── vpc/
└── gcp/
    └── production/
        └── us-central1/
            └── vpc/
```

### Структура по командам

```yaml
structure:
  pattern: "{team}/{environment}/{region}/{module}"
```

```
infrastructure/
├── platform/
│   └── prod/
│       └── eu-west-1/
│           └── eks/
└── data/
    └── prod/
        └── eu-west-1/
            └── redshift/
```

### Простая плоская структура

```yaml
structure:
  pattern: "{environment}/{module}"
```

```
infrastructure/
├── production/
│   ├── vpc/
│   └── eks/
└── staging/
    └── vpc/
```

## Решение проблем

### Модули не обнаруживаются

Запустите валидацию для проверки:

```bash
terraci validate -v
```

Проверьте:
1. Глубина директорий соответствует паттерну
2. Директории содержат `.tf` файлы
3. Директории не скрытые (нет префикса `.`)

### Неправильные ID модулей

Если ID модулей не совпадают с путями state-файлов, настройте `structure.pattern` так, чтобы он отражал, как модули размещаются на диске:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
```

Тот же паттерн используется для сопоставления ключей `terraform_remote_state` с модулями — выводите ключи state из пути файловой системы (например, через `abspath(path.module)`), чтобы они совпадали с обнаруженными ID модулей. Подробнее см. [Разрешение зависимостей](/ru/guide/dependencies).

## Следующие шаги

- [Разрешение зависимостей](/ru/guide/dependencies) — как TerraCi извлекает и разрешает зависимости между модулями
- [Фильтры](/ru/config/filters) — include/exclude паттерны для выбора модулей
