# TODO

- `intent-revealing-names` runs at function granularity, so file-scope names — exported
  constants, type names, top-level vars — fall outside every judged unit and its
  "searchable constants" bullet is partially unenforced. Two ways to close it:
  - Add a small file-granularity companion rule for top-level names.
  - Preferred: let `granularity` take a list (e.g. `[function, top_level]`) so one
    rule enumerates and judges both unit kinds. Needs a new `top_level` granularity
    (top-level declarations: constants, types, vars) plus frontmatter accepting a
    list — touches ParsePrompt/ParseGranularity in `internal/domain/rule/rule.go`
    and NeedsEnumeration/UnitsAt/BuildNamesPrompt in `internal/domain/lint/lint.go`.
