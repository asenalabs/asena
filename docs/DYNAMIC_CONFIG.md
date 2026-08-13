#   Dynamic Configuration

Asena supports **dynamic routing** through a `dynamic.yaml` file. 
This allows you to add, remove, or update routers and services at runtime
without restarting the server.
Changes are detected automatically [fsnotify](https://github.com/fsnotify/fsnotify)
and applied in real time.

---
##  File Structure

The configuration is organized into two main sections:

```yaml
http:
    routers:    #Incoming request rules
    services:   #Upstream services (backends)
```
### 1. Routers

Routers define matching rules and map incoming requests to a service.

| Field   | Type   | Description                                                                                                                                                 |
|---------|--------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| rule    | string | Matching expression built from one or more matchers, combined with `&&`, <code>&#124;&#124;</code>, `!`, and parentheses for grouping. |
| service | string | Name of the target service (must exist under `services`).                                                                                                   |

#### Examples
```yaml
http:
  routers:
    api-router:
      rule: "Host(`localhost`)"
      service:  api-service
    api-v2-writes:
      rule: "Host(`api.example.com`) && PathPrefix(`/v2`) && !Method(`GET`)"
      service: api-service
```
Supported matchers:
- ``Host(`example.com`)`` - matches the request's Host header, ignoring port and letter case.
- ``PathPrefix(`/v2`)`` - matches when the request path starts with the given prefix.
- ``Method(`GET`)`` - matches the HTTP method exactly (case-insensitive on input, normalized to uppercase).
- ``Header(`X-Api-Key`, `secret`)`` - matches when the named header is present with exactly this value.

Matchers can be combined with `&&` (AND), `||` (OR), `!` (NOT), and parentheses for grouping - `&&` binds tighter than `||`, the same as most C-family languages, so use parentheses when you want an OR to span an AND.

When two or more routers' rules could both match the same request, the **more specific** rule wins — roughly: a `Header` match outranks a `Method` match, which outranks `PathPrefix`, which outranks a bare `Host` match, and combining matchers with `&&` always outranks any single one of them alone. This is computed automatically from the rule; you don't configure it directly.

### 2. Services
Services define load-balancing to one or more upstream servers.

| Field            | Type   | Description                                                                                                                            |
|------------------|--------|----------------------------------------------------------------------------------------------------------------------------------------|
| algorithm        | string | Load balancing algorithm. Supported: `round-robin`(default)/(`weighted-round-robin`,`least-connections` and others are coming soon...) | 
| flash_interval   | string | How often server weights/health are refreshed (e.g. `500ms`, `10s`).                                                                   |
| pass_host_header | bool   | Forward original `Host` header to backend.                                                                                             |
| servers          | list   | Array of backend servers with `url` (and optional `weight`).                                                                           |

#### Examples
```yaml
http:
  services:
    api-service:
      load_balancer:
        algorithm: round-robin
        flash_interval: 500ms
        pass_host_header: false
        servers:
          - url: "http://localhost:9000"
          - url: "http://localhost:9001"
```

---

## Fallback Behavior

- If `dynamic.yaml` is missing or contains no valid routers/services,
Asena automatically falls back to static configuration and returns **503 Service Unavailable** for any requests not matched by static routes.
- If routers/services are entered correctly, but `algorithm`, `flash_interval`, `pass_host_header` are not attached, Asena will enter default values (`round-robin`, `500ms`, `false`) for these.

---

##  Error Handling

During reload, Asena validates the `dynamic.yaml` file.
If the file is missing fields or contains invalid values, reload will fail and the previous configuration will remain active.

Common error messages include:
- `http section is missing` → the root `http` section is not defined.
- `routers section is missing` → no `routers` block found under `http`.
- `services section is missing` → no `services` block found under `http`.
- `load_balancer section is missing` → a service does not define a `load_balancer`.
- `load_balancer.servers section is missing` → no backend servers are listed for a service.
- `algorithm is not set` → missing or nil load balancing algorithm.
- `unknown algorithm` → algorithm field has an unsupported value.
- `failed to parse dynamic config file` → invalid YAML format.

✅ On error:
- Asena logs the detailed message via `zap.Logger`.
- The server continues running with the **last valid configuration** to avoid downtime.

**Note on invalid rules specifically:** an unparsable `rule` (e.g. a typo'd matcher name, or unbalanced parentheses) is *not* one of the reload-failing errors above. It's handled one level down: that single router is logged as skipped and excluded from matching, while the rest of a valid `dynamic.yaml` — every other router, and all services — reloads normally. A mistake in one team's router definition doesn't take down anyone else's.

---

## Updates Channel Semantics

The `Updates()` method exposes a channel that delivers new configs on reload.
- The channel is created with a buffer size of `1`.
- If the consumer is slow, new events overwrite the old one (last-write-wins).
- This avoids blocking reloads, but means intermediate updates can be dropped.

⚠️ If your application must process **every change**, you should either:
- Increase channel buffer size, or
- Implement your own fan-out/broadcast mechanism.

---

##  Example Layout

Repository suggestion:
```csharp
.
├── asena.yaml          # Static base config
├── dynamic.yaml        # Dynamic routing (hot reload)
└── docs/
    └── DYNAMIC_CONFIG.md
```
