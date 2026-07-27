---
include_source: false
granularity: function
---
A test is read far more often than it is written, and almost always by someone who was
not there when it was written. That reader must be able to answer three questions:

1. **What does this test establish?**
2. **From what?** Which arrangement and which inputs produce the outcome it checks.
3. **And when it turns red, what broke?** The failure alone has to say.

A test satisfies this rule when a stranger gets all three in one pass over the body. Judge
what such a reader can extract, not what you can reconstruct — you have the whole file and
unlimited attention, and they have one reading and a failure message.

## What a failure has to say

The third question is the one most easily misjudged, because ecosystems differ in how much
they hand you for free. Some rewrite a plain comparison into a full report of both sides;
some print a structured difference; some print an expected and an actual; some print only
the text the author wrote and nothing else. Do not require any particular assertion call,
message argument or output format.

The property is this: **after a failing run, can the reader tell which assertion failed and
what it was checking?** Decide that from what the test file shows, never from a runner you
cannot see. Where a body runs its assertions once, a bare comparison is complete and owes
nothing more: the unit is named and the failing line is its own, so which assertion broke
and what it was checking are already answered.

Where one comparison stands in for many, the file itself has to supply the rest, and two
things in it can. Either the loop declares each case to the framework as its own reported
unit, run and named separately, so a failure arrives under that case's name; or what the
failing comparison emits carries the case's identity — its label, or its input. Either one
is enough, and the rule does not care which. Neither present is the shape that fails: every
case fails on the same line, and the reader is left to work out which one got there. The
same holds for several calls to one assertion helper inside a single unit when nothing
handed to it or emitted by it tells the calls apart. A helper called once in a unit is
never this problem — the runner names the unit.

Judge the pairing of assertion and reporting, never the assertion on its own.

Whether a case's label is a well-chosen description is not this rule's question. Whether a
failure can be traced back to a case at all is.

SATISFIES the rule:

- **Arrangement that pulls its weight.** Every value constructed is one the act consumes,
  one an assertion inspects, or one a constructor demands. A required-but-irrelevant field
  set to an obvious placeholder is arrangement doing its job: it tells the reader this field
  does not matter.
- **The deciding values in view.** The input that produces the asserted outcome appears in
  the act or in a named binding beside it, and the expected value sits with the assertion
  that checks it, so a reader traces input to outcome without leaving the body.
- **Helper calls whose names say what they build or what they check.**
  `newOrderTotalling(120)` and `assertRejected(result)` compress a test; they do not hide
  it. A helper is noise only when its name tells the reader nothing.
- **Shared arrangement the test then bends to its own case.** Building on common setup is
  fine when the body shows which part of it this case depends on — the one field that
  matters overridden in view, next to the act.
- **A long, linear body.** Length carries no penalty. Twenty lines a reader can follow top
  to bottom beat five that hide their inputs behind a call.
- **A bare comparison where the framework reports it.** No message is owed when what failed
  and what it was checking are already in the output.
- **An assertion on a single condition, where that condition is what the subject produces.**
  When the outcome under test is itself one boolean — accepted or rejected, present or
  absent, locked or unlocked — the condition is the difference, and asserting it answers
  "what broke". What withholds an answer is joining outcomes that could fail separately.
- **One comparison of a whole produced value against a whole expected value**, where the
  expected value is visible to the reader. Seeing the difference between two complete values
  is a clear answer to "what broke".
- **A loop that declares each case as its own reported unit**, or whose failing comparison
  emits the case's label or its input. Either is enough; the rule does not care which
  supplies it.

DISQUALIFIES the rule:

- **Arrangement for a world the behaviour never touches.** Three records built when the act
  reads one; a customer, an address and a payment method assembled for a test about a
  line-item total; fields populated that the act never consumes and no assertion inspects.
  Every line of setup a reader must consider and then discard is a line that claimed to
  matter and did not.
- **The deciding values out of sight.** The input that produces the asserted outcome is set
  somewhere the body does not show — inside an unnamed setup routine, in a distant shared
  fixture, in a default the test never restates — so the reader sees the conclusion and not
  the premise.
- **The deciding value present but buried.** The one character that makes the input invalid
  sitting in the middle of a two-hundred-character literal, the one significant entry in a
  long list of identical ones, with nothing in the test pointing at it. Present is not the
  same as visible.
- **A helper that swallows the test.** A body that is one call to `setup()` and one call to
  `check()`, where the act, its input and its expectation all live inside routines whose
  names describe none of them. The whole distinction from a good helper is the name: one
  that says what it builds or what it asserts leaves the reader oriented; `run`, `doIt` and
  `setup` leave them to open it.
- **A failure that cannot be traced to a case.** One comparison inside a loop that neither
  declares each case as its own reported unit nor puts the case's label or input into what
  the failure emits: every input fails on the same line and the reader is left to work out
  which one got there. The same applies to several calls to one assertion helper inside a
  unit when nothing handed to it or emitted by it tells the calls apart.
- **A failure that collapses several outcomes into one condition.** Several comparisons
  joined into one boolean and asserted once; a flag set inside a loop and checked after it;
  a count of mismatches asserted to be zero. Each reports that something was wrong and
  withholds which of the joined outcomes it was and how — exactly what the reader needed.

Readability is not style. Formatting, line length, blank lines, comment density, the choice
of one arrangement idiom over another, and how the cases are laid out decide nothing here. A
test that answers the three questions passes however it is written, and a test that leaves
one of them unanswered fails however tidy it looks.

**Arrangement that runs before each test is arrangement, not concealment.** A hook or method
that the framework runs before every test, or a scope enclosing the unit that prepares what
the unit then uses, is a normal way to build a fixture and satisfies this rule when a reader
can see what it builds — because it sits in the same file, and because its name or its body
says what it produces. The second question asks whether the deciding values are *findable*,
not whether they are inline. What fails is arrangement whose deciding value the reader
cannot locate at all, or one whose name says nothing about what it built.

You are given the test file and not the code it exercises. Judge what a reader can extract
from what is written here; do not speculate about what a call not shown to you might do, and
do not fail a unit because a helper you cannot see *might* be doing something surprising.

How many behaviours the unit pins down, whether its name states one, whether it stands in
for the subject, whether another unit says the same thing, and whether the file leaves an
outcome uncovered are all somebody else's questions. A unit that pins down four behaviours
in one body satisfies this rule outright if a reader can tell what all four are, from what,
and which one broke.

## Writing the reason

Say which of the three questions the reader cannot answer, and name the thing that blocks
it: the arrangement that goes unused, the identifier holding the deciding value out of view,
or the assertion whose failure would not identify itself. When the problem is a loop or a
shared helper, say so — the fix is to carry the case's identity into the failure, and that
is different from the fix for unused setup.

Weak: `the test is hard to follow`.
Better: `builds a Customer, an Address and two prior Invoices when the act only reads
Invoice.Total, so a reader cannot tell which part of the arrangement decides the result`.
