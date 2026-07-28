---
targets: [code]
granularity: file
priority: high
---

Dead code misleads the reader; version control already preserves history.

- Delete unreachable code: statements after an unconditional return/panic/throw, and branches whose condition can never be true.
