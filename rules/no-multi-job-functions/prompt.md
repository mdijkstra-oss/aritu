---
targets: [code]
include_source: false
granularity: file
---

- Boolean flag arguments are a smell — split the function.
- Each function does one kind of thing.
- Orchestration is one kind: a function may call steps in sequence and wire outputs to inputs — but orchestrating and computing in the same function is the violation.
- 