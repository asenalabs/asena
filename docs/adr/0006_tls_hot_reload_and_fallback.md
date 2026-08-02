## ADR-0006: TLS hot-reload and fallback to HTTP

* **Status:** Accepted (backfilled - this ADR was written after the code already existed)

## Context

When HTTPS is enabled, Asena needs to serve TLS certificates that get renewed periodically (e.g. via Let's Encrypt / certbot) - without dropping existing connections or restarting the process. It also needs to decide what happens if TLS is enabled in config but the certificate or key can't actually be loaded at startup.

## Decision

* Load certificates through a `CertManager` and server TLS using a dynamic `GetCertificate` callback (so certs can be swapped without restarting the listener).
* Listen for **SIGHUP** and reload the cert/key files from the disk in place when reviced - this matches the signal common ACME renewal tooling already sends.
* If the **initial** certificate load fails, log a warning and **fall back to plain HTTP** instead of refusing to start.
* When HTTPS is active, also run a small HTTP listener on `:80` whose only job is to redirect everything to HTTPS.

## Consequences

**Good:**

* Certificate rotation doesn't require restarting Asena or dropping in-flight connection.
* Plays nicely with existing SIGHUP-based renewal workflows.
* A broken TLS config at boot doesn't take the whole proxy offline - it degrades to HTTP instead of falling to start.

**Costs:**

* That same fallback is a silent one: if TLS is misconfigured, Asena will serve **plain HTTP** on what supposed to be an HTTPS deployment, with only a log line warking the difference. Anyone not watching logs could end up running unencrypted traffic without realizing it - this is a real operational risk worth flagssing to anyone deploying Asena.
* The HTTP->HTTPS redirect listener's port (`:80`) is hardcoded, not configurable.

## Alternatives Considered

* **Fail fast on bad TLS config at startup.** Rejected (so far) - the project prioritized "the proxy stays up, even degraded" over "refuse to run with a misconfiguration." This trade-off is debatable and could be revisited - e.g. making the fallback-to-HTTP behavior configurable, or at minimum raising it from a `Warn` log to something louder.
* **Built-in ACME/autocert support.** Not decided - a real option for a future ADR if manual cert management becomes a paint point.

## Related Code Location

`internal/server/`