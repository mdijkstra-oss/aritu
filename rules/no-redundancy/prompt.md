---
targets: [tests]
include: [tests]
include_source: true
granularity: file
---
The unit here is the whole file, and the verdict is about its tests taken together rather
than about any one of them. The file satisfies this rule when no two of its tests assert
the same behaviour with examples drawn from the same equivalence class.

Two tests pinning down one case cost twice: two places to edit when the behaviour moves,
two red lines for one defect, and a reader who has to diff them character by character to
discover they differ in nothing that matters. But a test deleted as a duplicate that was
not one takes a real case out of the suite, and nobody finds out until the case breaks in
production. That asymmetry is the whole shape of this rule. Rejecting is expensive; most
of what follows is about not rejecting too readily.

## What an equivalence class is

Two inputs belong to the same equivalence class when the implementation cannot tell them
apart: they land on the same side of the same comparisons, take the same branches, and
reach the same outcome by the same route. Sameness is a fact about the code. It is not a
judgement about how alike the two values look, and it is not a judgement about how alike
the two tests read.

You are given the implementation for exactly this reason, and it is what decides the
question. Take a call passing an empty string and a call passing a string of spaces. If
`ParseConfig` trims its argument and then makes one emptiness check, both calls arrive at
that check having become the same value: one branch, one class, two tests where one would
do. If instead it rejects an empty argument on one branch and a wholly blank one on
another — a separate condition, a separate error, a separate path — those are two classes
and two tests are precisely right. Nothing in the two test bodies distinguishes these
situations. Only the source does.

So work in this order. Find pairs of tests that assert the same outcome of the same entry
point. For each pair, follow both inputs through the implementation. Reject only when you
can point at the branches and show that both inputs take the same ones. A pair you cannot
trace is not a pair you can reject. The one exception is a pair supplying the same input:
there is no difference to trace, and the repetition is itself the evidence.

SATISFIES the rule:

- **A boundary pair.** A value at the limit and a value just past it are two classes,
  always, however alike the two tests read and however small the difference between the
  inputs. The comparison *is* the behaviour, and it takes both sides to pin it down: a test
  at the limit alone cannot tell a strict comparison from an inclusive one, so removing
  either test leaves the boundary unguarded. The same holds for a triple just under, at,
  and just over. This shape is the single most common false alarm on this rule — two tests
  differing by one in one literal, asserting outcomes that look related. It is not
  redundancy.
- **Two tests with identical shape asserting different behaviours.** Same arrange lines,
  same construction helper, same assertion form, different claim: one asserts what was
  returned, another asserts the error, another asserts the state left behind. Copied
  structure is how a suite stays legible, and a family of tests that read as variations on
  one template is a well-kept file, not a duplicated one. Shape is never the evidence here.
- **A table of many inputs to one behaviour, which is one test and not many.** Twelve cases
  feeding twelve different values into one question are twelve examples of a single test.
  That every case runs the same assertion is the shape doing its job, and cases whose
  inputs differ are not counted against one another. Decompose a table into its cases only
  to check for the two shapes below that live inside tables — an example repeated verbatim,
  and a case a standalone test covers a second time.
- **Several inputs sharing one outcome but reaching it by different routes.** Four distinct
  causes of rejection all producing a rejection are four classes, because the
  implementation reaches each by its own branch. A shared outcome does not merge them, and
  a shared error type does not either.
- **The same input used by two tests that claim different things.** One asserts the value
  returned, another asserts what was written or handed onward from that same call. Same
  fixture, different behaviour, no redundancy.
- **A unit tested directly and again through a caller that uses it.** The second test pins
  down the composition — the wiring, the ordering, the propagation of a result — which is a
  behaviour of the caller and not a repeat of the first test.
- **A file with a great many distinct tests.** Count is not the subject. A long file whose
  tests each land in their own class is a thorough file.

DISQUALIFIES the rule:

- **The same case tested twice under two names.** Two tests calling the same entry point
  with different literal values that the implementation normalises, trims, lowercases or
  coerces into the same value before the only branch that matters, each asserting the same
  outcome. The two names promise two cases; the code sees one.
- **A copied test with a changed constant that changes nothing.** The literal moved from
  `3` to `4` but the implementation only compares against zero, so both runs take the same
  path to the same result. Varying an input that the code does not branch on produces a
  second test with no second case behind it.
- **The same example listed twice.** Two cases in one table, or two entries in one list of
  inputs, carrying the same input and the same expectation under different labels. This is
  the exception to the table shape above, and it needs no tracing: the example is literally
  repeated.
- **A standalone test repeating a case a table already covers.** A table case already feeds
  that input to that behaviour and a separate test does it again with a different spelling.
  The two shapes differ; the class does not.
- **A test that re-asserts what another already established about the same call.** One test
  asserts every field of a result produced from a given input; a second builds the same
  input and asserts one of those fields again. The second adds no case.

Judge only duplication. A file that tests too little is not rejected here — a hole is not a
duplicate, and the absence of a case is outside this question entirely. Nor does it matter
here whether a test is well named, whether it asserts more than one thing, whether it would
survive a refactor, or whether a stranger could read it. A file with no two tests in one
class satisfies this rule however else it reads.

**When the implementation does not settle it, the file satisfies the rule.** If the branch
is not visible, if the two calls run through code you were not given, if the inputs might
differ in a way that matters and you cannot demonstrate that they do not — that is a pass.
Suspicion is not evidence. Reject only when you can name the pair and either show that both
supply the same input or name the branch that erases the difference between the inputs they
supply.

## Writing the reason

A reason on this rule has to survive being read on its own by someone about to delete a
test, so it must name **both** tests — spelled as the file spells them — and the class they
share. Where the two supply different inputs, that means the input each supplies and the
point in the implementation where the difference between those inputs disappears; where
they supply the same input, it means the repeated example itself. Without that second half
the reader cannot check your work, and this is the rule where they must be able to. If the
file holds more than one redundant pair, give the clearest one; a single located pair is
worth more than a count.

Weak: `several tests cover the same ground`.
Better: `RejectsEmptyName and RejectsBlankName both call ParseConfig, which trims its
argument before its one emptiness check, so the empty string and the all-spaces string
reach that branch as the same value and both tests assert the same rejection`.
