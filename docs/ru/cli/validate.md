# terraci validate

Валидация проекта и конфигурации TerraCi.

## Синтаксис

```bash
terraci validate [flags]
```

## Описание

Команда `validate` проверяет:
- Корректность конфигурации `.terraci.yaml`
- Обнаружение Terraform-модулей
- Парсинг HCL-файлов
- Разрешение зависимостей
- Отсутствие циклических зависимостей

## Флаги

| Флаг | Тип | По умолчанию | Описание |
|------|-----|--------------|----------|
| `--verbose` | bool | false | Подробный вывод |

## Примеры

### Базовая валидация

```bash
terraci validate
```

Успешный вывод:
```
Configuration: valid
Modules discovered: 15
Dependencies resolved: 23
Cycles detected: 0

Validation passed
```

### Подробный вывод

```bash
terraci validate --verbose
```

Вывод:
```
Configuration:
  Pattern: {service}/{environment}/{region}/{module}
  Min depth: 4
  Max depth: 5
  Submodules: enabled

Discovered modules:
  platform/prod/eu-central-1/vpc
  platform/prod/eu-central-1/eks
  platform/prod/eu-central-1/rds
  platform/prod/eu-west-1/vpc
  platform/prod/eu-west-1/eks
  ...

Dependencies:
  platform/prod/eu-central-1/eks -> platform/prod/eu-central-1/vpc
  platform/prod/eu-central-1/rds -> platform/prod/eu-central-1/vpc
  platform/prod/eu-west-1/eks -> platform/prod/eu-west-1/vpc
  ...

Execution levels:
  Level 0: vpc (eu-central-1), vpc (eu-west-1)
  Level 1: eks (eu-central-1), eks (eu-west-1), rds (eu-central-1)
  ...

Validation passed
```

### Валидация с конфигом

```bash
terraci validate -c production.terraci.yaml
```

### Валидация директории

```bash
terraci validate -d /path/to/terraform
```

## Проверки

### Конфигурация

- `structure.pattern` указан
- `structure.min_depth >= 1`
- `structure.max_depth >= min_depth`
- `gitlab.terraform_image` указан

### Модули

- Директории содержат `.tf` файлы
- Глубина соответствует `min_depth`/`max_depth`
- Модули соответствуют паттерну структуры

### HCL

- Синтаксис `.tf` файлов корректен
- Блоки `terraform_remote_state` парсятся
- Переменные в путях разрешаются

### Зависимости

- Все ссылки на remote_state разрешаются
- Нет циклических зависимостей
- Граф зависимостей корректен

## Сообщения об ошибках

### Ошибка конфигурации

```
Error: structure.pattern is required
```

Решение: Добавьте pattern в `.terraci.yaml`:
```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
```

### Модули не найдены

```
Warning: No modules discovered
```

Возможные причины:
- Неверная глубина директорий
- Нет `.tf` файлов
- Все модули исключены фильтрами

### Ошибка парсинга HCL

```
Error: Failed to parse platform/prod/eu-central-1/vpc/main.tf: ...
```

Решение: Проверьте синтаксис указанного файла.

### Неразрешённая зависимость

```
Warning: Unresolved dependency in platform/prod/eu-central-1/eks:
  data.terraform_remote_state.unknown_module
```

Модуль ссылается на несуществующий remote_state.

### Циклическая зависимость

```
Error: Circular dependency detected:
  module-a -> module-b -> module-c -> module-a
```

Решение: Устраните циклическую зависимость в Terraform-коде.

## Использование в CI

```yaml
validate:
  stage: test
  script:
    - terraci validate
  rules:
    - changes:
        - "**/*.tf"
        - ".terraci.yaml"
```

## Коды возврата

| Код | Описание |
|-----|----------|
| `0` | Валидация успешна |
| `1` | Ошибка валидации |
| `2` | Ошибка конфигурации |

## Рекомендации

### Pre-commit hook

```bash
#!/bin/sh
# .git/hooks/pre-commit

terraci validate || exit 1
```

### Makefile

```makefile
.PHONY: validate
validate:
	terraci validate --verbose
```

### CI pipeline

```yaml
stages:
  - validate
  - generate
  - deploy

validate:
  stage: validate
  script:
    - terraci validate
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"

generate:
  stage: generate
  needs: [validate]
  script:
    - terraci generate -o pipeline.yml
```
