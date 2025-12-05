# Генерация пайплайнов

TerraCi генерирует GitLab CI пайплайны с учётом зависимостей модулей и параллельным выполнением.

## Базовая генерация

```bash
terraci generate -o .gitlab-ci.yml
```

## Структура пайплайна

### Стадии

Стадии создаются для каждого уровня выполнения:

```yaml
stages:
  - deploy-plan-0    # Plan для модулей уровня 0
  - deploy-apply-0   # Apply для модулей уровня 0
  - deploy-plan-1    # Plan для модулей уровня 1
  - deploy-apply-1   # Apply для модулей уровня 1
```

### Переменные

Глобальные переменные из конфигурации:

```yaml
variables:
  TERRAFORM_BINARY: "terraform"
  TF_IN_AUTOMATION: "true"
```

### Джобы

Два джоба на модуль (если `plan_enabled: true`):

```yaml
plan-platform-prod-vpc:
  stage: deploy-plan-0
  script:
    - cd platform/prod/us-east-1/vpc
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan
  artifacts:
    paths:
      - platform/prod/us-east-1/vpc/plan.tfplan
    expire_in: 1 day

apply-platform-prod-vpc:
  stage: deploy-apply-0
  script:
    - cd platform/prod/us-east-1/vpc
    - ${TERRAFORM_BINARY} apply plan.tfplan
  needs:
    - job: plan-platform-prod-vpc
  when: manual
```

## Зависимости джобов

Джобы используют `needs` для выражения зависимостей:

```yaml
plan-platform-prod-eks:
  stage: deploy-plan-1
  needs:
    - job: apply-platform-prod-vpc  # Ждёт VPC
```

## Пайплайны только для изменений

Генерация для изменённых модулей и их зависимых:

```bash
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml
```

### Пример

Если изменился `vpc/main.tf`:
- `vpc` включён (изменён)
- `eks` включён (зависит от vpc)
- `rds` включён (зависит от vpc)
- `app` включён (зависит от eks и rds)

## Опции конфигурации

### Стадия plan

```yaml
gitlab:
  plan_enabled: true   # Генерировать plan джобы
  # plan_enabled: false  # Сразу к apply
```

### Auto-approve

```yaml
gitlab:
  auto_approve: false  # Требовать ручной запуск (по умолчанию)
  # auto_approve: true   # Автоматический apply
```

### Пользовательские скрипты

```yaml
gitlab:
  before_script:
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} workspace select ${TF_ENVIRONMENT}
```

## Dry Run

Предпросмотр без генерации:

```bash
terraci generate --dry-run
```

Вывод:
```
Dry Run Summary:
  Total modules: 12
  Affected modules: 5
  Stages: 6
  Jobs: 10

Execution Order:
  Level 0: [vpc]
  Level 1: [eks, rds]
  Level 2: [app]
```
