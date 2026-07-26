# aritu

A shield against tests that do not earn their keep.

aritu is an LLM linter for Go tests. You give it a rule and a test file; it asks a
model whether each test function in that file satisfies the rule, several times
over, and reports how many runs agreed. Run it in CI or from a pre-commit hook to
stop tests that assert nothing, name nothing, or mock away the thing they claim to
cover.

It does no code parsing. There is no AST, no matcher DSL, no suppression comments —
just files, a prompt, and a vote.

## Build

```sh
go build -o aritu ./cmd/aritu
```

Requires Go 1.25+ and the `claude` CLI on `PATH`, already authenticated.

## Use

```sh
# judge one file against one rule
aritu apply one-reason-to-fail internal/parser/parser_test.go --model sonnet --votes 4

# run a rule against its whole fixture corpus and check each result
# against the expectation its directory name carries
aritu selftest one-reason-to-fail --model sonnet --votes 4
```

`apply` prints the counts as JSON:

```json
{
  "rule": "one-reason-to-fail",
  "file": "internal/parser/parser_test.go",
  "votes": 4,
  "verdicts": {
    "TestParsesHostAndPort": 4,
    "TestRejectsMalformedPort": 0
  }
}
```

Each value is how many of the `votes` runs judged that test to satisfy the rule.
**A test passes only at full agreement.** Every other count is a failure — `0` and
`3` of `4` alike.

The count is not a second verdict. It is how close the prompt is. `3` of `4` is a
rule nearly working; `0` of `4` on a test that should pass is a rule that is
broken. Both fail, and the number is what tells them apart while you tune.

### Exit codes

| code | meaning |
|---|---|
| `0` | every test function unanimously satisfies the rule |
| `1` | one or more do not, whether the votes were unanimously against or split |
| `2` | could not run — model unreachable, file not found, bad response, name/verdict mismatch |

A split vote is not a third outcome and never becomes `2`. The rule is "all votes
agree"; a split does not meet it, so it fails. Filing it under "could not run"
would invite a hook to treat an unsure model as a tooling problem and skip past
exactly the test aritu exists to catch.

Output is always written before exiting, including on `2`. The counts are the whole
diagnostic; suppressing them on failure removes the reason to have them.

## Rules

A rule is a directory. One rule per directory, under `rules/` by default
(`--rules` to point elsewhere).

```
rules/one-reason-to-fail/
  prompt.md
  fixtures/
    pass-single-assert/
      scenario.go
      scenario_test.go
    pass-multiple-asserts-one-behavior/
    fail-two-unrelated-behaviors/
    fail-act-assert-chain/
```

`prompt.md` carries YAML frontmatter and the criterion:

```markdown
---
include_source: false
---
A test must have one reason to fail: it pins down a single behaviour, however
many assertions it takes to do so...
```

`include_source` decides what the model sees. With `false` only the test file is
sent. With `true` the file under test goes too, resolved by Go convention
(`parser_test.go` → `parser.go`); if it cannot be found, the file is skipped and
reported rather than silently judged on partial context.

Ships with three rules:

- **`named-for-behavior`** — named for the behavior it protects, specifically
- **`one-reason-to-fail`** — one behavior, however many assertions
- **`no-mocking-under-test`** — doesn't mock the thing under test

### Writing a rule

State **both poles**: the property a test must have, and the shapes that
disqualify it. A prompt given only one pole drifts toward answering everything the
same way, and a rule that never fires is indistinguishable from a rule that always
passes.

Then prove it with fixtures. Each fixture is its own directory — Go files in one
directory share a package, so two fixtures both declaring `TestFoo` would not
compile otherwise. The `pass-`/`fail-` prefix carries the expectation, and
`selftest` checks every fixture against it:

```
rule: one-reason-to-fail  model: sonnet  votes: 4

FIXTURE                             EXPECT  RESULT  VERDICTS
fail-act-assert-chain               fail    hold    TestStackHandlesPushingPoppingAndDraining=0
fail-two-unrelated-behaviors        fail    hold    TestLowercasesSurroundedNamesAndRejectsBlankOnes=0
pass-multiple-asserts-one-behavior  pass    hold    TestSplitsAWellFormedAddressIntoHostAndPort=4
pass-single-assert                  pass    hold    TestReturnsTheFallbackWhenTheKeyIsMissing=4
pass-table-driven-one-behavior      pass    hold    TestPullsValuesOutsideThePercentageRangeToTheNearestBound=4

5/5 fixtures hold
```

`selftest` is `apply` in a loop — same prompt, same voting, same code path. It adds
only the comparison against the directory prefix and the table. It compares counts,
never exit codes: a `pass-` fixture holds at `votes`, a `fail-` fixture holds at
`0`. Anything between the poles fails, including a `fail-` fixture the model only
mostly rejected — a rule that needs a dissenting vote to fire is one bad test away
from missing.

## How it calls the model

Two calls per file, both through the `claude` CLI via `exec`, with tools disabled
and a replaced system prompt. The first enumerates the test functions; the second
judges them. If the second returns a name the first did not, or drops one it did,
that is an error and exit `2` — never a silent merge. Models are unreliable at
exhaustive enumeration, and a quietly dropped test is the precise failure this tool
exists to catch.

## Flags

| flag | default | |
|---|---|---|
| `--model` | `sonnet` | model passed to the claude CLI |
| `--votes` | `4` | rounds that must all agree before a test passes |
| `--effort` | — | reasoning effort; empty leaves the CLI default |
| `--rules` | `./rules` | directory holding one subdirectory per rule |
| `--claude` | `claude` | claude CLI binary to invoke |
| `--timeout` | `10m` | deadline for the whole run, so a hung CLI cannot hang a commit hook |
