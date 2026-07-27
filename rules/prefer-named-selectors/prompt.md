---
targets: [code]
granularity: file
---

Derive nested data through named selectors instead of inline constructions — a selector is a pure function: data in, derived data out.

- Name selectors with a `get*`, `find*`, or `filter*` prefix and reuse them.
- Do not scatter inline loops that repeatedly compute the same derived value — extract a selector.
