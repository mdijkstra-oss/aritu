# Repo notes

## Rules
Run `make rulebook` before doing everything else. These rules you must follow and are enforced.

## Features

Feature specs live in `docs/features/`. Each feature is its own folder named
`<nn>-<type>-<slug>` — e.g. `09-feat-dark-mode-toggle`, `10-ref-split-render-engine`,
`11-fix-token-overflow` — containing a `spec.md` (and later a `plan.md`).

`<nn>` is the next unused number, so the directory lists in the order the specs
were written. It is an ordinal, not a priority or a dependency: a spec keeps its
number once it has one, and nothing is renumbered when one is abandoned.
