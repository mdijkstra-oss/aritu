---
targets: [code]
granularity: file
priority: med
---

A comment is an unchecked claim: nothing fails when it lies, so it drifts. The
default is none. One earns its place only where a rewriter who never read it
would get the code wrong; anything else belongs in a name or in the structure,
which cannot drift because the compiler reads them too.

- Refactor code whose comment explains *what* or *how* until the comment is no
  longer needed for clarity.
- Move a why about order or dependency into the structure, so the wrong version
  cannot compile.
- Delete a comment that argues the design is right — the test: a rewriter who
  never read it would still write correct code. One they would break without is
  a constraint and stays.
- Delete a comment whose opening names the declaration and then says what the
  signature already says — `parseConfig parses the config file`.
- Delete code that has been commented out — example code inside a doc comment is
  documentation, not a leftover.
- Do not restate adjacent names, signatures, types, or metadata.
- Do not describe distant, nonlocal code — it won't get updated when that code
  changes.
- Do not state facts that time makes false: counts, versions, the current
  behavior of code elsewhere.
- Keep a comment that carries an outside constraint the code cannot encode — an
  upstream quirk, a protocol, a wire format, a legal or business reason. Length
  is never the fault: a three-sentence quirk stays where a one-line rationale
  goes.
- Allow the doc comment an ecosystem mandates on an exported symbol (godoc,
  JSDoc, docstrings), but not one that only restates the signature — the
  convention asks for a line, not for that line to say nothing.
- Allow legal headers, warnings about consequences, and TODO comments.
- Judge the file as a whole: a file where more than a third of the declarations
  carry a rationale comment fails even if every comment reads as a why.
