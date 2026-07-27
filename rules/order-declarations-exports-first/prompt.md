---
targets: [code]
include_source: false
granularity: file
---

- Order within a file: imports, then exported types, then exported constants, then exported functions.
- Place internal (non-exported) declarations after the exports, grouped by the exported code they support.
