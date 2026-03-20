---
layout: home

hero:
  name: TerraCi
  text: Генератор пайплайнов Terraform
  tagline: CI пайплайны с учётом зависимостей для Terraform/OpenTofu монорепозиториев — GitLab CI и GitHub Actions, с оценкой стоимости и проверкой политик
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
      src: /icons/graph.svg
    title: Граф зависимостей
    details: Парсит terraform_remote_state для построения DAG. Топологическая сортировка гарантирует правильный порядок выполнения.
    link: /ru/guide/dependencies
    linkText: Как это работает
  - icon:
      src: /icons/zap.svg
    title: Параллельное выполнение
    details: Группирует модули по уровням. Независимые модули выполняются параллельно — ждут только зависимые.
    link: /ru/guide/pipeline-generation
    linkText: Структура пайплайна
  - icon:
      src: /icons/git.svg
    title: Режим изменений
    details: Определяет изменённые файлы через git diff. Генерирует пайплайны только для затронутых модулей и их зависимых.
    link: /ru/guide/git-integration
    linkText: Git интеграция
  - icon:
      src: /icons/shield.svg
    title: Проверка политик
    details: Контроль соответствия с OPA-политиками на каждом плане. Блокировка или предупреждение — результаты в MR-комментариях.
    link: /ru/config/policy
    linkText: Настроить политики
  - icon:
      src: /icons/dollar.svg
    title: Оценка стоимости
    details: Оценка месячной стоимости AWS из terraform планов. Разница before/after по каждому модулю в MR.
    link: /ru/config/cost
    linkText: Настроить оценку
  - icon:
      src: /icons/tofu.svg
    title: Поддержка OpenTofu
    details: Полноценная поддержка Terraform и OpenTofu. Переключение одной опцией в конфиге.
    link: /ru/guide/opentofu
    linkText: Настройка
---

## Быстрый старт

```bash
# Установка
brew install edelwud/tap/terraci

# Инициализация и генерация (GitLab)
terraci init
terraci generate -o .gitlab-ci.yml

# Инициализация и генерация (GitHub Actions)
terraci init --provider github
terraci generate -o .github/workflows/terraform.yml

# Только изменённые модули
terraci generate --changed-only --base-ref main
```

## Как это работает

```mermaid
flowchart LR
  subgraph repo["Репозиторий"]
    r1["vpc/"]
    r2["eks/"]
    r3["rds/"]
  end
  subgraph terraci["TerraCi"]
    t1["Поиск"] --> t2["Парсинг"] --> t3["Сортировка"] --> t4["Генерация"]
  end
  subgraph ci["CI пайплайн"]
    g1["plan vpc"] --> g2["apply vpc"]
    g2 --> g3["plan eks + rds"]
    g3 --> g4["apply eks + rds"]
  end
  repo --> terraci --> ci
```

[Полный справочник конфигурации →](/ru/config/)
