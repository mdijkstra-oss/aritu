---
include_source: true
---
Judge whether the test exercises the real implementation of the unit it claims to test.

The subject is the production function, method or type the test is named for and whose
behaviour the test reports on. A test satisfies this rule when the subject is the real
production code and the assertions are about what that real code computed, returned or
changed.

The decisive question: if the real implementation of the subject were deleted from the
source file, would the test still pass on its own scaffolding? If yes, the test does not
test the subject and does not satisfy the rule.

SATISFIES the rule:

- The real production function or type is called directly and its return value, error or
  resulting state is asserted.
- The subject is reached through an interface, constructor or dependency parameter, but
  the concrete value behind it is the production implementation.
- A genuine out-of-scope collaborator is replaced by a fake, stub, spy, in-memory double
  or canned fixture — a clock, a random source, a UUID generator, an outbound HTTP
  transport or round-tripper, a payment gateway, an SMTP sender, a database, a message bus
  — while the real subject still runs and the assertions are about what the real subject
  did with it. Test doubles are not banned by this rule; substituting the *subject* is what
  this rule catches.
- The subject's whole job is to hand something to a collaborator (a notifier, a publisher,
  an exporter), the collaborator is faked, and the test asserts on *what the real subject
  handed it* — the message, the payload, the arguments. There the recorded call is the real
  subject's actual output, not a stand-in for it.
- No test double appears at all. A test that only calls real code cannot violate this rule.

DISQUALIFIES the rule:

- The subject itself is replaced by a stub, fake, spy or mock declared in the test file,
  and the assertions are on that substitute's canned return value. The production
  implementation is never invoked.
- A package-level function variable, constructor hook or struct field that *is* the unit
  under test is swapped for a test-supplied function, and the test then asserts that
  test-supplied function's result.
- The only assertion is that a double standing in for the subject was called — a call
  count, a `wasCalled` flag, a recorded-arguments slice — with no assertion about any value
  returned, state changed or effect produced by the real subject.
- The real subject is constructed but every assertion lands on a double's pre-programmed
  return value, so the subject's own logic never influences the outcome.
- Every assertion in the test would still hold with the real implementation deleted; the
  test pins down its own scaffolding and nothing else.

Go tests come in many shapes: table-driven with a slice of cases, subtests via t.Run, plain
linear arrange-act-assert, golden-file comparisons, functional or integration tests that
stand up real dependencies. Judge the behaviour the test pins down, never the syntax it
uses. A table-driven test is not a violation of anything, and neither is a test without a
table. A type with "mock", "stub", "fake" or "spy" in its name is not by itself a
violation either. What decides it is whether the substituted thing is the subject or a
collaborator, and whether the assertions are about real behaviour.
