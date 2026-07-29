# TODO

- Everything an agent learns about this repository belongs here or in the rules
  directory, not in its own memory. A memory travels with one person: it fixes
  the tool for whoever wrote it and leaves every other contributor, and every
  fresh checkout, running the unfixed version — and the fix looks like it works,
  which is worse than it plainly not working. A misfiring rule, a prompt that
  needs rewording, a hole in the auto-fix skill: all of those are the
  repository's, and they go in this file until somebody lands them. Reserve
  memory for how one person likes to be worked with.

- Enumeration is a model call. Every granularity below `file` asks the splitter
  to list a file's units, which costs a call per file per granularity and can
  disagree with itself between runs. A static parse per language would be
  cheaper and deterministic — the unit kinds are all syntax: declarations,
  functions, comments, test cases. The splitter prompts are the specification
  to port, and the cache in `internal/domain/audit/audit.go` already keys on file
  plus granularity, so a parser could fill the same seam. Prompts are the
  stopgap, not the destination.

- A rule can ask a question its granularity cannot answer. The linter renders
  every unit of a file in one call, so a criterion that depends on the whole
  file gets resolved once and stamped across each unit — the verdicts correlate
  instead of standing alone. Three rules currently do this:
  - `no-explanatory-comments`: the doc-comment allowance turns on whether the
    module publishes anything, which is a property of the file. Measured across
    six files: the restating comments never split, they pass or fail as a bloc.
  - `single-purpose-functions`: "parameters that only ever travel together in
    this file" reaches past the function being judged.
  - `intent-revealing-names`: "flag a noise suffix only when both look-alike
    names appear in this file" does the same.
  Either move the file-scoped half out of the criterion and hand it to the
  linter as context, or accept that these clauses are file-scoped and drop them
  from a per-unit rule.

- `no-duplication` and `prefer-named-selectors` judge a relation between two
  sites, so `file` is the finest granularity that can hold both. Neither sees
  duplication across files, which is where most of it lives.

- Generic helpers are sitting in domain packages, and the imports they force
  say things that are not true. `internal/domain/audit` imports
  `internal/domain/selftest` for one function — `FormatDuration`, a duration in
  and a string out, which knows nothing about fixtures — so the import reads as
  the audit run depending on the selftest runner. `plural` is written out twice,
  byte for byte, in `audit/render.go` and `lint/format.go`. `singleLine` in
  `selftest/table.go` is exactly the first step of `snippetOf` in `lib/service`,
  which flattens the same way before truncating. None of the three is reachable
  by a rule that judges one file at a time, which is the entry above seen from
  the other side: the duplication that costs most is the duplication no run will
  ever be pointed at.

  The home already exists — `internal/lib` is where this repository keeps what
  is about nothing in particular, and a helper does not become domain code by
  having been written for a domain. What makes this a decision rather than a
  sweep is that moving them asserts a shape: that formatting a duration, saying
  how many, and flattening whitespace are library concerns rather than three
  packages' private business. Worth settling once, deliberately, rather than
  letting an auto-fix run relocate code on a judgement nobody made.

- `no-untyped-maps` reads a `map[string]T` as a known shape without checking
  whether the file ever writes a literal key. It fired three times in one sweep
  on maps that cannot be anything else: `lint.Report.Verdicts`, keyed by unit
  names the splitter discovers at runtime; `schemaNode.Properties`, which
  carries one key per judged unit; and the `map[string]any` the OpenAI SDK's
  `ResponseFormatTextConfigParamOfJSONSchema` takes. Every failing vote argued
  from what "looks like a known set of keys" while the file subscripts nothing.
  Two fixtures are missing: a map initialised empty and filled from names read
  at runtime, and one whose type an external signature fixes. The criterion to
  write is the keys the file actually accesses — a map it never subscripts with
  a literal passes.

- The auto-fix skill does not say how to pick the next file. Given a broad
  scope it will hand the whole list to one `apply` run, which spends a long
  wall-clock on a report too wide to act on and buries the ordering the run
  is supposed to follow. What it should say: walk the scope one package
  directory at a time — `internal/domain/audit`, then `internal/domain/config`,
  through `internal/lib/glob`, `internal/lib/kind` and the rest — judging and
  fixing a directory to green before opening the next. A package is the unit
  because a severe finding relocates code within one.

- Nothing checks reachability now. `no-dead-code` was removed for asking a
  model to decide what a parser decides for free: it cost a vote per
  declaration per file and got it wrong in the direction that matters — judging
  `cmd/aritu/targets.go` it called the last statement of `applyOptions`
  unreachable behind an "unconditional `return opts, err`" that is the body of
  an `if err != nil`, three votes to zero, twice, one of them writing "wait,
  `if err != nil` is conditional" before failing it anyway. What is still
  missing is the tool that does settle it — `go vet` and `staticcheck` here —
  wired into the same workflow that runs `apply`. The general form: a criterion
  a parser can decide belongs to the parser, and aritu keeps the ones that need
  a reader.
