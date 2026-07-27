---
targets: [code]
granularity: file
---

Everything a function depends on is visible at its call site; nothing flows in from the side.

- Pass dependencies in explicitly — no global state.
- Perform IO at the boundaries; keep the core pure.
