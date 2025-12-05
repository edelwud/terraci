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

Каждый модуль идентифицируется компонентами пути:

| Путь | Service | Environment | Region | Module |
|------|---------|-------------|--------|--------|
| `platform/production/us-east-1/vpc` | platform | production | us-east-1 | vpc |
| `analytics/production/eu-west-1/redshift` | analytics | production | eu-west-1 | redshift |

ID модуля — это полный путь: `platform/production/us-east-1/vpc`

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
Родительский модуль и сабмодули могут существовать одновременно. TerraCi обнаруживает все директории с `.tf` файлами в пределах диапазона глубины.
:::

## Конфигурация

Настройте структуру в `.terraci.yaml`:

```yaml
structure:
  # Паттерн директорий
  pattern: "{service}/{environment}/{region}/{module}"

  # Минимальная глубина (вычисляется из паттерна, если не задана)
  min_depth: 4

  # Максимальная глубина (для поддержки сабмодулей)
  max_depth: 5

  # Включить обнаружение сабмодулей
  allow_submodules: true
```

### Пользовательские паттерны

Можно настроить паттерн для разных структур:

**3-уровневая структура:**
```yaml
structure:
  pattern: "{environment}/{region}/{module}"
  min_depth: 3
  max_depth: 3
```

**5-уровневая структура:**
```yaml
structure:
  pattern: "{org}/{service}/{environment}/{region}/{module}"
  min_depth: 5
  max_depth: 6
```

## Что является модулем?

Директория считается Terraform-модулем, если:

1. Она находится на правильной глубине (между `min_depth` и `max_depth`)
2. Содержит хотя бы один `.tf` файл

TerraCi игнорирует:
- Скрытые директории (начинающиеся с `.`)
- Директории без `.tf` файлов
- Директории вне диапазона глубины

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
  min_depth: 2
  max_depth: 2
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
1. Глубина директорий соответствует `min_depth`/`max_depth`
2. Директории содержат `.tf` файлы
3. Директории не скрытые (нет префикса `.`)

### Неправильные ID модулей

Если ID модулей не совпадают с путями state-файлов, настройте паттерн:

```yaml
backend:
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

Этот паттерн используется для сопоставления ключей `terraform_remote_state` с модулями.
