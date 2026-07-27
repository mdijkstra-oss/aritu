---
targets: [code]
granularity: file
---

A file has one job. Everything in it — types, helpers, constants — exists to serve that job.

- Split any file that mixes unrelated responsibilities (say parsing plus HTTP handling plus persistence).
- Do not group by theme — "utils", "helpers" and "misc" grab-bag files are the violation, not an exception.
