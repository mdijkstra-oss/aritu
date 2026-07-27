---
targets: [code]
include_source: false
granularity: file
description: '-'
---
Never mutate inputs

- A function receives data but does not own it. To change input data, copy it first.
- Do not modify the caller's memory. Reducers return new state rather than mutating existing state.
