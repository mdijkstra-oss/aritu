---
targets: [code]
granularity: function
---

A switch over a bounded set either handles every case or fails loudly.

- Make the `default` case panic when switching over a bounded set of values and the language does not enforce that all cases are handled.
- Prefer explicit loud failure (`default: panic("unknown: " + t)`) over silent fall-through — silent fall-through allows silent data corruption.
