# Разрешение зависимостей

TerraCi автоматически обнаруживает зависимости между Terraform-модулями, анализируя data-источники `terraform_remote_state`.

## Как это работает

### 1. Парсинг remote state

TerraCi парсит все `.tf` файлы в каждом модуле, ища `terraform_remote_state`:

```hcl
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "my-terraform-state"
    key    = "platform/production/us-east-1/vpc/terraform.tfstate"
    region = "us-east-1"
  }
}
```

### 2. Извлечение путей

Из каждого блока remote state извлекается:
- **Тип бэкенда** (s3, gcs, azurerm и т.д.)
- **Путь к state-файлу** (из `key`, `prefix` или аналогичных полей)
- **Наличие `for_each`**

### 3. Сопоставление с модулями

Путь state-файла сопоставляется с обнаруженными модулями:

```
key: platform/production/us-east-1/vpc/terraform.tfstate
     ↓
Module ID: platform/production/us-east-1/vpc
```

### 4. Построение графа

Зависимости добавляются в направленный ациклический граф (DAG):

```
eks → vpc     (eks зависит от vpc)
rds → vpc     (rds зависит от vpc)
app → eks    (app зависит от eks)
app → rds    (app зависит от rds)
```

## Поддерживаемые бэкенды

| Бэкенд | Поле пути |
|--------|-----------|
| s3 | `key` |
| gcs | `prefix` |
| azurerm | `key` |
| http | `address` |
| consul | `path` |

## Динамические ссылки с `for_each`

TerraCi обрабатывает `for_each` в remote state:

```hcl
locals {
  dependencies = {
    vpc = "platform/production/us-east-1/vpc"
    iam = "platform/production/us-east-1/iam"
  }
}

data "terraform_remote_state" "deps" {
  for_each = local.dependencies

  backend = "s3"
  config = {
    bucket = "my-terraform-state"
    key    = "${each.value}/terraform.tfstate"
  }
}
```

Это создаёт зависимости от модулей `vpc` и `iam`.

## Разрешение локальных переменных

TerraCi разрешает ссылки на локальные переменные в путях:

```hcl
locals {
  env        = "production"
  region     = "us-east-1"
  state_key  = "platform/${local.env}/${local.region}/vpc/terraform.tfstate"
}

data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    key = local.state_key
  }
}
```

## Уровни выполнения

TerraCi группирует модули по уровням выполнения:

```
Уровень 0: [vpc, iam]           # Нет зависимостей
Уровень 1: [eks, rds]           # Зависят от уровня 0
Уровень 2: [app]                # Зависит от уровня 1
```

Модули одного уровня могут выполняться параллельно.

## Кросс-окружающие зависимости

TerraCi поддерживает зависимости, пересекающие границы окружений или регионов. Это полезно, когда модулю в одном окружении нужно ссылаться на ресурсы из другого:

```hcl
# В модуле: cdp/stage/eu-central-1/ec2/db-migrate

# Зависимость в том же окружении/регионе
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    key = "${local.service}/${local.environment}/${local.region}/vpc/terraform.tfstate"
  }
}

# Кросс-окружающая зависимость (захардкоженный путь)
data "terraform_remote_state" "vpn_vpc" {
  backend = "s3"
  config = {
    key = "${local.service}/vpn/eu-north-1/vpc/terraform.tfstate"
  }
}
```

Обе зависимости будут обнаружены:
- `cdp/stage/eu-central-1/vpc` (из динамического пути)
- `cdp/vpn/eu-north-1/vpc` (из захардкоженного кросс-окружающего пути)

TerraCi резолвит переменные `local.*` из структуры пути модуля, позволяя смешивать динамические и захардкоженные пути в одном модуле.

Смотрите [пример cross-env-deps](../../examples/cross-env-deps/) для полного рабочего примера.

## Детекция циклов

TerraCi обнаруживает циклические зависимости:

```bash
terraci validate
```

Вывод:
```
✗ Circular dependency detected:
  module-a → module-b → module-c → module-a
```

Циклические зависимости блокируют генерацию пайплайна.

## Визуализация

Экспортируйте граф зависимостей:

```bash
# DOT формат для GraphViz
terraci graph --format dot -o deps.dot
dot -Tpng deps.dot -o deps.png

# Текстовый список
terraci graph --format list

# Уровни выполнения
terraci graph --format levels
```
