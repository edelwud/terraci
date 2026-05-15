---
title: Плагин Summary
description: "Сводные комментарии MR/PR: конфигурация и параметры"
outline: deep
---

# Плагин Summary

Плагин summary публикует сводку результатов планирования как комментарий в MR/PR. Он **включён по умолчанию** и собирает результаты всех плагинов (cost, policy) в единый комментарий.

## Конфигурация

```yaml
extensions:
  summary:
    enabled: true            # по умолчанию: true (отключить через false)
    on_changes_only: false   # комментировать только при наличии изменений
    include_details: true    # включить полный вывод плана в раскрываемых секциях
    labels:
      - terraform
      - "{environment}"
      - "{module}"
      - "resource:{resource_type}"
```

## Параметры

### enabled

Включение/отключение плагина. Так как summary использует политику `EnabledByDefault`, он активен пока не отключён явно.

```yaml
extensions:
  summary:
    enabled: false   # отключить сводные комментарии
```

### on_changes_only

Публиковать комментарий только когда план содержит изменения (add/change/destroy).

```yaml
extensions:
  summary:
    on_changes_only: true
```

### include_details

Включить полный вывод плана в раскрываемых `<details>` секциях комментария.

```yaml
extensions:
  summary:
    include_details: true   # по умолчанию
```

### labels

Синхронизировать управляемые TerraCI метки MR/PR после публикации summary-комментария.

```yaml
extensions:
  summary:
    labels:
      - terraform
      - "{environment}"
      - "{module}"
      - "resource:{resource_type}"
```

Статические метки добавляются один раз. Шаблоны без resource-плейсхолдеров раскрываются один раз на измененный или упавший модуль. Шаблоны с плейсхолдерами `resource_*` раскрываются один раз на измененный Terraform-ресурс в измененных модулях.

Поддерживаемые плейсхолдеры:

| Плейсхолдер | Значение |
|-------------|----------|
| `{module_id}` | ID модуля в результате plan |
| `{module_path}` | Путь модуля в результате plan |
| `{status}` | Статус результата plan |
| `{service}`, `{environment}`, `{region}`, `{module}` | Компоненты из `structure.pattern` |
| пользовательские имена сегментов | Любой пользовательский компонент из `structure.pattern` |
| `{resource_address}` | Адрес Terraform-ресурса |
| `{resource_type}` | Тип Terraform-ресурса |
| `{resource_name}` | Имя Terraform-ресурса |
| `{resource_action}` | Действие Terraform (`create`, `update`, `delete`, `replace`, `read`) |

Пустые или неразрешенные метки пропускаются с предупреждением. Сгенерированные метки обрезаются по пробелам, дедуплицируются, сортируются и сохраняют регистр.

Синхронизация управляемая: TerraCI удаляет только метки, которые были сгенерированы предыдущим summary-комментарием TerraCI и отсутствуют в текущем запуске. Пользовательские метки не удаляются. Ошибки синхронизации меток являются только предупреждениями; ошибки публикации комментария по-прежнему завершают `terraci summary` с ошибкой.

## CLI-команда

```bash
terraci summary
```

Команда `terraci summary` сканирует результаты планирования в service directory, загружает отчёты плагинов (cost, policy, tfupdate), формирует markdown-комментарий, публикует его в MR/PR через сервис комментариев активного CI-провайдера и синхронизирует настроенные управляемые метки, если провайдер поддерживает labels.

## См. также

- [terraci summary CLI](/ru/cli/summary)
- [Конфигурация GitLab CI](/ru/config/gitlab)
- [Система плагинов](/ru/plugins/)
