---
include: [tests]
include_source: true
granularity: test_case
---
A unit's verdict must hang on the behaviour it is named for. Two things have to be true
at once, and the unit fails this rule if either is missing: the name must state a
behaviour specific enough that something could contradict it, and the body must be
capable of contradicting it.

The decisive question, asked of the whole identifier: **if this behaviour were removed
from the implementation, would this unit go red?** No is a fail. It is a fail for a name
so vague that nothing could ever contradict it, and just as much a fail for a precise
name whose body stays green while the thing it promises is broken. A well-named unit
that proves nothing is the shape this rule exists to catch.

You are given the implementation as well as the test, and it is what answers this
question. Find in the source the behaviour the name promises, then ask what would still
hold in this unit if you deleted it. Do not settle this from the test alone — and when the
source you were given does not show the behaviour the name promises, the unit satisfies the
rule. Suspicion is not evidence.

The name satisfies its half when it states an observable outcome of the code under test,
specifically enough that the name would have to change if that outcome changed. In
practice it carries the outcome, and usually the condition that produces it:

- `ReturnsNotFoundForUnknownUser`
- `retries_once_after_timeout`
- `truncates titles longer than the limit`
- `keepsInsertionOrderAcrossFlushes`

How a name is spelled carries no weight whatever. Word-joined, underscored, spaced or
sentence-shaped are the same name to this rule; judge what it claims. The name need not be
long. It must be falsifiable: read the name, then read the body, and the name should
already have told you what the body checks.

Every shape below is judged against the composite identifier, never one part of it. The
parts share the work and either may do it: the test may state the behaviour and leave the
case to vary the input, or the test may be a bare namespace and the case may carry the
claim.

A composite QUALIFIES when the parts together state a behaviour:

- `ParseConfig (extracts host before colon)` and `ParseConfig (rejects input with no
  separator)`. The test names the unit, the case makes the claim, and a reader who sees
  only that line fail knows what regressed.
- `TrimsSurroundingWhitespaceFromEachTag (leading spaces)`. The test already states the
  behaviour and the case only varies the input, which is what a table of cases is for. A
  case is not required to restate a behaviour its test has already stated.

A composite is DISQUALIFIED when:

- **Neither part states a behaviour.** `ParseConfig (host and port)` and `ParseConfig
  (missing port)`. The test names the unit under test, the cases name the input, and
  nothing anywhere says what the code should do with it. A reader seeing either line fail
  learns only which function was involved.
- **The case is numbered or lettered and the test states no behaviour either.**
  `ParseConfig (case 1)`, `Validate (variant 3)`. The label separates this case from its
  siblings and says nothing about what it protects, and with the test half naming only
  the unit, nothing in the identifier could be contradicted. A numbered case under a test
  that does state the behaviour varies only the input and is not this shape.

The body satisfies its half when the act reaches the named behaviour and the assertions
land on something that behaviour determines — so that changing the behaviour changes the
verdict.

SATISFIES the rule:

- **The act exercises the named behaviour and the assertions describe what it produced**
  — the returned value, the error classification, the state it changed, the message it
  emitted — with each asserted value one the behaviour actually decides.
- **The behaviour is reached indirectly.** The body calls a higher entry point that routes
  through the named behaviour rather than naming it. What decides this rule is whether
  removing the behaviour turns the unit red, not which identifier the body mentions.
- **An absence-of-failure assertion when acceptance is itself the named behaviour.** A
  unit named for accepting well-formed input, which feeds well-formed input and asserts
  that no error came back, does go red when the code starts rejecting it. The same
  assertion under a name that promises a produced value does not, and fails.
- **Expected values written from intent, however large.** A full expected structure
  spelled out by hand, or a comparison against stored expected output whose content a
  reader can open and evaluate against the requirement, is derived from what the code
  should do rather than from what it does.
- **A terse but precise name.** Wording carries no credit and length carries none. A short
  name that states an outcome satisfies the first half as well as a long one does.

DISQUALIFIES the rule:

- **Names the unit under test, with no behaviour at all.** `Validate`, `Parse`, `User`,
  `Handler`, `NewClient`. The name of a function, method, type or file is not a behaviour,
  and this stays true however descriptive that function's own name is.
- **Names the setup or the input and never the outcome.** `WithEmptyConfig`,
  `EmptyInput`, `TwoUsers`. What went in is not what the code should do with it.
- **Names ceremony, mechanics or a ticket rather than behaviour.** `HappyPath`, `SadPath`,
  `Bug123`, `ItWorks`, `EdgeCases`, `Misc`, `Basic`, `Smoke`, `Sanity`, `FullFlow`, and
  any name whose only claim is "works", "succeeds", "is correct" or "handles it". Those
  report that a test ran, not what it defends.
- **Numbered or lettered variants.** `Parse2`, `FooB`, `ValidateCase3`. The suffix
  separates the unit from its siblings and says nothing about what it protects.
- **So vague it would still fit after the behaviour changed.** If the implementation were
  changed to produce a different result and the name would still read as true, the name is
  protecting nothing.
- **A body that does not exercise the named behaviour.** The name promises one outcome and
  the body drives a different path, or asserts something adjacent to it. Delete the named
  behaviour from the implementation and every assertion here still holds.
- **Only that nothing failed.** The sole assertion is that no error came back, that
  nothing was raised, that the call returned. That proves the code ran; it says nothing
  about what it produced. A unit named for extracting the port from an address whose body
  asserts only that no error was returned is this shape exactly: the extraction could
  return the wrong port, or none, and the unit stays green.
- **Only what the signature or the language already guarantees.** That the returned value
  has its declared type; that a value the constructor was handed and never transformed
  comes back unchanged; that a collection built unconditionally on the path taken is not
  the absent value; that a constant equals itself. No change to the implementation's
  behaviour could make these fail.
- **The code as its own oracle.** The expected value is a snapshot of what the
  implementation currently emits rather than something derived from what it should emit —
  a literal copied out of a failing run, a pasted digest, a recorded blob.

That last shape needs care, because comparing against stored expected output is a
legitimate technique and this rule does not attack it. The violation is an expectation
that **could not have been written before the code ran** and that nothing in the test
justifies. Ask whether someone who knew the requirement but had never seen the
implementation could have produced this expectation, and whether a reviewer can look at it
and say whether it is right. If both answers are yes it is derived from intent, however
long it is. If the answer is no — nobody could have written this without running the code,
and nobody reads it — the expectation agrees with the implementation by construction and
can never contradict it.

Whether the unit pins down more than one behaviour, and whether it binds to internals a
refactor would move, are not asked here. Neither is whether the file as a whole leaves an
outcome uncovered. **Assume the real subject runs**: whether a substitute stood in its place
is somebody else's question, so judge the mutation question against the code you were given
rather than against what the test may have swapped out. Judge only whether the behaviour
this unit names is the behaviour its verdict depends on.

In the reason, say which half broke and point at the thing that has to change. If the name
is the problem, quote the identifier and say what outcome it fails to state. If the body is
the problem, name the assertion that carries the verdict and say what it would still allow
— the wrong result the implementation could return with that assertion still green. A
reader who has only your sentence must know whether to rename the unit or to strengthen an
assertion, and which one.

Weak: `the test does not prove what it claims`.
Better: `named for extracting the port, but the only assertion is that no error came back,
so any port value at all — or none — leaves it green`.
