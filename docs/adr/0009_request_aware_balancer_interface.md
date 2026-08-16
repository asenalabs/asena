# ADR-0009: Request-aware Balancer interface and a Done() completion hook

* **Status:** Accepted

## Context

ADR-0004 defined `Balancer.Next() config.ServerCfg` and predicted that adding
more algorithms would just mean "a new type and one case in the factory -
call sites don't change". That held for `weighted-round-robin`, but the next
four algorithms on the roadmap (`least-connections`, `least-time`,
`ip-hash`, `sticky-session`) all needed something `Next()` alone could not
give them: either the incoming request itself, or a signal that a request
had finished, and how long it took.
 
This decision was made and implemented as part of #64. This ADR documents
it as a short follow-up rather than landing in the same PR - the change was
already merged by the time we got to writing it down.

Extend `Balancer` to two methods:
 
    type Balancer interface {
        Next(r *http.Request) *config.ServerCfg
        Done(server *config.ServerCfg, duration time.Duration, err error)
    }
 
`Next` now receives the request, so algorithms can read the client IP or
cookies. `Done` is called once per request after it finishes (success or
failure), so algorithms can track connection counts or response time.
 
 Calling `Done` reliably turned out to be the hard part: `httputil.ReverseProxy`
clones the request before `Rewrite` runs, and its `ErrorHandler` is later
called with the *original* request, not the clone - data attached during
`Rewrite` never reaches the error path. The fix: `Manager.ServeProxy`
attaches an empty mutable result box to the request's context *before*
calling `ServeHTTP`, so both the clone (seen by `Rewrite`/`ModifyResponse`)
and the original (seen by `ErrorHandler`) can reach the same box.
 
## Consequences

**Good:**
 
* Unblocks every algorithm on the roadmap that needs request data or
  completion tracking - 4 of the next 5.
* `Done()` is called exactly once per request, success or failure, without
  either hook needing to know about the other.
* Algorithms that don't need one or both parameters (Round Robin, Weighted
  Round Robin) just ignore them - no forced complexity for the simple cases.

**Cost:**

* Every existing `Balancer` implementation needed a signature change and an
  empty `Done()`, even ones that don't use it.
* The result-box-in-context mechanism isn't obvious on its own - a future
  reader needs `balancerResult`'s doc comment (or this ADR) to understand
  why it exists instead of something simpler.
* `ServeProxy` is now the only correct way to serve a request through a
  managed proxy. Calling `GetProxy(name).ServeHTTP(w, r)` directly silently
  skips the whole mechanism instead of failing loudly.

## Alternatives Considered

* **Create a request-scoped context value fresh inside `Rewrite`.** Rejected
  - this is exactly what doesn't work, since `ErrorHandler` receives the
  original request, not the one `Rewrite` modified.
* **A shared map keyed by request pointer, instead of context.** Rejected -
  needs its own locking for concurrent in-flight requests, with no real
  benefit over context, and is a less idiomatic Go pattern.
* **Keep `Next()` as-is; handle IP/cookie-based algorithms as special cases
  inside `manager.go`.** Rejected - breaks the pluggable design from
  ADR-0004; every new algorithm would mean editing `manager.go`, not just
  adding a type.


## Related Code Location
 
`internal/proxy/balancer/`, `internal/proxy/`