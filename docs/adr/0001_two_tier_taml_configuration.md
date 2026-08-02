# ADR-001: Two-tier YAML configuration (static + dynamic)

- **Status:** Accepted (backfilled - this ADR was written after the code already existed)

## Context

Asena is a reverse proxy, so it has two very different kinds of setting:

* Settings that rarely change and are risky to change at runtime - the port it listens on, whether HTTPS is enabled, where TLS certs live, how logging is configured.
* Settings that describe *what traffic goes where* \- routers and backend services\. There need to change often \(adding a service\, adjusting a backend URL\) and restarting the whole proxy every time would cause downtime for anything already routed through it\.

## Decision

Split configuration into two files with two different lifecycles;

* `asena.yaml` - **static**. Read once at startup (`AsenaConfigService`). Holds process-level settings: `asena` (port, TLS), `log`, `proxy_transport`.
* `dynamic.yaml` - **dynamic**. Watched and hot-reload at runtime (`DynamicConfigService`). Holds `http.routers` and `http.services` \- the actual proxy behavior\.

## Consequensces

**Good:**

* Routing changes take effect without restarting the process or dropping live connections.
* Clear mental model: "if it's about the process, it's static; if it's about traffic routing, it's dynamic."

**Costs:**

* Two config files and two schemas to document and keep straight (see `docs/DYNAMIC_CONFIG.md` for the dynamic side).
* Needs its own reload/validation/fallback machinery (see ADR-0005), which is more code than a single flat config file would need.

## Alternatives Considered

* **Single config file, reload everything on change.** Rejected — reloading

things like the listen port or TLS cert paths at runtime is either

meaningless (you can't rebind without restarting anyway) or risky, and it

would make the reload logic more complex for little benefit.

* **Environment variables for everything.** Rejected — the dynamic side

(routers/services, potentially many of them, nested) doesn't map well to

flat env vars.

## Related Code Location

`internal/config/`