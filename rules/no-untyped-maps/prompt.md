---
targets: [code]
granularity: file
---

A known shape deserves a type; an untyped map hides what the code actually depends on.

- Model keys you access repeatedly as struct fields.
- Reserve `map[string]T` for genuinely dynamic key lookups.
