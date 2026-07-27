---
targets: [code]
include_source: false
granularity: file
description: '-'
---
Dependencies flow in one direction

- The library layer imports nothing above it. The domain layer imports the library. The application layer (UI, routes) imports the domain and the library.
- The folder structure must mirror this dependency graph.
