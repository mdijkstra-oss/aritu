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

## The levels

A **test** is the smallest thing this file's framework runs and reports under its own name.

An **enclosing scope** is anything that groups tests and qualifies their names without being
run as a test itself: a grouping block, a suite, a fixture class, a module. Scopes are
namespaces, not tests.

A **case** is one leaf of a single test: one row of a table of cases, one generated or
parametrised argument set, or one subdivision declared inside the test body. A case is not a
test of its own; it is one execution of one test.

**The subject** is the production function, method, type or module a test drives and reports
on. It is what the test exists to pin down, as distinct from the collaborators it uses along
the way and from the scaffolding the test builds around it.

## How a unit is named

Write a name by joining its enclosing scopes, outermost first, with " > ", then the test's
own name, then its case in parentheses. A name has up to three parts, and any of them may be
absent:

- **A namespace path**, written `Outer > Inner > ` before the rest. Enclosing scopes say
  where the test lives, not what it claims.
- **The test**: the smallest thing the framework runs and reports under its own name.
- **A case**, written `(in parentheses at the end)`: one leaf of that test, such as one row
  of a table or one parametrised argument set.

When a unit carries a case, the unit is the whole identifier — the string a test runner
prints when that case fails. Read the parts as one name. No part is judged alone, and no
part has to carry the whole meaning by itself:

- In `TrimsSurroundingWhitespaceFromEachTag (leading spaces)` the test states the
  behaviour and the case only varies the input.
- In `ParseAddress (extracts host before colon)` the test name is a namespace and the
  case carries the claim.
- In `ParseConfig (host and port)` neither part states a behaviour.

All three are one unit each. Whatever the rule asks of a unit, ask it of the composite.
