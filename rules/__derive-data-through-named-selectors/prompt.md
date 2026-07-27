---
targets: [code]
include_source: false
granularity: file
description: '-'
---
Derive data through named selectors

- A selector is a pure function: data in, derived data out.
- Name selectors with `Get*`, `Find*`, or `Filter*` prefixes and reuse them.
- Do not scatter inline loops that repeatedly compute the same derived value; extract a selector.
