---
targets: [code]
granularity: file
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
- Allow the doc comment an ecosystem mandates on an exported symbol (godoc,
  JSDoc, docstrings), even where it only gives the signature back. Judge
  whatever it says beyond that by the test above. Length is never the fault:
  a paragraph of foreign fact stays where a single restating line on an
  unexported name goes.
- Allow legal headers, warnings about consequences, and TODO comments.
