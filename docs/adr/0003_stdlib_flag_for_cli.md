## ADR-0003: Standard library `flag` package for CLI args

* **Status:** Accepted (backfilled - this ADR was written after the code already existed)

## Context

Asena needs to accept a small number of startup flags: `-http-port`, `-https-port`, `-cert-file`, `-key-file`. Common Go options for this range from the standard library's `flag` package up to full CLI frameworks like [Cobra](https://github.com/spf13/cobra) or [urfave/cli](https://github.com/urfave/cli), which add subcommands, help generation, shell completion, etc.

## Decision

Use the standard library `flag` package, wrapped in a small `pkg/cli.Parse()` helper that returns a plain `Options` structure.

## Consequences

**Good:**

* Zero extra dependencies for something this small.
* The whole CLI surface is \~20 lines and easy to read end to end.

**Costs:**

* No subcommends (e.g. a future `asena config validate` or `asena version`) without restructuring - \`flag is built around a single flat set of flags.
* No built-in shell completion or automatic grouped help output the way Cobra/urfave provide.
* If asena's CLI surface grows significantly, this will likely need to be replaced - that migration is straightforward but not free.

## Alternatives Considered

* **Cobra.** Rejected for now - brings subcommand routing, generated docs, and completion scripts that are overkill for four flags, plus added dependency.
* **urfave/cli.** Samereasoning as Cobra - more capability than currently needed.

## Related Code Location

`pkg/cli/`