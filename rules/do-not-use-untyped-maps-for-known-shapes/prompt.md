---
targets: [code]
include_source: false
granularity: file
description: '-'
---
Do not use untyped maps for known shapes

- If you access the same keys repeatedly, model them as struct fields.
- Parse JSON at boundaries; use structs internally.
- Reserve `map[string]T` for genuinely dynamic key lookups.
