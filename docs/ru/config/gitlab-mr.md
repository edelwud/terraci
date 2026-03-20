---
title: "Merge Request"
description: "Комментарии в MR с результатами plan, оценкой стоимости и проверкой политик"
outline: deep
---

# Интеграция с GitLab MR

TerraCi может автоматически публиковать результаты terraform plan в виде комментариев к Merge Request в GitLab.

## Обзор

При запуске в пайплайне GitLab MR, TerraCi:
1. Сохраняет вывод terraform plan для каждого модуля
2. Собирает все результаты в summary-джобе
3. Публикует форматированный комментарий в MR со сводкой всех планов

## Конфигурация

### Базовая настройка

```yaml
gitlab:
  mr:
    comment:
      enabled: true
    summary_job:
      image:
        name: "ghcr.io/edelwud/terraci:latest"
```

### Все опции

```yaml
gitlab:
  mr:
    # Настройка комментариев
    comment:
      # Включить MR комментарии (по умолчанию: true, если секция mr существует)
      enabled: true
      # Комментировать только при наличии изменений (по умолчанию: false)
      on_changes_only: false
      # Включать полный вывод плана в раскрывающихся секциях (по умолчанию: true)
      include_details: true

    # Метки для добавления к MR (поддерживают плейсхолдеры)
    labels:
      - "terraform"
      - "env:{environment}"
      - "service:{service}"

    # Конфигурация summary job
    summary_job:
      # Docker-образ с terraci
      image:
        name: "ghcr.io/edelwud/terraci:latest"
      # Теги раннера
      tags:
        - docker
```

## Плейсхолдеры в метках

Метки поддерживают следующие плейсхолдеры, которые раскрываются для каждого модуля:

| Плейсхолдер | Описание | Пример |
|-------------|----------|--------|
| `{service}` | Имя сервиса | `platform` |
| `{environment}` | Окружение | `production` |
| `{env}` | Сокращение для environment | `prod` |
| `{region}` | Регион облака | `eu-central-1` |
| `{module}` | Имя модуля | `vpc` |

## Как это работает

### 1. План-джобы

При включенной MR-интеграции план-джобы модифицируются для:
- Использования флага `-detailed-exitcode` для определения изменений
- Сохранения вывода в файл `plan.txt`
- Сохранения `plan.txt` как артефакт (с `when: always`)

```yaml
plan-platform-stage-eu-central-1-vpc:
  script:
    - cd platform/stage/eu-central-1/vpc
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan -detailed-exitcode 2>&1 | tee plan.txt; exit ${PIPESTATUS[0]}
  artifacts:
    paths:
      - platform/stage/eu-central-1/vpc/plan.tfplan
      - platform/stage/eu-central-1/vpc/plan.txt
    expire_in: 1 day
    when: always
```

### 2. Summary Job

Добавляется джоб `terraci-summary`, который:
- Запускается после завершения всех план-джобов
- Запускается только в MR-пайплайнах (`$CI_MERGE_REQUEST_IID`)
- Сканирует файлы `plan.txt` из артефактов
- Публикует/обновляет комментарий в MR через GitLab API

```yaml
terraci-summary:
  stage: summary
  image: ghcr.io/edelwud/terraci:latest
  script:
    - terraci summary
  needs:
    - job: plan-platform-stage-eu-central-1-vpc
      optional: true
  rules:
    - if: $CI_MERGE_REQUEST_IID
      when: always
```

### 3. Формат комментария

Комментарий в MR включает:
- Таблицу-обзор с иконками статусов
- Количество модулей с изменениями/без изменений/с ошибками
- Раскрывающиеся детали с полным выводом плана (если включено)

Пример:
```
## 🔄 Terraform Plan Summary

| Модуль | Статус | Сводка |
|--------|--------|--------|
| `platform/stage/eu-central-1/vpc` | ✅ Изменения | Plan: 2 to add, 1 to change, 0 to destroy |
| `platform/stage/eu-central-1/eks` | ➖ Без изменений | Infrastructure is up-to-date |

<details>
<summary>📋 platform/stage/eu-central-1/vpc</summary>

```
Plan: 2 to add, 1 to change, 0 to destroy.
...
```

</details>
```

## Аутентификация

Summary job требует токен GitLab API:

### Использование CI_JOB_TOKEN

Стандартный `CI_JOB_TOKEN` работает для MR в том же проекте:
```yaml
# Дополнительная конфигурация не требуется
```

### Использование GITLAB_TOKEN

Для кросс-проектных MR или расширенных прав:
```yaml
variables:
  GITLAB_TOKEN: $GITLAB_API_TOKEN
```

Требуемые права: `api` или `write_repository`

## Переменные окружения

Summary job использует эти CI/CD переменные:

| Переменная | Описание |
|------------|----------|
| `CI_MERGE_REQUEST_IID` | Номер MR (определяется автоматически) |
| `CI_PROJECT_ID` | ID проекта (определяется автоматически) |
| `CI_PROJECT_PATH` | Путь проекта (определяется автоматически) |
| `GITLAB_TOKEN` | API токен (откат к `CI_JOB_TOKEN`) |

## Устранение неполадок

### Комментарий не публикуется

1. Проверьте, что MR-интеграция включена:
   ```yaml
   gitlab:
     mr:
       comment:
         enabled: true
   ```

2. Убедитесь, что пайплайн запущен из MR:
   ```bash
   echo $CI_MERGE_REQUEST_IID
   ```

3. Проверьте права токена:
   ```bash
   curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
     "https://gitlab.com/api/v4/projects/$CI_PROJECT_ID/merge_requests/$CI_MERGE_REQUEST_IID"
   ```

### Результаты планов не найдены

1. Убедитесь, что файлы plan.txt существуют в артефактах:
   ```bash
   find . -name "plan.txt"
   ```

2. Проверьте, что план-джобы завершились (даже с ошибками):
   ```yaml
   artifacts:
     when: always  # Обязательно для сохранения при ошибках
   ```

### Summary job отсутствует

Summary job появляется только когда:
1. MR-интеграция включена (секция `gitlab.mr` существует)
2. Планы включены (`gitlab.plan_enabled: true`)

## Смотрите также

- [Summary CLI](/ru/cli/summary) — команда публикации результатов plan в комментариях MR/PR
- [GitLab CI](/ru/config/gitlab) — настройка генерации пайплайнов, включая образы, стейджи и джобы
- [Конфигурация GitHub Actions](/ru/config/github) — эквивалентная интеграция с GitHub PR через секцию `github.pr`
