# aritu

A shield against tests that do not earn their keep.

aritu is an LLM linter for Go tests. You give it a rule and a test file; it asks a
model whether each unit of that file satisfies the rule, several times over, and
reports how many runs agreed. Run it in CI or from a pre-commit hook to stop tests
that assert nothing, name nothing, or mock away the thing they claim to cover.

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
aritu apply one-reason-to-fail internal/parser/parser_test.go

# run a rule against its whole fixture corpus and check each result
# against the expectation its directory name carries
aritu selftest one-reason-to-fail --votes 4
```

`apply` prints the counts as JSON:

```json
{
  "rule": "named-for-behavior",
  "file": "internal/parser/parser_test.go",
  "votes": 2,
  "verdicts": {
    "TestParseConfig (extracts host before colon)": 2,
    "TestParseConfig (host and port)": 0
  },
  "reasons": {
    "TestParseConfig (host and port)": [
      "names the input the case supplies rather than the outcome it protects, so a reader seeing it fail learns nothing about what regressed"
    ]
  }
}
```

Each value is how many of the `votes` runs judged that unit to satisfy the rule.
**A unit passes only at full agreement.** Every other count is a failure — `0` and
`3` of `4` alike.

The count is not a second verdict. It is how close the prompt is. `3` of `4` is a
rule nearly working; `0` of `4` on a test that should pass is a rule that is
broken. Both fail, and the number is what tells them apart while you tune.

`reasons` carries one sentence per dissenting run, for units that fell short. A
unanimous pass has nothing to explain, so it is omitted.

### Exit codes

| code | meaning |
|---|---|
| `0` | every unit unanimously satisfies the rule |
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
rules/
  base.md                        # shared prompt, prepended to every rule
  one-reason-to-fail/
    prompt.md
    fixtures/
      pass-single-assert/
        scenario.go
        scenario_test.go
      pass-multiple-asserts-one-behavior/
      fail-two-unrelated-behaviors/
      fail-act-assert-chain/
```

`base.md` holds what every rule would otherwise repeat: that Go tests come in many
shapes and the behaviour is judged rather than the syntax, what a unit is, and how
to write a reason. A rule's `prompt.md` is then only its criterion.

`prompt.md` carries YAML frontmatter and that criterion:

```markdown
---
include_source: false
granularity: test
---
A test's name must say which behaviour breaks when the test fails...
```

`include_source` decides what the model sees. With `false` only the test file is
sent. With `true` the file under test goes too, resolved by Go convention
(`parser_test.go` → `parser.go`); if it cannot be found, the file is skipped and
reported rather than silently judged on partial context.

Ships with three rules:

- **`named-for-behavior`** — named for the behavior it protects, specifically
- **`one-reason-to-fail`** — one behavior, however many assertions
- **`no-mocking-under-test`** — doesn't mock the thing under test

### Granularity

`granularity` declares what a rule judges. The levels form a scale, each a
refinement of the one above:

| level | keys returned | key is |
|---|---|---|
| `file` | one | `internal/parser/parser_test.go` |
| `function` | one per `func Test*` | `TestParseConfig` |
| `test` | one per independently nameable leaf | `TestParseConfig (rejects blank input)` |

At `test` granularity a table row, a `map[string]struct{...}` key and a `t.Run`
subtest are each a leaf; a plain function that declares none of them is one leaf by
itself.

**The unit judged is the whole identifier**, which is also what Go prints when a
case fails. Neither half has to carry the meaning alone:

```
TestTrimsSurroundingWhitespaceFromEachTag (leading spaces)   ← function states it
TestParseAddress (extracts host before colon)                ← case states it
TestParseConfig (host and port)                              ← neither does: fails
```

That is why a table of many inputs to one behaviour is fine with input-named rows,
and why deleting your case names does not help — it pushes judgement up onto a
function name that now has to carry every claim at once.

Both keys are required. Defaulting either one silently changes what the model sees
or what it judges, and nothing would report it.

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
rule: named-for-behavior  model: sonnet  votes: 4

FIXTURE                                   EXPECT  RESULT  VERDICTS
fail-namespace-function-input-rows        fail    hold    TestParseClock (hour of 24)=0 TestParseClock (midnight)=0 ...
fail-numbered-rows                        fail    hold    TestRunningTotals (case 1)=0 TestRunningTotals (case 2)=0 ...
pass-namespace-function-behavioural-rows  pass    hold    TestParseSemver (rejects a version missing the patch component)=4 ...
pass-subtests-named-for-behaviour         pass    hold    TestDeduplicate (drops values that already appeared)=4 ...

11/11 fixtures hold
```

`selftest` is `apply` in a loop — same prompt, same voting, same code path. It adds
only the comparison against the directory prefix and the table. It compares counts,
never exit codes: a `pass-` fixture holds at `votes`, a `fail-` fixture holds at
`0`. Anything between the poles fails, including a `fail-` fixture the model only
mostly rejected — a rule that needs a dissenting vote to fire is one bad test away
from missing.

## How it calls the model

Two calls per file, both through the `claude` CLI via `exec`, with tools disabled
and a replaced system prompt. The first enumerates the units at the rule's declared
granularity; the second judges them, and is handed that list explicitly rather than
left to re-derive it.

If the second returns a unit the first did not list, or drops one it did, that is an
error and exit `2` — never a silent merge. Models are unreliable at exhaustive
enumeration, and a quietly dropped test is the precise failure this tool exists to
catch.

At `file` granularity the first call is skipped entirely: the unit is the path, which
costs nothing to know and cannot be disagreed with.

## Flags

| flag | default | |
|---|---|---|
| `--model` | `sonnet` | model passed to the claude CLI |
| `--votes` | `2` | rounds that must all agree before a unit passes |
| `--jobs` | `5` | model calls allowed in flight at once |
| `--effort` | — | reasoning effort; empty leaves the CLI default |
| `--rules` | `./rules` | directory holding `base.md` and one subdirectory per rule |
| `--claude` | `claude` | claude CLI binary to invoke |
| `--timeout` | `10m` | deadline for the whole run, so a hung CLI cannot hang a commit hook |

`--jobs` bounds concurrency at the one seam every model call passes through, so
fixture-level and vote-level parallelism cannot multiply into a process storm.

`--timeout` covers the entire run rather than a single call, so it scales with how
many fixtures a rule has. A large corpus at high `--votes` may need more than the
default.
