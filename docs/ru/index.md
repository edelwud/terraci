---
layout: home

hero:
  name: TerraCi
  text: Генератор пайплайнов Terraform
  tagline: Генерация GitLab CI пайплайнов с учётом зависимостей для Terraform/OpenTofu монорепозиториев
  image:
    src: /logo.svg
    alt: TerraCi
  actions:
    - theme: brand
      text: Начать
      link: /ru/guide/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/edelwud/terraci

features:
  - icon:
      src: /icons/search.svg
    title: Поиск модулей
    details: Сканирует структуру директорий для поиска Terraform-модулей. Настраиваемая глубина (4-5 уровней).
    link: /ru/guide/project-structure
    linkText: Подробнее
  - icon:
      src: /icons/graph.svg
    title: Граф зависимостей
    details: Извлекает зависимости из terraform_remote_state. Строит DAG с топологической сортировкой.
    link: /ru/guide/dependencies
    linkText: Как это работает
  - icon:
      src: /icons/zap.svg
    title: Параллельное выполнение
    details: Группирует модули по уровням. Независимые модули выполняются параллельно.
    link: /ru/guide/pipeline-generation
    linkText: Пример
  - icon:
      src: /icons/git.svg
    title: Режим изменений
    details: Определяет изменённые файлы через git diff. Генерирует пайплайны только для затронутых модулей.
    link: /ru/guide/git-integration
    linkText: Git интеграция
  - icon:
      src: /icons/tofu.svg
    title: Поддержка OpenTofu
    details: Работает с Terraform и OpenTofu. Переключение одной опцией в конфиге.
    link: /ru/guide/opentofu
    linkText: Настройка
  - icon:
      src: /icons/chart.svg
    title: Визуализация
    details: Экспорт графа зависимостей в DOT. Визуализация через GraphViz.
    link: /ru/cli/graph
    linkText: Команды
---

## Установка

```bash
# Homebrew (macOS/Linux)
brew install edelwud/tap/terraci

# Go
go install github.com/edelwud/terraci/cmd/terraci@latest

# Docker
docker run --rm -v $(pwd):/workspace ghcr.io/edelwud/terraci:latest generate
```

## Использование

```bash
# Инициализация конфига
terraci init

# Генерация пайплайна
terraci generate -o .gitlab-ci.yml

# Только изменённые модули
terraci generate --changed-only --base-ref main
```

## Как это работает

**1. Поиск модулей** по структуре директорий:

```
platform/prod/eu-central-1/
├── vpc/        → platform/prod/eu-central-1/vpc
├── eks/        → platform/prod/eu-central-1/eks
└── rds/        → platform/prod/eu-central-1/rds
```

**2. Извлечение зависимостей** из `terraform_remote_state`:

```hcl
# eks/main.tf
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    key = "platform/prod/eu-central-1/vpc/terraform.tfstate"
  }
}
```

**3. Построение порядка выполнения**:

```
Уровень 0: vpc (нет зависимостей)
Уровень 1: eks, rds (зависят от vpc)
```

**4. Генерация пайплайна**:

```yaml
stages:
  - plan-0
  - apply-0
  - plan-1
  - apply-1

plan-vpc:
  stage: plan-0

apply-vpc:
  stage: apply-0
  needs: [plan-vpc]

plan-eks:
  stage: plan-1
  needs: [apply-vpc]
```

## Конфигурация

```yaml
# .terraci.yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

gitlab:
  image: hashicorp/terraform:1.6
  plan_enabled: true

exclude:
  - "*/test/*"
```

[Полный справочник конфигурации →](/ru/config/)
