# Cost Plugin Refactor Plan

**Цель:** уменьшить количество сущностей, убрать нарушения слоёв, сделать границы между
доменами явными и контракты функций понятными — следуя идиоматичному Go.

---

## Контекст и диагноз

### Текущая структура пакетов

```
plugins/cost/                     ← Layer 1: plugin surface
plugins/cost/internal/
  engine/                         ← Layer 2: orchestration
  runtime/                        ← Layer 2: pricing resolution kernel
  model/                          ← shared domain types (нейтральный)
  results/                        ← assembly helpers
  handler/                        ← interfaces для cloud handlers
  pricing/                        ← cache + fetcher abstraction
  cloud/                          ← provider registry + AWS impl
    aws/
    awskit/
  handlertest/, runtimetest/, enginetest/   ← test helpers
```

### Ключевые проблемы

| # | Нарушение | Файл | Серьёзность |
|---|-----------|------|-------------|
| 1 | `model` импортирует `awskit` — AWS в нейтральном слое | `model/config_types.go:8,50-53` | Высокая |
| 2 | `runtime` импортирует `awskit` — хардкод `ProviderID` | `runtime/provider_runtime_registry.go:13,119` | Средняя |
| 3 | `lifecycle.go` строит `pricing.CacheInspector` напрямую, обходя `Estimator` | `lifecycle.go:9,24-33` | Средняя |
| 4 | Blob cache конструируется дважды: в `Preflight` и в `Runtime` | `lifecycle.go:24`, `runtime.go:25` | Низкая |
| 5 | 9 публичных конструкторов `Estimator` в двух файлах | `engine/estimator.go:27-63`, `engine/factory.go:52-83` | Средняя |
| 6 | `engine` re-exports `results.EstimateAction` через type alias | `engine/scanner.go:17-25` | Низкая |
| 7 | Два несвязанных `LookupBuilder` с одним именем | `handler/handler.go:49`, `awskit/lookup.go:6` | Низкая |
| 8 | Prefetch разбит на два объекта с interface-мостом без причины | `engine/prefetch.go`, `runtime/prefetch.go` | Низкая |
| 9 | `cloud.Reset()` — тестовый код в production файле | `cloud/registry.go:83-89` | Низкая |

---

## Задачи

### P0 — Нарушения слоёв (domain boundary violations)

---

#### [x] Задача 1: Убрать `awskit` из `model/config_types.go`

**Проблема:** `model/config_types.go:8` импортирует `awskit` ради одной строки
`awskit.ProviderID`. Добавление GCP/Azure заставит менять нейтральный domain-слой.

**Решение:**
1. Удалить метод `EnabledProviderIDs()` из `model.CostConfig`
2. Удалить import `awskit` из `model/config_types.go`
3. Перенести логику отбора провайдеров в `engine/factory.go` как package-level функцию
4. Обновить все места вызова `cfg.EnabledProviderIDs()` — в `engine/factory.go`
   (`configuredProviders` уже там и вызывает её)
5. Метод `HasEnabledProviders()` в `model` — переписать без вызова `EnabledProviderIDs()`,
   используя только поля конфига напрямую

**После:**
```go
// model/config_types.go — только данные
func (c *CostConfig) HasEnabledProviders() bool {
    if c == nil { return false }
    return c.Providers.AWS != nil && c.Providers.AWS.Enabled
}

// engine/factory.go — логика с AWS-знанием
func enabledProviderIDs(cfg *model.CostConfig) []string {
    var ids []string
    if cfg.Providers.AWS != nil && cfg.Providers.AWS.Enabled {
        ids = append(ids, awskit.ProviderID)
    }
    return ids
}

func configuredProviders(cfg *model.CostConfig) ([]cloud.Provider, error) {
    enabled := map[string]bool{}
    for _, id := range enabledProviderIDs(cfg) {
        enabled[id] = true
    }
    // ... остальная логика без изменений
}
```

**Файлы:** `model/config_types.go`, `engine/factory.go`, `plugin.go` (если вызывает)

---

#### [x] Задача 2: Убрать `awskit` из `runtime/provider_runtime_registry.go`

**Проблема:** `SetPricingFetcher` хардкодит fallback на `awskit.ProviderID` при
нескольких провайдерах (`provider_runtime_registry.go:119`). Это сломает любой
второй провайдер молча.

**Решение:**
1. Удалить import `awskit` из `runtime/provider_runtime_registry.go`
2. Переименовать метод: `SetPricingFetcher(f) → SetFetcherForProvider(providerID string, f pricing.PriceFetcher)`
3. Удалить магический fallback: если `providerID` не найден — no-op, без паники
4. Обновить `engine/estimator.go:88` — прокси-метод `Estimator.SetPricingFetcher`
   превратить в `SetFetcherForProvider(providerID string, f pricing.PriceFetcher)`
5. Обновить все вызывающие стороны (тесты `runtimetest`, `enginetest`)

**После:**
```go
// runtime/provider_runtime_registry.go
func (r *ProviderRuntimeRegistry) SetFetcherForProvider(providerID string, f pricing.PriceFetcher) {
    if rt, ok := r.runtimes[providerID]; ok {
        rt.Cache.SetFetcher(f)
    }
}
```

**Файлы:** `runtime/provider_runtime_registry.go`, `engine/estimator.go`,
`runtimetest/suite.go`, `enginetest/helpers.go`

---

#### [x] Задача 3: Упростить `lifecycle.go` — убрать прямой доступ к `pricing`

**Проблема:** `lifecycle.go:9` импортирует `pricing` напрямую, строит `CacheInspector`
и дублирует конструкцию blob cache, которая уже происходит в `runtime.go`.

**Решение:**
1. Убрать import `pricing` из `lifecycle.go`
2. Убрать вызов `resolveBlobCache` из `Preflight`
3. `Preflight` делает только: `validateRuntimeConfig(cfg)` + лёгкую диагностику
   без blob I/O
4. Cache state logging перенести в `runtime.go:newRuntime()` — после создания
   `Estimator` вызвать `logCacheState(ctx, estimator)` как private helper
5. `logCacheState` использует уже существующие методы `Estimator`:
   `CacheDir()`, `CacheEntries(ctx)`, `CleanExpiredCache(ctx)`

**После:**
```go
// lifecycle.go — только валидация конфига
func (p *Plugin) Preflight(ctx context.Context, appCtx *plugin.AppContext) error {
    if !p.IsEnabled() {
        return nil
    }
    return validateRuntimeConfig(p.Config())
}

// runtime.go — cache logging при построении runtime
func newRuntime(ctx context.Context, appCtx *plugin.AppContext, cfg *model.CostConfig) (*costRuntime, error) {
    // ...
    estimator, err := engine.NewEstimatorFromConfigWithBlobCache(cfg, cache)
    // ...
    logCacheState(ctx, estimator)
    return &costRuntime{estimator: estimator}, nil
}

func logCacheState(ctx context.Context, e *engine.Estimator) {
    // использует e.CacheDir(), e.CacheEntries(ctx), e.CleanExpiredCache(ctx)
}
```

**Файлы:** `lifecycle.go`, `runtime.go`

---

### P1 — Сокращение конструкторов и re-export

---

#### [x] Задача 4: Сократить конструкторы `Estimator` с 9 до 2 публичных

**Проблема:** 5 `New*` в `engine/estimator.go` + 4 `NewEstimatorFrom*` в
`engine/factory.go` — компенсация за отсутствие одного чёткого пути создания.

**Текущие 9 конструкторов:**
- `NewEstimator(store, namespace, ttl, fetcher)`
- `NewEstimatorWithBlobCache(cache, fetcher)`
- `NewEstimatorWithRuntimeRegistry(runtimeRegistry)`
- `NewEstimatorWithCatalogAndRuntimeRegistry(catalog, runtimeRegistry)`
- `NewEstimatorWithResolver(catalog, runtimeRegistry, resolver)`
- `NewEstimatorFromConfig(cfg, store)` — только в тестах
- `NewEstimatorFromConfigWithBlobCache(cfg, cache)` — используется в `runtime.go`
- `NewEstimatorFromConfigWithProvider(cfg, cp, store)` — только в тестах
- `NewEstimatorFromConfigWithProviderAndBlobCache(cp, cache)` — только в тестах

**Решение:**
Оставить два публичных канонических конструктора:
1. `NewEstimatorFromConfig(cfg *model.CostConfig, cache *blobcache.Cache) (*Estimator, error)` — production path
2. `NewEstimatorWithDeps(catalog *ProviderCatalog, registry *ProviderRuntimeRegistry) *Estimator` — explicit DI для тестов

Все остальные → `unexported` helpers или удалить:
- `newEstimator(catalog, registry, resolver)` — остаётся как private
- `NewEstimatorWithBlobCache` — удалить (заменить использование на `NewEstimatorFromConfig`)
- `NewEstimatorWithRuntimeRegistry` — удалить (только тесты — перейти на `NewEstimatorWithDeps`)
- `NewEstimatorWithCatalogAndRuntimeRegistry` → переименовать в `NewEstimatorWithDeps`
- `NewEstimatorWithResolver` — удалить (тестовая перегрузка, использовать `Resolver().Use()`)
- `NewEstimatorFromConfigWithProvider*` — удалить (только тесты, заменить на `NewEstimatorWithDeps`)
- `NewEstimator(store, ns, ttl, fetcher)` — удалить, конструкция cache — ответственность вызывающей стороны

`engine/factory.go` полностью удалить как файл — вся логика переходит в `engine/estimator.go`.

**После:**
```go
// engine/estimator.go

// NewEstimatorFromConfig — production path, строит Estimator из конфига и готового cache.
func NewEstimatorFromConfig(cfg *model.CostConfig, cache *blobcache.Cache) (*Estimator, error) { ... }

// NewEstimatorWithDeps — для тестов и advanced use-cases с явными зависимостями.
func NewEstimatorWithDeps(catalog *ProviderCatalog, registry *ProviderRuntimeRegistry) *Estimator { ... }
```

**Файлы:** `engine/estimator.go`, `engine/factory.go` (удалить), `runtime.go`,
`enginetest/helpers.go`, `runtimetest/suite.go`

---

#### [x] Задача 5: Перенести `EstimateAction` в `model`, убрать re-export из `engine`

**Проблема:** Тип определён в `results/assembler.go:13`, re-exported в
`engine/scanner.go:17-25` через type alias. Потребители не понимают, откуда тип.

**Решение:**
1. Перенести `EstimateAction` + все 5 констант из `results/assembler.go` в новый файл
   `model/action.go`
2. В `results/assembler.go` — импортировать из `model`
3. В `engine/scanner.go` — удалить все alias-объявления, импортировать из `model`
4. Во всём остальном коде, использующем `engine.ActionCreate` и т.п. — обновить на
   `model.ActionCreate`

**После:**
```go
// model/action.go — единственное место определения
type EstimateAction string
const (
    ActionCreate  EstimateAction = "create"
    ActionDelete  EstimateAction = "delete"
    ActionUpdate  EstimateAction = "update"
    ActionReplace EstimateAction = "replace"
    ActionNoOp    EstimateAction = "no-op"
)
```

**Файлы:** новый `model/action.go`, `results/assembler.go`, `engine/scanner.go`,
`engine/executor.go`, `engine/batch.go`

---

### P2 — Именование и упрощение prefetch

---

#### [x] Задача 6: Переименовать `awskit.LookupBuilder` → `awskit.PriceLookupSpec`

**Проблема:** `awskit.LookupBuilder` — конкретная struct, `handler.LookupBuilder` — интерфейс.
Одно имя, разные вещи, разные сигнатуры (`Build` vs `BuildLookup`).

**Решение:**
1. Переименовать `awskit.LookupBuilder` → `awskit.PriceLookupSpec` в `awskit/lookup.go`
2. Переименовать метод `Build` → `Lookup` (сигнатура остаётся)
3. Обновить все места использования в `awskit/standard_lookup.go` и всех handler-файлах
   в `cloud/aws/ec2/`, `rds/`, `elb/`, `eks/`, `elasticache/`, `serverless/`, `storage/`

**После:**
```go
// awskit/lookup.go
type PriceLookupSpec struct {
    Service       pricing.ServiceID
    ProductFamily string
}

func (s *PriceLookupSpec) Lookup(region string, attrs map[string]string) *pricing.PriceLookup { ... }
```

**Файлы:** `awskit/lookup.go`, `awskit/standard_lookup.go`, все handler-файлы в `cloud/aws/`

---

#### [x] Задача 7: Объединить `PrefetchPlanner` + `PricingPrefetcher` — убрать `ServicePlan` interface

**Проблема:** `PrefetchPlanner` (engine) строит план, `PricingPrefetcher` (runtime)
исполняет его. Между ними — `ServicePlan` interface-мост. Это две структуры и один
интерфейс ради одной логической операции.

**Решение:**
1. Удалить `engine/prefetch.go` как отдельный файл
2. Перенести `buildPrefetchRequirements` как private функцию в `engine/batch.go`:
   принимает `[]ModulePlan + ProviderCatalogRuntime`, возвращает
   `map[pricing.ServiceID][]string`
3. Добавить метод `WarmIndexes(ctx, map[pricing.ServiceID][]string) error` на
   `ProviderRuntimeRegistry` в `runtime/provider_runtime_registry.go`
4. Удалить `runtime/prefetch.go` как отдельный файл
5. Удалить `ServicePlan` interface из `runtime/prefetch.go`
6. Обновить `engine/batch.go` — вместо `b.prefetcher.PrefetchPricing(ctx, plan)` →
   `b.runtimes.WarmIndexes(ctx, buildPrefetchRequirements(b.catalog, modulePlans))`
7. Обновить `newEstimator` в `estimator.go` — убрать `PrefetchPlanner` и `PricingPrefetcher`
   из полей `Estimator`

**После:**
```go
// runtime/provider_runtime_registry.go — добавить метод
func (r *ProviderRuntimeRegistry) WarmIndexes(ctx context.Context, services map[pricing.ServiceID][]string) error {
    for serviceID, regions := range services {
        rt, ok := r.runtimes[serviceID.Provider]
        if !ok { continue }
        for _, region := range regions {
            if _, err := rt.Cache.GetIndex(ctx, serviceID, region); err != nil {
                return err
            }
        }
    }
    return nil
}

// engine/batch.go — private helper вместо PrefetchPlanner struct
func buildPrefetchRequirements(catalog ProviderCatalogRuntime, plans []*ModulePlan) map[pricing.ServiceID][]string { ... }
```

**Файлы:** `engine/prefetch.go` (удалить), `runtime/prefetch.go` (удалить),
`engine/batch.go`, `runtime/provider_runtime_registry.go`, `engine/estimator.go`

---

### P3 — Мелкие качественные улучшения

---

#### [x] Задача 8: Перенести `cloud.Reset()` в тестовый файл

**Проблема:** `cloud/registry.go:83-89` содержит `Reset()` с комментарием
`// Only for testing` в production коде.

**Решение:**
1. Удалить `Reset()` из `cloud/registry.go`
2. Создать `cloud/testing.go` с build tag `//go:build !production` (или просто
   `_test`-suffix-based helper) — переместить `Reset()` туда
3. Переименовать в `ResetForTesting()` для ясности намерения

**Файлы:** `cloud/registry.go`, новый `cloud/testing.go`

---

#### [x] Задача 9: Удалить wrapper-функцию `newProviderCatalog` из `engine/factory.go`

**Проблема:** `engine/factory.go:48-50` — однострочная обёртка без добавленной ценности:

```go
func newProviderCatalog(providers []cloud.Provider, registry *handler.Registry) *costruntime.ProviderCatalog {
    return costruntime.NewProviderCatalogFromProviders(providers, registry)
}
```

**Решение:** Удалить функцию. Вызывать `costruntime.NewProviderCatalogFromProviders` напрямую.
Это делается автоматически при выполнении Задачи 4 (удаление `factory.go`).

---

## Порядок выполнения и зависимости

```
Задача 1 (model ← awskit)
    └── не блокирует другие, но должна идти первой
        (убирает import awskit из model — снимает риск цикла)

Задача 2 (runtime ← awskit)
    └── независима от Задачи 1

Задача 3 (lifecycle)
    └── зависит от Задачи 4 (нужны финальные сигнатуры Estimator)

Задача 4 (конструкторы Estimator)
    └── зависит от Задач 1 и 2 (нужны чистые слои)
    └── блокирует Задачу 3

Задача 5 (EstimateAction в model)
    └── независима, можно делать в любой момент

Задача 6 (переименование LookupBuilder)
    └── независима, механическое переименование

Задача 7 (prefetch слияние)
    └── зависит от Задачи 4 (структура Estimator меняется)

Задачи 8-9 — независимы, тривиальны
```

**Рекомендуемый порядок:**
1 → 2 → 5 → 6 → 4 → 3 → 7 → 8 → 9

---

## Критерии готовности

- [x] `go build ./plugins/cost/...` без ошибок
- [x] `go test ./plugins/cost/...` без регрессий
- [x] `model/config_types.go` не импортирует ни один пакет из `cloud/`
- [x] `runtime/provider_runtime_registry.go` не импортирует `awskit`
- [x] `lifecycle.go` не импортирует `pricing` или `blobcache`
- [x] Публичных конструкторов `Estimator` ровно 2
- [x] `EstimateAction` определён только в одном месте (`model/action.go`)
- [x] Нет двух сущностей с именем `LookupBuilder` в разных пакетах
- [x] `engine/prefetch.go` и `runtime/prefetch.go` удалены
- [x] `cloud.Reset()` недоступен из production кода
