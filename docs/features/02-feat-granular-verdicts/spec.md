---
slug: granular-verdicts
type: feat
status: draft
created: 2026-07-26
---

------

# Feature: Granular verdicts

## Purpose

aritu currently judges the wrong unit, and says too little about what it found.

In a table-driven test each row is a test. The function name is a namespace. Judging
`TestCollect` and reporting `TestCollect=0` is judging the folder, not the file — the
behaviours were one level down, well named, and never looked at. Running the first build
against aritu's own suite proved it: `named-for-behavior` rejected all four of
`vote_test.go`'s tests unanimously, while their rows read
`"a split vote counts only the rounds that agreed"` and
`"orders results by round when completion order is reversed"`. The rule was right that
`TestCollect` says nothing. It was pointed at the wrong string.

Not every rule wants that resolution, though, and some want less. Whether a table mixes
several behaviours is a property of the whole function. Whether tests share mutable state
is a property of the whole file and cannot be expressed at all today. So the unit under
judgement becomes a per-rule declaration rather than a fixed choice.

Second: a verdict is currently a bare count. A calling agent learns that something failed
and nothing about what to change. The model already knows why; it just is not asked.

## Scope

### Granularity

New required frontmatter key, alongside `include_source`:

```yaml
granularity: file | function | test
```

| level | keys returned | key is |
|---|---|---|
| `file` | 1 | `internal/parser/parser_test.go` |
| `function` | one per `func Test*` | `TestParseConfig` |
| `test` | one per independently nameable leaf | `TestParseConfig (rejects input with no separator)` |

The levels form a scale: each is a refinement of the one above, so key count is
non-decreasing as you go finer.

Assignment for the three existing rules:

- `named-for-behavior` — `test`. The row name is the thing making the claim.
- `one-reason-to-fail` — `function`. How many behaviours a table pins down is a property
  of the whole loop, not of any row. Asked of a single row the question is close to
  vacuous.
- `no-mocking-under-test` — `function`. The subject is substituted once in setup and
  shared by every row.

**Required, never defaulted.** `include_source` is already required because silently
defaulting it changes which files reach the model. This is the same hazard and worse:
forget it on a row-naming rule, get function granularity, and the rule quietly judges
namespaces instead of behaviours. It does not error. It returns plausible verdicts that
mean nothing — which is the failure the first spec opens with, a rule that never fires
being indistinguishable from a rule that always passes.

Model it as a typed enum, not a string, so the value is validated once at load and the
dispatch over it fails loudly on an unknown (R-12, R-13).

### What counts as a leaf at `test` granularity

A leaf is a unit that can independently fail and be named. Concretely: a table row
carrying a `name` field, a `map[string]struct{...}` case keyed by its name, or a
`t.Run("...", ...)` subtest. A plain test function with none of these is itself one leaf,
so it yields exactly one key.

**Nameless tables fall back to the function, and this is correct rather than a special
case.** No `t.Run` means no subtests, so no leaves exist and the function is judged as a
plain test. That gives the right answer in both directions without a rule for it:

- `TestParsesHostFromAddress` over a nameless table of addresses — one behaviour, many
  inputs, function name states it — passes.
- `TestParseConfig` over a nameless table of six behaviours — the function name now has to
  carry all six claims and cannot — fails.

The property worth preserving: **you cannot dodge the rule by deleting names.** Stripping
row names does not hide cases from judgement, it pushes judgement up to a name that is
harder to pass. The lazy path is the failing path.

**Duplicate row names disambiguate, they do not error.** Go emits `empty_input` and
`empty_input#01`; enumeration must do the same. Two rows collapsing to one map key would
otherwise trip the duplicate-verdict check and exit `2`, which is the wrong diagnosis —
the tool is fine, the table has two cases a reader cannot tell apart. That is the rule
firing correctly through the wrong channel. It should come out as a `0`.

The general principle: unnameable, indistinguishable and absent all resolve to **verdicts**.
Exit `2` stays reserved for "the tool could not run", which is the only thing that makes it
trustworthy in a commit hook.

### Verdict identity

The key is the identity of the judged unit at the declared level. Function name, file path,
or function plus case:

```
TestParseConfig (rejects input with no separator)
```

Spaces to underscores recovers the `go test -run` selector, so the readable form loses
nothing.

### Reasons

The verdict call gains a `reason` field beside `satisfies`: one sentence on why this unit
does or does not satisfy this rule.

Surfaced as a **sibling map, not nested**:

```json
{
  "rule": "named-for-behavior",
  "file": "internal/parser/parser_test.go",
  "votes": 4,
  "verdicts": { "TestSlugify": 4, "TestParse": 0 },
  "reasons": {
    "TestParse": ["names the unit under test with no stated outcome, so it would still read as true whatever parsing produced"]
  }
}
```

`verdicts` stays `map[string]int` exactly as specified, so tallying, unanimity, `Holds` and
the exit logic are untouched. Nesting the count and reasons into an object would change
five signatures and the documented output shape to buy nothing.

Populate `reasons` only for keys that did not reach `votes`. A unanimous pass has nothing to
explain, and an array per failing key naturally carries one entry per dissenting run — which
is also the tuning signal, since four differently-worded reasons for the same rejection say
something different from four identical ones.

Field order within the schema was measured and does not affect verdicts: reason-before-
`satisfies` and `satisfies`-before-reason returned identical counts across four votes on
three tests, with no failed calls. Order it for readability.

### Base prompt

Extract the boilerplate the tool wraps around every rule body into `rules/base.md`, beside
the rule directories. Rule loading resolves `<rulesDir>/<name>/prompt.md`, so a loose file
at the root is invisible to it.

Three layers, cleanly separated:

| layer | holds | scope |
|---|---|---|
| system prompt | no tools, answer only in JSON | every call |
| `rules/base.md` | what judging means, that test shapes vary, judge behaviour not syntax, how to write a reason | every verdict call |
| `prompt.md` | the criterion and its two poles, nothing else | one rule |

A file rather than a Go constant because this tool exists to iterate on prompts, and a base
prompt locked in a constant needs a rebuild to tune. The safety net already exists: break
`base.md` and the fixtures across every rule stop holding. `selftest` is the regression test
for that file.

Reason guidance belongs here rather than in each rule: one sentence, about *this* unit
against *this* rule rather than the rule in the abstract, addressed to whoever has to fix
the test. Left to each rule author you get "does not satisfy the rule" from one and an essay
from another.

**Delete rather than relocate:** the prompts currently say
``Answer with a JSON object holding a "results" array…``. `--json-schema` already enforces
the shape. That prose is redundant and should not move into the base.

### Enumeration

The names call gains the granularity it is enumerating for. Two variants: `function` lists
top-level `func Test*`; `test` is the same instruction plus an expansion clause for rows and
subtests. `file` needs no variant — the unit is the file path, known statically.

**At `file` granularity, skip the names call entirely.** There is nothing to enumerate, so
the level costs one call per vote instead of two, and the name/verdict comparison becomes a
single-key check that cannot spuriously fail.

**Pass the enumerated list into the verdict prompt.** Today the verdict call re-derives the
list independently and is cross-checked against the names call. That redundancy is cheap
across four functions and expensive across twenty-five rows, where two independent
derivations must agree exactly and every disagreement is a could-not-run. Handing the
verdict call the list does not weaken the guarantee: the names call remains the independent
authority on what is in the file, and a verdict call given an explicit list can still drop
an entry, which the existing comparison still catches. It only removes the false exit `2`s
where the two calls phrased a row name slightly differently.

Granularity must reach **both** prompt builders. The name/verdict comparison enforces that
the two calls agree exactly, so enumerating eleven rows while judging one function is an
exit `2` on a healthy file.

## Out of scope

- `package` granularity — judging several files together is where "the tests collectively
  cover the exported surface" lives, but it needs multi-file input and is a real feature
  rather than a fourth enum value. Leave room for it; do not build it.
- Still no code parsing. Enumeration remains a model call, not an AST walk.
- Changing the `verdicts` map shape, the exit codes, or `selftest`'s hold semantics.
- Deciding whether aritu's own tests should be restructured to satisfy its own rules. That
  question is recorded in `docs/learnings/aritu-fails-its-own-test-suite.md` and is a
  maintainer call, not part of this work.
- Making `reason` available to `selftest`'s table. Useful later for prompt tuning; not
  required here.

## Constraints

- Go. One binary. The rules directory stays the only configuration.
- Bound by `CONSTITUTION.md`. Specifically: granularity is a bounded set, so it is a typed
  enum whose dispatch panics on an unknown value (R-12), carried as a struct field rather
  than a raw string (R-13), and selected through a dispatch table rather than a switch on
  the value (R-4).
- `verdicts` remains `map[string]int`. `reasons` is additive and omitted when empty.
- Exit codes are unchanged, and nothing added here may route a judgement into exit `2`.
- Existing rule prompts keep their two-pole structure. `named-for-behavior` gains a second
  pole for row names — `"host and port"` names the input, `"extracts host before colon"`
  names the behaviour.

## Done criteria

All three rules carry an explicit `granularity`, `rules/base.md` exists and the rule bodies
no longer restate format or output shape, and `named-for-behavior` has gained table-driven
fixtures whose rows carry the pass/fail signal rather than the function name.

`selftest` holds for every fixture of every rule on sonnet at 4 votes, as before.

And the case that started this: `aritu apply named-for-behavior internal/lib/vote/vote_test.go`
exits `0`. Those rows were always well named; the tool has to be able to see them.
