---
targets: [code]
include_source: false
granularity: file
description: '-'
---
Do not duplicate code

- Write a given piece of logic once and reuse it.
- If two types overlap in meaning, unify them into one.
- Before writing new code, search the codebase for existing similar patterns. Report the result explicitly, in one of these two forms: "Found X in Y, can be reused/refactored" or "Searched for X, nothing suitable exists."
- New logic must include tests. The only exception is thin wrappers, which are covered by testing the underlying implementation they wrap.
