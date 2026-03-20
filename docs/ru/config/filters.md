---
title: "Фильтры"
description: "Фильтрация модулей с помощью glob-паттернов include/exclude и library-модулей"
outline: deep
---

# Фильтры

TerraCi поддерживает гибкую фильтрацию модулей через glob-паттерны.

## Параметры

| Параметр | Тип | Описание |
|----------|-----|----------|
| `exclude` | []string | Паттерны для исключения модулей |
| `include` | []string | Паттерны для включения модулей |

## Синтаксис паттернов

TerraCi использует glob-паттерны со следующими спецсимволами:

| Паттерн | Совпадает с |
|---------|-------------|
| `*` | Любые символы кроме `/` |
| `**` | Любые символы включая `/` (любая глубина) |
| `?` | Одиночный символ |
| `[abc]` | Класс символов |
| `[!abc]` | Инвертированный класс символов |

## Exclude

Исключает модули, соответствующие паттернам:

```yaml
exclude:
  - "*/test/*"           # Исключить test-окружения
  - "*/sandbox/*"        # Исключить sandbox
  - "*/.terraform/*"
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

Включает только модули, соответствующие паттернам. Если не задан, включаются все модули (после исключений).

```yaml
include:
  - "platform/*/*/*"     # Только сервис platform
  - "analytics/*/*/*"
```

### Примеры include

```yaml
# Только production
include:
  - "*/production/*/*"

# Только конкретный сервис
include:
  - "platform/*/*/*"
```

## Фильтрация через CLI

Фильтры можно переопределить через командную строку:

```bash
# Исключить паттерн
terraci generate --exclude "*/test/*" --exclude "*/sandbox/*"

# Включить только паттерн
terraci generate --include "platform/*/*/*"

# Фильтрация по сегменту (синтаксис key=value, работает с любым именем сегмента из паттерна)
terraci generate --filter service=platform
terraci generate --filter environment=production
terraci generate --filter region=us-east-1

# Несколько значений для одного сегмента (логика ИЛИ)
terraci generate --filter environment=stage --filter environment=prod

# Комбинирование фильтров по сегментам (логика И между разными ключами)
terraci generate --filter service=platform --filter environment=production --filter region=us-east-1
```

Флаг `--filter` работает с любым именем сегмента, определённым в вашем `structure.pattern`. Например, если ваш паттерн — `{team}/{stack}/{datacenter}/{component}`, используйте `--filter team=infra`.

## Порядок фильтрации

Фильтры применяются в следующем порядке:

1. **Обнаружение** — поиск всех модулей на нужной глубине
2. **Exclude** — удаление модулей, совпадающих с exclude-паттернами
3. **Include** — если задан, оставить только совпадающие модули
4. **Фильтры по сегментам** — применение фильтров `--filter key=value`

## Фильтры по сегментам

Фильтрация по сегменту паттерна через CLI с помощью `--filter key=value`:

```bash
# По сервису
terraci generate --filter service=platform

# По окружению
terraci generate --filter environment=production

# По региону
terraci generate --filter region=us-east-1

# Комбинирование (И между разными ключами)
terraci generate --filter service=platform --filter environment=production --filter region=us-east-1

# Несколько значений для одного ключа (ИЛИ внутри одного ключа)
terraci generate -f environment=stage -f environment=prod
```

## Комбинирование фильтров

Фильтры можно комбинировать:

```yaml
exclude:
  - "*/sandbox/*"
  - "*/test/*"

include:
  - "platform/*/*/*"
  - "analytics/*/*/*"
```

Результат:
1. Исключаются все sandbox и test модули
2. Затем включаются только модули platform и analytics

## Примеры использования

### Production пайплайн

```yaml
exclude:
  - "*/sandbox/*"
  - "*/test/*"
  - "*/development/*"

include:
  - "*/production/*/*"
```

### Региональный деплой

Генерация пайплайна для конкретного региона:

```bash
terraci generate --filter region=us-east-1
```

Или в конфиге:

```yaml
include:
  - "*/*/us-east-1/*"
```

### Пайплайн для конкретного сервиса

```bash
terraci generate --filter service=platform
```

### Исключение конкретных модулей

```yaml
exclude:
  - "platform/production/us-east-1/legacy-vpc"
  - "analytics/*/*/deprecated-*"
```

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

## Диагностика фильтров

Посмотреть, какие модули будут включены:

```bash
terraci validate -v
```

Вывод:
```
Discovered 20 modules
After exclude patterns: 15 modules
After include patterns: 10 modules
Final module count: 10

Modules:
  - platform/production/us-east-1/vpc
  - platform/production/us-east-1/eks
  ...
```

## Подстановочные символы в путях

### Один уровень (`*`)

```yaml
include:
  - "platform/*/us-east-1/*"  # Любое окружение, только us-east-1
```

Совпадает с:
- `platform/production/us-east-1/vpc`
- `platform/staging/us-east-1/vpc`

### Несколько уровней (`**`)

```yaml
include:
  - "platform/**"  # Все модули platform на любой глубине
```

Совпадает с:
- `platform/production/us-east-1/vpc`
- `platform/production/us-east-1/ec2/rabbitmq`

## Библиотечные модули

Библиотечные модули (shared modules) — это переиспользуемые Terraform-модули без собственных провайдеров и remote state. Они используются исполняемыми модулями через блок `module`.

### Конфигурация

```yaml
library_modules:
  paths:
    - "_modules"
    - "shared/modules"
```

### Как это работает

При настройке `library_modules.paths` TerraCi:

1. **Парсит блоки module** в исполняемых модулях для поиска локальных вызовов (`source = "../_modules/kafka"`)
2. **Отслеживает зависимости** от библиотечных модулей в графе зависимостей
3. **Детектирует изменения** в библиотечных модулях при использовании `--changed-only`
4. **Включает зависимые модули** при изменении библиотечного модуля

### Пример структуры

```
terraform/
├── _modules/               # Библиотечные модули
│   ├── kafka/              # Переиспользуемая конфигурация Kafka
│   │   └── main.tf
│   └── kafka_acl/          # Модуль ACL для Kafka
│       └── main.tf
├── platform/
│   └── production/
│       └── eu-north-1/
│           └── msk/        # Исполняемый модуль, использующий _modules/kafka
│               └── main.tf
```

В `platform/production/eu-north-1/msk/main.tf`:

```hcl
module "kafka" {
  source = "../../../../_modules/kafka"
  # ...
}

module "kafka_acl" {
  source = "../../../../_modules/kafka_acl"
  # ...
}
```

### Детекция изменений

При изменении `_modules/kafka/main.tf`:

```bash
terraci generate --changed-only
```

TerraCi включит `platform/production/eu-north-1/msk` в пайплайн, потому что он использует библиотечный модуль `kafka`.

### Транзитивные зависимости

Если библиотечный модуль `kafka_acl` внутри использует модуль `kafka`, то при изменении `kafka` все модули, использующие `kafka_acl`, также будут детектированы как затронутые.

### Verbose-вывод

Используйте verbose-режим для отображения детекции библиотечных модулей:

```bash
terraci generate --changed-only -v
```

Вывод:
```
Changed library modules: 1
  - /project/_modules/kafka
Affected modules (including dependents): 3
  - platform/production/eu-north-1/msk
  - platform/production/eu-north-1/streaming
  - platform/production/eu-west-1/msk
```

### Пример

Смотрите [пример library-modules](https://github.com/edelwud/terraci/tree/main/examples/library-modules) для полного рабочего примера.

## Смотрите также

- [Структура](/ru/config/structure) — настройка паттернов директорий и обнаружения модулей
