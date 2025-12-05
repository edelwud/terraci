# Фильтры

TerraCi поддерживает гибкую фильтрацию модулей через glob-паттерны.

## Параметры

| Параметр | Тип | Описание |
|----------|-----|----------|
| `exclude` | []string | Паттерны для исключения модулей |
| `include` | []string | Паттерны для включения модулей |

## Логика фильтрации

1. Если `include` не задан — включаются все обнаруженные модули
2. Если `include` задан — включаются только совпадающие модули
3. `exclude` всегда применяется после `include`

```
Все модули → include (если задан) → exclude → Результат
```

## Glob-синтаксис

| Паттерн | Описание |
|---------|----------|
| `*` | Любой сегмент пути |
| `**` | Любое количество сегментов |
| `?` | Любой одиночный символ |
| `[abc]` | Любой символ из набора |
| `[a-z]` | Любой символ из диапазона |

## Exclude

Исключает модули, соответствующие паттернам:

```yaml
exclude:
  - "*/test/*"           # Исключить test-окружения
  - "*/sandbox/*"        # Исключить sandbox
  - "legacy/*/*/*/*"     # Исключить legacy-сервис
```

### Примеры exclude

```yaml
# Исключить конкретный регион
exclude:
  - "*/*/eu-north-1/*"

# Исключить несколько окружений
exclude:
  - "*/dev/*/*"
  - "*/test/*/*"

# Исключить конкретные модули
exclude:
  - "*/*/*/deprecated-*"
  - "*/*/*/old-*"
```

## Include

Включает только модули, соответствующие паттернам:

```yaml
include:
  - "platform/*/*/*"     # Только сервис platform
```

### Примеры include

```yaml
# Только production
include:
  - "*/prod/*/*"

# Только определенные регионы
include:
  - "*/*/eu-central-1/*"
  - "*/*/eu-west-1/*"

# Только конкретный сервис
include:
  - "billing/*/*/*"
```

## Комбинирование фильтров

```yaml
# Только production, кроме legacy-модулей
include:
  - "*/prod/*/*"
exclude:
  - "*/prod/*/legacy-*"
```

```yaml
# Все, кроме тестовых окружений и sandbox
exclude:
  - "*/test/*"
  - "*/dev/*"
  - "*/sandbox/*"
```

## Фильтрация через CLI

Фильтры можно переопределить через командную строку:

```bash
# Исключить паттерн
terraci generate --exclude "*/test/*"

# Включить только паттерн
terraci generate --include "platform/prod/*/*"

# Комбинирование
terraci generate \
  --include "platform/*/*/*" \
  --exclude "*/dev/*"
```

CLI-флаги добавляются к фильтрам из конфига.

## Фильтрация сабмодулей

```yaml
# Только сабмодули (глубина 5)
include:
  - "*/*/*/*/*"

# Исключить сабмодули
exclude:
  - "*/*/*/*/*"

# Конкретный сабмодуль
exclude:
  - "platform/prod/eu-central-1/ec2/rabbitmq"
```

## Фильтрация по окружению

Фильтрация через флаг `--environment`:

```bash
# Только production
terraci generate --environment prod

# Только staging
terraci generate --environment stage
```

Эквивалентно:
```yaml
include:
  - "*/prod/*/*"
```

## Примеры конфигураций

### Разделение по окружениям

```yaml
# production.terraci.yaml
include:
  - "*/prod/*/*"
exclude:
  - "*/prod/*/deprecated-*"

# staging.terraci.yaml
include:
  - "*/stage/*/*"
```

### Исключение тестовых модулей

```yaml
exclude:
  - "*/test/*"
  - "*/sandbox/*"
  - "*/*/*/test-*"
  - "*/*/*/mock-*"
```

### Только критичная инфраструктура

```yaml
include:
  - "core/*/*/*"
  - "*/prod/eu-central-1/*"
exclude:
  - "*/*/*/monitoring"
```

## Диагностика

### Проверка фильтрации

```bash
# Показать, какие модули будут обработаны
terraci generate --dry-run
```

### Отладка паттернов

```bash
# Verbose-вывод показывает совпадения
terraci validate --verbose
```

## Частые ошибки

### Паттерн не совпадает

```yaml
# Неправильно — слишком конкретно
include:
  - "platform/prod/eu-central-1/vpc"

# Правильно — используйте glob
include:
  - "platform/prod/*/*"
```

### Неверная глубина

```yaml
# Для 4-уровневой структуры
include:
  - "*/*/*"      # Неправильно — 3 уровня
  - "*/*/*/*"    # Правильно — 4 уровня
```

### Конфликт include/exclude

```yaml
# Всё исключено — ничего не будет обработано
include:
  - "*/prod/*/*"
exclude:
  - "*/*/*/*"    # Исключает всё
```
