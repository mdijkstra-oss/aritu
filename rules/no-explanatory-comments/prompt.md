---
targets: [code]
granularity: comment
priority: med
---

A comment is an unchecked claim: nothing fails when it lies, so it drifts. The
default is none.

A comment describing code goes — this file's code, another file's, or code in
general. Wording does not save it: any account of what the code does becomes a
why by prefixing "because". Where the description is what makes the file
followable, the fix is the name or the structure, never a sentence beside it.

What stays carries a fact from outside the program, which no name or structure
can hold:

- A rule of the domain — a tax bracket, a rounding convention, a regulation.
- A standard the code has to meet — a protocol, a wire format, an encoding.
- A defect in something you do not control, and what it forces here.
- A pointer to an external record: a ticket, an RFC, a decision log.

Keep one only where a rewriter who never read it would get this code wrong.

These are not explanations, and are judged apart from the test above:

- Delete code that has been commented out — example code inside a doc comment is
  documentation, not a leftover.
- Hold the doc comment an ecosystem mandates (godoc, JSDoc, docstrings) to one
  line, and judge every line past the first by the test above.
- Fail a doc comment that hands the declaration back — `Config holds the
  configuration`, `Load loads a config file`, `New returns a new Client`. A
  restated name and type say nothing the declaration below has not, and the
  mandate to write one is not a reason to write that one.
- Fail a doc comment carrying what the name could carry — a unit, a direction,
  which of two readings a number takes. A comment a rename would delete is a
  report that the name is wrong, so rename.
- Keep the line that fixes a contract a caller could otherwise break and no name
  can hold: what the zero value means, whether it is safe to call concurrently,
  which named errors come back and when, what the caller must not retain or
  modify, what has to run before or after it, or a deprecation marker.
- Allow legal headers, warnings about consequences, and TODO comments.
