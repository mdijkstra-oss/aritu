You are judging the units of a test file against one rule. The rule follows this text;
the file follows the rule. For each unit you are given, return a verdict: whether that
unit satisfies that rule.

Only the rule below applies. A unit may be badly named, may test two things at once, may
stub out half the world — none of that is your concern unless the rule below asks about
it. Judge one criterion, on the units you are given, and nothing else.

## Test shapes

Tests come in many shapes, and every ecosystem writes them differently: a table of cases
walked by one loop, parametrised cases generated from a list, grouping blocks nested
inside each other, one class per fixture, plain linear arrange-act-assert, golden-file
comparison, integration tests that stand up real dependencies.

Judge the behaviour a test pins down, never the syntax it is written in. No shape is by
itself a pass or a fail: a table of cases is not a violation of anything, and neither is
a test without one. A property that holds of a test written one way holds of the same
test written another way, and a rule that fires on the spelling rather than the substance
is being read wrongly.

## What a unit is

A unit is whatever the rule below judges, and the rule declares which. Sometimes it is each
test in the file, sometimes each leaf of each test, and sometimes the file itself — in which
case you are given one unit, its path, and you return one verdict covering everything in it.
Judge each unit you are given on its own, exactly as identified.

**The subject** is the production function, method, type or module a test drives and reports
on. It is what the test exists to pin down, as distinct from the collaborators it uses along
the way and from the scaffolding the test builds around it.

When units are named individually, a name has up to three parts, and any of them may be
absent:

- **A namespace path**, written `Outer > Inner > ` before the rest. Enclosing scopes — a
  grouping block, a fixture class, an outer suite — qualify a name. They say where the
  test lives, not what it claims.
- **The test**: the smallest thing the framework runs and reports under its own name.
- **A case**, written `(in parentheses at the end)`: one leaf of that test, such as one
  row of a table or one parametrised argument set.

When a unit carries a case, the unit is the whole identifier — the string a test runner
prints when that case fails. Read the parts as one name. No part is judged alone, and no
part has to carry the whole meaning by itself:

- In `TrimsSurroundingWhitespaceFromEachTag (leading spaces)` the test states the
  behaviour and the case only varies the input.
- In `ParseAddress (extracts host before colon)` the test name is a namespace and the
  case carries the claim.
- In `ParseConfig (host and port)` neither part states a behaviour.

All three are one unit each. Whatever the rule asks of a unit, ask it of the composite.

## Writing the reason

Every unit's answer carries a reason, so write one for every unit. For a unit that does not
satisfy the rule the reason is the whole diagnostic and the guidance below applies to it.
For a unit that does satisfy it, one short clause naming what carries it is enough; nobody
reads those, and they exist so that every answer has the same shape. When the unit is a
whole file, the reason is the only guidance a reader gets, so it has to do the locating work
the unit name cannot.

- Exactly one sentence.
- About this unit against this rule. The rule restated in the abstract is not a reason,
  and neither is asserting non-compliance: "does not satisfy the rule", "violates the
  criterion" and "a test must pin down one behaviour" tell the reader nothing they did
  not already have from the verdict.
- **Name the thing in the file that causes it** — the identifier, the case label, the
  assertion, the second act, the substituted type, the call that reaches outside the
  test. A reader who has this sentence and nothing else must know where in the file to
  look and what to change. Quote the offending fragment when a name alone would not
  locate it.
- Write to the person who has to fix this test, not about them.

Weak: `the test name is not behaviour-focused`.
Better: `names the unit under test with no stated outcome, so it would still read as true
whatever parsing produced`.
