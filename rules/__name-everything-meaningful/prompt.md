---
targets: [code]
include_source: false
granularity: file
description: '-'
---
Name everything meaningful

- Extract and name non-trivial inline functions. Anonymous callbacks are acceptable only for trivial transforms (for example, `.map(x => x.id)`).
- Give a name to any condition that requires interpretation, using an `is*`, `has*`, `can*`, or `should*` prefix. For example, replace `if strings.HasPrefix(s, "<") && len(s) > 3` with `if isOpeningTag(s)`.
