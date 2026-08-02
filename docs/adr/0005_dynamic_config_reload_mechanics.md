## ADR-0005: fsnotify-based dynamic config reload

* **Status:** Accepted (backfilled - this ADR was written after the code already existed)

## Context

`dynamic.yaml` (see ADR-0001) needs to be reloaded automatically when it changes on disk, without downtime. Two practical problems show up as soon as you try this:

1. A single logical "save" from an editor or `mv` often fires *several* filesystem events (write, rename, chmod) -naively reloading on every event means reloading the same content multiple times in a row.
2. Whatever reads the reloaded config (the proxy manages) shouldn't be able to block the file watcher just by being slow to consume an update.

## Decision

* Watch `dynamic.yaml` with [fsnotify](https://github.com/fsnotify/fsnotify).
* **Debounce** events for 500ms - if more events arrive within that window, only the last one triggers a reload.
* **Hash the file** (SHA-256) after parsing and skip publishing an update if the content hash hasn't actually changed since the last reload.
* Publish new configs on a **buffered channel of size 1** an unconsumed one already sitting in the buffer (last-write-wins) instead of blocking.

## Consequences

**Good:**

* Resilient to editors/tools that touch the file multiple times per save.
* The watcher loop blocks waiting on a slow or stalled consumer.
* Reloading unchanged content is a no-op - no unnecessary rebuilds downstream (e.g. the proxy manager doesn't rebuild routes for nothing).

**Costs:**

* A consumer that genuinely needs to see **very** intermediate config state (not just the last) will silently miss updates -this is documented behavior in `docs/DYNAMIC_CONFIG.md`, but it's easy to forget and would need a bigger buffer or a custom fan-out if that requirment ever shows up.
* The 500ms debounce is a fixed delay between a file save and the change taking effect - small, but worth knowing about if someone is debugging "why didn't my change apply immediately."

## Alternatives Considered

* **Reload on every raw fsnotify event, no debounce.** Rejected - causes redundant reloads and log noise for a single logical save.
* **Unbuffered channel with a blocking send.** Rejected - would let a slow or stuck consumer stall the entire file watcher goroutine.

## Related Code Location

`internal/config/`