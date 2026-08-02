## ADR-0007: Atomic proxy and router hot-swap

* **Status:** Accepted (backfilled - this ADR was written after the code already existed)

## Context

When `dynamic.yaml` changes, the set of routers and reverse proxies inside `proxy.Manager` must be updated without dropping in-flight requests and without restarting the process. ADR-0005 covers *when* a new config is picked up (the file watcher); this ADR covers *how* the proxy actually switches over to it safely while requests are still arriving.

## Decision

`proxy.Manager` stores its current maps of routers and reverse proxies inside an `atomic.Value`. When a new configuration is loaded:

1. A completely new set of proxies and routers is built from scratch.
2. The new maps are stored atomically, replacing the old ones in one step.
3. Requests already in flight keep using the old proxies until they finish.
4. Any new request that arrives after the swap immediately uses the new configuration.

A mutex is used only briefly, during the swap itself - not on the request path.

## Consequences

**Good:**

* Zero-downtime configuration updates - no dropped requests during a reload.
* Simple and safe for concurrent access: readers on the request path never need to take a lock.
* No complex locking logic needed where it metters most (the hot request path).

**Costs:**

* Slightly higher memory use during the transition, since the old and new maps briefly exist at the same time.
* The full set of proxies is rebuilt on every config change, even a small one - there's no partial/incremental update.

**Neutral:**

* This pairs directly with the dynamic config watcher in ADR-0005 - that ADR triggers the reload, this one makes the reload safe to apply while serving traffic.

## Alternatives Considered

* **Mutex around the whole request path.** Rejected - would add lock contention to every single request, not just the rare moment a config reloads.

## Related Code Location

`internal/proxy/`