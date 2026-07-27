---
targets: [code]
include_source: false
granularity: file
---

- If you access the same keys repeatedly, model them as struct fields.
- Parse JSON/YAML etc at boundaries; use structs internally.
- Reserve `map[string]T` for genuinely dynamic key lookups.
