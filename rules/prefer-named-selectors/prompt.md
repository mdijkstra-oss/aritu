---
targets: [code]
granularity: file
priority: med
---

A derived value computed in more than one place deserves a name — a selector: a pure function, data in, derived data out.

- Extract a named selector when the same derivation appears more than once; a one-off inline derivation is fine where it stands.
- Give a selector a name that states what it returns (`activeUsers`, `visibleTodos`); use a `get*`, `find*`, or `filter*` prefix where the language's idiom expects one.
