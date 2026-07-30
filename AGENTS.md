# Repo notes

## Rules
Run `make rulebook` before doing everything else. These rules you must follow and are enforced.

## Running aritu while changing aritu

No one cooks in the kitchen they are still building.

`selftest` is the only run allowed. It judges the fixtures under `rules/` and
`prompts/`, which is what a change to a rule or a prompt has to be measured
against, and it touches no file you are editing.

`apply` is not, and neither is the `aritu-review` skill that drives it. Judging
this tree with a binary you are halfway through rewriting reports on a moving
target, and its findings pull the work towards cleaning the codebase and away
from the change you were asked to make.

A change to the Go code is verified by `go build ./...` and `go test ./...`.
`make rulebook` is fine at any time: it calls no model.

## Features

Feature specs live in `docs/features/`. Each feature is its own folder named
`<nn>-<type>-<slug>` — e.g. `09-feat-dark-mode-toggle`, `10-ref-split-render-engine`,
`11-fix-token-overflow` — containing a `spec.md` (and later a `plan.md`).

`<nn>` is the next unused number, so the directory lists in the order the specs
were written. It is an ordinal, not a priority or a dependency: a spec keeps its
number once it has one, and nothing is renumbered when one is abandoned.
