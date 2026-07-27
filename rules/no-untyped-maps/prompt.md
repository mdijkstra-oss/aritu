---
targets: [code]
granularity: file
---

A known shape deserves a type; an untyped map hides what the code actually depends on.

- Model keys you access repeatedly as struct fields.
- Parse JSON/YAML etc. at the boundaries; use structs internally.
- Reserve `map[string]T` for genuinely dynamic key lookups.
