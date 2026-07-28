# aritu

A shield against tests that do not earn their keep.

aritu is an LLM linter for tests. You give it a rule and a test file; it asks a
model whether each unit of that file satisfies the rule, several times over, and
reports how many runs agreed. Run it in CI or from a pre-commit hook to stop tests
that assert nothing, prove nothing, or mock away the thing they claim to cover.

**A rule you put in a prompt is a request.** Nothing reads the code back
afterwards, nothing disagrees, and nothing fails: whether the rule was followed
comes down to whether the model was inclined to follow it, and the only account of
that is a diff summarised by the same model that wrote it. aritu is that same
standard made checkable — a verdict per unit with a reason, several runs that have
to agree before anything passes, and a non-zero exit code out of a hook. Being told
becomes being held to it.

The same rules come back out as a document. `aritu rulebook` writes them out in
full, which is what you hand an agent as its `AGENTS.md` — and it is the very text
the model is judged against, so the standard an agent is given and the standard it
is held to are one file and cannot drift apart.

It does no code parsing. There is no AST, no matcher DSL, no suppression comments —
just files, a prompt, and a vote. That is also why it is not tied to a language: a
model reads the file and reports what it sees, and nothing in a rule names an
ecosystem. There is no language flag, and there is nothing to configure — the file
is in the prompt, and the model can see what it is.

## Build

```sh
go build -o aritu ./cmd/aritu
```

Requires Go 1.25+ and a Responses-compatible endpoint to point it at. Nothing is
shelled out to and no vendor CLI has to be installed or authenticated.

`aritu --help` lists everything; it is generated from the flags rather than written
beside them, so it cannot drift.

## Use

```sh
# every rule that is about it, over one file
aritu apply internal/parser/parser_test.go

# everything the enabled rules target
aritu apply

# every rule over everything a pattern matches
aritu apply 'internal/**/*_test.go'

# one rule, several patterns
aritu apply --rule proves-what-it-claims 'internal/**/*_test.go' 'cmd/**/*_test.go'

# the same rules over another ecosystem, with no flag to say so
aritu apply 'src/**/*.test.ts' 'tests/test_*.py'

# check the rules themselves against their own fixtures
aritu selftest --votes 4

# write the same rules out as instructions, before anything is judged
aritu rulebook > AGENTS.md
```

Patterns are globs, including `**` across directories. aritu expands them itself,
so a quoted pattern and one your shell already expanded reach the same set — the
run does not change because you moved it from zsh to bash. A pattern matching
nothing is an error rather than a quiet success, and overlapping patterns judge a
file once.

Naming no pattern sweeps everything the enabled rules target, so enabling a rule is
the whole of enabling it — there is no second list of paths to remember to widen.

A pattern says which files to consider; each rule's `targets` says which of them it
is handed. Name a document alongside a test file and the rules about tests judge
only the test file. Name one that **no** enabled rule is about and that is an error
and exit `2`, not a silent skip: nothing could have judged it, and a sweep that
quietly dropped it would read exactly like one that covered everything.

Omit `--rule` and every rule in the rules directory runs; repeat it to pick a few.

`apply` prints a report grouped by file, then by rule:

```
internal/parser/parser_test.go
  proves-what-it-claims
    ✓ TestParseConfig (extracts host before colon)
    ! TestParseConfig (rejects blank input) (1 of 2)
      one run read the name as stating an outcome and one did not
    ✗ TestParseConfig (host and port)
      names the input the case supplies rather than the outcome it protects
  tests-one-thing
    ✓ TestParseConfig

  2 passed  ·  2 failed  ·  1 split  ·  1 file, 2 rules, 2 votes  ·  4.1s
```

`✓` is a majority judging the unit to satisfy the rule, `✗` a majority judging
that it does not, and `!` an exact tie, which fails: half the votes is not a
majority. The count appears whenever the vote was not unanimous — the mark
carries the decision, the count how close it was.

Each block is printed as it finishes rather than at the end, so a sweep of any
size shows its verdicts as they land. They still print in reading order: a file
whose rules are all judged waits for the file above it, so the same run always
reports the same way rather than in whichever order the model happened to answer.
The line naming what the sweep covers goes to stderr before the first call, so a
redirected run captures the report alone and still shows a person it has started.

Colour goes to a terminal and nowhere else: a pipe, a redirect or `NO_COLOR` all
get plain text, so what you capture is what you read.

`--output json` gives the same run as data:

```json
{
  "reports": [
    {
      "rule": "proves-what-it-claims",
      "file": "internal/parser/parser_test.go",
      "votes": 2,
      "verdicts": { "TestParseConfig (host and port)": 0 },
      "reasons": {
        "TestParseConfig (host and port)": ["names the input the case supplies rather than the outcome it protects"]
      }
    }
  ]
}
```

Each verdict is how many of the `votes` runs judged that unit to satisfy the rule.
**A unit passes only at full agreement.** Every other count is a failure — `0` and
`3` of `4` alike.

The count is not a second verdict. It is how close the prompt is. `3` of `4` is a
rule nearly working; `0` of `4` on a test that should pass is a rule that is
broken. Both fail, and the number is what tells them apart while you tune.

`reasons` carries one sentence per dissenting run, for units that fell short. A
pass has nothing to explain, so it is omitted.

### Exit codes

| code | meaning |
|---|---|
| `0` | every unit of every rule over every file satisfies its rule by a majority of votes |
| `1` | one or more do not, whether the votes were unanimously against or tied |
| `2` | one or more targets could not be run, which outranks `1` |

`2` outranking `1` matters: a run where one file was unreadable and another
genuinely failed did not check everything, and reporting that as an ordinary rule
failure would let a hook treat a partial sweep as a complete one.

A split vote is not a third outcome and never becomes `2`. The rule is "all votes
agree"; a split does not meet it, so it fails. Filing it under "could not run"
would invite a hook to treat an unsure model as a tooling problem and skip past
exactly the test aritu exists to catch.

Output is always written before exiting, including on `2`. The counts are the whole
diagnostic; suppressing them on failure removes the reason to have them.

## The rulebook

`apply` tells you what is wrong after the file exists. `rulebook` writes the same
rules out beforehand, as the thing to follow rather than the thing to be measured
against:

```sh
aritu rulebook > AGENTS.md
```

```markdown
# Coding rules

The following rules must be abided by. Read them before writing anything they are
about, and check what you wrote against them before calling it done.

## Tests one thing

A test must have one reason to fail: it pins down a single behaviour, however many
assertions it takes to do so. ...
```

One heading per rule and the whole of that rule under it, in the order the rules
were enabled. It is the selection `apply` uses, so a rule you have not enabled is
not preached either, and `--rule` narrows the document the same way it narrows a
sweep.

**The heading is read off the directory name**, so a rule has one name and not two:
`my-pretty-rule` becomes `## My pretty rule`. Any parking prefix comes off first —
whether a rule is currently enforced says nothing about what it asks for.

**It is the same block the model is judged against.** `rule.Section` renders it
once and both readers get that rendering: the rulebook stacks the sections into a
document, and the verdict prompt frames one of them for the model. There is no
second, friendlier statement of a rule kept beside the real one — a summary and a
criterion drift apart the moment either is edited, and the drift is invisible
because nothing compares them. One text, read by the model that judges and by the
person who writes, cannot disagree with itself.

**No model is called.** The rules are already written down, so a repository can
produce its rulebook offline and with no `service.endpoint` configured — and the
same rule set always renders the same document, which is what makes it worth
committing. Point it at a `CLAUDE.md`, an `AGENTS.md`, a contributing guide, or a
prompt.

Write your prompts accordingly. A rule is read in both directions now, so it wants
to state the property and the shapes that break it plainly enough to be followed,
not just adjudicated.

## Configuration

`aritu.yml` at the repository root. Every key is optional except `service.endpoint`,
which has no default:

```yaml
service:
  endpoint: https://gateway.internal/v1
  auth_token_var: ARITU_TOKEN   # omit for an endpoint that ignores auth
  model: sonnet
  effort: medium                # omit to leave the endpoint's own default standing

votes: 2
jobs: 5
timeout: 10m
output: pretty

rules:
  dir: ./rules
  enabled: [proves-what-it-claims, tests-one-thing]   # omit for all

targets:
  tests: ['internal/**/*_test.go', 'cmd/**/*_test.go']   # replaces the built-in
  migrations: ['db/migrate/**/*.sql']                    # a kind of your own
```

`service.endpoint` is the base URL of any Responses-compatible API — a gateway, a
proxy, a self-hosted model. There is no default: a run that silently reached a
vendor's servers because a key was missing from the file would be a surprising
place for a repository's source to end up.

`model` and `effort` sit in the same block because which model names are valid is a
property of the endpoint serving them: a file that moved its endpoint and left its
model behind would be naming a model nobody serves. Both are still flags —
`--model opus` beats the file — because which model answers is worth trying once
from a shell in a way a gateway URL is not.

`auth_token_var` is **the name of an environment variable, never a token**. Its
value is read at startup and sent as `Authorization: Bearer <value>`. Omit it and
no `Authorization` header is sent at all. Name a variable that is unset and the run
stops before its first request, rather than paying for the typo as a wall of 401s
minutes into a sweep:

```
aritu: service.auth_token_var names $ARITU_TOKEN, which is not set
```

Precedence is built-in defaults, then the file, then flags — so `--votes 2` beats
`votes: 4` in the file.

`targets` is this repository's answer to which of its files are of a given kind. A
key matching a built-in kind **replaces** it outright, patterns and refinement
together: a repository overriding `tests` is saying it knows better than the naming
conventions aritu ships, and quietly keeping those as a filter over your patterns
would make the override a lie. A key nobody built in defines a new kind, which a
rule you write can then be about.

You do not have to override anything to keep the sweep off your own rules. Nothing
under the rules directory is ever derived into it: what sits there is rule material
rather than your work, and a `fail-` fixture is a bad test written on purpose, which
`selftest` judges against the expectation its directory name carries. Name one as a
pattern and it is still judged — that was asked for.

aritu searches upward from the working directory, so running from a subdirectory
behaves the same as running from the root, and `--config` points somewhere else
entirely. Paths inside the file resolve against the file; patterns you type
resolve against your shell. Each resolves in the frame it was written in.

An unknown key is an error naming it. A setting that silently does nothing is
worse than one that refuses to load.

## Rules

A rule is a directory. One rule per directory, under `rules/` by default
(`--rules` to point elsewhere).

```
rules/
  tests-one-thing/
    prompt.md
    fixtures/
      pass-go-table-of-inputs-to-one-behavior/
        discount.go
        discount_test.go
      pass-ts-grouped-tests-are-separate-units/
        cart.ts
        cart.test.ts
      fail-go-table-mixes-values-and-rejections/
        duration.go
        duration_test.go
```

### Parking a rule

A directory whose name starts with `_` is **parked**. It stays on disk, keeps its
prompt and its fixtures, and still runs when you name it — `--rule _tests-one-thing`
works exactly as it did — but no sweep picks it up on its own, and `rulebook` does
not preach it.

Parking is what to do with a rule you are not ready to enforce. Deleting it loses
the prompt that took the work to write, and listing the rules you *do* want in
`aritu.yml` means editing that list every time you add one.

It follows the same principle as the rules directory being kept out of the derived
sweep: what was **derived** leaves parked rules alone, and what was **asked for**
overrules that.

A rule's `prompt.md` is only its criterion. Everything every rule would otherwise
repeat — what judging means, that tests come in many shapes across ecosystems and
the behaviour is judged rather than the syntax, what a unit is, how to write a
reason — lives in `prompts/`, embedded in the binary, one folder per model call
and one file per unit kind:

```
prompts/
  splitter/                  # the call that lists a file's units
    instructions.md          #   the frame ahead of the file
    task.md                  #   the call to action after it
    function.md              #   what a function unit is
    test_case.md             #   what a test, a scope and a case are, and how to name one
  linter/                    # the call that judges the units
    instructions.md          #   what a verdict is, and how to write its reason
    task.md
    file.md                  #   one verdict covering the whole file
    function.md
    test_case.md
```

A prompt is those pieces joined as named sections — `<instructions>`, `<unit>`,
`<rule>`, `<units>`, `<file>`, `<task>` — and the rule's granularity picks the
unit kind that rides along. A rule at `file` or `function` granularity is told
nothing about tests, which is what makes a rule about anything else possible.

`prompt.md` carries YAML frontmatter and that criterion:

```markdown
---
targets: [tests]
include_source: false
granularity: test_case
---
A test's verdict must hang on the behaviour it is named for...
```

There is no key for the rule's name or its summary. The heading comes from the
directory name, and the body is the whole of what the rule says — to the model that
judges it and, through [`rulebook`](#the-rulebook), to whoever is about to write the
file. Nothing is stated twice, so nothing can disagree with itself.

`targets` names the kinds of file the rule is about, and is what makes a rule about
something other than tests runnable: aritu ships `tests`, `code` and `docs`, and
`aritu.yml` can replace any of them or add its own. `code` deliberately overlaps
`tests` — a test file has comments like any other source file — because kinds are
named matchers rather than a partition of the tree.

`targets` and `granularity` are required, and a missing one is an error naming the
key. Defaulting either silently changes which files reach the model or what it
judges, and nothing would report it. A
`targets` typo is the sharpest case: `[test]` for `[tests]` would match no file, run
nothing, and exit `0`. So an unknown kind fails when the rule is loaded, naming the
ones there are, before a single model call — and so does an empty list, which is a
rule that could never run.

`include_source` and `granularity` are also what decides which properties can share
a rule: two properties that disagree about either cannot share a verdict call
however similar they read.

### The seven rules

Roughly twenty-five separate properties make a test worth keeping. One rule per
property would be twenty-five verdict calls per file per vote and twenty-five
prompts to keep from contradicting each other. So they group, by **judgement** — the
same question asked of the same unit with the same evidence in front of the model:

| rule | granularity | `include_source` | |
|---|---|---|---|
| **`tests-one-thing`** | `function` | `false` | every assertion serves one claim |
| **`tests-behavior-not-implementation`** | `function` | `true` | binds to the caller's seam, asserts what and not how |
| **`proves-what-it-claims`** | `test_case` | `true` | remove the named behaviour and this unit goes red |
| **`self-contained`** | `file` | `false` | same verdict on any machine, in any order, at any time |
| **`readable`** | `function` | `false` | a stranger can tell what it establishes, and what broke |
| **`no-redundancy`** | `file` | `true` | no two tests assert one behaviour from one equivalence class |
| **`no-gaps`** | `file` | `true` | every distinct outcome the code can produce is asserted |

Per file, per vote: one enumeration call per granularity that needs one, shared by
every rule at that level, plus one verdict call per rule. Seven rules cost nine
calls. The three `file`-granularity rules are the cheap ones — `file` needs no
enumeration and its schema has one key.

The cost that is not calls: a `file`-granularity failure returns one verdict and one
sentence for a whole test file. `reasons` carries one entry per dissenting run, which
softens it, and coverage genuinely is a property of the whole file — but it is
thinner guidance than a per-test rejection, and that is a known trade.

### Finding the file under test

`include_source` decides what the model sees. It defaults to `false`, which sends
the file alone: finding a file under test is only meaningful for a rule about
tests, so a rule writes the key out only when it wants the pairing. With `true` the
implementation goes too — four of the seven test rules need it,
because whether an identifier is a subject's surface or its internals, whether two
inputs land in one equivalence class, and what outcomes are missing are all facts
about the code rather than about the test.

Resolution is a **search, not a derivation**: a mirrored source tree cannot be
reached by swapping a suffix, so aritu offers ordered candidates and takes the first
that exists.

| test file | candidates, in order |
|---|---|
| `parser_test.go` | `parser.go` |
| `parser.test.ts`, `parser.spec.ts`, `__tests__/parser.ts` | `parser.ts` beside it, then `test`/`tests`/`spec` swapped for `src` |
| `test_parser.py`, `parser_test.py` | `parser.py` beside it, then the same name under the package root |
| `ParserTest.java`, `ParserTests.java`, `TestParser.java` | `Parser.java` beside it, then `src/test/java` swapped for `src/main/java` |

When nothing exists the file is skipped and **reported**, naming every path aritu
looked at, rather than silently judged on partial context. If a real layout is
missing from the table it is a row to add, not a knob to expose.

### Granularity

`granularity` declares what a rule judges. The levels are defined by the role a
construct plays, never by the syntax that declares it — that is what lets one rule
set judge every ecosystem.

- **`function`** — one per function or method the file declares under its own
  name.
- **`test_case`** — one per **leaf** of that: one row of a table, one parametrised
  argument set, one subdivision declared inside the test, or the test itself when it
  has none.
- **`file`** — one, keyed by the path. Relations *between* tests live only here.

**Enclosing scopes are namespace prefixes, not levels.** A grouping block, a fixture
class or an outer suite qualifies a name and is joined into it with ` > `. It does
not change what is being judged.

| source | `function` | `test_case` |
|---|---|---|
| `func TestParseConfig` + table rows | `TestParseConfig` | `TestParseConfig (rejects blank input)` |
| `func TestParseConfig` + `t.Run("rejects")` | `TestParseConfig` | `TestParseConfig (rejects)` |
| `describe("formatDate")` + `it("pads days")` | `formatDate > pads days` | `formatDate > pads days` |
| `it.each` rows under that `it` | `formatDate > pads days` | `formatDate > pads days (2026-01-05)` |
| module-level `def test_x` | `test_x` | `test_x` |
| `class TestParser` + `parametrize` | `TestParser > test_x` | `TestParser > test_x (blank input)` |
| `class ParserTest` + `@Test rejectsBlank` | `ParserTest > rejectsBlank` | `ParserTest > rejectsBlank` |

The asymmetry in rows two and three is load-bearing. Two subtests in one function
are **one** `function`-level unit, so `tests-one-thing` sees both and rejects them —
correct, because that is one language's shape for two behaviours sharing a name. Two
`it`s in one `describe` are **two** `function`-level units, so each is judged alone
and both pass — also correct, because a `describe` is a grouping construct and
grouping is what it is for.

**The unit judged is the whole identifier**, which is also what a test runner prints
when a case fails. No part has to carry the meaning alone:

```
TestTrimsSurroundingWhitespaceFromEachTag (leading spaces)   ← the test states it
TestParseAddress (extracts host before colon)                ← the case states it
TestParseConfig (host and port)                              ← neither does: fails
```

That is why a table of many inputs to one behaviour is fine with input-named rows,
and why deleting your case names does not help — it pushes judgement up onto a name
that now has to carry every claim at once.

### Writing a rule

State **both poles**: the property a test must have, and the shapes that
disqualify it. A prompt given only one pole drifts toward answering everything the
same way, and a rule that never fires is indistinguishable from a rule that always
passes. A grouped rule needs the second pole more than a narrow one did, because it
has several ways to be wrong and several near-misses to spare.

Then prove it with fixtures. Each fixture is its own directory holding exactly one
test file and the implementation it covers — one directory per fixture because files
in one directory often share a namespace, so two fixtures both declaring `TestFoo`
would collide. The `pass-`/`fail-` prefix carries the expectation and a language
segment follows it, so the four coverage floors are visible in the listing:

1. Every rule carries at least one pass and one fail fixture **in each of Go,
   TypeScript, Python and Java**. This is the only thing that proves a prompt is not
   quietly shaped around one ecosystem.
2. Every disqualifying shape a prompt names has a fail fixture, in whichever
   language expresses that shape most naturally.
3. Every near-miss a prompt calls out as satisfying has a pass fixture. These are
   what stop a rule over-firing, and they are the ones that break during tuning.

`selftest` checks every fixture against its prefix:

```
rule: proves-what-it-claims  model: sonnet  votes: 4

FIXTURE                                          EXPECT  RESULT  TIME     VERDICTS
fail-java-asserts-only-that-nothing-threw        fail    hold    1m23.2s  parsesTheHostFromAnAddress=0 parsesThePortFromAnAddress=0
fail-py-numbered-cases-under-a-bare-test         fail    hold    30.7s    test_slugify (case 1)=0 test_slugify (case 2)=0 ...
pass-py-numbered-cases-under-a-stated-behaviour  pass    hold    45.1s    test_rejects_ports_above_the_maximum (case 1)=4 ...
pass-ts-terse-names-state-outcomes               pass    hold    14.1s    clampPercent > caps at 100=4 clampPercent > floors at 0=4 ...

14/14 fixtures hold in 2m27.7s
```

The two `numbered-cases` fixtures are the pair that keeps this rule honest: numbered case
labels are a violation under a test that states no behaviour and are not one under a test
that does, and only having both in the set proves the rule can tell them apart.

A fixture's files are named by its directory, never by a kind: a rule's `targets`
are not consulted anywhere in a `selftest` run. Otherwise a rule's self-test would
depend on the `aritu.yml` of whichever repository the rule happened to sit in.

`selftest` is `apply` in a loop — same prompt, same voting, same code path. It adds
only the comparison against the directory prefix and the table. It compares counts,
never exit codes: a `pass-` fixture holds at `votes`, a `fail-` fixture holds at
`0`. Anything between the poles fails, including a `fail-` fixture the model only
mostly rejected — a rule that needs a dissenting vote to fire is one bad test away
from missing.

## How it calls the model

Two kinds of call, both one non-streaming HTTP request to the configured endpoint,
with a replaced system prompt and no tools offered. Each asks for a strict
`json_schema` format, so the reply is the structured value rather than prose to be
salvaged.

The first — the splitter — lists the units in a file at one granularity. It
depends on the file and the granularity and nothing else — the rule's text never
reaches it — so **a file is enumerated once per granularity however many rules
judge it there**. Running seven rules over nine files pays for one listing per
file and level, not one per rule.

It asks for roles rather than for syntax: at test granularity, the smallest thing
the framework runs and reports under its own name, the scopes enclosing it, and
the leaves it subdivides into. No language is named to the model, which is what
lets one splitter prompt serve every ecosystem.

The second judges those units against one rule, and is handed the enumerated list
explicitly rather than left to re-derive it. Its schema is generated per call with
every unit named as a required key, so a duplicated, dropped or invented unit is
rejected by the endpoint's strict schema enforcement rather than becoming aritu's
problem.

If a verdict still arrives naming a unit the enumeration did not list, that is an
error and exit `2` — never a silent merge. Models are unreliable at exhaustive
enumeration, and a quietly dropped test is the precise failure this tool exists to
catch.

At `file` granularity the first call is skipped entirely: the unit is the path,
which costs nothing to know and cannot be disagreed with.

## Flags

| flag | default | |
|---|---|---|
| `--rule` | all | rule to run; repeat for several |
| `--config` | search upward | config file to use instead of `aritu.yml` discovery |
| `--model` | `sonnet` | model name sent to the endpoint |
| `--output` | `pretty` | `pretty` for reading, `json` for parsing |
| `--votes` | `1` | rounds run per unit; a strict majority must agree it passes |
| `--jobs` | `5` | model calls allowed in flight at once |
| `--effort` | — | reasoning effort; empty leaves the endpoint default |
| `--rules` | `./rules` | directory holding one subdirectory per rule |
| `--timeout` | `10m` | deadline for the whole run, so a hung endpoint cannot hang a commit hook |
| `--debug` | off | print each prompt on stderr instead of calling the model — placeholder units stand in for the splitter's answer, no report is written, no endpoint is needed |

`--jobs` bounds concurrency at the one seam every model call passes through, so
fixture-level and vote-level parallelism cannot multiply into a process storm.

`--timeout` covers the entire run rather than a single call, so it scales with how
many fixtures a rule has. A large corpus at high `--votes` may need more than the
default.
