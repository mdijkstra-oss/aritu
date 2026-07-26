# Repo notes

## Features

Feature specs live in `docs/features/`. Each feature is its own folder named
`<type>-<slug>` — e.g. `feat-dark-mode-toggle`, `ref-split-render-engine`,
`fix-token-overflow` — containing a `spec.md` (and later a `plan.md`).

Scaffold a new one with the `/md-feature` skill (it creates the folder and an
empty spec template; you fill in the spec). Then pressure-test it with the
`/md-spar` skill — an evidence-backed critique loop that sharpens the spec
through back-and-forth. Once the spec holds up, author the architecture with
`/md-plan` (prose + Mermaid data-flow graphs, component contracts, test tables),
and spar that too — the plan lens attacks it with independent adversaries.
Neither `/md-spar` nor `/md-plan` builds.

Flow: `/md-scaffold` (once per repo) → `/md-feature` → `/md-spar` (spec) →
`/md-plan` → `/md-spar` (plan) → build.

One worktree per feature: the spec and the code that satisfies it share a
branch and land together in the same PR.

## Learnings

`docs/learnings/` holds markdown notes worth keeping from building features —
gotchas, decisions, and patterns discovered along the way. One note per file.
