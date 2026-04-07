# Cost Plugin Refactor — Plan v4
**Date:** 2026-04-07
**Target score:** 9.5+ / 10
**Status:** PENDING

---

## Контекст

После трёх итераций рефакторинга (v1–v3, 38 задач суммарно) плагин достиг **8.2/10**.
Аудит выявил 133 проблемы, из которых 42 подлежат исправлению в этом плане.
Остальные 91 либо уже решены, либо слишком мелки (пунктуация в doc-comments) для отдельных задач.

---

## P0 — Корректность (проблемы, влияющие на поведение)

### Задача 1: `FormatCost` — мёртвое условие `&& cost != 0` на `:16`

**Файл:** `plugins/cost/internal/model/formatting.go:16`

**До:**
```go
if cost < 0.01 && cost > -0.01 && cost != 0 {
```

**После:**
```go
if cost < 0.01 && cost > -0.01 {
```

`cost == 0` уже обработан строкой выше — последнее условие мёртво.

---

### Задача 2: `BuildSegmentTree` — фильтр пропускает модули с `DiffCost != 0`

**Файл:** `plugins/cost/internal/model/tree.go:26`

**До:**
```go
if mc.AfterCost == 0 && mc.BeforeCost == 0 && mc.Error == "" {
    continue
}
```

**После:**
```go
if mc.AfterCost == 0 && mc.BeforeCost == 0 && mc.DiffCost == 0 && mc.Error == "" {
    continue
}
```

Без этой проверки модуль, у которого `BeforeCost - AfterCost = DiffCost != 0` при обоих равных нулю (что теоретически невозможно, но защита дешёвая), исчезает из дерева.
Важнее: будущий refund/credit сценарий, где `AfterCost=0, BeforeCost=0, DiffCost!=0` — защита корректная.

---

### Задача 3: `WarmIndexes` — fail-fast без документации семантики

**Файл:** `plugins/cost/internal/runtime/provider_runtime_registry.go:121-136`

Текущий код возвращает первую ошибку и прекращает прогрев. Это оставляет кеш в частично прогретом состоянии без сообщения об этом.

**После:**
```go
// WarmIndexes downloads any missing pricing data for the given service/region requirements.
// Warming is best-effort: all service/region pairs are attempted even if some fail.
// Returns a joined error if one or more downloads fail.
func (r *ProviderRuntimeRegistry) WarmIndexes(ctx context.Context, services map[pricing.ServiceID][]string) error {
    if len(services) == 0 {
        return nil
    }
    var errs []error
    for serviceID, regions := range services {
        rt, ok := r.runtimes[serviceID.Provider]
        if !ok {
            continue
        }
        for _, region := range regions {
            if _, err := rt.Cache.GetIndex(ctx, serviceID, region); err != nil {
                errs = append(errs, fmt.Errorf("%s/%s: %w", serviceID, region, err))
            }
        }
    }
    return errors.Join(errs...)
}
```

**Затронутые файлы:** `provider_runtime_registry.go`

---

### Задача 4: `NewProviderRuntimeRegistryFromProvidersWithBlobCache` — fetcher override молча ломается при >1 провайдере

**Файл:** `plugins/cost/internal/runtime/provider_runtime_registry.go:64`

**До:**
```go
if len(providers) == 1 && fetcher != nil {
    runtimeFetcher = fetcher
}
```

**После:** добавить panic или явный комментарий, раскрывающий ограничение:
```go
// fetcher override is only supported for single-provider setups (typically tests).
// For multi-provider setups, pass nil and use SetFetcherForProvider per provider.
if fetcher != nil {
    if len(providers) != 1 {
        panic("runtime: fetcher override requires exactly one provider; use SetFetcherForProvider for multi-provider setups")
    }
    runtimeFetcher = fetcher
}
```

---

### Задача 5: `nil` guards в `ProviderRuntimeRegistry` — несогласованность `cache` vs `inspect`

**Файл:** `plugins/cost/internal/runtime/provider_runtime_registry.go:156-163`

`CleanExpiredCache` проверяет `r.cache == nil`, остальные методы проверяют `r.inspect == nil`.
Оба поля инициализируются вместе в конструкторе, но guard'ы разные — потенциальная бомба.

**После:** унифицировать все guards на `r.inspect == nil`:
```go
func (r *ProviderRuntimeRegistry) CleanExpiredCache(ctx context.Context) {
    if r.inspect == nil {
        return
    }
    if err := r.cache.CleanExpired(ctx); err != nil {
        log.WithError(err).Debug("failed to clean expired cache")
    }
}
```

---

## P1 — Дизайн и слои (structural improvements)

### Задача 6: `CacheTTLFromConfig` — перенести из `engine` в `model`

**Файл:** `plugins/cost/internal/engine/estimator.go:68-79`

Функция читает `*model.CostConfig` и возвращает `time.Duration` — она принадлежит `model`, а не `engine`.
`engine` знает о конфигурационной детали, которую не должен знать.

**После:**
- Переместить `CacheTTLFromConfig` в `model/config_types.go` как метод:
```go
// CacheTTLDuration returns the configured blob cache TTL, or the default 24h.
func (c *CostConfig) CacheTTLDuration() time.Duration {
    if c == nil {
        return 24 * time.Hour
    }
    if ttl := c.BlobCacheTTL(); ttl != "" {
        if d, err := time.ParseDuration(ttl); err == nil {
            return d
        }
    }
    return 24 * time.Hour
}
```
- В `engine/estimator.go` — удалить `CacheTTLFromConfig`, константу `defaultCacheTTL`
- В `runtime.go:84` — заменить `engine.CacheTTLFromConfig(cfg)` на `cfg.CacheTTLDuration()`

**Затронутые файлы:** `model/config_types.go`, `engine/estimator.go`, `runtime.go`

---

### Задача 7: Тройной дубль `"cost/pricing"` — единственный источник

**Файл:** `engine/estimator.go:19`, `model/config_types.go:26`, `runtime.go` (через `cfg.BlobCacheNamespace()`)

`DefaultCacheNamespace` в `engine` и `DefaultBlobCacheNamespace` в `model` — одна и та же строка.

**После:**
- Удалить `DefaultCacheNamespace` из `engine/estimator.go:19`
- `model.DefaultBlobCacheNamespace` остаётся единственным источником
- В тестах, которые используют `engine.DefaultCacheNamespace` — заменить на `model.DefaultBlobCacheNamespace`

**Затронутые файлы:** `engine/estimator.go`, `engine/estimator_test.go` (если используется)

---

### Задача 8: `ValidateAndPrefetch` — удалить мёртвый production export

**Файл:** `plugins/cost/internal/engine/estimator.go:160-166`

`ValidateAndPrefetch` не вызывается ни в одном production code path. Production использует `coord.Estimate`, который встраивает prefetch. Метод — мёртвый export.

**После:** удалить метод полностью из `Estimator`.
Если нужна тестовая точка входа — создать в `engine_test` пакете wrapper.

**Затронутые файлы:** `engine/estimator.go`, тесты использующие его

---

### Задача 9: `Estimator` — поля `resolver`, `scanner`, `executor` не нужны на struct

**Файл:** `plugins/cost/internal/engine/estimator.go:22-29`

`resolver`, `scanner`, `executor` хранятся на `Estimator` как поля, но используются только через `coord`. Единственные публичные методы, которые идут мимо `coord`:
- `EstimateModule` → `e.scanner.Scan` + `e.executor.Execute`

Но все эти три поля уже содержатся в `coord`. Это дублирование состояния.

**После:** убрать `resolver`, `scanner`, `executor` с `Estimator`. `EstimateModule` делегирует через `coord` или держит minimal-path:
```go
type Estimator struct {
    coord    *estimateCoordinator
    catalog  *costruntime.ProviderCatalog
    runtimes *costruntime.ProviderRuntimeRegistry
}
```
`EstimateModule` переписать через прямое обращение к `coord.scanner` и `coord.executor` (они есть на `estimateCoordinator`).

**Затронутые файлы:** `engine/estimator.go`, `engine/batch.go` (добавить accessors на coordinator если нужны)

---

### Задача 10: `AggregateCost` в `executor.go` — удалить ненужный facade

**Файл:** `plugins/cost/internal/engine/executor.go:46-48`

```go
func AggregateCost(result *model.ModuleCost, rc model.ResourceCost, action EstimateAction) {
    results.AggregateCost(result, rc, action)
}
```

Это точный re-export без добавления ценности. Используется только в тестах (проверить grep). Тесты должны импортировать `results` напрямую.

**После:** удалить функцию из `executor.go`. Обновить вызывающие тесты.

**Затронутые файлы:** `engine/executor.go`, тесты

---

### Задача 11: `ModuleExecutor.resolver` — конкретный тип вместо интерфейса

**Файл:** `plugins/cost/internal/engine/executor.go:12-13`

```go
type ModuleExecutor struct {
    resolver *costruntime.CostResolver
}
```

Принимает конкретный тип — нельзя подменить в тестах без построения полного `CostResolver`.

**После:** ввести локальный интерфейс:
```go
type resourceResolver interface {
    ResolveWithSubResourcesState(ctx context.Context, req costruntime.ResolveRequest, state *costruntime.ResolutionState) []model.ResourceCost
    ResolveBeforeCostWithState(ctx context.Context, rc *model.ResourceCost, resourceType handler.ResourceType, beforeAttrs map[string]any, region string, state *costruntime.ResolutionState)
}

type ModuleExecutor struct {
    resolver resourceResolver
}
```

**Затронутые файлы:** `engine/executor.go`, `engine/estimator.go` (конструктор)

---

### Задача 12: `errgroup` → `sync.WaitGroup` в `batch.go`

**Файл:** `plugins/cost/internal/engine/batch.go:71-79`

`errgroup` используется при том, что ни одна goroutine никогда не возвращает ошибку (`return nil` всегда). Это неверная семантика.

**После:**
```go
var wg sync.WaitGroup
wg.Add(len(executablePlans))
for _, scanned := range executablePlans {
    go func() {
        defer wg.Done()
        moduleResults[scanned.Index] = *b.executor.Execute(ctx, scanned.Plan)
    }()
}
wg.Wait()
```

И удалить `"golang.org/x/sync/errgroup"` из импортов.

**Затронутые файлы:** `engine/batch.go`

---

### Задача 13: `logCacheState` — разделить logging и cache cleanup

**Файл:** `plugins/cost/runtime.go:87-118`

Функция с именем `logCacheState` вызывает `cache.CleanExpired(ctx)` — side effect mutation внутри read-only named function.

**После:** разделить на два явных вызова в `newRuntime`:
```go
logCacheState(ctx, estimator.Cache())
estimator.Cache().CleanExpired(ctx)
```

И убрать `cache.CleanExpired(ctx)` из `logCacheState`. Функция `logCacheState` должна принимать `CacheInspector` (не `*Estimator`), чтобы явно показывать что она read-only.

**Затронутые файлы:** `runtime.go`

---

### Задача 14: `cloud.Get` — удалить мёртвый production API

**Файл:** `plugins/cost/internal/cloud/registry.go:80-85`

`Get(name string)` нигде не вызывается в production коде. Только в тестах (если вообще).

**После:** переместить в `cloud/export_test.go` по аналогии с `ResetForTesting`.

**Затронутые файлы:** `cloud/registry.go`, `cloud/export_test.go`

---

### Задача 15: `cloud` registry — `sync.Mutex` → `sync.RWMutex`

**Файл:** `plugins/cost/internal/cloud/registry.go:49-50`

`Providers()` и `Get()` — read-only операции, но используют полную блокировку `sync.Mutex`.

**После:**
```go
var cpMu sync.RWMutex

func Providers() []Provider {
    cpMu.RLock()
    defer cpMu.RUnlock()
    ...
}
```

`Register` продолжает использовать `cpMu.Lock()`.

**Затронутые файлы:** `cloud/registry.go`

---

### Задача 16: `aws/resources.go` — `var Definition` → `unexported` + accessor

**Файл:** `plugins/cost/internal/cloud/aws/resources.go:22`

`var Definition = cloud.Definition{...}` — экспортированная mutable переменная, которую можно перезаписать в runtime. Это unintended.

**После:**
```go
var definition = cloud.Definition{...}

func providerDefinition() cloud.Definition { return definition }
```

И в `init()` использовать `providerDefinition()`. Либо просто сделать `var definition` unexported.

**Затронутые файлы:** `cloud/aws/resources.go`, `cloud/aws/provider.go` (если есть)

---

### Задача 17: `aws/resources.go` — один `deps` вместо N вызовов `NewRuntimeDeps`

**Файл:** `plugins/cost/internal/cloud/aws/resources.go:30-59`

`awskit.NewRuntimeDeps(providerRuntime)` вызывается 16 раз подряд — по одному на каждый handler.
Все возвращают одно и то же значение.

**После:**
```go
var (
    providerRuntime = awskit.NewRuntime(awskit.Manifest)
    deps            = awskit.NewRuntimeDeps(providerRuntime)
)

var definition = cloud.Definition{
    ...
    Resources: []cloud.ResourceRegistration{
        {Type: handler.ResourceType(awskit.ResourceInstance), Handler: &ec2.InstanceHandler{RuntimeDeps: deps}},
        ...
    },
}
```

**Затронутые файлы:** `cloud/aws/resources.go`

---

## P2 — Качество кода (Go idioms, naming, contracts)

### Задача 18: `formatting.go` — заменить custom helpers на stdlib

**Файл:** `plugins/cost/internal/model/formatting.go`

Три функции реализуют то, что есть в stdlib:

| Функция | Замена |
|---------|--------|
| `hasDecimal(s)` | `strings.ContainsRune(s, '.')` |
| `splitDecimal(s)` | `strings.Cut(s, ".")` → `(intPart, decPart, found)` |
| `trimZeros(s)` | `strings.TrimRight(s, "0")` + обрезка trailing `.` |

**После** — убрать три функции, добавить `"strings"` в imports, использовать stdlib в `formatWithCommas` и `trimTrailingZeros`:

```go
func trimTrailingZeros(cost float64, precision int) string {
    s := sprintfFloat(cost, precision)
    if !strings.ContainsRune(s, '.') {
        return s
    }
    s = strings.TrimRight(s, "0")
    return strings.TrimRight(s, ".")
}

func formatWithCommas(cost float64) string {
    s := trimTrailingZeros(cost, 2)
    intPart, decPart, _ := strings.Cut(s, ".")
    ...
}
```

**Затронутые файлы:** `model/formatting.go`

---

### Задача 19: `output.go` — `sort.Strings` → `slices.Sort`

**Файл:** `plugins/cost/output.go:196-202`

```go
sort.Strings(keys)
```

**После:**
```go
slices.Sort(keys)
```

Убрать импорт `"sort"` из `output.go`.

**Затронутые файлы:** `output.go`

---

### Задача 20: `output.go` — `default: return true` → `default: return false`

**Файл:** `plugins/cost/output.go:192-193`

`shouldShowResource` с `default: return true` молча показывает любой будущий `CostErrorKind` — новые ошибки становятся видимыми без явного решения.

**После:**
```go
default:
    return false // unknown error kinds are hidden until explicitly handled
```

**Затронутые файлы:** `output.go`

---

### Задача 21: `output.go` — убрать `isZeroCost` однострочник

**Файл:** `plugins/cost/output.go:205-207`

```go
func isZeroCost(cost float64) bool { return cost == 0 }
```

Используется один раз. Прямое `submodule.MonthlyCost == 0` читается лучше.

**После:** убрать функцию, заменить вызов на прямое сравнение.

**Затронутые файлы:** `output.go`

---

### Задача 22: `output.go` — `renderSummary` — `defer` для `DecreasePadding`

**Файл:** `plugins/cost/output.go:27-44`

Два пути выхода, оба должны вызвать `log.DecreasePadding()`. Паттерн хрупкий — добавление ещё одного return создаст утечку.

**После:**
```go
func renderSummary(result *model.EstimateResult) {
    log.Info("summary")
    log.IncreasePadding()
    defer log.DecreasePadding()
    ...
}
```

**Затронутые файлы:** `output.go`

---

### Задача 23: `commands.go` — именованная константа для timeout и default output format

**Файл:** `plugins/cost/commands.go:36,44`

```go
context.WithTimeout(cmd.Context(), 5*time.Minute)   // magic literal
cmd.Flags().StringVarP(&costOutputFmt, "output", "o", "text", ...)  // magic string
```

**После:**
```go
const (
    costDefaultTimeout   = 5 * time.Minute
    costDefaultOutputFmt = "text"
)
```

**Затронутые файлы:** `commands.go`

---

### Задача 24: `commands.go` — `appCtx` вместо `ctx` для `*plugin.AppContext`

**Файл:** `plugins/cost/commands.go:15`

```go
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
```

По всей кодовой базе `*plugin.AppContext` называется `appCtx`. Здесь `ctx` — конфликт с конвенцией и с `cmd.Context()`.

**После:**
```go
func (p *Plugin) Commands(appCtx *plugin.AppContext) []*cobra.Command {
```

**Затронутые файлы:** `commands.go`

---

### Задача 25: `pipeline.go` — `AllowFailure: true` — добавить комментарий

**Файл:** `plugins/cost/pipeline.go:29`

Молчаливое `AllowFailure: true` — решение без обоснования.

**После:**
```go
// AllowFailure: cost estimation is best-effort; it should not block deployments.
AllowFailure: true,
```

**Затронутые файлы:** `pipeline.go`

---

### Задача 26: `pipeline.go` — `resultsFile`/`reportFile` переехать в `report.go`

**Файл:** `plugins/cost/pipeline.go:10-13`

Константы используются как в `pipeline.go`, так и в `report.go`. Семантически они принадлежат `report.go` (artifact naming).

**После:** перенести константы в `report.go`, убрать из `pipeline.go` (добавить import если нужен — но это тот же пакет `cost`, поэтому просто переместить объявление).

**Затронутые файлы:** `pipeline.go`, `report.go`

---

### Задача 27: `usecases.go` — добавить `workDir` в error при отсутствии планов

**Файл:** `plugins/cost/usecases.go:54`

```go
return nil, errors.New("no plan.json files found")
```

**После:**
```go
return nil, fmt.Errorf("no plan.json files found in %s", workDir)
```

**Затронутые файлы:** `usecases.go`

---

### Задача 28: `usecases.go` — log prefix `"cost: "` для consistency

**Файл:** `plugins/cost/usecases.go:45,57`

```go
log.WithField("dir", workDir).Info("scanning for plan.json files")
log.WithField("count", len(modulePaths)).Info("modules with plan.json found")
```

**После:**
```go
log.WithField("dir", workDir).Info("cost: scanning for plan.json files")
log.WithField("count", len(modulePaths)).Info("cost: modules with plan.json found")
```

**Затронутые файлы:** `usecases.go`

---

### Задача 29: `usecases.go` — `make([]string, 0, 1)` → `var filtered []string`

**Файл:** `plugins/cost/usecases.go:79`

**После:**
```go
var filtered []string
```

**Затронутые файлы:** `usecases.go`

---

### Задача 30: `model/tree.go` — `FindChild` и `SplitPath` → unexported

**Файл:** `plugins/cost/internal/model/tree.go:54,92`

Обе функции используются только внутри `tree.go`. Экспорт создаёт ненужный public API.

**После:** `FindChild` → `findChild`, `SplitPath` → `splitPath`.

Обновить все вызовы внутри `tree.go` и в тестах, которые могут на них ссылаться.

**Затронутые файлы:** `model/tree.go`, `model/tree_test.go`

---

### Задача 31: `model/config_types.go` — loop variable `p` → `pc`

**Файл:** `plugins/cost/internal/model/config_types.go:43`

```go
for _, p := range c.Providers {
```

`p` — конвенциональное имя для `*Plugin`. Здесь это `ProviderConfig`.

**После:**
```go
for _, pc := range c.Providers {
    if pc.Enabled {
```

**Затронутые файлы:** `model/config_types.go`

---

### Задача 32: `model/config_types.go` — `BlobCacheTTL()` возвращает `time.Duration` а не `string`

**Файл:** `plugins/cost/internal/model/config_types.go:83-89`

После задачи 6 метод `CacheTTLDuration()` уже будет добавлен как правильная версия. `BlobCacheTTL() string` становится реализационной деталью, используемой только внутри `CacheTTLDuration`. Можно сделать unexported.

**После:** `BlobCacheTTL()` → `blobCacheTTL()` (unexported), используется только в `CacheTTLDuration`.

**Затронутые файлы:** `model/config_types.go`, `engine/estimator.go` (если использовалось там)

---

### Задача 33: `handler/registry.go` — `SupportedTypes()` возвращает `[]ResourceType`

**Файл:** `plugins/cost/internal/handler/registry.go:44-57`

**До:** возвращает `[]string` с потерей типа
**После:** возвращает `[]ResourceType`

```go
func (r *Registry) SupportedTypes() []ResourceType {
    typeSet := make(map[ResourceType]bool)
    for _, providerHandlers := range r.handlers {
        for t := range providerHandlers {
            typeSet[t] = true
        }
    }
    types := make([]ResourceType, 0, len(typeSet))
    for t := range typeSet {
        types = append(types, t)
    }
    return types
}
```

**Затронутые файлы:** `handler/registry.go`, вызовы `SupportedTypes()` в тестах

---

### Задача 34: `handler/attrs.go` — добавить `json.Number` и `string` в type switch

**Файл:** `plugins/cost/internal/handler/attrs.go:14-26`

Terraform JSON может содержать числа как `string` или `json.Number`. Текущий код молча возвращает `0`.

**После** `GetFloatAttr`:
```go
case json.Number:
    f, _ := val.Float64()
    return f
case string:
    f, _ := strconv.ParseFloat(val, 64)
    return f
```

Аналогично для `GetIntAttr`.

**Затронутые файлы:** `handler/attrs.go`

---

### Задача 35: `pricing/types.go` — добавить период в doc-comments

**Файл:** `plugins/cost/internal/pricing/types.go:72,82,91`

```go
// PriceIndex represents a compact pricing index for a service/region
// Price represents a single product price
// PriceLookup represents criteria for finding a price
```

**После:** добавить точки на конце каждого doc-comment.

**Затронутые файлы:** `pricing/types.go`

---

### Задача 36: `pricing/cache.go` — `ExpiresAt` от `time.Now()`, не от timestamp данных

**Файл:** `plugins/cost/internal/pricing/cache.go:242`

TTL должен отсчитываться от момента сохранения в кеш, а не от `idx.UpdatedAt` (который устанавливается fetcher'ом и может быть в прошлом).

**После:** заменить `idx.UpdatedAt.Add(c.ttl)` на `time.Now().Add(c.ttl)`.

**Затронутые файлы:** `pricing/cache.go`

---

### Задача 37: `awskit/lookup.go` — `Lookup` не должен мутировать `attrs` вызывающего

**Файл:** `plugins/cost/internal/cloud/awskit/lookup.go:13-17`

```go
func (b *PriceLookupSpec) Lookup(region string, attrs map[string]string) *pricing.PriceLookup {
    if attrs == nil {
        attrs = make(map[string]string)
    }
    attrs["location"] = ResolveRegionName(region)  // мутирует аргумент
```

**После:** не мутировать аргумент, создать копию:
```go
merged := make(map[string]string, len(attrs)+1)
for k, v := range attrs {
    merged[k] = v
}
merged["location"] = ResolveRegionName(region)
```

**Затронутые файлы:** `awskit/lookup.go`

---

### Задача 38: `awskit/describe.go` — `Map()` убрать type cast

**Файл:** `plugins/cost/internal/cloud/awskit/describe.go:50-52`

```go
func (d DescribeBuilder) Map() map[string]string {
    return map[string]string(d)
}
```

Тип `DescribeBuilder` уже `map[string]string`. Type cast — бесполезная конверсия.

**После:**
```go
func (d DescribeBuilder) Map() map[string]string { return d }
```

**Затронутые файлы:** `awskit/describe.go`

---

### Задача 39: `results/assembler.go` — `append([]model.ModuleCost(nil), ...)` → `slices.Clone`

**Файл:** `plugins/cost/internal/results/assembler.go:102`

```go
Modules: append([]model.ModuleCost(nil), a.modules...),
```

**После:**
```go
Modules: slices.Clone(a.modules),
```

**Затронутые файлы:** `results/assembler.go`

---

### Задача 40: `results/assembler.go` — `NewErroredModule` — инициализировать `Resources: []model.ResourceCost{}`

**Файл:** `plugins/cost/internal/results/assembler.go:130-136`

`NewModuleAssembler` явно инициализирует `Resources: make([]model.ResourceCost, 0)`.
`NewErroredModule` оставляет `Resources: nil` → JSON выдаёт `"resources": null` вместо `"resources": []`.

**После:**
```go
func NewErroredModule(modulePath, region string, err error) model.ModuleCost {
    return model.ModuleCost{
        ModuleID:   strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
        ModulePath: modulePath,
        Region:     region,
        Error:      err.Error(),
        Resources:  []model.ResourceCost{},
    }
}
```

**Затронутые файлы:** `results/assembler.go`

---

### Задача 41: `runtime/resolver.go` — `CostCategoryUsageBased` в `ResolveBeforeCostWithState` — добавить комментарий

**Файл:** `plugins/cost/internal/runtime/resolver.go:222-223`

```go
case handler.CostCategoryUsageBased:
```

Пустой case без комментария — непонятно это баг или намерение.

**После:**
```go
case handler.CostCategoryUsageBased:
    // Usage-based resources have no fixed before-cost: skip.
```

**Затронутые файлы:** `runtime/resolver.go`

---

### Задача 42: `runtime/resolver.go` — логировать `CostErrorNoHandler` аналогично `CostErrorNoProvider`

**Файл:** `plugins/cost/internal/runtime/resolver.go:150-154`

`CostErrorNoProvider` → `logUnsupportedResource`. `CostErrorNoHandler` → нет лога. Несогласованная наблюдаемость.

**После:**
```go
h, ok := r.runtime.ResolveHandler(providerID, req.ResourceType)
if !ok {
    result.ErrorKind = model.CostErrorNoHandler
    result.ErrorDetail = "no handler"
    logUnsupportedResource(req.ResourceType.String(), req.Address)
    return result
}
```

**Затронутые файлы:** `runtime/resolver.go`

---

## Порядок выполнения

```
P0 (корректность): 1 → 2 → 3 → 4 → 5
P1 (дизайн):       6 → 7 → 8 → 9 → 10 → 11 → 12 → 13 → 14 → 15 → 16 → 17
P2 (качество):     18 → 19 → 20 → 21 → 22 → 23 → 24 → 25 → 26 → 27 → 28 → 29 → 30 → 31 → 32 → 33 → 34 → 35 → 36 → 37 → 38 → 39 → 40 → 41 → 42
```

---

## Acceptance Criteria

- [ ] `go build ./plugins/cost/...` — чистая сборка
- [ ] `go test ./plugins/cost/... -count=1 -race` — 19/19 пакетов PASS
- [ ] `go vet ./plugins/cost/...` — без замечаний
- [ ] `engine.CacheTTLFromConfig` не существует — есть `model.CostConfig.CacheTTLDuration()`
- [ ] `engine.DefaultCacheNamespace` не существует — единственный источник `model.DefaultBlobCacheNamespace`
- [ ] `engine.ValidateAndPrefetch` не существует
- [ ] `engine.AggregateCost` facade не существует
- [ ] `Estimator` struct не имеет полей `resolver`, `scanner`, `executor`
- [ ] `errgroup` импорт удалён из `engine/batch.go`
- [ ] `sort` импорт удалён из `output.go`
- [ ] `logCacheState` не вызывает `CleanExpired`
- [ ] `cloud.Get` не присутствует в production файлах
- [ ] `var Definition` в `aws/resources.go` — unexported
- [ ] `FindChild` → `findChild`, `SplitPath` → `splitPath` в `model/tree.go`
- [ ] `WarmIndexes` использует `errors.Join` (best-effort)
- [ ] `PriceLookupSpec.Lookup` не мутирует аргумент `attrs`

---

## Ожидаемый результат

| Измерение | До v4 | После v4 |
|-----------|-------|----------|
| Слои | 8.5 | 9.5 |
| Именование | 7.5 | 9.0 |
| Контракты | 7.5 | 9.5 |
| SRP | 8.0 | 9.5 |
| Абстракции | 8.0 | 9.0 |
| Go-идиомы | 7.5 | 9.5 |
| Мёртвый код | 7.5 | 9.5 |
| Документация | 8.5 | 9.5 |
| **Итог** | **8.2** | **9.5** |
