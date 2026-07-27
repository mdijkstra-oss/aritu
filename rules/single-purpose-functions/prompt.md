---
targets: [code]
granularity: function
---

A function does one kind of thing, and its signature tells the whole story.

- Split any function that takes a boolean flag argument — the flag means it does two things.
- Keep each function to one kind of thing. Orchestration is one kind: calling steps in sequence and wiring outputs to inputs — but orchestrating and computing in the same function is the violation.
- Treat 4 or more arguments as a sign the function does too much.
- Wrap groups of related arguments into an object (`Point center`, not `double x, double y`).
- Do not hide side effects — a `checkPassword()` that also initializes a session is lying about what it does.
