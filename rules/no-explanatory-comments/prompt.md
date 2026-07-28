---
targets: [code]
granularity: file
---

A comment is an unchecked claim: nothing fails when it lies, so it drifts. Every
fact belongs in the strongest representation that can hold it — a name, the
structure, a test — and a comment is the last resort for what none of them can
carry.

- Refactor code whose comment explains *what* or *how* until the comment is no
  longer needed for clarity.
- Move a why about behavior into a test — a comment describes an intent; a test
  defends it.
- Move a why about order or dependency into the structure, so the wrong version
  cannot compile.
- Delete a comment that justifies the design — an argument for the reviewer is
  not information for the maintainer.
- Do not restate adjacent names, signatures, or metadata.
- Do not describe distant, nonlocal code — it won't get updated when that code
  changes.
- Do not state facts that time makes false: counts, versions, the current
  behavior of code elsewhere.
- Keep a comment that carries an outside constraint the code cannot encode — an
  upstream quirk, a protocol, a legal or business reason.
- Allow legal headers, warnings about consequences, and TODO comments.
- Judge the file as a whole: a file where most declarations carry rationale
  fails even if every comment reads as a why.
