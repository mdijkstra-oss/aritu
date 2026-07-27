---
targets: [code]
include_source: false
granularity: file
description: '-'
---
Errors are values

- Return, transform, and compose errors as data.
- Panic only for programmer errors: broken invariants or impossible states.
- Use `Must*` for bootstrap-time failures that should abort startup. Use `Should*` for tolerable failures that callers can handle.
