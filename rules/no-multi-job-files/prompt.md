---
targets: [code]
granularity: file
priority: severe
---

A file has one job. Everything in it — types, helpers, constants — exists to serve that job.

- Split any file that mixes unrelated responsibilities (say parsing plus HTTP handling plus persistence).
- Judge the job at the altitude of one type, one flow, or one entry point — a type with all its operations, or a `main` that parses flags and wires the program together, is one job.
- Confine an entry point to grammar, dispatch and construction — a formatter, a fake or a resolution pass it calls is a job of its own, and being its only caller does not make it wiring.
- Take the entry point out and judge what is left standing: serving one feature is not what makes one job.
- Do not group by theme — "utils", "helpers" and "misc" grab-bag files are the violation, not an exception.
- Allow a helper file scoped to one narrow domain (string manipulation, date math) — the violation is a file with no domain, not the word "util" in its name.
