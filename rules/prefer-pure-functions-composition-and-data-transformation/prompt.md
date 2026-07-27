---
targets: [code]
include_source: false
granularity: file
description: '-'
---
Prefer pure functions, composition, and data transformation

- Each function has a single purpose. Prefer standalone functions over methods, and composition over inheritance.
- Structure logic as: input data → transformation → output data. Use map, filter, and reduce.
- Keep data and behavior separate. Pass dependencies in explicitly. Do not rely on hidden or global state.
- Compose small functions into larger ones. If a function is complex, decompose it.
- Multiple boolean flags multiply code paths; prefer separate functions or a dispatch table (`handlers[type]()`) over branching on a type value (`if type == "A"`).
- Perform IO at the boundaries of the system. Keep the core logic pure.
- When writing new code: if the logic is correct for a generic type parameter rather than a concrete type, write it generic. The library provides the engine; the domain provides the configuration.
- When reviewing existing code: if replacing a concrete type with a type parameter changes nothing, the code is generic plumbing miscategorized as domain logic. Extract and parameterize it.
