---
title: Плагин Summary
description: "Сводные комментарии MR/PR: конфигурация и параметры"
outline: deep
---

# Плагин Summary

Плагин summary публикует сводку результатов планирования как комментарий в MR/PR. Он **включён по умолчанию** и собирает результаты всех плагинов (cost, policy) в единый комментарий.

## Конфигурация

```yaml
plugins:
  summary:
    enabled: true            # по умолчанию: true (отключить через false)
    on_changes_only: false   # комментировать только при наличии изменений
    include_details: true    # включить полный вывод плана в раскрываемых секциях
```

## Параметры

### enabled

Включение/отключение плагина. Так как summary использует политику `EnabledByDefault`, он активен пока не отключён явно.

```yaml
plugins:
  summary:
    enabled: false   # отключить сводные комментарии
```

### on_changes_only

Публиковать комментарий только когда план содержит изменения (add/change/destroy).

```yaml
plugins:
  summary:
    on_changes_only: true
```

### include_details

Включить полный вывод плана в раскрываемых `<details>` секциях комментария.

```yaml
plugins:
  summary:
    include_details: true   # по умолчанию
```

## CLI-команда

```bash
terraci summary
```

Команда `terraci summary` сканирует результаты планирования в service directory, загружает отчёты плагинов (cost, policy), формирует markdown-комментарий и публикует его в MR/PR через сервис комментариев активного CI-провайдера.

## См. также

- [terraci summary CLI](/ru/cli/summary)
- [GitLab MR интеграция](/ru/config/gitlab-mr)
- [Система плагинов](/ru/plugins/)
