---
include_source: true
granularity: file
---
Every distinct outcome the implementation can produce must be asserted somewhere in this
file. You are given the implementation as well as the tests, and the question runs from the
implementation outwards: read what the code can do, then look for the test that would turn red
if it stopped doing it. An outcome nothing asserts is free to change without anything
noticing, and that is a gap.

An **outcome** is a result a caller can tell apart from another result: a value returned, a
distinct failure reported, a state changed, an effect produced. Two inputs that travel the
same branch and produce the same kind of result are one outcome, and one test for them is
enough. Two inputs that produce results a caller would handle differently are two outcomes,
and each needs its own assertion. The set of outcomes is read off the implementation you are
shown — not off what the entry point's name suggests it ought to do, and not off outcomes you
would like it to have.

This is not line coverage. A branch that runs during a test but whose distinguishing result
nothing observes is uncovered for this rule's purposes, because deleting that branch's effect
would leave the file green.

The work, in order: enumerate the outcomes of the implementation; for each, find the
assertion that pins it down; the file passes only when every outcome has one.

SATISFIES the rule:

- Every outcome in the implementation has at least one assertion in the file that would fail
  if that outcome changed. That is the whole rule. Everything below is what does not count
  against it.
- **A table of cases, or parametrised cases, covering several classes at once.** Rows for the
  empty input, the single element, the many-element case and the over-limit case cover four
  outcomes. Coverage counts classes reached, never tests declared, and a file with four
  well-chosen cases may cover more than a file with thirty near-identical ones.
- **A private helper covered through its caller.** When the helper's outcomes are reached and
  asserted through the entry point that calls it, they are covered. A helper does not owe a
  test of its own, and an untested helper name is not by itself a gap.
- **An unreachable branch.** A branch that exists only to abort loudly once an invariant the
  rest of the implementation maintains has been broken, and that no input the surface accepts
  can reach. A guard over a bounded set of values that fails on a value no accepted
  input produces, a check for a state an earlier step already made impossible, a defensive
  path after a condition that has already returned — all of them are this. Demanding a test
  for a branch no caller can trigger would punish the pattern that makes broken invariants
  visible. Not a gap. Say nothing about them.
- **Hostile inputs on a surface that is not exposed to them.** A function whose parameters
  arrive already parsed, already validated or already typed — values the surrounding
  implementation produced rather than read — does not owe malformed, oversized or adversarial
  cases; whatever handed them over has already established the shape. Only a surface that
  decodes, parses, scans or otherwise interprets raw text, bytes or externally-shaped data
  owes them. When the implementation you were given does not show a surface taking in
  something it did not construct, treat it as not exposed and say nothing about hostile
  inputs.
- **A degenerate value that is not its own class.** The empty value, zero and one are gaps
  when the implementation branches on them, or when they produce a result a caller would
  handle differently from the ordinary one. When the code treats them the same as any other
  value they belong to the ordinary case and are already covered by it: a loop that simply
  runs no iterations, or an accumulation that simply accumulates nothing, is not by itself a
  gap.
- **A failure that is only passed through.** When the implementation forwards a
  collaborator's failure unchanged, one test showing the failure reaches the caller covers
  it. The file does not owe a case for every failure that collaborator could ever produce.
- **Outcomes asserted indirectly.** An effect observed through the state it left, or a
  returned value checked through a helper that inspects it, is asserted. How the assertion is
  spelled is not the subject.

DISQUALIFIES the rule:

- **A missing happy path.** An entry point the file never calls at all, or one it calls
  without asserting what it ordinarily produces. This is the largest gap the rule can find
  and the easiest to miss, because the file looks busy.
- **A reachable error path never exercised.** The implementation reports a failure for a
  condition, and no test in the file constructs an input that reaches it. Each distinct
  failure is its own outcome: a file that proves the valid input works has said nothing about
  what happens to the invalid one.
- **Boundaries untested.** A comparison against a limit splits inputs into three classes —
  below it, exactly at it, past it — and off-by-one lives in the middle one. A file that only
  ever passes values comfortably inside the range leaves the limit itself unasserted.
- **Degenerate inputs missing.** The empty collection, the absent value, zero, exactly one,
  and the maximum wherever the implementation itself imposes one. Each is a class of its own
  as soon as it takes a branch of its own or produces a result the ordinary case never
  produces — a guard that rejects the empty input, an accumulation that yields zero out of
  nothing, a divisor that can be zero, a limit the code enforces — and a file that only
  passes typical values never asks.
- **Hostile inputs missing where the surface is exposed to them.** A surface that decodes,
  parses, scans or interprets raw text, bytes or externally-shaped data owes the malformed,
  the truncated, the oversized, the wrongly encoded, the deeply nested and the deliberately
  crafted. Proving it handles well-formed input says nothing about the input it will actually
  be handed.
- **An outcome the file runs but never observes.** The branch executes, and every assertion
  would hold whether it executed or not: only that a call completed, only that some failure
  occurred when the code distinguishes three, only one field of a result whose other fields
  the code also computes. Executed is not asserted.
- **Any distinct outcome nothing asserts.** The catch-all, and the one to fall back on: a
  state change no test reads back, an effect no test observes, a returned field no test looks
  at, a condition no test makes true.

Judge the set, not the count. A file with few tests that reaches every class passes; a file
with many tests crowded onto one class fails, however thorough it looks. And judge only what
this implementation produces — an outcome you cannot point to in the code you were given is
not a gap, it is a feature request.

Nothing else about these tests moves this verdict. How the covering tests are named, whether
one of them pins down two behaviours at once, whether a reader could follow them, whether
they reach outside the machine, whether two of them overlap, and whether they stand in for
the subject are all somebody else's questions. A file of badly named, tangled, duplicated
tests satisfies this rule as long as between them they assert every outcome.

## Writing the reason

You return one verdict and one sentence for an entire file, so this sentence carries more
than it does under a rule that judges each test. It must let the reader write the missing
test without reopening the implementation. Name three things: **the entry point that produces
the uncovered outcome**, **the outcome itself**, and **the condition that reaches it**. Use
the identifier as it appears in the implementation, not the name of the test that came
closest.

When several outcomes are uncovered, name one concretely rather than gesturing at all of
them. A reader who is handed one specific missing case will find its neighbours; a reader
handed "coverage is incomplete" will not.

Weak: `error paths are not covered`.
Better: `nothing reaches ParseConfig's failure for input with no separator, so the file never
asserts which failure that returns`.
