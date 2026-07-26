# aritu fails its own test suite, and that is the interesting part

Pointing the first working build at its own tests, sonnet, 4 votes:

```
named-for-behavior    internal/lib/vote/vote_test.go
  TestCollect=0  TestTally=0  TestIsUnanimous=0  TestIsRejected=0     exit 1

one-reason-to-fail    internal/domain/lint/lint_test.go
  TestApply=0  TestBuildNamesPrompt=4  TestBuildVerdictPrompt=4  TestExitFor=4   exit 1

no-mocking-under-test internal/domain/lint/lint_test.go
  all four at 4                                                       exit 0
```

Every one of those verdicts is correct, which is what makes them worth writing down.

## The rules are not trivially permissive

The third result is the reassuring one. `lint_test.go` drives `Apply` through a
`claudecli.Ask` backed by a lookup table of canned JSON. That is a test double by
any ordinary definition, and a rule that pattern-matched on "is there a fake here"
would have flagged it. It did not, unanimously, because the *subject* — `Apply` —
is the real production function and the assertions are about what it actually
computed. The prompt's distinction between substituting the subject and
substituting a collaborator survived contact with real code.

The second result is the discriminating one: three of four tests in the same file
passed 4/4 while `TestApply` failed 0/4. A rule that always fires or never fires
would not produce that.

## The tension the tool exposes

`TestCollect` fails `named-for-behavior` because it names the unit under test.
`TestApply` fails `one-reason-to-fail` because its table answers several different
questions — unanimous pass, split vote, extra key, dropped name, votes below one —
rather than feeding many inputs to one question.

Both tests are, at the same time, exactly what the engineering constitution asks
for: R-3 requires a table-driven test whenever more than one case exists, and Go
convention names a test after the function it covers. Satisfying aritu's rules
here would mean splitting `TestApply` into eight behaviour-named functions, which
pushes back against the rule that produced the table in the first place.

So this is not a bug in the rules and not a bug in the tests. It is a real
disagreement between two defensible standards, and the tool's job was to surface
it rather than to settle it. Which standard wins is a maintainer decision, and
the options are roughly:

1. Rename and split the project's own tests to satisfy its rules, accepting more
   test functions and some duplicated table scaffolding.
2. Narrow `named-for-behavior` so a table-driven test named for its unit passes
   when its cases collectively pin one behaviour down.
3. Accept that aritu does not pass aritu, and record why.

Nothing was changed either way. Picking one is the point at which the tool stops
being a linter and starts being a policy.

## Worth keeping as a habit

Running the linter against its own source was cheaper than any of the fixtures
(three files, twelve verdict calls, about $0.07) and told us more than the
15-fixture selftest did. The fixtures prove a rule fires on cases written to make
it fire. Dogfooding proves it fires on code nobody wrote for it.
