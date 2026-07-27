---
targets: [code]
granularity: file
---

Code states *what* it does. Comments are reserved for *why*, and are rarely needed.

- Refactor code whose comment explains *what* until the comment is no longer needed for clarity.
- Allow legal headers, warnings about consequences, and TODO comments.
- Do not let a comment go stale or mislead — a wrong comment is worse than no comment.
- Do not describe distant, nonlocal code in a comment — it won't get updated when that code changes.
