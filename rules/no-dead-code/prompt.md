---
targets: [code]
granularity: file
---

Dead code misleads the reader; version control already preserves history.

- Delete all code that has been commented out.
- Delete unreachable code: statements after an unconditional return/panic/throw, and branches whose condition can never be true.
- Delete code kept "in case we need it later" — unused parameters nothing reads, feature flags nothing sets, half-wired escape hatches.
