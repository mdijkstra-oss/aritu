---
include_source: false
granularity: test
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

When the unit is a function with a case — `TestParseConfig (rejects input with no
separator)` — every shape above is judged against the whole identifier, never the
function name alone. The two halves share the work and either one may do it: the
function may state the behaviour and leave the case to vary the input, or the
function may be a bare namespace and the case may carry the claim.

A case name QUALIFIES when the composite states a behaviour:

- `TestParseConfig (extracts host before colon)` and `TestParseConfig (rejects input
  with no separator)`. The function names the unit, the case makes the claim, and a
  reader who sees only that line fail in CI knows what regressed.
- `TestTrimsSurroundingWhitespaceFromEachTag (leading spaces)`. The function already
  states the behaviour and the case only varies the input, which is what a table is
  for. A case is not required to restate a behaviour its function has already stated.

A case name is DISQUALIFIED when:

- **Neither half states a behaviour.** `TestParseConfig (host and port)` and
  `TestParseConfig (missing port)`. The function names the unit under test, the cases
  name the input, and nothing anywhere says what the code should do with it. A reader
  seeing either line fail learns only which function was involved.
- **The case is numbered or lettered.** `case 1`, `first`, `b`, `variant 3`. These are
  disqualified for the same reason `TestParse2` is: the label separates this case from
  its siblings and says nothing about what it protects. A case label that names neither
  the behaviour nor the input it varies fails whatever the function is called.
