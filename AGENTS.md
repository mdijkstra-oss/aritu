# Repo notes

## Rules
Run `make rulebook` before doing everything else. These rules you must follow and are enforced.

## Running aritu while changing aritu

The line is drawn at one call, not at the command. In this tree, `apply` cannot
ask the model for a verdict: judging this tree with a binary you are halfway
through rewriting reports on a moving target, and its findings pull the work
towards cleaning the codebase and away from the change you were asked to make.

Everything else runs. `apply` sweeps, loads rules, enumerates units against the
model and fills its caches, then refuses at the verdict call and reports why,
per file. That is deliberate: the machinery under this rewrite only runs over a
real tree, and running it is how you find out whether it works.

`selftest` is untouched and judges as normal. Its fixtures under `rules/` and
`prompts/` are what a change to a rule or a prompt has to be measured against,
and they are not the files you are editing.

`--debug` is untouched too: it fabricates every reply locally, so it prints both
prompts here and calls nothing.

The `aritu-review` skill is still not a run to make here: it exists to act on
findings, and it will collect refusals.

The one thing this costs: `apply` pointed at a fixture under `rules/` is refused
along with everything else, because the refusal is per call and not per file.
Judging a fixture is what `selftest` is for.

A change to the Go code is verified by `go build ./...` and `go test ./...`.
`make rulebook` is fine at any time: it calls no model.

## Features

Feature specs live in `docs/features/`. Each feature is its own folder named
`<nn>-<type>-<slug>` — e.g. `09-feat-dark-mode-toggle`, `10-ref-split-render-engine`,
`11-fix-token-overflow` — containing a `spec.md` (and later a `plan.md`).

`<nn>` is the next unused number, so the directory lists in the order the specs
were written. It is an ordinal, not a priority or a dependency: a spec keeps its
number once it has one, and nothing is renumbered when one is abandoned.
