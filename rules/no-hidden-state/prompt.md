---
targets: [code]
include_source: false
granularity: file
---

- Pass dependencies in explicitly; no global state.
- Perform IO at the boundaries. Keep the core pure.