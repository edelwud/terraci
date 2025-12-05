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

## Сопоставление имён

TerraCi использует умное сопоставление имён для сабмодулей:

| Имя remote state | Соответствует |
|------------------|---------------|
| `ec2_rabbitmq` | `ec2/rabbitmq` |
| `ec2-rabbitmq` | `ec2/rabbitmq` |
| `rabbitmq` | `ec2/rabbitmq` (в том же контексте) |

## Фильтрация сабмодулей

```bash
# Только сабмодули
terraci generate --include "*/*//*/*/**"

# Исключить конкретный сабмодуль
terraci generate --exclude "*/*/us-east-1/ec2/rabbitmq"

# Только родительские модули
terraci generate --exclude "*/*/*/*/*"
```
