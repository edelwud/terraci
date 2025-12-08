---
layout: home

hero:
  name: "TerraCi"
  text: "Генератор пайплайнов Terraform"
  tagline: Автоматическая генерация GitLab CI пайплайнов с учётом зависимостей для ваших Terraform/OpenTofu монорепозиториев
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
    title: Умное обнаружение
    details: Автоматически находит Terraform-модули на основе структуры директорий. Поддерживает вложенные сабмодули на глубине 4 и 5.
    link: /ru/guide/project-structure
    linkText: Подробнее
  - icon:
      src: /icons/graph.svg
    title: Разрешение зависимостей
    details: Парсит блоки terraform_remote_state для построения точного графа зависимостей. Поддерживает for_each и динамические ссылки.
    link: /ru/guide/dependencies
    linkText: Как это работает
  - icon:
      src: /icons/zap.svg
    title: Параллельное выполнение
    details: Группирует независимые модули в уровни выполнения для максимальной параллелизации при соблюдении зависимостей.
    link: /ru/guide/pipeline-generation
    linkText: Пример
  - icon:
      src: /icons/git.svg
    title: Пайплайны для изменений
    details: Git-интеграция определяет изменённые файлы и генерирует пайплайны только для затронутых модулей и их зависимых.
    link: /ru/guide/git-integration
    linkText: Git интеграция
  - icon:
      src: /icons/tofu.svg
    title: Поддержка OpenTofu
    details: Полноценная поддержка Terraform и OpenTofu. Достаточно изменить одну опцию в конфиге.
    link: /ru/guide/opentofu
    linkText: Настройка
  - icon:
      src: /icons/chart.svg
    title: Визуализация графа
    details: Экспорт графа зависимостей в формат DOT для визуализации с помощью GraphViz.
    link: /ru/cli/graph
    linkText: Команды
---

## Быстрый пример

```bash
# Инициализация конфигурации
terraci init

# Генерация пайплайна для всех модулей
terraci generate -o .gitlab-ci.yml

# Генерация пайплайна только для изменённых модулей
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml
```

## Как это работает

TerraCi анализирует структуру вашего Terraform-проекта:

```
infrastructure/
├── service/
│   └── environment/
│       └── region/
│           ├── vpc/          # Модуль на глубине 4
│           ├── eks/          # Зависит от vpc
│           └── ec2/
│               └── rabbitmq/ # Сабмодуль на глубине 5
```

Парсит data-источники `terraform_remote_state` для определения зависимостей:

```hcl
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "terraform-state"
    key    = "cdp/stage/eu-central-1/vpc/terraform.tfstate"
  }
}
```

И генерирует GitLab CI пайплайн с правильным порядком выполнения:

```yaml
stages:
  - deploy-plan-0
  - deploy-apply-0
  - deploy-plan-1
  - deploy-apply-1

plan-cdp-stage-eu-central-1-vpc:
  stage: deploy-plan-0
  script:
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan

plan-cdp-stage-eu-central-1-eks:
  stage: deploy-plan-1
  needs:
    - apply-cdp-stage-eu-central-1-vpc
```
