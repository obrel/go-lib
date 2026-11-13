# go-lib

Golang shared lib for obrel projects.

- gst: GStreamer binding (cgo)
- [wrk](pkg/wrk): Simple worker and task management
- [log](pkg/log): Production-grade Logrus wrapper with Sentry integration (Grade B)
- [rr](pkg/rr): Simple round robin implementation

## pkg/log - Logger

Thread-safe, structured logger built on `logrus v1.9.3` + `sentry-go v0.50.0`. Replaces deprecated `raven-go` / `logrus_sentry`.

### Features (Grade B)

- **Safe level mapping**: `PanicLevel`, `FatalLevel`, `ErrorLevel`, `WarnLevel`, `InfoLevel`, `DebugLevel`, `TraceLevel`, `OFF` (truly disables via discard)
- **Thread-safe config**: `SetOutput`, `SetLevel`, `SetFormat`, `AddHook` protected by `RWMutex`; hot path uses `atomic.Bool` for OFF check
- **Structured API**: `WithField`, `WithFields`, `WithError`, `WithContext`, `IsLevelEnabled`, `Panic/Panicf`
- **Contextual logger**: `For(pkg, scope)` captures file:line via `Base + Itoa`, with `package`/`scope` fields
- **Isolated loggers**: `New()` returns non-global logger for tests/per-component use
- **Sentry v2**: 
  - `NewSentryHook` / `AddSentryHook` -> Sentry Logs product (Info+)
  - `NewSentryEventHook` / `AddSentryEventHook` -> Sentry Issues (Error+), backward compat with old raven behavior
  - Fixed nil deref bug when DSN invalid
  - Flush on `Fatal` via `RegisterExitHandler`
- **Graceful shutdown**: `Flush(timeout)` and `FlushWithContext(ctx)` respects deadline
- **Formats**: `JSONFormat` / `TextFormat` with RFC3339 timestamps
- **Tests**: `go test -race ./pkg/log` covers OFF, structured, concurrency, sentry error handling

### Usage

```go
import "github.com/obrel/go-lib/pkg/log"

func main() {
    // Basic
    log.SetOutput(os.Stderr)
    log.SetFormat(log.JSONFormat)
    log.SetLevel(log.InfoLevel)

    log.Info("server started")
    log.GetLogger().WithField("port", 8080).Info("listening")

    // Contextual
    logger := log.For("myapp", "http").WithError(err).WithField("req_id", id)
    logger.Error("failed")

    // With context (trace_id injection via Sentry hub)
    ctx := context.Background()
    logger.WithContext(ctx).Info("request")

    // Sentry Logs (new product)
    _ = log.AddSentryHook("https://...@sentry.io/1", "production", "v1.2.3")

    // Sentry Issues (errors as events) - old behavior
    _ = log.AddSentryEventHook("https://...@sentry.io/1", "production", "v1.2.3")

    // Graceful shutdown
    defer log.Flush(2 * time.Second)

    // OFF disables all
    log.SetLevel(log.OFF)

    // Isolated for tests
    testLog := log.New()
}
```

### Production Readiness

| Check | Status |
|-------|--------|
| No race on config | ✅ RWMutex + atomic |
| OFF truly disables | ✅ discard + atomic |
| Nil deref on bad DSN | ✅ fixed |
| Deprecated deps removed | ✅ sentry-go maintained |
| Structured logging | ✅ WithField/Fields/Error/Context |
| Caller accurate | ✅ For + SetReportCaller |
| Flush on Fatal | ✅ ExitHandler |
| Tests + -race | ✅ log_test.go |
| Perf (no lock per log) | ✅ atomic.Bool hot path |
| Global singleton escape | ⚠️ still global `std`, use `New()` for isolation |

Grade: **B** - Ready for internal services, medium traffic. For 10k+ RPS / zero-alloc, consider `slog` or `zap`.

### Migration from v0

- `OFF` now truly hides logs, previously enabled Trace.
- `Fatal/Fatalf` still call `os.Exit(1)` after flush; mock with `SetExitFunc` in tests.
- Sentry: old `AddSentryHook` sent Issues via raven. New `AddSentryHook` sends Logs. Use `AddSentryEventHook` if you need Issues.
- `Logger` interface extended with `Panic`, `Trace`, structured methods. Recompile dependents.

## Development

```bash
go vet ./pkg/log ./pkg/wrk ./pkg/rr
go test -race ./pkg/... -v
go mod tidy
```
