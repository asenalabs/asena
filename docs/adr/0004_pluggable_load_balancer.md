## ADR-0004: Pluggable load balancer, round-robin default

* **Status:** Accepted (backfilled - this ADR was written after the code already existed)

## Context

A `service` in `dynamic.yaml` can point at more than one backend server, so Asena needs pick which one handles each request. The project's roadmap already lists more algorithms to come(`weighted-round-robin`, `least-connection`, ...) so the very first algorithm shouldn't be wired in as a one-off.

## Decision

Define a small `Balancer` interface:

```go
type Balancer interface {
    Next() config.ServerCfg
}
```

and a `New(algorithm, servers)` factory. Today it only implements round-robin, and it also **falls back to round-robin** for any unset or unrecognized `algorithm` value rather than erroring.

## Consequences

**Good:**

* Adding `weight-round-robin` or `least-connection` later means adding a new type that satisfies `Balancer` and one `case` in the factory - call sites don't change.
* A service always gets a working balanceer, even if `alorithm` is left out of the config entirely.

**Costs:**

* Today, this fallback is safe to have. Before Asena uses an `algorithm` value, `internal/config` checks it first. If the value is missing or wrong, Asena stops and shows an error right there.. So the fallback never actually runs.
* One small thing to remember: the list of valid algorithm names is written in two places - once in that check, and once inside `balancer.New()`. When we add a new algorithm, we must update both places.

## Alternatives Considered

* **Hardcode round-robin only, no interface.** Rejected - the roadmap already commits to more algorithms, and retrofitting an interface later would mean touching every call site that currently assumes round-robin.

## Related Code Location

`internal/proxy/balancer/`