---
slug: multi-target-runs
type: feat
status: draft
created: 2026-07-26
---

------

# Feature: Multi-target runs

## Purpose

`apply` judges one file against one rule. Checking a package before a commit means
running it eighteen times by hand and reading eighteen reports, and a hook that
wants every rule over every changed file has to shell-loop and stitch the exit
codes together itself.

The shape a test runner already has is the right one: point it at a pattern, get
one report covering everything it found.

Running many things at once also makes the argument list long enough to be worth
writing down. A repository has one answer to which model, how many votes and which
files, and repeating it on every invocation is how the answer drifts between the
hook, CI and the person debugging. So this feature carries a config file too.

The saving is not only ergonomic. The enumeration call — "what units are in this
file" — takes the granularity and the file and nothing else. The rule body never
reaches it. So running three rules over one file asks the same question three
times and pays for three identical answers.

## Scope

### Selecting files

Positional arguments become glob patterns rather than a single file:

```sh
aritu apply internal/parser/parser_test.go        # a literal path
aritu apply 'internal/**/*_test.go'               # everything underneath
aritu apply 'internal/*/[a-z]*_test.go'           # a narrower glob
aritu apply 'src/**/*.spec.ts' 'tests/**/test_*.py'
```

Globs rather than a language's own convention. The runtime is Go and the rules
that ship are about Go tests, but nothing in target selection should assume that:
a glob is understood everywhere, and the day a rule judges TypeScript the argument
handling should not need a second syntax bolted beside it.

**The pattern is the selector, so aritu holds no opinion about what a test file
is.** Everything a pattern matches is judged. There is no `*_test.go` filter, no
"is this really a test file" check, and consequently no rule about naming a
non-test file explicitly — that question only existed because the tool was being
asked to guess at a language convention it should not know.

`filepath.Glob` matches a single path segment, so `internal/**/*_test.go` finds
nothing through it. `bmatcuk/doublestar/v4` supplies `**` and costs nothing: its
module graph is itself alone. Preferred over hand-rolling because the edges are
where globs go wrong — leading and trailing `**`, `a/**/b` matching a bare `a/b`,
escaping, character classes — and none of that is worth rediscovering.

Behaviour must not depend on the calling shell. zsh expands `**` and bash without
`globstar` does not, so a quoted pattern and a shell-expanded list of paths have to
reach the same result.

A pattern matching no files at all is an error. Silently succeeding over an empty
set is the failure mode where a hook reports green because its path was wrong.

Duplicate matches across patterns are judged once. Two overlapping globs are a
convenience, not a request to pay twice.

### Selecting rules

The rule stops being positional and becomes a repeatable flag:

```sh
aritu apply --rule named-for-behavior internal/parser/parser_test.go
aritu apply --rule named-for-behavior --rule one-reason-to-fail 'internal/**/*_test.go'
aritu apply 'internal/**/*_test.go'               # every rule in the rules dir
```

Omitting `--rule` runs every rule the rules directory holds, in name order. An
unnamed rule is an error naming it, not a silent skip.

Listing the rules needs a `rule.List(rulesDir)` that returns the directories and
ignores `base.md`, which is a file at the root rather than a rule.

### Configuration

`aritu.yml` at the repository root, every key optional, absent is fine:

```yaml
model: sonnet
effort: medium
votes: 2
jobs: 5
timeout: 10m
output: pretty
claude: claude

rules:
  dir: ./rules
  enabled: [named-for-behavior, one-reason-to-fail]   # omit for all

include:
  - 'internal/**/*_test.go'
  - 'cmd/**/*_test.go'
```

Visible rather than a dotfile, beside the `rules/` directory that is already
visible configuration. YAML because `yaml.v3` is already the only dependency, and
one filename rather than a `.yml`/`.yaml`/`.json` family, so there is never a
question of which one won.

`rules` is a block because the word names two different things — the directory the
rules live in and the subset to run — and flattening them would need `rulesDir`
beside `rules` for no gain. `include` supplies the targets for a bare
`aritu apply` with no positional arguments.

#### Precedence

```
built-in defaults  ->  aritu.yml  ->  command-line flags
```

`alecthomas/kong` owns this layering. Its resolvers exist for exactly this, so
precedence is library behaviour rather than something reinvented here — and the
trap it avoids is real: a flag carrying its default value is indistinguishable from
a flag nobody typed, so a merge written by hand lets flag defaults silently
overwrite the file.

Adopting kong replaces the `flag.FlagSet` plumbing in `cmd/aritu`. A repeatable
`--rule` comes free instead of needing a `flag.Value`; the subcommands stop needing
a FlagSet each; and `--help`, including the exit-code table, is generated from the
struct rather than maintained by hand beside the flags it describes and drifting
from them. This is a rewrite of code that currently works and is tested, and that
churn belongs to this feature rather than arriving as a surprise inside it.

Validation runs **once, on the resolved result** — votes at least one, a known
output, a known effort. Validating each source separately is how a config file
ends up accepting something the flag rejects.

#### Discovery

Walk up from the working directory to the first `aritu.yml`, so running from a
subdirectory behaves the same as running from the root. `--config <path>` overrides
the search. No file found is not an error.

**Each path resolves in the frame it was written in.** `rules.dir` and `include`
are relative to the config file, because that is where they were typed. Patterns
given as arguments stay relative to the shell's working directory. Resolving both
against the same base surprises the reader in one direction or the other.

#### Unknown keys are an error

Decoded with `KnownFields(true)`, so a typo'd `vote: 4` fails rather than being
skipped. This is the argument `include_source` and `granularity` already make: a
setting that silently does nothing is worse than one that refuses to load, because
the run still produces a confident-looking answer to a question nobody asked.

#### What the file may not set

`granularity` and `include_source` stay in each rule's `prompt.md`. They are
properties of the rule, and letting a repository override them would mean the same
named rule judges different things in different checkouts.

### One enumeration per file

The enumeration prompt is built from the granularity and the file. The rule body is
not in it, so two rules at the same granularity over the same file ask an identical
question.

Rather than cache per `(file, granularity)`, enumerate each file **once** at `test`
granularity and roll the coarser levels up from the result:

| level | derived by |
|---|---|
| `test` | the enumeration itself |
| `function` | the distinct function halves of those units, in first-seen order |
| `file` | the path, which needed no call to begin with |

A function that declares no cases appears in the `test` list as its bare name, and
one that declares cases appears once per case, so the distinct function halves are
exactly the set of test functions. The roll-up is a string split, not a second
question.

For the three rules that ship today this turns three enumeration calls per file
into one. Over six files that is eighteen calls down to six.

**The cost of this is coupling.** Every rule now depends on the `test`-granularity
enumeration being right, where a `function`-granularity rule used to ask a simpler
question and carry its own risk. A `test` enumeration that invents a case now
corrupts every rule's unit list for that file rather than one. The mitigation is
that the generated schema already forces the verdict call to answer exactly the
enumerated set, so an enumeration error surfaces as a wrong unit list rather than a
silent drop — but it surfaces everywhere at once.

The cache is in memory and lives for one run. Nothing is written to disk: a stale
enumeration keyed by a path would be judged against a file that had since changed,
and no amount of invalidation logic is worth that.

### Output

`apply` reports grouped by file, then by rule, because the reader is looking at
their own code rather than at the rule set:

```
internal/parser/parser_test.go
  named-for-behavior
    ✓ TestParseConfig (extracts host before colon)
    ✗ TestParseConfig (host and port)
      names the input the case supplies rather than the outcome
  one-reason-to-fail
    ✓ TestParseConfig

internal/lib/vote/vote_test.go
  named-for-behavior
    ✓ ...

  14 passed  ·  3 failed  ·  2 files, 2 rules, 4 votes  ·  28.4s
```

`--output json` gains an envelope, because the top level is no longer one report:

```json
{
  "reports": [ { "rule": "…", "file": "…", "votes": 2, "verdicts": {}, "reasons": {} } ]
}
```

This is a breaking change to the JSON shape and is deliberate. Emitting a bare
report for one target and an envelope for many would mean every consumer writes
the same branch, and the single-target case is not special enough to earn it.

Timing follows `selftest`: per-target durations and a wall-clock total.

### Exit codes

Unchanged in meaning, aggregated across targets:

- `0` — every unit of every rule over every file unanimously satisfied its rule
- `1` — one or more did not
- `2` — one or more targets could not be run

`2` outranks `1`. A run where one file was unreadable and another genuinely failed
is a run that did not check everything, and reporting it as an ordinary rule
failure would let a hook treat a partial sweep as a complete one. Every report is
printed either way, including the ones that errored.

### selftest

Takes the same treatment, for the same reason: `--rule` repeatable, all rules when
omitted, so `aritu selftest` alone exercises the whole corpus.

## Out of scope

- Watch mode. A runner-shaped CLI invites it; it is a separate feature.
- Per-rule configuration, such as `named-for-behavior: {votes: 4}`. Genuinely
  useful for a rule that splits, but it turns `enabled` from a list into a map and
  doubles the merge, and nothing yet demands it.
- A second, machine-local override file. `claude` is the one key that is about the
  machine rather than the project, and until that is actually painful a flag covers
  it.
- Reporters beyond `pretty` and `json`.
- Language-agnosticism beyond argument handling. Two Go assumptions remain and are
  known: `rule.SourcePathFor` resolves `file_test.go` to `file.go` for
  `include_source`, and the shipped enumeration prompts describe Go test shapes.
  Both are real couplings and neither belongs in this feature.
- Caching enumeration between runs, for the staleness reason above.
- Changing granularity, the rule prompts, the fixtures, or `selftest`'s hold
  semantics.
- Deciding what to do about the findings a full sweep produces on this repo's own
  tests. Running everything at once makes that list longer, not different.

## Constraints

- Go. Bound by `CONSTITUTION.md`.
- Two dependencies join `yaml.v3`. `bmatcuk/doublestar/v4` for `**` matching, whose
  module graph is itself alone. `alecthomas/kong` for flags, subcommands, generated
  help and config resolution, at four modules.
- `spf13/viper` was measured and rejected. At twenty-six modules it is the heaviest
  option on offer, but the disqualifying part is behavioural: it does not reject
  unknown keys and it lowercases them, so `vote: 4` would load silently as nothing.
  A config layer that cannot be strict cannot implement this spec.
- The file itself is still decoded by `yaml.v3` with `KnownFields(true)`, which is
  what makes that strictness available at all.
- Glob matching is a pure function of pattern and path, table-testable without a
  filesystem. Target expansion, config loading and rule listing are testable
  against `t.TempDir()` rather than a model.
- The three-layer merge is one pure function over three inputs, so precedence can
  be pinned by a table rather than inferred from flag registration order.
- Concurrency stays bounded by the existing `--jobs` throttle at the ask seam.
  Fanning out over files must not introduce a second, separate limit.
- A file's rules may only be judged after that file's enumeration resolves, and
  the enumeration must happen once even when several rules start together.
- `Report` keeps its shape. What changes is that a run now produces several.

## Done criteria

Every invocation below has been run and its output looked at. Reading the code and
concluding it would work is not evidence; the point of the list is that the shapes
interact — a glob with a config and an overriding flag exercises three things whose
seams are where this will break.

### Costing nothing, because they resolve before the first model call

These fail fast and can be re-run freely.

| command | expected |
|---|---|
| `aritu --help` | both subcommands and every flag, generated rather than hand-written |
| `aritu apply --help` | the flags and the exit-code table |
| `aritu apply --rule no-such-rule 'internal/**/*_test.go'` | exit `2`, names the missing rule |
| `aritu apply 'nowhere/**/*_test.go'` | exit `2`, says the pattern matched nothing |
| `aritu apply --output yaml internal/lib/vote/vote_test.go` | exit `2`, `pretty or json` |
| `aritu apply` with no targets and no `include` in config | exit `2`, says what to pass |
| a config carrying `vote: 4` | exit `2`, names the unknown key |
| a config carrying `votes: 0` | exit `2`, the same message the flag gives |

### Against a stand-in `claude`, so verdicts are fixed and the run is free

A script that logs each invocation and returns canned JSON, as the tests already do.

| command | expected |
|---|---|
| `aritu apply 'internal/**/*_test.go'` | **one enumeration call per file**, counted in the log, whatever the rule count |
| `aritu apply --rule named-for-behavior --rule one-reason-to-fail <one file>` | one enumeration in the log, two verdict calls |
| `aritu apply 'internal/**/*_test.go' 'internal/lib/**/*_test.go'` | overlapping matches judged once, not twice |
| `aritu apply --output json 'internal/**/*_test.go'` | a `reports` envelope, one entry per file-rule pair |
| a run where one file is unreadable and another fails its rule | exit `2`, and both reports printed |
| a run where every file passes | exit `0` |
| a run where one unit fails and nothing errors | exit `1` |

### Config and precedence

| command | expected |
|---|---|
| no `aritu.yml` present | behaves exactly as before the feature |
| `aritu.yml` with `votes: 4` | a run reports four votes |
| the same, plus `--votes 2` | a run reports two — the flag wins |
| `aritu.yml` with `include:` and no positional arguments | those patterns are used |
| `cd internal/lib/vote && aritu apply vote_test.go` | the root config is found, and `rules.dir` resolves against the config rather than the working directory |
| `--config <path>` pointing elsewhere | that file is used and the search is skipped |

### Globs behave the same however the shell treats them

`aritu apply 'internal/**/*_test.go'` and `aritu apply internal/**/*_test.go`
produce the same set of targets under zsh, which expands `**`, and under
`bash --noglobstar`, which does not.

### Against the real model, once the rest holds

`aritu apply 'internal/**/*_test.go'` completes across every rule and file and
prints one grouped report, and `aritu selftest` with no rule named runs all three
rules and still holds for every fixture on sonnet.
