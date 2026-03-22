---
title: "Структура"
description: "Настройка паттернов директорий для обнаружения Terraform-модулей"
outline: deep
---

# Конфигурация структуры

Секция `structure` определяет, как TerraCi обнаруживает Terraform-модули в вашем проекте.

## Параметры

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `pattern` | string | `{service}/{environment}/{region}/{module}` | Паттерн структуры директорий |

## Pattern

Паттерн описывает структуру директорий вашего проекта с помощью плейсхолдеров. Имена сегментов задаются пользователем и могут быть любыми — они не ограничены значениями по умолчанию, указанными ниже.

| Плейсхолдер | Описание | Пример |
|-------------|----------|--------|
| `{service}` | Название сервиса/продукта | `platform`, `billing` |
| `{environment}` | Окружение | `prod`, `stage`, `dev` |
| `{region}` | Регион/датацентр | `eu-central-1`, `us-east-1` |
| `{module}` | Название модуля | `vpc`, `eks`, `rds` |

Вы можете использовать любые пользовательские имена сегментов:

```yaml
# Пользовательские имена сегментов
structure:
  pattern: "{team}/{stack}/{datacenter}/{component}"
```

Каждое имя сегмента в паттерне становится:
- Переменной окружения в сгенерированном пайплайне (например, `TF_TEAM`, `TF_STACK`, `TF_DATACENTER`, `TF_COMPONENT`)
- Фильтруемым ключом через CLI-флаг `--filter` (например, `--filter team=infra`)

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
```

## Сабмодули

Сабмодули — вложенные модули внутри родительского модуля. Они обнаруживаются автоматически в директориях глубже уровня паттерна:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
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
```

### Один продукт, несколько окружений

```yaml
structure:
  pattern: "{environment}/{region}/{module}"
```

### По регионам без окружений

```yaml
structure:
  pattern: "{region}/{module}"
```

## Диагностика

### Модули не обнаруживаются

1. Убедитесь, что глубина директорий соответствует паттерну

2. Запустите валидацию:
   ```bash
   terraci validate --verbose
   ```

### Неверные ID модулей

Если ID модулей выглядят некорректно, проверьте соответствие паттерна вашей структуре:

```yaml
# Если структура: env/region/module
structure:
  pattern: "{environment}/{region}/{module}"
```

## Смотрите также

- [Фильтры](/ru/config/filters) — фильтрация модулей с помощью glob-паттернов
- [Структура проекта](/ru/guide/project-structure) — руководство по организации Terraform-проекта
