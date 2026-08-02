# ADR-0002: Structured logging with Zap + Lumberjack

* **Status:** Accepted (backfilled - this ADR was written after the code already existed)

## Context

As a server handling live traffic, Asena needs structured, leveled logs (not just `fmt.Println`), and it needs log rotation so log files don't grow forever. There's also a chicken-and-egg problem: the *real* logger's settings (output path, rotation, dev vs. prod format) live inside `asena.yaml` but something needs to be able to log errors if loading that very file fails.

## Decision

Use [Zap] (https://github.com/uber-go/zap) for structured logging, in two phases:

1. **`InitFallbackZapLogger()`** \- a bare console logger with no config dependency\, used only until the real config is loaded\.
2. **`InitProductionZapLogger(env, cfg)`** \- a config\-driven logger\. In [Lumberjack](https://github.com/natefinch/lumberjack); in development it also echoes to the console in a human-readable format.

## Consequences

**Good:**

* Config loading can log real errors (e.g. "failed to read asena config file) before the "real" logger even exists.
* Startup code doesn't need to pass a logger around before one is ready.

**Costs:**

* `logger.Get()` **panics** if called before either init function has run - there's an implicit ordering dependency (`InitFallbackZapLogger()` or `InitProductionZapLogger()` must run first).
* A package-level global logger is convenient but implicit - it's harder to swap or mock in tests than an injected logger would be, and any package that imports `pkg/logger` has a hidden dependency on init order.

## Alternatives Considered

* **Dependency-inject a logger everywhere from process start,** Rejected for now - would require restructuring startup to build the logger before anything else and thread it through every constructor. More plumbing than the project needs at its current size; worth revisiting if the global singleton starts causing test pain.
* **Standard library `log` package.** Rejected - no structured fields or levels, and no rotation without extra glue code.

## Related Code Location

`pkg/logger/`