# Быстрый старт

Это руководство поможет настроить TerraCi и сгенерировать первый пайплайн.

## Установка

### Через Go

```bash
go install github.com/edelwud/terraci/cmd/terraci@latest
```

### Через Docker

```bash
docker pull ghcr.io/edelwud/terraci:latest
docker run --rm -v $(pwd):/workspace ghcr.io/edelwud/terraci generate
```

### Из исходников

```bash
git clone https://github.com/edelwud/terraci.git
cd terraci
make build
./terraci version
```

## Быстрый старт

### 1. Инициализация конфигурации

Перейдите в корень вашего Terraform-проекта и выполните:

```bash
terraci init
```

Это создаст файл `.terraci.yaml`:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true

gitlab:
  terraform_binary: "terraform"
  terraform_image: "hashicorp/terraform:1.6"
  plan_enabled: true
  auto_approve: false
```

### 2. Валидация проекта

Проверьте, что TerraCi корректно обнаруживает модули:

```bash
terraci validate
```

Ожидаемый вывод:

```
✓ Found 12 modules
✓ Built dependency graph with 15 edges
✓ No circular dependencies detected
✓ 4 execution levels identified

Execution order:
  Level 0: vpc, iam
  Level 1: eks, rds, elasticache
  Level 2: app-backend, app-frontend
  Level 3: monitoring
```

### 3. Визуализация зависимостей (опционально)

Экспортируйте граф зависимостей:

```bash
terraci graph --format dot -o deps.dot
dot -Tpng deps.dot -o deps.png
```

### 4. Генерация пайплайна

Сгенерируйте GitLab CI пайплайн:

```bash
terraci generate -o .gitlab-ci.yml
```

### 5. Генерация только для изменённых модулей

Для инкрементальных деплоев:

```bash
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml
```

## Структура проекта

TerraCi ожидает следующую структуру директорий:

```
your-project/
├── .terraci.yaml          # Конфигурация TerraCi
├── service-a/
│   ├── production/
│   │   └── us-east-1/
│   │       ├── vpc/
│   │       │   └── main.tf
│   │       └── eks/
│   │           └── main.tf
│   └── staging/
│       └── us-east-1/
│           └── vpc/
│               └── main.tf
└── service-b/
    └── production/
        └── eu-west-1/
            └── rds/
                └── main.tf
```

Паттерн `{service}/{environment}/{region}/{module}` соответствует:
- `service-a/production/us-east-1/vpc`
- `service-a/production/us-east-1/eks`
- `service-b/production/eu-west-1/rds`

## Определение зависимостей

TerraCi обнаруживает зависимости из data-источников `terraform_remote_state`:

```hcl
# В eks/main.tf
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "my-terraform-state"
    key    = "service-a/production/us-east-1/vpc/terraform.tfstate"
    region = "us-east-1"
  }
}

resource "aws_eks_cluster" "main" {
  vpc_config {
    subnet_ids = data.terraform_remote_state.vpc.outputs.private_subnet_ids
  }
}
```

TerraCi парсит путь `key` и сопоставляет его с модулем `vpc`.

## Следующие шаги

- [Структура проекта](/ru/guide/project-structure) — поддерживаемые структуры директорий
- [Разрешение зависимостей](/ru/guide/dependencies) — как определяются зависимости
- [Справочник конфигурации](/ru/config/) — все опции конфигурации
- [Справочник CLI](/ru/cli/) — все доступные команды
