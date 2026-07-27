---
targets: [tests]
include: [tests]
include_source: true
granularity: function
---
A test must bind to the seam a caller would use, and assert what the code produced rather
than how it produced it. Such a test survives a rewrite of the thing it tests: change the
algorithm, swap the internal data structure, reorder the internal steps, and as long as a
caller still gets what it was promised, the test stays green. A test bound to the internals
goes red on a change that broke nothing — and once that has happened twice, the suite starts
being edited to match the code instead of the code being fixed to match the suite.

The subject is the production function, method or type the test drives and reports on. Its
**surface** is what a caller can reach and rely on: what it returns, what it raises or
signals, the state it changes that a caller can observe afterwards, what it hands to the
collaborators it was given. Its **internals** are everything it owns and may rearrange
freely: the fields it stores its work in, the helpers it calls on the way, the order and
number of those calls, the strategy it picked.

**You are given the implementation as well as the test, and it is what decides this rule.**
Whether an identifier the test touches belongs to the subject's surface or to the state the
subject owns is a fact about the source, not about the test. Read the implementation, find
where the subject's boundary actually sits, and judge which side of it each assertion lands
on. Do not guess this from the test alone — and when the source you were given does not show
where that boundary sits, the unit satisfies the rule. Suspicion is not evidence.

**Visibility keywords do not decide this.** A test living in the same compilation or
namespace unit as its subject may call things the language spells without a public marker,
and does so routinely; some ecosystems have no enforced privacy at all; and a test may be
placed inside the subject's namespace deliberately, precisely so it can drive the thing it
means to test. The violation is reaching *past* the subject into state the subject owns —
not calling an identifier that happens to lack a public marker. When such an identifier
**is** the subject — the test drives it as a unit, feeds it inputs and asserts what it
returned — testing it directly is this rule being satisfied, not broken.

SATISFIES the rule:

- The subject is driven through its surface and the assertions are about what it produced:
  the returned value, the error or signal raised, the observable state left behind, the
  message it emitted.
- The subject is reached through an abstraction, constructor argument or dependency
  parameter, and the concrete thing behind it is the real production code.
- The subject is an internal helper called directly, given inputs and asserted on by what it
  returns. That the helper is not part of a published surface is irrelevant; it is this
  test's subject and the test binds to its own edge.
- **A genuine out-of-scope collaborator is replaced by a fake, stub, spy, in-memory double
  or canned fixture** — a clock, a random source, an identifier generator, an outbound
  transport, a payment gateway, a mail sender, a database, a message bus — while the real
  subject runs and the assertions are about what the real subject did with what it got back.
  Doubles are not what this rule is about; substituting the *subject* is.
- **The subject's whole job is to hand something to a collaborator** — a notifier, a
  publisher, an exporter, a writer — the collaborator is a double, and the test asserts on
  *what the real subject handed it*: the payload, the message, the arguments. There the
  recorded call is the subject's actual output and there is nowhere else to observe it. What
  separates this from the call-record shape below is that no closer observation was
  available: the subject returned nothing, raised nothing and changed nothing else the test
  could have asserted, so the content it handed over is the whole of what it produced.
- Assertions on the order or grouping of a returned sequence, when that ordering is part of
  what the caller is promised. Ordering a caller can observe is an outcome, not an algorithm.

DISQUALIFIES the rule:

- **Reaching past the subject into state it owns.** Reading a field the subject stores its
  work in and asserting on it instead of on what the subject returned; asserting on an
  intermediate value the subject left behind mid-way; writing directly into that state to
  set up a case the surface could have reached on its own. The implementation shows which
  state is the subject's own bookkeeping; assertions on it pin down a layout the subject is
  free to change.
- **Asserting details a change with no observable effect would break.** The strategy chosen
  — which algorithm, which lookup, which comparison — rather than the result it produced;
  the shape of the internal container rather than the values a caller gets out; the number
  of iterations, the intermediate query or command constructed on the way, the internal log
  line emitted by a step. Ask of each assertion: could the subject be rewritten so that every
  caller still gets exactly what it got before, and this assertion still fail? If yes, the
  assertion is about how, not what.
- **Asserting the order or count of calls to collaborators in place of an outcome.** That a
  double was called, called once, called twice, called before another one — or called with
  arguments that are not themselves what the subject produced — where a closer observation
  was available and went unasserted: a value the subject returned, an error it raised, state
  it changed, or the payload it handed over, asserted for its content rather than for the
  fact of the call. A call-count flag records that the subject reached a line, not what it
  computed. The narrow exception is a subject whose promised behaviour genuinely *is* the
  sequencing, where the ordering is the named behaviour under test and nothing observable
  records it otherwise; incidental sequencing that merely happens to hold today is not that.
- **Substituting the subject itself.** A double, stub or test-supplied replacement stands
  where the production code should be — a replaceable function reference, a constructor
  override or a settable hook that *is* the unit under test is swapped out — and the
  assertions land on that substitute's canned answer. The real logic never runs. The check:
  if the subject's production body were deleted and only the test's own scaffolding remained,
  would this test still pass? If yes, it is pinning down its own scaffolding.

A double named "mock", "stub", "fake" or "spy" is not by itself anything, and neither is the
absence of one. A test that drives real code from end to end is clear of the last two shapes
above, which need a double to occur; it can still fail this rule by reaching into state the
subject owns or by asserting on the strategy the subject picked. What decides the verdict is
whether the substituted thing was the subject or a collaborator, and whether the assertions
describe something the subject produced or something about how it went about it.

## Writing the reason

Name the exact assertion that binds to the internals, and say what a caller-visible
assertion would have been in its place. The reader must be able to find the line and know
what to replace it with, so identify the internal thing by name — the field reached into,
the strategy asserted on, the double whose call record stands in for an outcome — and say
what observable result was available instead.

Weak: `the test asserts implementation details rather than behaviour`.
Better: `asserts on cache.buckets, the storage the subject owns, instead of on what Get
returned for the same key, so any change to how entries are stored fails it`.
