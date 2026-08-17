# ADR-0010: Optional balancer capability interfaces (StickyCookieSetter)

* **Status:** Accepted

## Context

`sticky-session` needs to write a cookie onto the response so a client
returns to the same server next time. Nothing before this needed to touch
the response at all - every algorithm so far only read the request (via
`Next`) or reported back after the fact (via `Done`, see ADR-0009). Adding a
cookie method directly to `Balancer` would force every other algorithm -
Round Robin, Least Connections, everything else - to implement an empty
method it will never use.

This decision was made and implemented as part of #75. This ADR documents
it as a short follow-up rather than landing in the same PR - the change was
already merged by the time we got to writing it down.

## Decision

Define a small, separate interface that only `StickySession` implements:

    type StickyCookieSetter interface {
        SetStickyCookie(header http.Header, server *config.ServerCfg)
    }

`manager.go`'s `ModifyResponse` hook checks for it with a type assertion
(`bl.(balancer.StickyCookieSetter)`) and only calls it if the balancer
actually implements it - the same pattern the standard library uses for
`http.Flusher` and `http.Hijacker`.

## Consequences

**Good:**

* `Balancer` itself stays exactly as defined in ADR-0009 - no forced empty
  methods on algorithms that will never need this.
* Future algorithms that need to touch the response some other way can
  define their own small optional interface, same pattern, without another
  change to `Balancer` itself.

**Cost:**

* This capability is only discoverable by reading `manager.go` or this ADR
  - it doesn't show up in the `Balancer` interface itself.
* Each new optional interface is one more type assertion `manager.go` needs
  to know to check for. Fine with one; worth a registry if this pattern
  grows.

## Alternatives Considered

* **Add a cookie method directly to `Balancer`.** Rejected - forces an empty
  method onto every algorithm that doesn't need it, against the same
  small-interface principle ADR-0009 established.
* **Handle sticky-session cookie logic entirely inside `manager.go`.**
  Rejected - same reasoning as ADR-0009's rejected alternative: breaks the
  pluggable design, `manager.go` would need to know about sticky-session
  specifically.

## Related Code Location

`internal/proxy/balancer/`, `internal/proxy/`