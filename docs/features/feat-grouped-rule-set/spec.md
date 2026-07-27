---
slug: grouped-rule-set
type: feat
status: draft
created: 2026-07-27
---

------

# Feature: A grouped, language-agnostic rule set

## Purpose

aritu ships three rules. They are good rules and they cover almost nothing.

A test earns its keep by satisfying roughly twenty-five separate properties — it tests
one thing, it survives a refactor, its verdict actually hangs on the behaviour it is
named for, it does not read the wall clock, the suite has no hole where the error path
should be. The current three cover three of them. The obvious next move is one rule per
property, and it is the wrong move: twenty-five rules is twenty-five verdict calls per
file per vote, twenty-five prompts to keep from contradicting each other, and twenty-five
fixture sets to keep green.

So the properties group. Seven rules, each holding the handful of failure shapes that
share a **judgement**: the same question asked of the same unit with the same evidence in
front of the model. `tests-one-thing` does not need the implementation; `no-gaps` is
meaningless without it. That is the seam the grouping follows, and it is not cosmetic —
granularity and `include_source` are per-rule frontmatter, so two properties that disagree
about either cannot share a call however similar they read.

Second, and independently: the rules are Go rules. Not because the properties are Go
properties — every one of them is true of a Jest suite — but because the prompts say
"Go test file", the enumeration prompt says `*testing.T` and `t.Run`, and
`SourcePathFor` knows exactly one convention, `parser_test.go` → `parser.go`. Point aritu
at a TypeScript file today and the enumeration call is asked to find `func Test*` in it.

Nothing about aritu's design is Go-specific. It does no parsing; a model reads the file
and reports what it sees, and models read TypeScript. The Go assumptions are hardcoded
strings in three places, and removing them is most of what makes the tool general.

## Scope

### The seven rules

The three existing rule directories are deleted. Seven replace them:

| rule | granularity | `include_source` |
|---|---|---|
| `tests-one-thing` | `function` | `false` |
| `tests-behavior-not-implementation` | `function` | `true` |
| `proves-what-it-claims` | `test` | `true` |
| `self-contained` | `file` | `false` |
| `readable` | `function` | `false` |
| `no-redundancy` | `file` | `true` |
| `no-gaps` | `file` | `true` |

Existing material is salvaged, not rewritten from nothing. `one-reason-to-fail`'s body is
the core of `tests-one-thing`; `named-for-behavior`'s is the first half of
`proves-what-it-claims`; `no-mocking-under-test`'s is one of four shapes under
`tests-behavior-not-implementation`. Their fixtures move with them. What is deleted is the
three-rule *set*, not three prompts' worth of thinking.

#### 1. `tests-one-thing` — `function`, no source

Every assertion in the test serves one claim. The number of assertions is not the subject:
three assertions on the three fields of one returned value are one behaviour.

Disqualifying shapes: several unrelated behaviours in one function; a table or a set of
subtests whose cases answer different questions rather than feeding different inputs to one
question; act-assert-act-assert, a scenario walkthrough where every act after the first is a
new behaviour wearing the previous test's name.

The near-misses that must keep passing are what stop this rule over-firing: many assertions
walking one composite result, a guard assertion that setup succeeded before the real act, a
value and its error asserted together, and a table of many inputs to one behaviour.

`function` because how many behaviours a loop pins down is a property of the loop. Asked of
a single row the question is close to vacuous. No source: the acts and the claims are both
in the test.

#### 2. `tests-behavior-not-implementation` — `function`, source required

The test binds to the seam a caller would use, and asserts what the code produced rather
than how it produced it.

Disqualifying shapes: reaching past the subject into its internal state; asserting details a
refactor would break — the algorithm chosen, the internal data structure, the order
collaborators were called in; asserting a double's call count or recorded arguments in place
of an outcome; substituting the subject itself.

**Visibility keywords do not decide this.** A Go test in the same package calls unexported
functions routinely, Python has no privacy at all, and a Java test may sit in the same
package deliberately. The violation is reaching *past* the subject into state it owns, not
calling something the language happens to spell without `export`. When the unexported
function *is* the subject, testing it directly is the rule being satisfied.

Two near-misses must keep passing. A genuine out-of-scope collaborator replaced by a fake —
a clock, an HTTP transport, a payment gateway — while the real subject runs. And a subject
whose whole job is to hand something to a collaborator, where the recorded payload *is* the
subject's output rather than a stand-in for it.

Source required: whether an identifier is the subject's surface or its internals is a fact
about the implementation, and no reading of the test alone settles it.

#### 3. `proves-what-it-claims` — `test`, source required

The test's verdict must actually hang on the behaviour it is named for. This absorbs
`named-for-behavior` and adds its missing half: a precise name is worthless if the body
would stay green when the named behaviour broke.

The decisive question, asked of the composite identifier: **if this behaviour were removed
from the implementation, would this unit go red?** No is a fail, and it is a fail for a name
so vague nothing could contradict it as much as for a body that never exercises what it
promises.

Disqualifying shapes: a name that states the unit, the input or the ceremony rather than an
outcome — `TestValidate`, `TestBug123`, `TestHappyPath`, `TestWithEmptyConfig`; a body that
does not exercise the named behaviour; an assertion that nothing threw, which proves the
code ran and not what it did; an assertion of what the language or the signature already
guarantees; an expected value that is a snapshot of current output rather than derived from
intent, so the code is its own oracle.

The snapshot shape needs care, because golden-file testing is legitimate. The violation is an
expectation that **could not have been written before the code ran** and that nothing in the
test justifies — a pasted hash, a checked-in blob no reviewer reads. A golden file whose
content a reader can evaluate is not this.

`test` because the claim lives in the leaf: a table row is where the name and the assertion
meet. Source required for the mutation question, which cannot be answered from the test.

#### 4. `self-contained` — `file`, no source

The suite produces the same verdict on any machine, in any order, at any time.

Disqualifying shapes: reading the real clock, the network, the filesystem or a random
source; a test that only passes because an earlier one ran; mutable state shared across
tests.

The distinction that makes this usable: a test that **creates** the state it uses is
self-contained; one that **depends on** state it did not create is not. A temporary
directory made and torn down per test is fine. A hardcoded `/tmp` path, a checked-in fixture
read from a relative path, the user's home directory, a listening port chosen by hand are
not. Likewise a fresh instance per test is fine and a package-level variable the tests
assign is not.

`file` because ordering dependence and shared mutable state are relations *between* tests
and cannot be expressed at any finer level. No source: the ambient dependencies that
actually make a suite flaky are named in the test file — the clock read, the socket dialled,
the path opened, the global assigned. A subject that reaches for the clock behind the test's
back is real, but it is a defect in the subject, and at `file` granularity requiring source
would mean a whole file goes unjudged whenever resolution misses.

#### 5. `readable` — `function`, no source

Someone who has never seen this test can tell what it establishes, from what, and learn
what broke from the failure alone.

Disqualifying shapes: setup that constructs things the behaviour does not depend on; the
values that matter buried in fixture noise; a failure that does not say what broke.

That last shape is the one that must be written framework-neutrally rather than as "call
`t.Errorf` with a message". pytest rewrites assertions, Jest prints a structured diff, JUnit
prints expected and actual, and Go prints whatever you wrote and nothing else. The property
is whether **the reader of a failure can tell which assertion failed and what it was
checking** — so a bare assert is fine where the framework supplies that, and is not fine in
a loop over unlabelled cases where the framework cannot.

Length carries no penalty and helper calls are not noise when the helper's name says what it
builds.

#### 6. `no-redundancy` — `file`, source required

No two tests in the file assert the same behaviour with different examples drawn from the
same equivalence class.

The near-misses matter more here than the violation, because this rule fires wrongly very
easily. A boundary pair — at the limit and just past it — is two classes and not
redundancy. Two tests with identical shape asserting different behaviours are not
redundancy. A table of many inputs to one behaviour is one test.

Source required: whether two inputs land in the same class is a fact about the
implementation's branches. Given `parse("")` and `parse("   ")`, only the code says whether
that is one case tested twice or two cases tested once.

#### 7. `no-gaps` — `file`, source required

Every distinct outcome the code under test can produce is asserted somewhere.

Disqualifying shapes: a missing happy path; a reachable error path never exercised;
boundaries untested — at the limit and on either side; degenerate inputs missing — empty,
nil, zero, one, max; hostile inputs missing where the surface is exposed to them; any
distinct outcome the code can produce that nothing asserts.

Two carve-outs, both of which this repository's own code would otherwise trip:

- **Unreachable branches are not gaps.** A `default: panic("unknown: " + t)` guarding an
  exhaustive switch exists to fail loudly on a state the type system forbids. Demanding a
  test for it would punish the pattern the constitution requires.
- **Hostile inputs are demanded only where the surface is exposed.** A parser reading
  untrusted bytes owes them; an internal helper called from three known sites does not.

A private helper covered through its caller is covered.

`file` because coverage is a property of the set. Source required, obviously: what is
missing is defined entirely by what exists.

### What this costs

Stated plainly, because the number moves the wrong way in one direction and the right way
in another.

Per file, per vote: one enumeration call, shared by every rule, plus one verdict call per
rule. Three rules cost four calls. **Seven rules cost eight.** Against today's rule set this
is twice the calls; against one rule per property — the alternative this feature exists to
avoid — it is roughly a third.

The three `file`-granularity rules are the cheap ones. `file` needs no enumeration and its
schema has one key, so `self-contained`, `no-redundancy` and `no-gaps` cost one call each
with nothing to cross-check.

The cost that is not calls: a `file`-granularity failure returns one verdict and one
sentence for an entire test file. `no-gaps` saying `internal/parser/parser_test.go: 0` with
a single reason is thinner guidance than a per-test rejection. The `reasons` array carries
one entry per dissenting run, which softens it, and coverage genuinely is a property of the
whole file. Named here so it is a known trade rather than a surprise.

### Language agnosticism

Three hardcoded Go assumptions, removed in three different ways.

#### The prompts stop naming a language

`rules/base.md` opens `You are judging the units of a Go test file`. Its "Test shapes"
section lists table-driven, `t.Run`, arrange-act-assert, golden-file. Both become
framework-neutral: the shapes section names the shapes across ecosystems — a table of cases,
`describe`/`it` nesting, parametrised cases, class-per-fixture, arrange-act-assert,
golden-file, integration tests standing up real dependencies — and reasserts what it already
says, that the behaviour a test pins down is judged and never the syntax it is written in.

The two enumeration prompts in `lint.go` are the worse offenders, because they instruct
rather than describe: `buildFunctionNamesPrompt` asks for "a top-level func whose name
begins with Test and which takes a single `*testing.T` parameter", and
`buildTestNamesPrompt` asks for "a subtest declared with `t.Run`". Both are rewritten
against roles rather than syntax (below).

**No language is ever named to the model, and no language flag exists.** The file is in the
prompt; the model can see what it is. Adding a `language:` key would be asking the operator
to tell a reader what it is reading.

#### The two levels, defined by role

The composite identifier — `Function (case)` — already generalises, and `splitUnit`,
`UnitsFor`, `distinctFunctions` and the key derivation all keep working untouched. What
changes is the definition each level is enumerated against.

**Function level: the smallest thing the framework runs and reports as a named test.** Go
`func TestX`, Jest/Vitest `it`/`test`, pytest `def test_x`, JUnit `@Test` method.

**Test level: one leaf of that.** A parametrised case — a table row, an `it.each` row, a
`parametrize` case, a `@ParameterizedTest` argument set — or a subdivision declared inside
the test, or the declared test itself when it has neither.

**Enclosing scopes are namespace prefixes, not levels.** A Jest `describe`, a pytest
`class Test*`, a JUnit class qualify the name and are joined into the function half with
` > `. They do not change what is being judged.

| source | `function` | `test` |
|---|---|---|
| Go `func TestParseConfig` + table rows | `TestParseConfig` | `TestParseConfig (rejects blank input)` |
| Go `func TestParseConfig` + `t.Run("rejects")` | `TestParseConfig` | `TestParseConfig (rejects)` |
| Jest `describe("formatDate")` + `it("pads days")` | `formatDate > pads days` | `formatDate > pads days` |
| Jest `it.each` rows under that `it` | `formatDate > pads days` | `formatDate > pads days (2026-01-05)` |
| pytest module-level `def test_x` | `test_x` | `test_x` |
| pytest `class TestParser` + `parametrize` | `TestParser > test_x` | `TestParser > test_x (blank input)` |
| JUnit `class ParserTest` + `@Test rejectsBlank` | `ParserTest > rejectsBlank` | `ParserTest > rejectsBlank` |

This mapping is load-bearing rather than cosmetic, and the asymmetry in row two versus row
three is the reason. Two `t.Run` subtests in one Go function are **one** function-level unit,
so `tests-one-thing` sees both and rejects them — which is correct, because that is Go's
shape for two behaviours sharing a name. Two `it`s in one `describe` are **two**
function-level units, so `tests-one-thing` judges each alone and passes both — which is also
correct, because a `describe` is a grouping construct and grouping is what it is for.
Collapsing `describe` onto `func Test` by shape rather than by role would make idiomatic
Jest fail a rule it does not violate.

One code consequence: `splitUnit` takes the **first** ` (` in a name. A namespace path is
now arbitrary prose, so it must take the last.

#### Path resolution is filesystem work, not model work

`SourcePathFor` maps `_test.go` → `.go` and returns one path. Four of seven rules need the
implementation, so this becomes the load-bearing piece.

A new library package — `internal/lib/testpath` — holds one table of naming conventions and
two pure functions over it:

- `IsTestFile(path) bool`
- `SourceCandidates(testPath) []string` — ordered candidate paths, most likely first

The domain layer checks which candidate exists. The library stays pure and the IO stays at
the boundary.

| language | test file | candidates, in order |
|---|---|---|
| Go | `parser_test.go` | `parser.go` |
| TypeScript / JavaScript | `parser.test.ts`, `parser.spec.ts`, `__tests__/parser.ts` | `parser.ts` beside it, then the same path with `test`/`tests`/`spec` swapped for `src` |
| Python | `test_parser.py`, `parser_test.py` | `parser.py` beside it, then the same name under the package root |
| Java | `ParserTest.java`, `ParserTests.java`, `TestParser.java` | `Parser.java` beside it, then `src/test/java` swapped for `src/main/java` |

Resolution stays a **search, not a derivation**, because Java's mirrored tree and
TypeScript's `src`/`test` split cannot be reached by a suffix swap. First candidate that
exists wins. When none exists the behaviour is unchanged from today: the file is skipped and
reported, never judged on partial context.

This makes an existing failure mode much more common — four rules need source instead of
one, and three of the four conventions above are searches. The mitigation is that it stays a
*reported* skip rather than a silent one, and that the three rules a project will hit first
(`tests-one-thing`, `self-contained`, `readable`) need no source at all.

`findTestFile`, which currently demands exactly one `*_test.go` per fixture directory, uses
`IsTestFile`. The "exactly one" invariant stays: it is what makes a fixture unambiguous.

### Fixtures

Fixture directory names gain a language segment after the pass/fail prefix:

```
rules/tests-behavior-not-implementation/fixtures/
  pass-go-fake-clock-collaborator/          scenario.go            scenario_test.go
  pass-ts-asserts-payload-handed-to-fake/   scenario.ts            scenario.test.ts
  fail-py-asserts-mock-call-count/          scenario.py            test_scenario.py
  fail-java-reaches-into-private-field/     Scenario.java          ScenarioTest.java
```

`ParseExpectation` only reads the `pass-`/`fail-` prefix, so this needs no code change. One
fixture per directory stays the rule for every language, not only the two where a shared
package would collide.

Three coverage floors, all of which `selftest` enforces by construction:

1. **Every rule carries at least one pass and one fail fixture in each of Go, TypeScript,
   Python and Java.** This is the only thing that proves a prompt is not quietly Go-shaped,
   and it is not negotiable per rule. A prompt that passes its Go fixtures and fails its
   pytest ones is a prompt describing `func Test*`.
2. **Every disqualifying shape a prompt names has a fail fixture**, in whichever language
   expresses that shape most naturally. Asserting call counts belongs in a Jest or Mockito
   fixture; a nameless table belongs in Go.
3. **Every near-miss a prompt calls out as satisfying has a pass fixture.** These are what
   stop a rule over-firing, and they are the fixtures that will actually break during
   tuning — a fake collaborator, a boundary pair under `no-redundancy`, a `default: panic`
   under `no-gaps`.

That is 56 fixtures at the floor and more where a rule has many shapes. `selftest` over the
whole set at four votes is a slow command; it is a tuning tool, not a hook.

### Config and docs

`aritu.yml` **names no rules today** — it carries `votes: 1` and two `include` globs, and
nothing else. There is nothing stale in it to correct, and its Go-shaped `include` patterns
are correct because this repository is Go. It is listed here so the absence is recorded
rather than assumed.

The stale rule names are all in `README.md`: the config example at line 141, the rules list
at lines 198-200, and the worked examples at lines 34, 52, 58, 84 and 246. Sections needing
more than a name swap: **Rules** (seven, with the granularity/`include_source` table),
**Granularity** (the role-based definitions and the mapping table), and the opening
paragraph, which says "an LLM linter for Go tests".

`cmd/aritu/main.go`'s `description` constant says the same thing and is what `--help`
prints.

The rule names appearing throughout `*_test.go` files are test data — arbitrary strings
standing in for a rule name — and are not a reference to the shipped rules. They stay.

## Out of scope

- **Batching rules into one call.** Rules sharing a `(granularity, include_source)` pair
  could be judged together: the seven collapse to five groups, so eight calls become six.
  A 25% saving, bought with a verdict key that has to carry a rule dimension, reasons that
  have to be attributed back per rule, and a fixture set that no longer isolates one rule
  from another. Not worth it at seven rules. Worth revisiting at twenty.
- **A `language:` key, anywhere.** Not in frontmatter, not in `aritu.yml`, not as a flag.
  The model reads the file; the filesystem layer resolves paths; neither needs to be told.
- **Configurable source resolution.** If the built-in candidate list misses a real layout,
  that is a candidate to add to the table, not a knob to expose.
- **`package` granularity.** Judging several files together is where "the suite covers the
  exported surface" properly lives, and `no-gaps` at `file` is the approximation available
  without multi-file input. Still a real feature, still not this one.
- **Still no parsing.** Enumeration remains a model call in every language.
- **Rewriting aritu's own tests to satisfy the new rules.** Seven rules over this repository
  will find a great deal. Acting on it is a separate decision.

## Constraints

- Bound by `CONSTITUTION.md`. Specifically: the naming-convention table is data driving a
  lookup, not a switch on a language value (R-4, R-13); `testpath` sits in `internal/lib`
  and is imported by `internal/domain/rule`, never the reverse (R-7); the old rule
  directories are deleted rather than kept beside the new ones (R-17).
- `SourcePathFor` returning a single path is replaced, not extended. Two functions that both
  answer "where is the source" would drift (R-2).
- No change to `verdicts`, `reasons`, the exit codes, or `selftest`'s hold semantics. This
  feature changes what is asked, not what comes back.
- Every rule declares `granularity` and `include_source` explicitly. Both stay required and
  neither is ever defaulted.
- Every prompt keeps the two-pole structure: the property a test must have, and the shapes
  that disqualify it. A grouped rule needs the second pole more than a narrow one did,
  because it now has several ways to be wrong and several near-misses to spare.
- Every prompt should have guidance on relaying what causes the wrong state so that the calling agent clearly knows where to look

## Done criteria

`rules/` holds exactly seven directories and none of the three old names.

`rules/base.md` and both enumeration prompts contain no occurrence of `Go`, `_test.go`,
`testing.T`, `t.Run` or `func Test`. Neither does `--help`.

`selftest` holds for every fixture of every rule, on sonnet at four votes — including,
for every rule, its TypeScript, Python and Java fixtures.

`aritu apply` over a `.test.ts` file, a `test_*.py` file and a `*Test.java` file produces
verdicts for all seven rules, with no flag naming a language and no fixture-only
scaffolding involved.

And the case that motivated the split: a test named `TestParsesPortFromAddress` whose body
asserts only that no error was returned passes `proves-what-it-claims`'s ancestor today and
must fail it now. A well-named test that proves nothing is the exact hole three rules left
open.
