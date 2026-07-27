---
targets: [code]
include_source: false
granularity: file
description: '-'
---
Exhaustive switches fail loudly

- When switching over a bounded set of values, the `default` case must panic.
- Prefer explicit loud failure (`default: panic("unknown: " + t)`) over silent fall-through, which allows silent data corruption.
