---
targets: [code]
granularity: file
priority: med
---

A comment is an unchecked claim: nothing fails when it lies, so it drifts. Every
fact belongs in the strongest representation that can hold it — a name, the
structure — and a comment is the last resort for what neither can carry.

- Refactor code whose comment explains *what* or *how* until the comment is no
  longer needed for clarity.
- Move a why about order or dependency into the structure, so the wrong version
  cannot compile.
- Delete a comment that argues the design is right — the test: a rewriter who
  never read it would still write correct code. One they would break without is
  a constraint and stays.
- Delete code that has been commented out — example code inside a doc comment is
  documentation, not a leftover.
- Do not restate adjacent names, signatures, or metadata.
- Do not describe distant, nonlocal code — it won't get updated when that code
  changes.
- Do not state facts that time makes false: counts, versions, the current
  behavior of code elsewhere.
- Keep a comment that carries an outside constraint the code cannot encode — an
  upstream quirk, a protocol, a legal or business reason.
- Allow the doc comment an ecosystem mandates on an exported symbol (godoc,
  JSDoc, docstrings), even when it restates the signature.
- Allow legal headers, warnings about consequences, and TODO comments.
- Judge the file as a whole: a file where more than half the declarations
  carry a rationale comment fails even if every comment reads as a why.
