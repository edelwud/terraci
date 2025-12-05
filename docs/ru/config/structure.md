# Конфигурация структуры

Секция `structure` определяет, как TerraCi обнаруживает Terraform-модули в вашем проекте.

## Параметры

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `pattern` | string | `{service}/{environment}/{region}/{module}` | Паттерн структуры директорий |
| `min_depth` | int | Авто | Минимальная глубина директории модуля |
| `max_depth` | int | Авто | Максимальная глубина директории модуля |
| `allow_submodules` | bool | `true` | Разрешить вложенные сабмодули |

## Pattern

Паттерн описывает структуру директорий вашего проекта. Поддерживаемые плейсхолдеры:

| Плейсхолдер | Описание | Пример |
|-------------|----------|--------|
| `{service}` | Название сервиса/продукта | `platform`, `billing` |
| `{environment}` | Окружение | `prod`, `stage`, `dev` |
| `{region}` | Регион/датацентр | `eu-central-1`, `us-east-1` |
| `{module}` | Название модуля | `vpc`, `eks`, `rds` |

### Примеры паттернов

**Стандартный (4 уровня):**
```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
```

Структура:
```
infrastructure/
└── platform/
    └── production/
        └── eu-central-1/
            ├── vpc/
            ├── eks/
            └── rds/
```

**Упрощенный (3 уровня):**
```yaml
structure:
  pattern: "{environment}/{region}/{module}"
  min_depth: 3
  max_depth: 3
```

Структура:
```
infrastructure/
└── production/
    └── eu-central-1/
        ├── vpc/
        ├── eks/
        └── rds/
```

**Расширенный (5 уровней):**
```yaml
structure:
  pattern: "{service}/{environment}/{region}/{layer}/{module}"
  min_depth: 5
  max_depth: 5
```

## Глубина директорий

### min_depth

Минимальная глубина, на которой ищутся модули:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4  # Модули на глубине 4
```

Если не указано, вычисляется автоматически из паттерна.

### max_depth

Максимальная глубина поиска модулей:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  max_depth: 5  # Разрешает сабмодули на глубине 5
```

Если не указано:
- При `allow_submodules: true` — `min_depth + 1`
- При `allow_submodules: false` — равно `min_depth`

## Сабмодули

Сабмодули — вложенные модули внутри родительского модуля:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true
```

Структура с сабмодулями:
```
platform/production/eu-central-1/
├── vpc/                    # Родительский модуль (глубина 4)
│   └── main.tf
├── ec2/                    # Родительский модуль (глубина 4)
│   ├── main.tf
│   ├── rabbitmq/           # Сабмодуль (глубина 5)
│   │   └── main.tf
│   └── redis/              # Сабмодуль (глубина 5)
│       └── main.tf
└── rds/
    └── main.tf
```

### Идентификация модулей

| Путь | ID модуля |
|------|-----------|
| `platform/production/eu-central-1/vpc` | `platform/production/eu-central-1/vpc` |
| `platform/production/eu-central-1/ec2` | `platform/production/eu-central-1/ec2` |
| `platform/production/eu-central-1/ec2/rabbitmq` | `platform/production/eu-central-1/ec2/rabbitmq` |

## Определение модуля

Директория считается Terraform-модулем, если содержит:
- Файлы `*.tf`, или
- Файлы `*.tf.json`

## Примеры конфигураций

### Монорепозиторий с несколькими продуктами

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true
```

### Один продукт, несколько окружений

```yaml
structure:
  pattern: "{environment}/{region}/{module}"
  min_depth: 3
  max_depth: 3
  allow_submodules: false
```

### По регионам без окружений

```yaml
structure:
  pattern: "{region}/{module}"
  min_depth: 2
  max_depth: 2
```

## Диагностика

### Модули не обнаруживаются

1. Проверьте глубину директорий:
   ```bash
   find . -name "*.tf" -type f | head -5
   ```

2. Убедитесь, что глубина соответствует паттерну

3. Запустите валидацию:
   ```bash
   terraci validate --verbose
   ```

### Неверные ID модулей

Если ID модулей выглядят некорректно, проверьте соответствие паттерна вашей структуре:

```yaml
# Если структура: env/region/module
structure:
  pattern: "{environment}/{region}/{module}"
  min_depth: 3
```
