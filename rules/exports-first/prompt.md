---
targets: [code]
granularity: file
---

A file reads top-down: public contract first, private detail after.

- Order a file as: imports, then exported types, then exported constants, then exported functions.
- Keep a type together with its methods and the constants tied to it — that grouping wins over the strict order, even when it interleaves exported and unexported names.
- Place internal (non-exported) declarations after the exports, grouped by the exported code they support; a helper several exports share goes at the end.
- Allow an internal declaration ahead of an export when the language's evaluation order forces it there.
