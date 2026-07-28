---
targets: [code]
granularity: function
priority: high
---

A function does one kind of thing, and its signature tells the whole story.

- Keep each function to one kind of thing. Orchestration is one kind: calling steps in sequence and wiring outputs to inputs — but orchestrating and computing in the same function is the violation.
- Allow a single-expression computation inline in an orchestrator — extracting a one-liner buys no clarity.
- Fail a function of 5 or more arguments, unless every argument is stored on the constructed object or the signature mirrors a named external API.
- Wrap two or more parameters that only ever travel together in this file into an object (`Point center`, not `double x, double y`) — unless an interface or callback contract fixes the signature.
- Do not hide side effects — a `checkPassword()` that also initializes a session is lying about what it does; transparent caching or lazy initialization is not hiding.
