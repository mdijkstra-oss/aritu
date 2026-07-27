---
targets: [code]
granularity: file
---

A file reads top-down: public contract first, private detail after.

- Order a file as: imports, then exported types, then exported constants, then exported functions.
- Place internal (non-exported) declarations after the exports, grouped by the exported code they support.
