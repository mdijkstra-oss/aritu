---
targets: [code]
include_source: false
granularity: file
description: '-'
---
Delete dead code

- Remove unused exports, unreachable functions, and orphaned files.
- Do not retain code "in case we need it later"; version control preserves history.
- Before adding new code, search for and remove dead code in the area you are touching.
