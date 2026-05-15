---
title: "terraci summary"
description: "Публикация результатов plan, стоимости и политик в комментарии MR/PR"
outline: deep
---

# terraci summary

Публикует результаты terraform plan в виде комментария к GitLab Merge Request или GitHub Pull Request.

## Синтаксис

```bash
terraci summary [flags]
```

## Описание

Команда `summary` собирает результаты terraform plan из артефактов и создаёт или обновляет комментарий с обзором в merge request (GitLab) или pull request (GitHub).

Эта команда предназначена для запуска как DAG-джоб, зависящий от нужных plan/report артефактов. Она загружает plan-результаты из артефактов каждого модуля, обогащает их файлами `{producer}-report.json` (cost, policy, tfupdate), найденными в служебной директории, публикует форматированный комментарий MR/PR и синхронизирует настроенные управляемые TerraCI метки.

Команда автоматически определяет CI-провайдер и контекст MR/PR пайплайна и создаёт комментарии только когда это уместно.

## Использование

Эта команда обычно используется в summary-джобе сгенерированного пайплайна.

### GitLab CI

```yaml
terraci-summary:
  stage: deploy-3
  image: ghcr.io/edelwud/terraci:latest
  script:
    - terraci summary
  needs:
    - job: plan-platform-stage-eu-central-1-vpc
      artifacts: true
  rules:
    - if: $CI_MERGE_REQUEST_IID
      when: always
```

### GitHub Actions

```yaml
summary:
  runs-on: ubuntu-latest
  needs: [plan-jobs...]
  if: github.event_name == 'pull_request'
  steps:
    - uses: actions/checkout@v4
    - run: terraci summary
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Переменные окружения

### GitLab

| Переменная | Описание | Обязательна |
|------------|----------|-------------|
| `CI_MERGE_REQUEST_IID` | Номер MR (определяется GitLab автоматически) | Да |
| `CI_PROJECT_ID` | ID проекта (определяется автоматически) | Да |
| `CI_SERVER_URL` | URL сервера GitLab (определяется автоматически) | Да |
| `GITLAB_TOKEN` | API токен для публикации комментариев | Нет* |
| `CI_JOB_TOKEN` | Резервный токен (предоставляется автоматически) | Нет* |

*Требуется либо `GITLAB_TOKEN`, либо `CI_JOB_TOKEN`.

### GitHub

| Переменная | Описание | Обязательна |
|------------|----------|-------------|
| `GITHUB_ACTIONS` | Индикатор среды GitHub Actions (устанавливается автоматически) | Да |
| `GITHUB_TOKEN` | Токен для публикации комментариев в PR | Да |
| `GITHUB_REPOSITORY` | Репозиторий в формате `owner/repo` (устанавливается автоматически) | Да |
| `GITHUB_EVENT_PATH` | Путь к JSON-файлу события (устанавливается автоматически) | Да |

Для управляемых меток токен также должен иметь право добавлять и удалять labels на MR/PR.

## Вывод

Команда публикует комментарий такого вида в MR/PR:

```markdown
## 🔄 Terraform Plan Summary

| Модуль | Статус | Сводка |
|--------|--------|--------|
| `platform/stage/eu-central-1/vpc` | ✅ Изменения | Plan: 2 to add, 1 to change, 0 to destroy |
| `platform/stage/eu-central-1/eks` | ➖ Без изменений | Infrastructure is up-to-date |

<details>
<summary>📋 platform/stage/eu-central-1/vpc</summary>

Plan: 2 to add, 1 to change, 0 to destroy.
...

</details>
```

## Конфигурация

Плагин `summary` **включён по умолчанию** — явно указывать `enabled: true` не требуется. Его можно отключить через `extensions.summary`:

```yaml
extensions:
  summary:
    enabled: false  # отключить плагин summary
```

Настройте поведение комментария MR/PR через плагин `summary` в `.terraci.yaml`:

```yaml
extensions:
  summary:
    enabled: true
    on_changes_only: false
    include_details: true
    labels:
      - terraform
      - "{environment}"
      - "{module}"
      - "resource:{resource_type}"
```

GitLab/GitHub провайдеры отвечают только за транспорт комментария и CI-контекст; настройки рендера принадлежат `summary`.

`labels` принимает статические значения и шаблоны. Шаблоны модулей могут использовать `{module_id}`, `{module_path}`, `{status}` и любой компонент `structure.pattern`, например `{environment}` или `{module}`. Resource-шаблоны также могут использовать `{resource_address}`, `{resource_type}`, `{resource_name}` и `{resource_action}` и раскрываются только для измененных ресурсов в измененных модулях. TerraCI удаляет только метки, записанные в metadata предыдущего summary-комментария TerraCI, поэтому пользовательские labels не затрагиваются.

## Коды завершения

| Код | Описание |
|-----|----------|
| 0 | Успех (или пропущено, если не в MR/PR) |
| 1 | Ошибка сканирования результатов или публикации комментария |

## Примеры

### Ручной запуск (для тестирования)

```bash
# Установите необходимые переменные окружения
export CI_MERGE_REQUEST_IID=42
export CI_PROJECT_ID=12345
export GITLAB_TOKEN=your-token

terraci summary
```

### С подробным выводом

```bash
terraci summary -v
```

## Смотрите также

- [Конфигурация summary](/ru/config/summary)
- [Конфигурация GitHub Actions](/ru/config/github)
- [terraci generate](/ru/cli/generate)
