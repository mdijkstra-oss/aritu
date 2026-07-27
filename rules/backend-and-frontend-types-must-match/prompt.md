---
targets: [code]
include_source: false
granularity: file
description: '-'
---
Backend and frontend types must match

- What the server sends is exactly what the client expects. Generate both sides from shared schemas.
- Do not use `any` or `interface{}` to bridge the two; they defeat the shared contract.
