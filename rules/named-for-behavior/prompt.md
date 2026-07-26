---
include_source: false
---
A test's name must say which behaviour breaks when the test fails. A reader who sees
only the failing name scroll past in CI output — without opening the file — should
know what regressed.

A test satisfies this rule when its name states an observable behaviour of the code
under test, specifically enough that the name would have to change if that behaviour
changed. In practice the name carries an outcome, and usually the condition that
produces it:

- `TestReturnsNotFoundForUnknownUser`
- `TestRetriesOnceAfterTimeout`
- `TestTruncatesTitlesLongerThanTheLimit`
- `TestKeepsInsertionOrderAcrossFlushes`

The name need not be a sentence and need not be long. It must be falsifiable: read
the name, then read the body, and the name should already have told you what the
body checks.

A test is DISQUALIFIED when its name takes any of these shapes:

- **Names only the unit under test, with no behaviour at all.** `TestParse`,
  `TestUser`, `TestHandler`, `TestNewClient`, `TestValidateConfig`. The name of a
  function, method, type or file is not a behaviour, and this stays true however
  descriptive the function's own name is.
- **Numbered or lettered variants.** `TestParse2`, `TestFooB`, `TestValidateCase3`,
  `TestServerThree`. The suffix separates the test from its siblings but says
  nothing about what it protects.
- **Describes mechanics, ceremony or vibes rather than behaviour.** `TestItWorks`,
  `TestHappyPath`, `TestSadPath`, `TestEdgeCases`, `TestMisc`, `TestBasic`,
  `TestSmoke`, `TestSanity`, `TestFullFlow`, and any name whose only claim is
  "works", "succeeds", "is correct", "returns OK" or "handles it". Those describe
  that a test ran, not what it defends.
- **So vague it would still fit after the behaviour changed.** If the code were
  changed to produce a different result and the name would still read as true, the
  name is protecting nothing.
- **Names only the setup or the input, never the outcome.** `TestWithEmptyConfig`,
  `TestNilInput`, `TestTwoUsers`.
- **Claims a behaviour the body does not actually pin down.** A precise-sounding
  name that does not match what the test asserts fails too — the name must be true
  of this test, not of some other test.

Judge the name against the behaviour the body actually establishes. If the body
pins down a specific outcome and the name reports it, the test satisfies the rule
even if the wording is terse.

Go tests come in many shapes: table-driven with a slice of cases, subtests declared
with `t.Run`, plain linear arrange/act/assert tests, and functional or integration
tests that drive a real system end to end. Shape is not the subject of this rule and
carries no penalty. A table-driven test is not a violation of anything — judge the
single behaviour its cases collectively pin down and ask whether the test function's
name states that behaviour. Case labels and `t.Run` strings inside the function are
not what is under judgement here; the test function's own name is.
