# Architecture Decision Records (ADRs)

This folder explains **why** Asena is built the way it is. Not what the code does - that's in the code. This folder is about the *reasons* behind the design.

## What is an ADR?

ADR means "Architecture Decision Record." It is one short for one important decision. For example: "why we use Zap for logging" or "why we split the config into two files."

## Why do we use them?

So people don't have to guess. Without this folder, a new contributor would have to read all the code and guess why things were done a certain way. With this folder, they can just read one short file.

Some decisions were already made in the code before this folder existed. Those first ADRs are marked **"Backfilled"**. This means: the decision came first, the ADR was written later, not after.

From now on, try to write the ADR *before* or *while* you build the feature, not after.

## How to add a new ADR

Before you start, please read [`CONTRIBUTING.md`](../../CONTRIBUTING.md) for the general rules of this project.

1. Copy `template.md` in this folder.
2. Give the new file the next number and a short name. Example: `0008-add-rate-limiting.md`
3. Fill in the sections (see below).
4. Add a row for it in the **Index** table below.
5. Open a pull request. If it's a big decision, it's fine to discuss it in an issue first, before writing the full ADR.

## Suggesting a feature or a decision

Do you want to suggest a new feature, or a change to how something works?
You don't need to write a full ADR yourself. Just open an issue and describe:

* What problem you want to solve
* Why the current behaviour doesn't solve it
* Any idea you have for a solution (this part is optional)

If the idea is accepted, and ADR will be written for it (by you or by a maintainer) once the decision is made.

## The section is each ADR

* **Status** — Accepted, Proposed, Backfilled, or Superseded (replaced by
    a newer ADR)
* **Context** — What problem were we solving? What made this hard or
    important?
* **Decision** — What did we choose to do? Keep it short and direct.
* **Consequences** — What do we gain? What does it cost us? Every decision
    has trade-offs — write both sides, not just the good part.
* **Alternatives Considered** — What other options did we think about, and
    why did we not pick them?
* **Related Code Location** *(optional)* — The package or folder where
    this decision lives (e.g. `internal/config/`). Point to a package, not a
    specific file or line number — file paths change often, and a stale
    pointer is worse than none. See ADR-0001 for an example.

Use plain, simple sentences. Many readers of this project are not native English speakers. Short sentences are easier for everyone.

## Index

| # | Title | Status |
| --- | ----- | ------ |
| [0001](0001_two_tier_yaml_configuration.md) | Two-tier YAML configuration (static + dynamic) | Backfilled |
| [0002](0002_structured_logging_zap_lumberjack.md) | Structured logging with Zap + Lumberjack | Backfilled |
| [0003](0003_stdlib_flag_for_cli.md) | Standard library `flag` package for CLI args | Backfilled |
| [0004](0004_pluggable_load_balancer.md) | Pluggable load balancer, round-robin default | Backfilled |
| [0005](0005_dynamic_config_reload_mechanics.md) | fsnotify-based dynamic config reload | Backfilled |
| [0006](0006_tls_hot_reload_and_fallback.md) | TLS hot-reload and fallback to HTTP | Backfilled |
| [0007](0007_atomic_proxy_router_hot_swap.md) | Atomic proxy and router hot-swap | Accepted |
| [0008](0008_ast_rule_engine.md)| AST-based rule engine for router matching | Accepted |

## When should I write a new ADR?

Write one when a decision would be hard or confusing to figure out just by reading the code. For example:

* Adding a new dependency
* Choosing what happens when something fails (a fallback)
* Picking a file format, protocol, or algorithm
* Any time someone might reasonably ask "why didn't we just do X instead?"

You do **not** need and ADR for small , everyday coding choices. Save it for decisions that shape the project.