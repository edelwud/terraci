# Cost Plugin Refactor — v2

**Цель:** устранить оставшиеся проблемы после v1-рефакторинга: баги корректности,
лишние обязанности у типов, дублирование кода и несоответствие Go-идиомам.

**Точка отсчёта:** после выполнения плана v1 — 7.5/10.
**Целевая оценка:** 9/10.

---

## Диагноз: что осталось

### Баг / Риск корректности

| # | Файл | Строки | Проблема |
|---|------|--------|----------|
| B1 | `engine/estimator.go` | 83–97 | `enabledProviderIDs` добавляет `awskit.ProviderID` дважды при `cfg.Providers.AWS.Enabled && cfg.Enabled`. Map-дедупликация скрывает это, но логика неверна. |
| B2 | `usecases.go` | 80 | `strings.HasSuffix(path, modulePath)` ловит ложные совпадения: путь `other/svc/foo/bar` совпадёт с `foo/bar` без проверки границы. |
| B3 | `output.go` | 205–207 | `isRenderedZeroCost` сравнивает строку `"$0"` вместо `cost == 0`. Хрупкая связь с `FormatCost`. |

### Дизайн / Архитектура

| # | Файл | Строки | Проблема |
|---|------|--------|----------|
| D1 | `engine/estimator.go` | 57–58 | Двухфазная инициализация: `coord.catalog = catalog` после конструктора. Объект на мгновение частично инициализирован. |
| D2 | `engine/estimator.go` | 23–30, 132–165 | `Estimator` держит `resolver`, `scanner`, `executor` — всё это уже внутри `coord`. Плюс 7 методов cache-lifecycle, которые не нужны поверхностным потребителям. |
| D3 | `engine/batch.go` | 22–24 | Анонимный интерфейс `runtimes` вместо именованного. Неочевиден без чтения кода. |
| D4 | `runtime/resolver.go` | 252 | `resolveStandardCost` — 7 параметров. |
| D5 | `runtime.go` | 89–95 | `resolveBlobCache` возвращает `BlobStoreInfo`, который тут же отбрасывается (`_`) в `newRuntime:26`. |
| D6 | `handler/registry.go` | 4–5, 64–69 | `handler/registry.go` импортирует app-logger (`caarlos0/log`) ради одной standalone-функции `LogUnsupported`. |

### Дублирование / Качество кода

| # | Файл | Строки | Проблема |
|---|------|--------|----------|
| Q1 | `engine/scanner.go` | 13 | `scanConcurrency = 4` дублирует `maxConcurrency = 4` из `batch.go:44`. |
| Q2 | `engine/scanner.go` | 92–161 | `ScanMany` и `ScanManyBestEffort` дублируют semaphore+WaitGroup шаблон. |
| Q3 | `results/assembler.go` | 12–21 | Action constants re-exported третий раз (уже есть в `engine/scanner.go`). |
| Q4 | `model/config_types.go` | 61–64 | Мёртвый `if !HasEnabledProviders() { return nil }` блок — no-op ветка в `Validate`. |
| Q5 | `model/config_types.go` | 11, 45 | `CostConfig.Enabled bool` — неясный internal-флаг без yaml/json тегов, источник бага B1. |
| Q6 | `model/formatting.go` | 74–75 | `string(rune('0'+precision))` — хрупкое форматирование, ломается при `precision > 9`. Решение: `fmt.Sprintf`. |
| Q7 | `model/formatting.go` | 83–92 | `sprintf` — лишний уровень косвенности без ценности; `trimTrailingZeros` мог бы вызывать `sprintfFloat` напрямую. |
| Q8 | `model/formatting.go` | 127–137 | `itoa` переизобретает `strconv.AppendInt`; нет задокументированной причины избегать stdlib. |
| Q9 | `handler/handler.go` | 1 | Устаревший doc-comment: `// Package provider defines...`, а пакет называется `handler`. |
| Q10 | `pricing/cache.go` | 100–122 | `Validate` возвращает срез анонимных структур вместо именованного типа. |
| Q11 | `runtime/provider_runtime_registry.go` | 63–65 | Молчаливое no-op при передаче fetcher для multi-provider. Не задокументировано. |
| Q12 | `lifecycle.go` | 11 | Комментарий говорит «cache logging happens in newRuntime», но не объясняет двойную валидацию config. |
| Q13 | `report.go` | 48 | `visibleReportModules` вызывается дважды: в `buildCostReport` и внутри `renderCostReportBody`. |

---

## Задачи

### P0 — Баги корректности

---

#### [x] Задача 1: Исправить `enabledProviderIDs` — убрать дубль + удалить `cfg.Enabled`

**Файл:** `engine/estimator.go:83–97`, `model/config_types.go:11,45`

**Проблема:**
```go
// engine/estimator.go:90-95 — двойной append при обоих флагах
if cfg.Providers.AWS != nil && cfg.Providers.AWS.Enabled {
    ids = append(ids, awskit.ProviderID)  // ← append 1
}
if cfg.Enabled {                           // ← internal флаг без yaml/json тегов
    ids = append(ids, awskit.ProviderID)  // ← append 2 — дубль
}
```

`cfg.Enabled` — внутреннее поле с тегами `yaml:"-" json:"-"`, которое нигде не устанавливается через конфиг. Оно появилось как legacy-путь, но `LegacyEnabled` уже покрывает миграцию. `cfg.Enabled` — мёртвый код, источник скрытого бага.

**Решение:**
1. Удалить поле `Enabled bool` из `model/config_types.go:11`
2. Обновить `HasEnabledProviders` в `model/config_types.go:45` — убрать `|| c.Enabled`
3. Упростить `enabledProviderIDs` в `engine/estimator.go:85–97` — убрать ветку `cfg.Enabled`

**После:**
```go
// model/config_types.go
func (c *CostConfig) HasEnabledProviders() bool {
    if c == nil { return false }
    return c.Providers.AWS != nil && c.Providers.AWS.Enabled
}

// engine/estimator.go
func enabledProviderIDs(cfg *model.CostConfig) []string {
    if cfg == nil { return nil }
    if cfg.Providers.AWS != nil && cfg.Providers.AWS.Enabled {
        return []string{awskit.ProviderID}
    }
    return nil
}
```

**Файлы:** `model/config_types.go`, `engine/estimator.go`

---

#### [x] Задача 2: Исправить `filterModulePaths` — проверка пути с границей

**Файл:** `usecases.go:78–80`

**Проблема:**
```go
// usecases.go:80 — HasSuffix без проверки разделителя
strings.HasSuffix(path, modulePath)
// "other_service/foo/bar" совпадёт с "foo/bar" — ложное совпадение
```

**Решение:** Добавить проверку через `filepath.FromSlash` + `filepath.HasPrefix` или сравнение через `path.Dir`-граничный тест:

```go
func matchesModulePath(path, target string) bool {
    if path == target {
        return true
    }
    // Проверяем suffix с гарантией границы сегмента пути
    sep := string(filepath.Separator)
    return strings.HasSuffix(path, sep+target) || strings.HasSuffix(path, "/"+target)
}
```

**Файлы:** `usecases.go`

---

#### [x] Задача 3: Исправить `isRenderedZeroCost` — прямое сравнение float

**Файл:** `output.go:205–207`

**Проблема:**
```go
// output.go:205-207 — связь со строковым представлением FormatCost
func isRenderedZeroCost(cost string) bool {
    return cost == "$0"
}
```

Если `FormatCost` когда-либо изменит формат нуля, эта функция тихо сломается.

**Решение:** Передавать `float64` вместо строки, сравнивать напрямую:
```go
func isZeroCost(cost float64) bool {
    return cost == 0
}
```
Обновить все места вызова `isRenderedZeroCost(costStr)` → `isZeroCost(costFloat)`.

**Файлы:** `output.go`

---

### P1 — Дизайн и структура

---

#### [x] Задача 4: Устранить двухфазную инициализацию `estimateCoordinator`

**Файлы:** `engine/estimator.go:52–68`, `engine/batch.go:27–40`

**Проблема:**
```go
// batch.go:37 — catalog: nil документирует неполную инициализацию
catalog: nil, // set by newEstimator after catalog is created

// estimator.go:58 — пост-конструктор мутация
coord := newEstimateCoordinator(scanner, executor, providerMetadata, runtimeRegistry)
coord.catalog = catalog  // ← объект уже возвращён
```

**Решение:** Передать `catalog` напрямую в `newEstimateCoordinator`:
```go
// batch.go — добавить catalog в параметры
func newEstimateCoordinator(
    scanner *ModuleScanner,
    executor *ModuleExecutor,
    catalog costruntime.ProviderCatalogRuntime,
    providerMetadata func() map[string]model.ProviderMetadata,
    runtimes indexWarmer,
) *estimateCoordinator {
    return &estimateCoordinator{
        scanner:          scanner,
        executor:         executor,
        catalog:          catalog,
        providerMetadata: providerMetadata,
        runtimes:         runtimes,
    }
}

// estimator.go — newEstimator убирает post-init
coord := newEstimateCoordinator(scanner, executor, catalog, catalog.ProviderMetadata, runtimeRegistry)
// coord.catalog = catalog  ← удалить
```

**Файлы:** `engine/batch.go`, `engine/estimator.go`

---

#### [x] Задача 5: Выделить именованный интерфейс `indexWarmer` в `batch.go`

**Файл:** `engine/batch.go:22–24`

**Проблема:**
```go
// batch.go:22-24 — анонимный интерфейс
runtimes interface {
    WarmIndexes(ctx context.Context, services map[pricing.ServiceID][]string) error
}
```

Читателю нужно заглянуть внутрь структуры, чтобы понять ограничение.

**Решение:** Именованный интерфейс на уровне пакета:
```go
// batch.go или scanner.go — добавить до estimateCoordinator
// indexWarmer downloads pricing indexes for a set of service/region pairs.
type indexWarmer interface {
    WarmIndexes(ctx context.Context, services map[pricing.ServiceID][]string) error
}
```
Заменить анонимный тип в `estimateCoordinator` на `indexWarmer`.

**Файлы:** `engine/batch.go`

---

#### [x] Задача 6: Упростить `resolveBlobCache` — убрать неиспользуемый `BlobStoreInfo`

**Файл:** `runtime.go:89–95` и вызывающий код `runtime.go:26`

**Проблема:**
```go
// runtime.go:26 — BlobStoreInfo отбрасывается через _
cache, _, err := resolveBlobCache(ctx, appCtx, cfg)

// runtime.go:89 — функция возвращает info, которая не нужна
func resolveBlobCache(...) (*blobcache.Cache, plugin.BlobStoreInfo, error)
```

`BlobStoreInfo` логируется внутри `resolveBlobStore` — наружу передавать незачем.

**Решение:**
1. `resolveBlobCache` возвращает `(*blobcache.Cache, error)` — убрать `BlobStoreInfo`
2. Обновить вызов в `newRuntime:26`

```go
func resolveBlobCache(ctx context.Context, appCtx *plugin.AppContext, cfg *model.CostConfig) (*blobcache.Cache, error) {
    blobStore, _, err := resolveBlobStore(ctx, appCtx, cfg)
    if err != nil {
        return nil, err
    }
    return blobcache.New(blobStore, cfg.BlobCacheNamespace(), engine.CacheTTLFromConfig(cfg)), nil
}
```

**Файлы:** `runtime.go`

---

#### [~] Задача 7: Переместить `LogUnsupported` из `handler/registry.go` в `engine`

**Файл:** `handler/registry.go:4–5, 64–69`

**Проблема:** `handler` — пакет чистых интерфейсов и реестра. Импорт application-logger (`caarlos0/log`) ради одной standalone-функции нарушает это. Logging — это policy уровня engine, не уровня handler-контрактов.

**Решение:**
1. Удалить `LogUnsupported` из `handler/registry.go`
2. Удалить import `caarlos0/log` из `handler/registry.go`
3. Перенести логику в `engine/resolver.go` или `runtime/resolver.go` как private helper в том месте, где она вызывается (`coreResolve`)

```go
// runtime/resolver.go — там где уже логируется resolver
func logUnsupportedResource(resourceType, address string) {
    log.WithField("type", resourceType).
        WithField("address", address).
        Debug("resource type not supported for cost estimation")
}
```
Заменить вызов `handler.LogUnsupported(...)` в `resolver.go:145` на `logUnsupportedResource(...)`.

**Файлы:** `handler/registry.go`, `runtime/resolver.go`

---

#### [x] Задача 8: Ввести `resolveStandardCostCtx` — убрать 7-параметрную функцию

**Файл:** `runtime/resolver.go:252`

**Проблема:**
```go
func (r *CostResolver) resolveStandardCost(
    ctx context.Context,
    providerID string,
    h handler.ResourceHandler,
    attrs map[string]any,
    region string,
    result model.ResourceCost,
    state *ResolutionState,
) model.ResourceCost
```

7 параметров — сигнал скрытого контекстного объекта.

**Решение:** Выделить `standardResolutionCtx` для группировки параметров разрешения:
```go
// runtime/resolver.go — новый внутренний тип
type standardResolutionCtx struct {
    providerID string
    handler    handler.ResourceHandler
    attrs      map[string]any
    region     string
    result     model.ResourceCost
    state      *ResolutionState
}

func (r *CostResolver) resolveStandardCost(ctx context.Context, rc standardResolutionCtx) model.ResourceCost
```
Обновить все места вызова.

**Файлы:** `runtime/resolver.go`

---

### P2 — Дублирование и качество

---

#### [x] Задача 9: Объединить константы параллелизма сканера и координатора

**Файлы:** `engine/scanner.go:13`, `engine/batch.go:44`

**Проблема:**
```go
// scanner.go:13
const scanConcurrency = 4

// batch.go:44
const maxConcurrency = 4
```
Два имени, одно значение, один смысл.

**Решение:** Объявить одну константу в `engine/batch.go` (или новом `engine/engine.go`):
```go
// maxModuleConcurrency — max concurrent module operations (scan + estimate).
const maxModuleConcurrency = 4
```
Использовать её в обоих местах. Удалить `scanConcurrency`.

**Файлы:** `engine/scanner.go`, `engine/batch.go`

---

#### [x] Задача 10: Устранить дублирование semaphore-шаблона в `ScanMany` / `ScanManyBestEffort`

**Файл:** `engine/scanner.go:92–161`

**Проблема:** `ScanMany` (строки 92–121) и `ScanManyBestEffort` (133–161) содержат идентичный concurrency boilerplate: `sync.WaitGroup` + semaphore channel + goroutine loop.

**Решение:** Выделить generic helper `scanConcurrently`:
```go
// engine/scanner.go — private helper
func scanConcurrently(modulePaths []string, regions map[string]string, fn func(i int, path, region string)) {
    var wg sync.WaitGroup
    sem := make(chan struct{}, maxModuleConcurrency)
    for i, modulePath := range modulePaths {
        region := regions[modulePath]
        if region == "" { region = DefaultRegion }
        wg.Go(func() {
            sem <- struct{}{}
            defer func() { <-sem }()
            fn(i, modulePath, region)
        })
    }
    wg.Wait()
}
```
`ScanMany` и `ScanManyBestEffort` становятся 10-строчными обёртками над ним.

**Файлы:** `engine/scanner.go`

---

#### [x] Задача 11: Удалить re-export action constants из `results/assembler.go`

**Файл:** `results/assembler.go:12–21`

**Проблема:** Action constants определены в `model/action.go`, re-exported в `engine/scanner.go`, и снова re-exported в `results/assembler.go`. Тройной re-export одного и того же.

**Решение:** Удалить строки 12–21 из `results/assembler.go`. Внутри `assembler.go` используется только `AggregateCost` — заменить `ActionCreate` и т.п. на `model.ActionCreate`:

```go
// было: assembler.go:76
case ActionCreate:
// стало:
case model.ActionCreate:
```

**Файлы:** `results/assembler.go`

---

#### [x] Задача 12: Убрать мёртвую ветку `Validate` в `config_types.go`

**Файл:** `model/config_types.go:61–64`

**Проблема:**
```go
if !c.HasEnabledProviders() {
    return nil  // ← ни одна проверка не добавлена, dead code
}
return nil      // ← тот же результат
```

Два `return nil` подряд — placeholder без содержимого.

**Решение:** Убрать блок `if !HasEnabledProviders()`:
```go
func (c *CostConfig) Validate() error {
    if c.LegacyEnabled != nil {
        return errors.New("plugins.cost.enabled is no longer supported; use plugins.cost.providers.aws.enabled")
    }
    if c.CacheDir != "" {
        return errors.New("plugins.cost.cache_dir is no longer supported; use plugins.diskblob.root_dir")
    }
    if c.BlobCache != nil && c.BlobCache.TTL != "" {
        if _, err := time.ParseDuration(c.BlobCache.TTL); err != nil {
            return fmt.Errorf("invalid blob_cache.ttl %q: %w", c.BlobCache.TTL, err)
        }
    }
    return nil
}
```

**Файлы:** `model/config_types.go`

---

#### [x] Задача 13: Исправить `formatting.go` — заменить хрупкое форматирование

**Файл:** `model/formatting.go:74–92, 127–137`

**Три точечных изменения:**

**13a — `trimTrailingZeros`: убрать `string(rune(...))` конструкцию и `sprintf` прослойку:**
```go
// было: через sprintf(format, cost) → sprintfFloat(cost, prec)
// стало: вызывать sprintfFloat напрямую
func trimTrailingZeros(cost float64, precision int) string {
    s := sprintfFloat(cost, precision)
    if hasDecimal(s) {
        s = trimZeros(s)
    }
    return s
}
// Удалить func sprintf полностью — она не нужна
```

**13b — `itoa`: заменить ручную реализацию на `strconv.FormatInt`:**
```go
import "strconv"

func itoa(n int64) string {
    return strconv.FormatInt(n, 10)
}
```

**13c — Добавить doc-comment к константам:**
```go
const (
    thousandThreshold = 1000 // costs >= $1000 use comma-separated formatting
    roundingOffset    = 0.5  // standard rounding for integer conversion
    digitsPerGroup    = 3    // digits between comma separators (1,000,000)
)
```

**Файлы:** `model/formatting.go`

---

#### [x] Задача 14: Исправить устаревший doc-comment в `handler/handler.go`

**Файл:** `handler/handler.go:1`

**Проблема:**
```go
// Package provider defines provider-agnostic interfaces for cloud cost estimation.
```
Пакет называется `handler`, а не `provider`.

**Решение:**
```go
// Package handler defines provider-agnostic interfaces for cloud cost estimation.
// AWS, GCP, and Azure handlers all implement these interfaces.
```

**Файлы:** `handler/handler.go`

---

#### [x] Задача 15: Именованный тип для `Validate` в `pricing/cache.go`

**Файл:** `pricing/cache.go:100–122`

**Проблема:** Метод `Validate` возвращает `[]struct{ Service ServiceID; Region string }` — анонимную структуру. Невозможно объявить переменную этого типа в вызывающем коде без повторения.

**Решение:** Ввести именованный тип:
```go
// MissingPricingEntry describes a service/region combination absent from the cache.
type MissingPricingEntry struct {
    Service ServiceID
    Region  string
}

func (c *Cache) Validate(ctx context.Context, services map[ServiceID][]string) []MissingPricingEntry
```

**Файлы:** `pricing/cache.go`

---

#### [x] Задача 16: Задокументировать ограничение fetcher в `provider_runtime_registry.go`

**Файл:** `runtime/provider_runtime_registry.go:63–65`

**Проблема:**
```go
// NewProviderRuntimeRegistryFromProvidersWithBlobCache:63-65
// Молчаливое no-op если providers > 1 и fetcher != nil
```

Разработчик, передающий fetcher для multi-provider сценария, не получает никакого сигнала об ошибке.

**Решение:** Добавить doc-comment к параметру `fetcher` и к условию:
```go
// fetcher optionally overrides the pricing fetcher for the single registered provider.
// Only applied when exactly one provider is configured; ignored for multi-provider setups.
// Pass nil to use each provider's default fetcher.
```

**Файлы:** `runtime/provider_runtime_registry.go`

---

#### [x] Задача 17: Убрать двойной вызов `visibleReportModules` в `report.go`

**Файл:** `report.go:43–68`

**Проблема:** `buildCostReport` вычисляет `visible := visibleReportModules(result)`, но затем `renderCostReportBody` на строке 48 снова вызывает `visibleReportModules(result)` — двойное вычисление одного результата.

**Решение:** `renderCostReportBody` принимает уже отфильтрованный `[]model.ModuleCost`:
```go
func renderCostReportBody(result *model.EstimateResult, modules []model.ModuleCost) string

// buildCostReport:
visible := visibleReportModules(result)
body := renderCostReportBody(result, visible)
```

**Файлы:** `report.go`

---

## Порядок выполнения и зависимости

```
Задача 1  (enabledProviderIDs + Enabled поле)
    └── независима, делать первой (убирает баг)

Задача 2  (filterModulePaths граница пути)
    └── независима

Задача 3  (isRenderedZeroCost → isZeroCost)
    └── независима

Задача 4  (двухфазная init → один конструктор)
    └── зависит от Задачи 5 (нужен именованный indexWarmer)

Задача 5  (именованный indexWarmer)
    └── независима, делать перед Задачей 4

Задача 6  (resolveBlobCache убирает BlobStoreInfo)
    └── независима

Задача 7  (LogUnsupported → engine)
    └── независима

Задача 8  (resolveStandardCost 7 params → ctx struct)
    └── независима

Задача 9  (объединить scanConcurrency)
    └── независима, но делать перед Задачей 10

Задача 10 (scanConcurrently helper)
    └── зависит от Задачи 9 (нужна итоговая константа)

Задача 11 (удалить re-export из assembler.go)
    └── независима

Задача 12 (мёртвая ветка Validate)
    └── делать после Задачи 1 (меняется HasEnabledProviders)

Задача 13 (formatting.go)
    └── независима

Задача 14 (doc-comment handler.go)
    └── тривиальна, независима

Задача 15 (MissingPricingEntry тип)
    └── независима

Задача 16 (doc fetcher constraint)
    └── тривиальна, независима

Задача 17 (visibleReportModules дубль)
    └── независима
```

**Рекомендуемый порядок:** 1 → 12 → 2 → 3 → 5 → 4 → 6 → 7 → 8 → 9 → 10 → 11 → 13 → 14 → 15 → 16 → 17

---

## Критерии готовности

- [x] `go build ./plugins/cost/...` без ошибок
- [x] `go test ./plugins/cost/...` без регрессий
- [x] `CostConfig` не содержит поле `Enabled bool` с `yaml:"-"`
- [x] `enabledProviderIDs` содержит ровно один `append` для AWS
- [x] `filterModulePaths` использует проверку с разделителем
- [x] `isRenderedZeroCost` удалена; сравнение через `float64`
- [x] `newEstimateCoordinator` принимает `catalog` в параметрах (нет post-init мутаций)
- [x] `indexWarmer` — именованный интерфейс в `engine/batch.go`
- [x] `resolveBlobCache` возвращает `(*blobcache.Cache, error)` без `BlobStoreInfo`
- [x] `handler/registry.go` не импортирует `caarlos0/log`
- [x] `resolveStandardCost` использует struct-контекст вместо 7 параметров
- [x] Одна константа параллелизма в `engine` — `maxModuleConcurrency`
- [x] `scanConcurrently` helper устраняет дублирование в `ScanMany`/`ScanManyBestEffort`
- [x] Action constants не re-exported в `results/assembler.go`
- [x] `Validate` в `config_types.go` без мёртвой ветки
- [x] `formatting.go`: `sprintf` функция удалена; `itoa` использует `strconv`
- [x] Doc-comment в `handler/handler.go:1` — правильное имя пакета
- [x] `MissingPricingEntry` — именованный тип вместо анонимной структуры
- [x] `renderCostReportBody` не вызывает `visibleReportModules` внутри
