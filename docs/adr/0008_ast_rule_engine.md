## ADR-0008: AST-based rule engine for router matching

* **Status:** Accepted

## Context

Before this change, a router rule like ``Host(`localhost`)`` was matched using simple string search. The code looked for the first `(` in the rule text and used a small map with only one entry: `Host`. This worked, but only for one very simple case.

The roadmap asks for more matchers: `PathPrefix`, `Method`, and `Header`. It also asks for combining them with `&&` (AND), `||` (OR), and `!` (NOT).

This is a problem for simple string search. Once you allow `&&`, `||`, and parenthesis, the same characters can mean different things depending on how they are grouped. For example, `(A || B) && C` and `A || (B && C)` are not the same rule, even though they use the same words. A string search has no way to know which grouping the user meant, it just reads left to right. So we could not just add more `if` checks on top of the old code. We needed a different way to read and understand a rule.

## Decision

We now read a rule in three steps:

1. **Lex** - Turn the rule text into a list of small pieces (tokens). Example: ``Host(`example.com`) && Method(`GET`)`` becomes ``[Host(`example.com`), &&, Method(`GET`)]``.
2. **Parser** - Turn that list of tokens into a tree. This tree is called AST (Abstract Syntax Tree). The tree remembers grouping and order correctly, so `&&` and `||` never get confused with each other.
3. **Match** - Walk th tree against each incoming request. Every part of the tree (`Host`, `Method`, `AND`, `OR`, `NOT`, ...) knows how to check itself against a request.

We also give every part of the tree a "specificity" score, a number that says how narrow or exact this rule is. `Header` checks are the most exact, so they get the highest score. A plain `Host` check is less exact, so it gets a lower score. When two rules could both match the same request, the rule with the higher score wins.

All of this reading and scoring happens **one time**, when `dynamic.yaml` is reloaded, not on every single request. After that, matching a real request is fast, we just walk the already-built tree.

If one router's rule is broken (for example, a type in the matcher name), we skip only that router. We write a warning in the logs. The rest of the routers still load and work normally.

## Consequences

**Good:**

* Rules can now use `&&`, `||`, `!`, and `()` correctly, with no confusion about grouping.
* Adding a new matcher later (for example `ClientIP`) only means adding one small new file. The lexer, the tree, and the matching code do not need to change.
* Reading and understanding a rule happens only once, at reload time. Answering real requests stays fast, because no text-parsing happens while serving traffic.
* When two rules could both match, the answer is now always the same, every time. Before, this could depend on random map order in Go, which is confusing and hard to debug.

**Cost:**
* This is more code than the old version. Instead of one map lookup, we now have a lexer, a tree, and a small parser.
* The "specificity" score is a rule of thumb, not a strict formula. In most cases it will do what you expect, but a very unusual rule could sort differently than a person expected. This is explained in code comments inside `internal/rule`.
* A broken rule is now caught when the config reloads, and a warning is printed. This is a good thing, but it is a change in behavior: before, a broken rule would just silently never match.

## Alternatives Considered

* **One big regular expression (regex).** Regex is good at matching text, but it cannot easily handle grouping with `&&`, `||`, and `()` the way a real grammar needs. We would hit the same problem as string search, just written differently.
* **Add `&&`/`||` on top of the old code, but check them left to right, with no grouping rules.** This was rejected because it has the exact bug described above, it cannot tell `A || B && C` from `(A || B) && C`. This would be a silent correctness bug waiting to happen.
* **Parse the rule again on every request.** This would be simpler to write, but wasteful, the rule text does not change between config reloads, only the request does. It is better to read the rule once and reuse the result many times, the same idea already used for proxies and routers in ADR-0007.

## Related Code Location
`internal/rule/`, `internal/proxy/`
