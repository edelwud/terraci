---
title: "Сабмодули"
description: "Вложенные сабмодули на глубине 5: примеры использования, зависимости и лучшие практики"
outline: deep
---

# Сабмодули

TerraCi поддерживает вложенные сабмодули на глубине 5 для организации связанной инфраструктуры внутри родительского модуля.

## Что такое сабмодули?

Сабмодули — это Terraform-модули, вложенные на один уровень глубже стандартного паттерна:

```
infrastructure/
└── platform/
    └── production/
        └── us-east-1/
            └── ec2/                    # Родительский модуль (глубина 4)
                ├── main.tf             # Файлы родительского модуля
                ├── rabbitmq/           # Сабмодуль (глубина 5)
                │   └── main.tf
                └── redis/              # Сабмодуль (глубина 5)
                    └── main.tf
```

## Идентификация модулей

| Путь | Тип | ID |
|------|-----|-----|
| `platform/production/us-east-1/ec2` | Родитель | `platform/production/us-east-1/ec2` |
| `platform/production/us-east-1/ec2/rabbitmq` | Сабмодуль | `platform/production/us-east-1/ec2/rabbitmq` |

## Конфигурация

Включите сабмодули в `.terraci.yaml`:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5              # Разрешить глубину 5
  allow_submodules: true    # Включить обнаружение сабмодулей
```

## Примеры использования

### Группировка EC2 инстансов

```
ec2/
├── main.tf           # Общие security groups, IAM roles
├── rabbitmq/
│   └── main.tf       # EC2 для RabbitMQ
└── elasticsearch/
    └── main.tf       # EC2 для Elasticsearch
```

### Кластеры баз данных

```
databases/
├── main.tf           # Общие VPC, подсети
├── postgresql/
│   └── main.tf       # PostgreSQL RDS
├── redis/
│   └── main.tf       # ElastiCache Redis
└── mongodb/
    └── main.tf       # DocumentDB
```

### Микросервисы

Группировка микросервисов по доменам:

```
services/
├── auth/
│   └── main.tf
├── payments/
│   └── main.tf
└── notifications/
    └── main.tf
```

## Зависимости сабмодулей

Сабмодули могут зависеть от:
- Родительского модуля
- Других сабмодулей того же родителя
- Модулей вне родителя

### Зависимость от родителя

```hcl
# В ec2/rabbitmq/main.tf
data "terraform_remote_state" "ec2_parent" {
  backend = "s3"
  config = {
    key = "platform/production/us-east-1/ec2/terraform.tfstate"
  }
}
```

### Зависимости между сабмодулями

```hcl
# В ec2/app/main.tf
data "terraform_remote_state" "rabbitmq" {
  backend = "s3"
  config = {
    key = "platform/production/us-east-1/ec2/rabbitmq/terraform.tfstate"
  }
}
```

## Сопоставление имён

TerraCi использует умное сопоставление имён для сабмодулей:

| Имя remote state | Соответствует |
|------------------|---------------|
| `ec2_rabbitmq` | `ec2/rabbitmq` |
| `ec2-rabbitmq` | `ec2/rabbitmq` |
| `rabbitmq` | `ec2/rabbitmq` (в том же контексте) |

## Сгенерированный пайплайн

Сабмодули отображаются как обычные джобы:

```yaml
plan-platform-prod-us-east-1-ec2:
  stage: deploy-plan-0
  # ...

plan-platform-prod-us-east-1-ec2-rabbitmq:
  stage: deploy-plan-1
  needs:
    - apply-platform-prod-us-east-1-ec2
  # ...
```

## Индекс родительских модулей

TerraCi поддерживает индекс связей родитель-потомок:

```go
type Module struct {
    components map[string]string // именованные сегменты из паттерна
    segments   []string          // упорядоченные имена сегментов

    Path         string
    RelativePath string
    Parent       *Module   // Ссылка на родителя (для сабмодулей)
    Children     []*Module // Ссылки на потомков (для родителей)
}
```

Это позволяет:
- Запрашивать все сабмодули родителя
- Находить родителя сабмодуля (через `m.Parent`)
- Проверять, является ли модуль сабмодулем (через `m.IsSubmodule()`)
- Строить точные цепочки зависимостей

## Лучшие практики

### 1. Группируйте связанные ресурсы

Объединяйте ресурсы, которые деплоятся вместе:

```
app/
├── main.tf           # ECS-кластер, балансировщик нагрузки
├── api/
│   └── main.tf       # API-сервис
└── worker/
    └── main.tf       # Фоновый воркер
```

### 2. Выносите общую конфигурацию в родитель

Размещайте общие ресурсы в родительском модуле:

```hcl
# В ec2/main.tf
resource "aws_security_group" "shared" {
  name = "ec2-shared-sg"
}

output "shared_security_group_id" {
  value = aws_security_group.shared.id
}
```

```hcl
# В ec2/rabbitmq/main.tf
data "terraform_remote_state" "parent" {
  # ...
}

resource "aws_instance" "rabbitmq" {
  vpc_security_group_ids = [
    data.terraform_remote_state.parent.outputs.shared_security_group_id
  ]
}
```

### 3. Единообразное именование

Используйте единообразные имена для сабмодулей:

```
✓ ec2/rabbitmq
✓ ec2/redis
✓ ec2/elasticsearch

✗ ec2/rabbit-mq
✗ ec2/Redis
✗ ec2/es
```

### 4. Ограничивайте глубину вложенности

TerraCi поддерживает только один уровень сабмодулей (глубина 5). Для более глубоких иерархий рекомендуется реструктуризация:

```
# Вместо:
platform/prod/us-east-1/services/backend/api/main.tf  # Слишком глубоко!

# Используйте:
platform/prod/us-east-1/backend-api/main.tf           # Плоская структура
```

## Фильтрация сабмодулей

```bash
# Только сабмодули
terraci generate --include "*/*//*/*/**"

# Исключить конкретный сабмодуль
terraci generate --exclude "*/*/us-east-1/ec2/rabbitmq"

# Только родительские модули
terraci generate --exclude "*/*/*/*/*"
```

## Устранение неполадок

### Сабмодули не обнаруживаются

1. Проверьте, что `max_depth` установлен в 5
2. Убедитесь, что `allow_submodules: true`
3. Убедитесь, что директория сабмодуля содержит `.tf` файлы

### Родитель не привязан

Если родитель сабмодуля не определяется:

1. Убедитесь, что родитель существует на глубине 4
2. Проверьте, что родитель содержит `.tf` файлы
3. Запустите `terraci validate -v` для просмотра деталей обнаружения

## Следующие шаги

- [Структура проекта](/ru/guide/project-structure) — паттерны директорий и настройка глубины
- [Фильтры](/ru/config/filters) — include/exclude паттерны для выбора модулей
