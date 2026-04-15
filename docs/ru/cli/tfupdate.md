---
title: "terraci tfupdate"
description: "Проверка и обновление версий провайдеров и модулей Terraform с синхронизацией lock-файлов"
outline: deep
---

# terraci tfupdate

Проверка доступных обновлений для провайдеров и модулей Terraform с опциональной записью и синхронизацией lock-файлов.

## Синтаксис

```bash
terraci tfupdate [flags]
```

## Описание

Команда `tfupdate` сканирует Terraform-файлы в обнаруженных модулях, запрашивает Terraform Registry и сообщает о доступных обновлениях версий провайдеров и модулей.

По умолчанию работает в режиме только чтения. Используйте `--write` для записи обновлённых версий в `.tf` файлы и синхронизации `.terraform.lock.hcl`.

Команда не завершается с ошибкой при наличии доступных обновлений — ненулевой код возврата означает ошибку выполнения.

## Флаги

| Флаг | Сокр. | Тип | По умолчанию | Описание |
|------|-------|-----|--------------|----------|
| `--target` | `-t` | string | `all` | Что проверять: `modules`, `providers`, `all` |
| `--bump` | `-b` | string | | Уровень версионирования: `patch`, `minor`, `major` |
| `--write` | `-w` | bool | `false` | Записать обновлённые версии и синхронизировать lock-файлы |
| `--pin` | | bool | `false` | Фиксировать точные версии при записи |
| `--module` | `-m` | string | | Проверить только указанный модуль |
| `--output` | `-o` | string | `text` | Формат вывода: `text`, `json` |
| `--timeout` | | string | | Общий таймаут (например, `15m`) |
| `--lock-platforms` | | []string | | Платформы для h1-хешей lock-файлов |

## Примеры

```bash
# Проверить все зависимости
terraci tfupdate

# Проверить только провайдеры
terraci tfupdate --target providers

# Проверить только модули с patch-уровнем
terraci tfupdate --target modules --bump patch

# Записать обновления и синхронизировать lock-файлы
terraci tfupdate --write

# Фиксировать точные версии
terraci tfupdate --write --pin

# Указать платформы для lock-файлов
terraci tfupdate --write --lock-platforms linux_amd64,darwin_arm64

# Задать таймаут
terraci tfupdate --timeout 15m

# Проверить конкретный модуль
terraci tfupdate --module platform/prod/eu-central-1/vpc

# JSON вывод
terraci tfupdate --output json
```

## Вывод

### Текстовый формат (по умолчанию)

```
• platform/prod/eu-central-1/vpc   updates=2
  • hashicorp/aws registry.terraform.io/hashicorp/aws   current=~> 5.0   available=~> 5.80
  • vpc registry.terraform.io/terraform-aws-modules/vpc   current=~> 5.0   available=~> 5.18
• summary
  • checked   count=15
  • updates available   count=2
```

Если обновлений нет:

```
• summary
  • checked   count=15
• all dependencies are up to date
```

### JSON формат

```bash
terraci tfupdate --output json
```

```json
{
  "providers": [
    {
      "module": "platform/prod/eu-central-1/vpc",
      "source": "registry.terraform.io/hashicorp/aws",
      "current": "~> 5.0",
      "available": "~> 5.80",
      "latest": "5.80.0",
      "status": ""
    }
  ],
  "modules": [],
  "summary": {
    "total_checked": 15,
    "updates_available": 1,
    "updates_applied": 0,
    "errors": 0,
    "skipped": 0
  }
}
```

## Уровни версионирования

| Значение | Описание | Пример: текущая `~> 5.1` |
|----------|----------|--------------------------|
| `patch` | Только patch-обновления | `~> 5.1` → `~> 5.1` |
| `minor` | Minor и patch-обновления | `~> 5.1` → `~> 5.80` |
| `major` | Major, minor и patch | `~> 5.1` → `~> 6.0` |

## Обработка ограничений версий

TerraCi читает и записывает ограничения версий Terraform, сохраняя их стиль.

### Поведение при записи

| Текущее ограничение | Последняя версия | Результат |
|---------------------|-----------------|-----------|
| `~> 5.0` | `5.82.0` | `~> 5.82` |
| `~> 5.0.1` | `5.1.3` | `~> 5.1.3` |
| `>= 1.0` | `2.0.0` | `>= 2.0` |

### Поддерживаемые операторы

`~>`, `>=`, `<=`, `>`, `<`, `=`, `!=`. Составные ограничения (`">= 1.0, < 2.0"`) также поддерживаются.

## Режим записи

При `--write` TerraCi обновляет ограничения версий в `.tf` файлах и автоматически синхронизирует `.terraform.lock.hcl`.

## Синхронизация lock-файлов

При использовании `--write` lock-файлы обновляются автоматически:

- Записи провайдеров создаются или обновляются с новой версией
- `zh:` хеши собираются для всех платформ из метаданных реестра
- `h1:` хеши вычисляются для настроенных платформ (`--lock-platforms`)
- Существующие хеши сохраняются и объединяются с новыми

## Требования

- `plugins.tfupdate.enabled: true` в `.terraci.yaml`
- `plugins.tfupdate.policy.bump` должен быть указан (через конфиг или `--bump`)
- Доступ к Terraform Registry

## Конфигурация

```yaml
plugins:
  tfupdate:
    enabled: true
    target: all
    policy:
      bump: minor
      pin: false
    ignore:
      - registry.terraform.io/hashicorp/null
    lock:
      platforms:
        - linux_amd64
        - darwin_arm64
    pipeline: false
    timeout: "15m"
```

Флаги командной строки имеют приоритет над настройками конфигурационного файла.

## Интеграция с CI пайплайном

При `plugins.tfupdate.pipeline: true` TerraCi добавляет джоб `tfupdate-check` в начало пайплайна. Джоб помечается как `allow_failure: true`.

## Артефакты

- `tfupdate-results.json` — полные результаты
- `tfupdate-report.json` — сводный отчёт для CI

## Коды завершения

| Код | Описание |
|-----|----------|
| 0 | Сканирование завершено успешно |
| ненулевой | Ошибка парсинга, реестра или записи |

## Смотрите также

- [Конфигурация tfupdate](/ru/config/tfupdate) — все параметры плагина
- [terraci generate](/ru/cli/generate) — генерация CI пайплайна
