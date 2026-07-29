# aritu

A shield against code that does not earn its keep.

![Claude is an expert](assets/claude-is-an-expert.jpeg)

aritu is an LLM linter. You give it a rule written in prose and a file; it asks a
model whether each unit of that file satisfies the rule, several times over, and
reports how many runs agreed. Run it in CI or from a pre-commit hook to hold code
to the standards no parser can check: the comment that restates its code, the name
that says nothing, the file doing three jobs, the test that proves nothing.

**A rule you put in a prompt is a request.** Nothing reads the code back
afterwards, nothing disagrees, and nothing fails: whether the rule was followed
comes down to whether the model was inclined to follow it, and the only account of
that is a diff summarised by the same model that wrote it. aritu is that same
standard made checkable — a verdict per unit with a reason, a majority of runs
that has to agree before anything passes, and a non-zero exit code out of a hook.
Being told becomes being held to it.

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
aritu apply internal/parser/parser.go

# everything the enabled rules target
aritu apply

# every rule over everything a pattern matches
aritu apply 'internal/**/*.go'

# one rule, several patterns
aritu apply --rule single-purpose-functions 'internal/**/*.go' 'cmd/**/*.go'

# the same rules over another ecosystem, with no flag to say so
aritu apply 'src/**/*.ts' 'lib/**/*.py'

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
internal/parser/parser.go
  intent-revealing-names  med
    ✓ parseConfig
    ! splitHostPort (1 of 2)
      one run read the name as carrying its intent and one did not
    ✗ handle
      names what the function is attached to, not what it does for the caller
  single-purpose-functions
    ✓ parseConfig

  2 passed  ·  2 failed  ·  1 split  ·  1 file, 2 rules, 2 votes  ·  4.1s
```

`✓` is a majority judging the unit to satisfy the rule, `✗` a majority judging
that it does not, and `!` an exact tie, which fails: half the votes is not a
majority. The count appears whenever the vote was not unanimous — the mark
carries the decision, the count how close it was.

A rule that fell short carries its [`priority`](#priority) beside its name, so a
sweep can be read top-down for what to fix first. A rule that passed carries
nothing: a clean target has nothing to triage, and a severity against every
passing rule would bury the handful that need one.

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
      "rule": "intent-revealing-names",
      "file": "internal/parser/parser.go",
      "votes": 2,
      "verdicts": { "handle": 0 },
      "reasons": {
        "handle": ["names what the function is attached to, not what it does for the caller"]
      }
    }
  ]
}
```

Each verdict is how many of the `votes` runs judged that unit to satisfy the rule.
**A unit passes on a strict majority.** A tie is not a majority, so it fails —
half the votes convinced is not a rule being followed.

The count is not a second verdict. It is how close the vote was, which is what you
tune a prompt against: `0` of `4` on code that should pass is a rule that is
broken, and a count hovering around the middle is a prompt that has not decided
what it thinks. The number is what tells them apart while you tune.

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

A split vote is not a third outcome and never becomes `2`. The rule is a strict
majority in favour; a tie does not meet it, so it fails. Filing it under "could
not run" would invite a hook to treat an unsure model as a tooling problem and
skip past exactly the case aritu exists to catch.

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

They are grouped by what a violation costs. ...

## Severe

Fix these before anything below them. ...

### No multi job files

A file has one job. Everything in it — types, helpers, constants — exists to
serve that job. ...
```

One heading per rule and the whole of that rule under it. It is the selection
`apply` uses, so a rule you have not enabled is not preached either, and `--rule`
narrows the document the same way it narrows a sweep.

**The rules are banded by [`priority`](#priority)**, hardest first, and within a
band they keep the order they were enabled in. Banding is the one thing a flat
list cannot say: which of two rules to satisfy first when a file breaks both. A
band no enabled rule declared is left out rather than printed empty.

**The heading is read off the directory name**, so a rule has one name and not two:
`my-pretty-rule` becomes `### My pretty rule`. Any parking prefix comes off first —
whether a rule is currently enforced says nothing about what it asks for.

**It is the same block the model is judged against.** `rule.SectionAt` renders it
once and both readers get that rendering: the rulebook stacks the sections into a
document under their band headings, and the verdict prompt frames one of them for
the model. Only the heading depth differs between the two. There is no
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
parallel: 5
timeout: 10m
output: pretty

rules:
  dir: ./rules
  enabled: [intent-revealing-names, single-purpose-functions]   # omit for all

targets:
  tests: ['internal/**/*_test.go', 'cmd/**/*_test.go']   # replaces the built-in
  migrations: ['db/migrate/**/*.sql']                    # a kind of your own

exclude: ['vendor/**', '**/*.gen.go']   # kept out of the derived sweep
```

`service.endpoint` is the base URL of any Responses-compatible API — a gateway, a
proxy, a self-hosted model. There is no default: a run that silently reached a
vendor's servers because a key was missing from the file would be a surprising
place for a repository's source to end up.

`model` and `effort` sit in the same block because which model names are valid is a
property of the endpoint serving them: a file that moved its endpoint and left its
model behind would be naming a model nobody serves. Neither has a flag, for the
same reason — they answer to the endpoint above them, and a shell that overrode one
without the other would be describing a pairing the file never sanctioned.

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
rather than your work, and a `fail-` fixture is a bad file written on purpose, which
`selftest` judges against the expectation its directory name carries. Name one as a
pattern and it is still judged — that was asked for.

aritu searches upward from the working directory, so running from a subdirectory
behaves the same as running from the root, and `--config` points somewhere else
entirely. Paths inside the file resolve against the file; patterns you type
resolve against your shell. Each resolves in the frame it was written in.

An unknown key is an error naming it. A setting that silently does nothing is
worse than one that refuses to load.

### Excluding files

`exclude` is what the derived sweep must not reach — vendored trees, generated
code, a directory you have decided not to fight about yet:

```yaml
exclude:
  - 'vendor/**'
  - '**/*.pb.go'
  - 'docs/legacy/**'
```

**The patterns are the same ones `targets` takes**, resolved against the config
file like every other path in it. There is no `.arituignore`, and that is
deliberate: a dotfile named for a tool carries gitignore's syntax by convention —
anchoring, `!` re-inclusion, directory-only trailing slashes — and a repository
would then be writing patterns two ways depending on which key they landed in.
One pattern language is worth more than the shorthand a second one buys. Write
`'**/node_modules/**'`, not `node_modules/`.

A pattern that does not parse fails the load naming itself, like any other bad
key.

**It bounds what is derived, not what you ask for.** `aritu apply` with no
arguments leaves excluded files out; `aritu apply internal/api/client.gen.go`
judges it. This is the rule the rules directory and a parked rule already follow —
what was derived respects the boundary, what was asked for overrules it — and it
is what keeps `aritu apply $(git diff --name-only)` meaning exactly what you typed.

The rules directory is excluded whether or not `exclude` mentions it, so a
repository never has to write that line.

## Rules

A rule is a directory. One rule per directory, under `rules/` by default
(`--rules` to point elsewhere).

```
rules/
  single-purpose-functions/
    prompt.md
    fixtures/
      pass-focused-render/
        render.ts
      fail-hidden-side-effect/
        report.go
      fail-many-arguments/
        billing.py
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
repeat — what judging means, that code comes in many shapes across ecosystems and
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
targets: [code]
granularity: function
priority: med
---
A name carries the intent of what it names; the reader should never have to
decode it...
```

There is no key for the rule's name or its summary. The heading comes from the
directory name, and the body is the whole of what the rule says — to the model that
judges it and, through [`rulebook`](#the-rulebook), to whoever is about to write the
file. Nothing is stated twice, so nothing can disagree with itself.

`targets` names the kinds of file the rule is about: aritu ships `tests`, `code`
and `docs`, and `aritu.yml` can replace any of them or add its own. `code`
deliberately overlaps `tests` — a test file has comments and names like any other
source file — because kinds are named matchers rather than a partition of the
tree.

`targets` and `granularity` are required, and a missing one is an error naming the
key. Defaulting either silently changes which files reach the model or what it
judges, and nothing would report it. A
`targets` typo is the sharpest case: `[test]` for `[tests]` would match no file, run
nothing, and exit `0`. So an unknown kind fails when the rule is loaded, naming the
ones there are, before a single model call — and so does an empty list, which is a
rule that could never run.

`priority` defaults instead, to `med`. It changes nothing about which files reach
the model or what it is asked about them — only how the finding is reported and
where the rule sits in the rulebook — so a rule that omits it judges exactly what
it always would and simply sorts last. An unknown *value* is still an error: the
scale is closed, and a typo that silently became `med` would quietly demote a rule
somebody meant to raise.

### Grouping properties into rules

Dozens of separate properties can make a file worth keeping. One rule per property
would be dozens of verdict calls per file per vote and as many prompts to keep from
contradicting each other. So properties group, by **judgement** — the same question
asked of the same unit with the same evidence in front of the model. Two
properties that agree on `granularity` and `include_source` can share a verdict
call; two that disagree about either cannot, however similar they read.

The grouping is what keeps the cost linear. Per file, per vote: one enumeration
call per granularity that needs one, shared by every rule at that level, plus one
verdict call per rule. `file`-granularity rules are the cheap ones — `file` needs
no enumeration and its schema has one key.

The cost that is not calls: a `file`-granularity failure returns one verdict and
one sentence for a whole file. `reasons` carries one entry per dissenting run,
which softens it, and some properties genuinely are properties of the whole file —
but it is thinner guidance than a per-unit rejection, and that is a known trade.

### Finding the file under test

`include_source` decides what the model sees. It defaults to `false`, which sends
the file alone: pairing a file with the implementation it covers is only
meaningful for a rule about tests, so a rule writes the key out only when it wants
the pairing. With `true` the implementation goes too — whether an identifier is a
subject's surface or its internals, whether two inputs land in one equivalence
class, and what outcomes are missing are all facts about the code rather than
about the test.

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

### Priority

`priority` declares what a violation costs. It answers one question — of the
things this file got wrong, which do I fix first — and the scale is built around
the fact that the answer is usually structural.

| level | what it means |
|---|---|
| **`severe`** | the fix relocates the code around it, and findings nested inside it often go with it |
| **`high`** | the violation is in a shape callers depend on, so the fix reaches past the declaration carrying it |
| **`med`** | local enough to fix where it stands: a rename, a move, a deletion |

**There is no `low`.** A property not worth fixing is not worth a rule, and a
band nobody acts on turns the whole scale into decoration. `med` is the floor,
which is also why it is what an omitted key parses to.

Priority does not change what is judged — the same files reach the model and it
is asked the same question. It changes what comes back: a rule that fell short
prints its level beside its name, and [`rulebook`](#the-rulebook) groups by it.
Fixing a `severe` first is not bookkeeping — its fix moves the code the lower
bands are about, so findings under it are often answered before they are read.

### Granularity

`granularity` declares what a rule judges. The levels are defined by the role a
construct plays, never by the syntax that declares it — that is what lets one rule
set judge every ecosystem.

- **`function`** — one per function or method the file declares under its own
  name.
- **`test_case`** — one per **leaf** of that: one row of a table, one parametrised
  argument set, one subdivision declared inside the test, or the test itself when it
  has none.
- **`file`** — one, keyed by the path. Relations *between* units live only here.

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
are **one** `function`-level unit, so a rule at that level sees both together —
correct, because that is one language's shape for two behaviours sharing a name. Two
`it`s in one `describe` are **two** `function`-level units, so each is judged alone —
also correct, because a `describe` is a grouping construct and grouping is what it
is for.

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

State **both poles**: the property a unit must have, and the shapes that
disqualify it. A prompt given only one pole drifts toward answering everything the
same way, and a rule that never fires is indistinguishable from a rule that always
passes. A grouped rule needs the second pole more than a narrow one did, because it
has several ways to be wrong and several near-misses to spare.

Then prove it with fixtures. Each fixture is its own directory holding exactly the
file to be judged — and, for a rule about tests, the implementation it covers. One
directory per fixture because files in one directory often share a namespace, so
two fixtures declaring the same name would collide. The `pass-`/`fail-` prefix
carries the expectation, so the coverage floors are visible in the listing:

1. A rule's fixtures span ecosystems — the same property proved in Go and
   TypeScript and Python. This is the only thing that shows a prompt is not
   quietly shaped around one language's idioms.
2. Every disqualifying shape a prompt names has a fail fixture, in whichever
   language expresses that shape most naturally.
3. Every near-miss a prompt calls out as satisfying has a pass fixture. These are
   what stop a rule over-firing, and they are the ones that break during tuning.

`selftest` checks every fixture against its prefix:

```
rule: intent-revealing-names  model: sonnet  votes: 4

FIXTURE                        EXPECT  RESULT  TIME   VERDICTS
fail-cryptic-names             fail    hold    31.2s  d=0 hndl=0 proc=0
fail-unsearchable-magic-number fail    hold    28.7s  applyDiscount=0
fail-unnamed-condition         fail    hold    30.1s  renderBanner=0
pass-descriptive-billing       pass    hold    45.1s  daysSinceLastInvoice=4 markInvoiceOverdue=4

14/14 fixtures hold in 2m27.7s
```

A fail fixture and the pass fixture nearest to it are the pair that keeps a rule
honest: the same surface shape is a violation in one and not in the other, and
only having both in the set proves the rule judges the property rather than the
shape.

A fixture's files are named by its directory, never by a kind: a rule's `targets`
are not consulted anywhere in a `selftest` run. Otherwise a rule's self-test would
depend on the `aritu.yml` of whichever repository the rule happened to sit in.

`selftest` is `apply` in a loop — same prompt, same voting, same code path. It adds
only the comparison against the directory prefix and the table. It compares counts,
never exit codes: a `pass-` fixture holds at `votes`, a `fail-` fixture holds at
`0`. Anything between the poles fails, including a `fail-` fixture the model only
mostly rejected — a rule that needs a dissenting vote to fire is one bad file away
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
enumeration, and a quietly dropped unit is the precise failure this tool exists to
catch.

At `file` granularity the first call is skipped entirely: the unit is the path,
which costs nothing to know and cannot be disagreed with.

## Flags

| flag | default | |
|---|---|---|
| `--rule` | all | rule to run; repeat for several |
| `--config` | search upward | config file to use instead of `aritu.yml` discovery |
| `--output` | `pretty` | `pretty` for reading, `json` for parsing |
| `--votes` | `1` | rounds run per unit; a strict majority must agree it passes |
| `--parallel` | `5` | model calls allowed in flight at once |
| `--rules` | `./rules` | directory holding one subdirectory per rule |
| `--timeout` | `10m` | deadline for the whole run, so a hung endpoint cannot hang a commit hook |
| `--debug` | off | print each prompt on stderr instead of calling the model — placeholder units stand in for the splitter's answer, no report is written, no endpoint is needed |

Which model answers, and at what effort, is `service` in the config file and
nowhere else. See [Configuration](#configuration).

`--parallel` bounds concurrency at the one seam every model call passes through, so
fixture-level and vote-level parallelism cannot multiply into a process storm.

`--timeout` covers the entire run rather than a single call, so it scales with how
many fixtures a rule has. A large corpus at high `--votes` may need more than the
default.
