---
targets: [tests]
include: [tests]
include_source: false
granularity: function
---
A test must have one reason to fail: it pins down a single behaviour, however many
assertions it takes to do so. When such a test goes red, the failing unit and the failing
line identify one thing that changed. When a test guards several behaviours, a failure
says only that something in a bundle broke, and the first failed assertion hides every
later one.

A test satisfies this rule when every assertion in it serves one claim about the code
under test. The number of assertions is not the subject. Three assertions checking the
three fields of one returned value are one behaviour. One assertion checking a boolean
that summarises four unrelated outcomes is still several.

You are judging the whole unit the framework runs, including every case and every named
subdivision inside it, because how many behaviours a loop pins down is a property of the
loop and not of any one row: asked of a single row the question is close to vacuous, since
one row is one input and answers whatever question its loop asks. Where a unit's name
carries a namespace path, the sibling units grouped beside it under that same path are
separate units and are not part of this one; judge only what this unit itself runs.

The question to ask: how many distinct behaviours would each, on its own, turn this unit
red? Count them. One is a pass, whatever else you think of the test. More than one is a
fail.

SATISFIES the rule:

- A single act followed by one or more assertions that all describe the result of that
  act — its return value, its error, the fields of the value it produced, the state it
  changed.
- Several assertions walking one composite result: a parsed structure checked field by
  field, a rendered string checked for several required parts, a collection checked for
  length and then for contents. One result inspected from several angles is one claim.
- Arrange steps, fixture construction, helper calls and cleanup before the act. Setup is
  not a second behaviour, and neither is a guard assertion that setup succeeded — a check
  that the fixture loaded, that the temporary directory was created, that the seeded
  record exists. A guard asserts a precondition the unit depends on: were it to fail, the
  unit could not run at all and nothing about the behaviour under test would have been
  learned. It is not an outcome the unit exists to prove, and it protects the diagnosis of
  the one behaviour that is. Asserting a record exists after seeding it, so that the
  deletion under test has something to delete, is a guard; asserting that creation worked
  and then that deletion worked is two outcomes, and belongs below.
- A value and its error asserted together, when the behaviour is "returns this value and
  no error", or "returns nothing and this error". That is one outcome described in two
  assertions — one of those two, held by the whole unit. A unit whose cases all expect a
  value and a unit whose cases all expect a rejection each hold one; a unit that mixes
  them is asking two questions, which is the shape below.
- A table of many inputs to one behaviour: many rows, one claim, each row a different
  input to the same question. Six rows feeding six different totals and discounts to one
  claim about the amount that comes back is one reason to fail. The same holds for named
  subdivisions inside the unit that vary the input to one behaviour rather than moving on
  to a different one.

DISQUALIFIES the rule:

- **Several unrelated behaviours in one function.** The valid case and the rejected case
  together, the happy path and the error path together, create-then-delete,
  encode-then-decode. A march through several functions or methods of a type in sequence
  to show the type works belongs here too: it demonstrates a surface rather than pinning
  down one thing that surface does. Each behaviour is its own reason to fail and belongs
  in its own unit.
- **Cases that answer different questions.** A table, or a set of named subdivisions, is
  one behaviour with many inputs, and stays one however its cases are spelled. It becomes
  several when the cases stop feeding inputs to one question and start asking different
  ones: some cases drive the unit to a result while others drive it to a rejection, one
  subdivision parses and the next one rejects, or the cases claim things about different
  subjects — one asserting what the call returned, another asserting what it wrote away.
  Then the loop has become several tests sharing one name, and grouping does not merge
  them. Judge what the cases claim, not the fields the case record happens to carry: a
  field every case fills in and a field only some cases use are the same evidence, and
  neither settles this on its own.
- **Act-assert-act-assert.** Act, assert, act again on the result of the first, assert
  again, act a third time — a scenario walkthrough rather than a test. Every act after
  the first is a new behaviour wearing the previous test's name, and an early failure
  stops the run before the later ones are ever exercised.

Length carries no penalty and assertion count carries no credit. A long test is not a
violation, and a test with a single assertion is not automatically a pass — a lone
assertion on a summary boolean can still be guarding a bundle.

You are given the test file and not the code it exercises. Count the behaviours the unit
itself acts on and asserts; do not speculate about what a call not shown to you does
underneath, and do not split one act into several because the implementation might have
several steps.

Whether the unit is named for what it pins down, whether a reader could tell which claim
broke, whether it stands in for the subject, and whether the file covers everything are all
somebody else's questions. One badly named unit pinning down exactly one behaviour satisfies
this rule.

In the reason, name the surplus behaviour and where it starts: the second act's call, the
assertion that belongs to a different claim, or the case labels that split one loop into
two questions. Say which behaviour should move out, so the reader knows what to cut and
what stays.

Weak: `the test does more than one thing`.
Better: `after asserting the parsed value it calls Save and asserts the written file, a
second behaviour under the first one's name`.
