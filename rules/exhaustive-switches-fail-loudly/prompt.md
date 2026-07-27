---
targets: [code]
include_source: false
granularity: file
---

- When switching over a bounded set of values, the `default` case must panic when the langauge does not enforce all cases to be handled.
- Prefer explicit loud failure (`default: panic("unknown: " + t)`) over silent fall-through, which allows silent data corruption.
