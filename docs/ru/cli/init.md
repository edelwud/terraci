---
title: "terraci init"
description: "Инициализация .terraci.yaml через интерактивный TUI-мастер или CLI-флаги"
outline: deep
---

# terraci init

Инициализация конфигурационного файла TerraCi.

## Синтаксис

```bash
terraci init [flags]
```

## Описание

Команда `init` создаёт файл `.terraci.yaml`. По умолчанию запускается интерактивный TUI-мастер, который проведёт вас через выбор конфигурации. Используйте `--ci` для неинтерактивного режима, подходящего для автоматизации, или передайте конкретные флаги для пропуска мастера.

## Флаги

| Флаг | Сокр. | Тип | По умолчанию | Описание |
|------|-------|-----|--------------|----------|
| `--force` | `-f` | bool | false | Перезаписать существующий файл |
| `--ci` | | bool | false | Неинтерактивный режим (пропустить TUI-мастер) |
| `--provider` | | string | | CI-провайдер: `gitlab` или `github` |
| `--binary` | | string | | Бинарный файл: `terraform` или `tofu` |
| `--pattern` | | string | | Паттерн структуры директорий |

При указании любого из флагов `--provider`, `--binary` или `--pattern` мастер автоматически пропускается и используется неинтерактивный режим.

## Примеры

### Интерактивный режим (по умолчанию)

```bash
terraci init
```

Запускает TUI-мастер со следующими группами:
1. **Основное** — CI-провайдер (GitLab CI или GitHub Actions), бинарный файл Terraform (Terraform или OpenTofu)
2. **Структура** — паттерн структуры директорий
3. **Опции пайплайна** — включение стадии plan и auto-approve
4. **Группы плагинов** — динамические группы от включённых плагинов (gitlab/github image, summary, cost, policy, tfupdate и т.д.)

### Неинтерактивный режим

```bash
terraci init --ci
```

Создаёт `.terraci.yaml` со значениями по умолчанию без запросов.

### Выбор провайдера

```bash
# Конфигурация для GitHub Actions
terraci init --provider github

# Конфигурация для GitLab CI
terraci init --provider gitlab
```

При `--provider github` сгенерированный конфиг будет содержать секцию `extensions.github` (с `runs_on`, `steps_before` и т.д.) и не будет содержать секцию `extensions.gitlab`. При `--provider gitlab` — наоборот.

### Настройка OpenTofu

```bash
terraci init --provider gitlab --binary tofu
```

Автоматически выбирается соответствующий образ (`ghcr.io/opentofu/opentofu:1.6`), а для GitHub Actions используется `opentofu/setup-opentofu@v1` вместо `hashicorp/setup-terraform@v3`.

### Пользовательский паттерн

```bash
terraci init --pattern "{team}/{stack}/{datacenter}/{component}"
```

### Полный неинтерактивный пример

```bash
terraci init --provider github --binary tofu --pattern "{service}/{environment}/{region}/{module}"
```

### Перезапись существующего

```bash
terraci init --force
```

Перезаписывает `.terraci.yaml` без запроса подтверждения.

### Инициализация в другой директории

```bash
terraci -d /path/to/project init
```

Создаёт конфигурацию в указанной директории.

## Генерируемая конфигурация

### Провайдер GitLab

При `--provider gitlab` (или по умолчанию) создаётся:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

execution:
  binary: terraform
  plan_enabled: true

extensions:
  gitlab:
    image:
      name: hashicorp/terraform:1.6
    stages_prefix: deploy
    cache_enabled: true
    auto_approve: false
    mr:
      comment:
        enabled: true
```

### Провайдер GitHub

При `--provider github` создаётся:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

execution:
  binary: terraform
  plan_enabled: true

extensions:
  github:
    runs_on: ubuntu-latest
    auto_approve: false
    permissions:
      contents: read
      pull-requests: write
    job_defaults:
      steps_before:
        - uses: actions/checkout@v4
        - uses: hashicorp/setup-terraform@v3
    pr:
      comment: {}
```

> Конфигурация backend (S3, GCS и т.д.) не генерируется командой `terraci init` — TerraCi читает существующие блоки `terraform { backend "..." }` из ваших модулей для разрешения путей state-файлов; ничего не добавляется в `.terraci.yaml`.

## Что создаётся

Команда создаёт:
- `.terraci.yaml` в текущей (или указанной) директории

Команда НЕ изменяет:
- Существующие Terraform-файлы
- Конфигурацию CI
- Другие файлы проекта

## После инициализации

1. **Проверьте конфигурацию**
   ```bash
   cat .terraci.yaml
   ```

2. **Настройте под свой проект**
   - Измените паттерн под вашу структуру
   - Обновите Docker-образ или метки раннеров
   - Добавьте исключения
   - Настройте backend

3. **Валидация**
   ```bash
   terraci validate
   ```

4. **Сгенерируйте первый пайплайн**
   ```bash
   terraci generate --dry-run
   # Для GitLab:
   terraci generate -o .gitlab-ci.yml
   # Для GitHub:
   terraci generate -o .github/workflows/terraform.yml
   ```

## Устранение проблем

### Файл уже существует

```
Error: config file already exists: .terraci.yaml (use --force to overwrite)
```

Решение: Используйте `--force` или вручную отредактируйте существующий файл.

### Нет прав на запись

```
Error: permission denied: .terraci.yaml
```

Решение: Проверьте права на директорию и файл.

## Смотрите также

- [Обзор конфигурации](/ru/config/) — справочник конфигурации .terraci.yaml
- [Быстрый старт](/ru/guide/getting-started) — начало работы с TerraCi
