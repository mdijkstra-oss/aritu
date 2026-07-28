# TODO

- Enumeration is a model call. Every granularity below `file` asks the splitter
  to list a file's units, which costs a call per file per granularity and can
  disagree with itself between runs. A static parse per language would be
  cheaper and deterministic — the unit kinds are all syntax: declarations,
  functions, comments, test cases. The splitter prompts are the specification
  to port, and the cache in `internal/domain/run/run.go` already keys on file
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
