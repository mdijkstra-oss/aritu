You are judging the units of a Go test file against one rule. The rule follows this
text; the file follows the rule. For each unit you are given, return a verdict: whether
that unit satisfies that rule.

Only the rule below applies. A unit may be badly named, may test two things at once, may
stub out half the world — none of that is your concern unless the rule below asks about
it. Judge one criterion, on the units you are given, and nothing else.

## Test shapes

Go tests come in many shapes: table-driven with a slice of cases, subtests declared with
`t.Run`, plain linear arrange-act-assert, golden-file comparisons, functional and
integration tests that stand up real dependencies. Judge the behaviour a test pins down,
never the syntax it is written in. No shape is by itself a pass or a fail: a table-driven
test is not a violation of anything, and neither is a test without a table.

## What a unit is

Each unit you are given is judged on its own, exactly as identified.

When a unit is written as `TestFunction (case name)`, the unit is the whole identifier —
the string Go prints in CI output when that case fails. Read the two halves as one name.
Neither half is judged alone, and neither half has to carry the whole meaning by itself:

- In `TestTrimsSurroundingWhitespaceFromEachTag (leading spaces)` the function states the
  behaviour and the case varies the input.
- In `TestParseAddress (extracts host before colon)` the function is a namespace and the
  case carries the claim.
- In `TestParseConfig (host and port)` neither half states a behaviour.

All three are one unit each. Whatever the rule asks of a unit, ask it of the composite.

## Writing the reason

Give a reason for every unit that does not satisfy the rule. A unit that satisfies it
needs none.

- Exactly one sentence.
- About this unit against this rule. The rule restated in the abstract is not a reason,
  and neither is asserting non-compliance: "does not satisfy the rule", "violates the
  criterion" and "a test must pin down one behaviour" tell the reader nothing they did
  not already have from the verdict.
- Point at the thing that is wrong — the name, the case label, the second act, the
  substituted type — so the reader knows what to change. Write to the person who has to
  fix this test.

Weak: `the test name is not behaviour-focused`.
Better: `names the unit under test with no stated outcome, so it would still read as true
whatever parsing produced`.
