---
include_source: false
---
A test must have one reason to fail: it pins down a single behaviour, however many
assertions it takes to do so. When such a test goes red, the failing name and the
failing line identify one thing that changed. When a test guards several behaviours,
a failure says only that something in a bundle broke, and the first failed assertion
hides every later one.

A test satisfies this rule when every assertion in it serves one claim about the code
under test. The number of assertions is not the subject. Three assertions checking the
three fields of one returned value are one behaviour. One assertion checking a boolean
that summarises four unrelated outcomes is still several.

The test to picture: if this behaviour were removed from the production code, this test
fails, and if any other behaviour were removed, it does not.

SATISFIES the rule:

- A single act followed by one or more assertions that all describe the result of that
  act — its return value, its error, the fields of the value it produced, the state it
  changed.
- Arrange steps, fixture construction, helper calls and cleanup before the act. Setup
  is not a second behaviour, and neither is a guard assertion that setup succeeded
  before the real act runs.
- A table-driven test whose cases are inputs to one behaviour: many rows, one claim,
  each row a different input to the same question. `TestReturnsZeroWhenDiscountExceedsTotal`
  with six rows of totals and discounts is one reason to fail.
- Subtests under `t.Run` that vary the input to one behaviour, rather than moving on to
  a different one.
- A test that asserts both a value and its error alongside it, when the behaviour is
  "returns this value and no error" — that is one outcome described in two assertions.
- Several assertions walking one composite result: a parsed struct, a rendered string
  checked for several required parts, a collection checked for length and contents.

DISQUALIFIES the rule:

- **Two behaviours in one function.** The valid case and the rejected case together, the
  happy path and the error path together, create-then-delete, encode-then-decode. Each
  is its own reason to fail and belongs in its own test.
- **An act-assert chain.** Act, assert, act again, assert again, act a third time. Every
  act after the first is a new behaviour wearing the previous test's name, and an early
  failure stops the run before the later ones are ever exercised.
- **A tour of an API.** A test that exercises several functions or methods of a type in
  sequence to show the type works, rather than pinning down one thing the type does.
- **Cases in a table that answer different questions.** A table is one behaviour with
  many inputs. When some rows assert a returned value and others assert an error
  classification, or the case struct grows a flag that switches which assertions run,
  the table has become several tests sharing a loop.
- **Subtests that are separate tests.** `t.Run("parses", ...)` and `t.Run("rejects", ...)`
  in one function are two behaviours nested under one name; the nesting does not merge
  them.
- **Assertions about incidental state.** A test for one behaviour that also asserts an
  unrelated invariant — a log line, a counter, an untouched field — takes on a second
  reason to fail.

Go tests come in many shapes: table-driven with a slice of cases, subtests via `t.Run`,
plain linear arrange-act-assert, golden-file comparisons, functional tests that drive a
real system end to end. Shape carries no penalty and no credit. A table-driven test is
not a violation of anything, a long test is not a violation, and a test with a single
assertion is not automatically a pass — a lone assertion on a summary boolean can still
be guarding a bundle. Judge how many distinct behaviours would each, on their own, turn
this test red. One is a pass. More than one is a fail.
