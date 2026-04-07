# Cost Plugin Refactor — Plan v3
**Date:** 2026-04-07  
**Target score:** 9.5 / 10  
**Status:** DONE — all 12 tasks completed (2026-04-07)

---

## Контекст

После двух итераций рефакторинга (v1: 9 задач, v2: 17 задач) плагин достиг оценки **8.0/10**.
Оставшиеся проблемы разбиты на три группы приоритетов.

---

## P0 — Системная граница: engine знает про AWS (3 задачи)

Это единственная причина, по которой оценка не выше 8. `engine/estimator.go` импортирует
`cloud/awskit` чтобы получить строку `"aws"`. Это должно быть устранено полностью.

### Задача 1: `CostProvidersConfig` → `map[string]ProviderConfig` [x]

**Файл:** `plugins/cost/internal/model/config_types.go:30-32`

**До:**
```go
type CostProvidersConfig struct {
    AWS *ProviderConfig `yaml:"aws,omitempty"`
}
```

**После:**
```go
// CostProvidersConfig maps provider IDs (e.g. "aws") to their enable flag.
// Keys must match the ConfigKey declared in cloud.Definition.
type CostProvidersConfig map[string]ProviderConfig
```

**Затронутые места:**
- `model/config_types.go:30-32` — заменить struct на map type
- `model/config_types.go:40-45` — `HasEnabledProviders()` итерирует по map
- `engine/estimator.go:84-92` — `enabledProviderIDs()` итерирует по map (awskit импорт уйдёт)
- `lifecycle_test.go` — все места `Providers.AWS` → `Providers["aws"]`
- `init_wizard.go` — если есть обращение к `Providers.AWS`

**Проверка:** `go build ./plugins/cost/...` + `go test ./plugins/cost/...`

---

### Задача 2: `cloud.Definition` добавляет поле `ConfigKey string` [x]

**Файл:** `plugins/cost/internal/cloud/registry.go:19-23`

**До:**
```go
type Definition struct {
    Manifest       pricing.ProviderManifest
    FetcherFactory func(pricing.ProviderManifest) pricing.PriceFetcher
    Resources      []ResourceRegistration
}
```

**После:**
```go
type Definition struct {
    // ConfigKey is the YAML key under plugins.cost.providers that enables this provider.
    // Example: "aws" maps to `plugins.cost.providers.aws.enabled: true`.
    ConfigKey      string
    Manifest       pricing.ProviderManifest
    FetcherFactory func(pricing.ProviderManifest) pricing.PriceFetcher
    Resources      []ResourceRegistration
}
```

**Затронутые места:**
- `cloud/registry.go:19-23` — добавить поле `ConfigKey`
- `cloud/aws/resources.go:22-60` — `Definition = cloud.Definition{ConfigKey: "aws", ...}`

**Проверка:** `go build ./plugins/cost/internal/cloud/...`

---

### Задача 3: `enabledProviderIDs` в `engine` итерирует `cloud.Providers()`, убрать `awskit` импорт [x]

**Файл:** `plugins/cost/internal/engine/estimator.go:84-116`

**До:**
```go
import "github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"

func enabledProviderIDs(cfg *model.CostConfig) []string {
    if cfg.Providers.AWS != nil && cfg.Providers.AWS.Enabled {
        return []string{awskit.ProviderID}
    }
    return nil
}
```

**После (awskit импорт исчезает из engine):**
```go
func enabledProviderIDs(cfg *model.CostConfig) []string {
    var ids []string
    for id, p := range cfg.Providers {
        if p.Enabled {
            ids = append(ids, id)
        }
    }
    return ids
}
```

Функция `configuredProviders` при этом не меняется — она уже итерирует `cloud.Providers()` и
фильтрует по `enabledIDs`. После Задачи 1 связь `id → awskit.ProviderID` становится
`id == def.ConfigKey`, что может быть проверено через `cloud.Providers()` если нужна валидация.

**Затронутые места:**
- `engine/estimator.go:10` — удалить `awskit` import
- `engine/estimator.go:84-92` — переписать `enabledProviderIDs`
- `engine/estimator_test.go` — тесты с `awskit.ProviderID` как map-ключом — заменить на строку `"aws"` там где тест строит `runtimes map`

**Проверка:** `go build ./plugins/cost/...` + `go test ./plugins/cost/...`

---

## P1 — Архитектурные улучшения (4 задачи)

### Задача 4: `Estimator` — убрать 8 pass-through методов, ввести `CacheInspector`

**Файл:** `plugins/cost/internal/engine/estimator.go:128-160`

Текущие 8 методов (`CacheDir`, `SetFetcherForProvider`, `CacheOldestAge`, `CacheTTL`,
`CleanExpiredCache`, `CacheEntries`, `SourceName`, `GetIndex`) делегируют напрямую в
`ProviderRuntimeRegistry`. Это делает публичный API `Estimator` раздутым диагностическими
поверхностями, которые не нужны при нормальной оценке.

**Решение:** выделить `CacheInspector` интерфейс; `Estimator` возвращает `Cache()` accessor.

```go
// engine/cache.go — новый файл
// CacheInspector provides diagnostic and maintenance access to the pricing cache.
type CacheInspector interface {
    Dir() string
    TTL() time.Duration
    OldestAge(ctx context.Context) time.Duration
    Entries(ctx context.Context) []pricing.CacheEntry
    CleanExpired(ctx context.Context)
}

// cacheInspector wraps ProviderRuntimeRegistry to implement CacheInspector.
type cacheInspector struct{ r *costruntime.ProviderRuntimeRegistry }

func (c *cacheInspector) Dir() string                                  { return c.r.CacheDir() }
func (c *cacheInspector) TTL() time.Duration                           { return c.r.CacheTTL() }
func (c *cacheInspector) OldestAge(ctx context.Context) time.Duration  { return c.r.CacheOldestAge(ctx) }
func (c *cacheInspector) Entries(ctx context.Context) []pricing.CacheEntry { return c.r.CacheEntries(ctx) }
func (c *cacheInspector) CleanExpired(ctx context.Context)             { c.r.CleanExpiredCache(ctx) }
```

```go
// Estimator — после
func (e *Estimator) Cache() CacheInspector {
    return &cacheInspector{r: e.runtimes}
}

// Убрать все 8 pass-through методов.
// Оставить только:
// - EstimateModule / EstimateModules / ValidateAndPrefetch — публичный функциональный API
// - Resolver() — нужен только для тестов; при необходимости тоже убрать
```

**Затронутые места:**
- `engine/estimator.go:128-160` — удалить 8 методов, добавить `Cache() CacheInspector`
- `engine/cache.go` — создать файл с интерфейсом и адаптером
- `runtime.go:101-129` — `logCacheState` использует `e.CacheDir()` → `e.Cache().Dir()` и т.д.
- `enginetest/helpers.go` — если использует старые методы напрямую
- `engine/estimator_test.go:604-628` — `TestEstimator_CacheAccessors` — обновить на новый API

**Проверка:** `go build ./plugins/cost/...` + `go test ./plugins/cost/...`

---

### Задача 5: `resolveBlobStore` + `resolveBlobCache` — слить в одну функцию

**Файл:** `plugins/cost/runtime.go:67-96`

`resolveBlobStore` существует только как промежуточный шаг `resolveBlobCache`. Он возвращает
`BlobStoreInfo`, которое тут же отбрасывается (`_`). Два названия для одной операции.

**До:**
```go
func resolveBlobStore(ctx, appCtx, cfg) (plugin.BlobStore, plugin.BlobStoreInfo, error)
func resolveBlobCache(ctx, appCtx, cfg) (*blobcache.Cache, error) {
    blobStore, _, err := resolveBlobStore(ctx, appCtx, cfg)
    ...
}
```

**После — одна функция:**
```go
// resolveBlobCache resolves the underlying blob store and wraps it in a blobcache.Cache
// configured with the plugin's namespace and TTL settings.
func resolveBlobCache(ctx context.Context, appCtx *plugin.AppContext, cfg *model.CostConfig) (*blobcache.Cache, error) {
    blobStore, info, err := appCtx.ResolveBlobStoreProvider(ctx, cfg.BlobCacheBackend())
    if err != nil { return nil, fmt.Errorf("blob store %q: %w", cfg.BlobCacheBackend(), err) }
    log.FromContext(ctx).WithField("backend", info.BackendName).
        WithField("root", info.RootPath).Debug("cost: blob store resolved")
    return blobcache.New(blobStore, cfg.BlobCacheNamespace(), cfg.BlobCacheTTL()), nil
}
```

`resolveBlobStore` удаляется полностью.

**Затронутые места:**
- `runtime.go:67-96` — слить в одну функцию
- `lifecycle_test.go` — если тест проверяет `resolveBlobStore` напрямую (нет, проверяет runtime)

**Проверка:** `go build ./plugins/cost/...` + `go test ./plugins/cost/...`

---

### Задача 6: `newRuntimeWithEstimator` — удалить мёртвый конструктор

**Файл:** `plugins/cost/runtime.go:41-43`

```go
func newRuntimeWithEstimator(estimator *engine.Estimator) *costRuntime {
    return &costRuntime{estimator: estimator}
}
```

Нигде не вызывается в production-коде. В тестах `enginetest/helpers.go` строит `costRuntime`
через `engine.NewEstimatorFromConfig`. Удалить.

**Затронутые места:**
- `runtime.go:41-43` — удалить функцию
- Поиск всех вызовов — `grep -r newRuntimeWithEstimator` в тестах

**Проверка:** `go build ./plugins/cost/...`

---

### Задача 7: `persistEstimateArtifacts` переехать в `report.go`, ошибки не глотать

**Файл:** `plugins/cost/usecases.go:97-110`

Текущие проблемы:
1. Вызов `buildCostReport` из `usecases.go` — это функция из `report.go`, сохранение отчёта
   — ответственность `report.go`
2. Ошибки сохранения артефактов логируются как `Warn` и проглатываются — тест не может
   проверить, что артефакты сохранились

**Решение:**
- Переименовать `persistEstimateArtifacts` → `saveArtifacts` в `report.go`
- Принимать и возвращать объединённую ошибку через `errors.Join`

```go
// report.go
func saveArtifacts(serviceDir string, result *model.EstimateResult) error {
    var errs []error
    if err := ci.SaveJSON(serviceDir, resultsFile, result); err != nil {
        errs = append(errs, fmt.Errorf("save results: %w", err))
    }
    report := buildCostReport(result)
    if err := ci.SaveReport(serviceDir, report); err != nil {
        errs = append(errs, fmt.Errorf("save report: %w", err))
    }
    return errors.Join(errs...)
}
```

```go
// usecases.go — вызов без проглатывания
if err := saveArtifacts(serviceDir, result); err != nil {
    log.FromContext(ctx).WithError(err).Warn("cost: failed to save artifacts")
}
```

**Затронутые места:**
- `usecases.go:97-110` — удалить `persistEstimateArtifacts`
- `report.go` — добавить `saveArtifacts`
- `usecases.go:35` — обновить вызов

**Проверка:** `go build ./plugins/cost/...` + `go test ./plugins/cost/...`

---

## P2 — Качество кода (5 задач)

### Задача 8: Исправить `FixedMonthlyCost` — убрать named return `monthly2`

**Файл:** `plugins/cost/internal/handler/calc.go:18-20`

**До:**
```go
func FixedMonthlyCost(monthly float64) (hourly, monthly2 float64) {
    return monthly / HoursPerMonth, monthly
}
```

**После:**
```go
func FixedMonthlyCost(monthly float64) (float64, float64) {
    return monthly / HoursPerMonth, monthly
}
```

**Затронутые места:** только `calc.go:18`

---

### Задача 9: Удалить мёртвый код в `ec2/instance.go` — AMI-блок в `BuildLookup`

**Файл:** `plugins/cost/internal/cloud/aws/ec2/instance.go:38-40`

```go
// Мёртвый код — всегда присваивает "Linux", ami не используется
if ami := handler.GetStringAttr(attrs, "ami"); ami != "" {
    operatingSystem = "Linux"
}
```

Удалить три строки. `operatingSystem` уже инициализирован `"Linux"` строкой выше.

**Затронутые места:** только `ec2/instance.go:38-40`

---

### Задача 10: Удалить мёртвые экспортируемые API из `pricing`

**Файл:** `plugins/cost/internal/pricing/cache.go`

Два экспортированных метода нигде не вызываются:
- `Cache.Validate(...)` — `cache.go:106-119` — scan без fetch; нет ни одного вызова в codebase
- `Cache.PrewarmCache(...)` — `cache.go:174-183` — прогрев; заменён на `WarmIndexes` в registry

Также: `ServiceCode = ServiceID` — type alias в `pricing/types.go:14` — нет ни одного
потребителя кроме комментария "backward compatibility".

**Затронутые места:**
- `pricing/cache.go:106-119` — удалить `Validate`
- `pricing/cache.go:174-183` — удалить `PrewarmCache`
- `pricing/types.go:14` — удалить `ServiceCode` type alias
- `pricing/cache_test.go` — если тестирует `Validate` — обновить или удалить тест

**Проверка:** `go build ./plugins/cost/...` + `go test ./plugins/cost/...`

---

### Задача 11: Мёртвая neg-ветка в `sprintfFloat` — убрать

**Файл:** `plugins/cost/internal/model/formatting.go:84-115`

`sprintfFloat` принимает `f float64` и обрабатывает отрицательные значения (`neg := f < 0`),
но вызывается исключительно из `formatPositive` (`:37`) которая по определению принимает
`f > 0`. Neg-ветка — мёртвый код.

**До:**
```go
func sprintfFloat(f float64, prec int) string {
    neg := f < 0
    if neg { f = -f }
    ...
    result := ""
    if neg { result = "-" }
    return result + intStr + decStr
}
```

**После:** убрать `neg` переменную и обе проверки. Функция принимает только positive values
(добавить комментарий или переименовать в `formatPositiveFixed`).

**Затронутые места:** только `formatting.go:84-115`

---

### Задача 12: `DefaultRegion` — убрать дублирование между `model` и `engine`

**Файл:** `plugins/cost/internal/engine/scanner.go:12`, `model/tree.go:9`

```go
// engine/scanner.go:12
DefaultRegion = model.DefaultRegion // re-export
```

Это re-export без добавления ценности. Потребители в `engine/` могут использовать
`model.DefaultRegion` напрямую. Убрать алиас из `scanner.go`.

**Затронутые места:**
- `engine/scanner.go:12` — удалить `DefaultRegion` re-export
- Поиск всех вызовов `engine.DefaultRegion` в тестах — заменить на `model.DefaultRegion`

**Проверка:** `go build ./plugins/cost/...` + `go test ./plugins/cost/...`

---

## Порядок выполнения

```
P0:  1 → 2 → 3      (зависимость: 1 должна быть до 3)
P1:  4 → 5 → 6 → 7  (независимые, порядок произвольный)
P2:  8 → 9 → 10 → 11 → 12  (независимые)
```

**Рекомендуемый глобальный порядок:** 1 → 2 → 3 → 5 → 6 → 4 → 7 → 8 → 9 → 10 → 11 → 12

---

## Acceptance Criteria

- [ ] `go build ./plugins/cost/...` — чистая сборка
- [ ] `go test ./plugins/cost/... -count=1 -race` — 19/19 пакетов PASS
- [ ] `go vet ./plugins/cost/...` — без замечаний
- [ ] `grep -r "awskit" plugins/cost/internal/engine/` — **ноль результатов**
- [ ] `grep -r "Providers\.AWS" plugins/cost/` — **ноль результатов**
- [ ] `Estimator` не имеет методов `CacheDir/CacheTTL/CacheEntries/CleanExpiredCache` — вместо них `Cache() CacheInspector`
- [ ] `resolveBlobStore` не существует — есть только `resolveBlobCache`
- [ ] `newRuntimeWithEstimator` не существует
- [ ] `FixedMonthlyCost` без named returns
- [ ] AMI dead-code block в `ec2/instance.go` удалён
- [ ] `pricing.Cache.Validate` и `PrewarmCache` удалены
- [ ] `ServiceCode` type alias удалён
- [ ] `engine.DefaultRegion` re-export удалён

---

## Ожидаемый результат

| Измерение | До v3 | После v3 |
|-----------|-------|----------|
| Слои | 6.5 | 9.5 |
| Именование | 8.0 | 9.0 |
| Контракты | 7.5 | 9.0 |
| SRP | 7.5 | 9.0 |
| Абстракции | 8.0 | 9.5 |
| Go-идиомы | 8.5 | 9.5 |
| Тестируемость | 8.5 | 9.0 |
| **Итог** | **8.0** | **9.5** |
