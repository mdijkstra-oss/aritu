---
slug: rule-file-targets
type: feat
status: draft
created: 2026-07-27
---

------

# Feature: A rule declares which files it is about

## Purpose

Every rule judges every file. `run.Run` takes the cross product of `opts.Files` and
`opts.Rules` and forms a goroutine for each pair, unconditionally. `ApplyCmd` records the
reasoning:

> The patterns are the selector, so aritu holds no opinion about what a test file is:
> everything they match is judged.

That was right when every rule was a rule about tests. It stopped being right two commits
ago. `32927e9` made prompt fragments per-rule so that a rule could include nothing, and the
README states what that buys: *"A rule that includes nothing is told nothing about tests,
which is what makes a rule about anything else possible."*

The prompt layer is ready for a rule about anything else. The file layer is not. A rule
about comments in source, or about the shape of a markdown document, can be written and
cannot be run: enabling it hands it every file the sweep selected, and the only lever is
`include`, which is global and shared by all seven existing rules. The first non-test rule
is blocked on this and nothing else.

There is a second, smaller failure already present. `include` and `rules.enabled` are
independent lists that can disagree without complaint — enable a rule, forget to widen
`include`, and the run is green because the rule never saw a file. This feature removes the
possibility by removing the second list.

## Scope

### The two questions, and who answers them

Scoping a rule to files is really two questions, and the prior art collapses them because
it can afford to.

**ESLint's flat config** puts globs and rule selection in one object — `{ files:
["**/*.js"], rules: { semi: "error" } }` — and merges every object that matches a path.
**Ruff** goes the other way, subtracting from a globally-enabled set:
`per-file-ignores = { "__init__.py" = ["E402"] }`. Both work because every rule in those
tools is a rule about one language, so "which files does this rule make sense for?" is
trivially *all of them*, and only "which files are in this project?" is left.

aritu has both questions live:

1. **Which files does this rule make sense for?** Intrinsic to the rule. `tests-one-thing`
   is about tests, in every repository, forever.
2. **Which files in *this* repository are of that kind?** Intrinsic to the repository.
   `internal/**/*_test.go` is a Go answer, and rules name no ecosystem.

So the answers live in different places. The rule names a **kind**. The repository maps
kinds to files. Neither ESLint's shape (every repo re-declares which rules are test rules,
and shipping a rule means editing every config) nor Ruff's (everything on everywhere by
default, which is nonsense the moment a markdown rule exists) can express that.

### A kind is patterns plus a refinement

`internal/lib/testpath` already establishes where ecosystem knowledge lives:

> `convention` is one ecosystem's test-file naming, held as data so that supporting a layout
> is a row in the table rather than a branch in the code.

Target kinds follow it. A kind is a row of data in `internal/lib`, holding two things: the
**patterns** that generate candidate files, and an optional **refinement** predicate that
decides membership exactly. Generation is coarse because a predicate cannot enumerate a
filesystem; refinement is exact because a glob cannot express `parser.test.ts` and
`__tests__/parser.ts` and `ParserTests.java` without becoming a second copy of a table that
already exists.

| kind | patterns | refinement |
|---|---|---|
| `tests` | every extension in `testpath`'s conventions table | `testpath.IsTestFile` |
| `code` | every extension in `testpath`'s conventions table | none |
| `docs` | `**/*.md`, `**/*.mdx` | none |

Two of the three derive their patterns from the conventions table's extension list — the
same input `indexByExtension` already consumes — so adding an ecosystem to `testpath`
widens `tests` and `code` with no second edit. Only `docs` introduces new data, and it
introduces two lines of it.

`tests` is the one kind that carries a refinement, and it is the reason the two-part shape
exists rather than a flat glob list. `testpath.IsTestFile` knows four ecosystems' affixes,
test directories and mirrored trees. Approximating that in globs would duplicate it and
then drift from it (R-2).

**`code` deliberately overlaps `tests`.** `foo_test.go` matches `**/*.go`, so a rule about
comments judges test files too — which is correct, because tests have comments. Kinds are
named matchers, not a partition of the tree. A repository wanting source-without-tests
writes that itself.

### The vocabulary is open

`aritu.yml` gains a `targets` block. Each key is a kind name; each value is a pattern list
with no refinement:

```yaml
targets:
  code: ['internal/**/*.go', 'cmd/**/*.go']   # replaces the built-in
  migrations: ['db/migrate/**/*.sql']          # a new kind, for a rule someone wrote
```

A key matching a built-in name **replaces** that built-in wholesale, patterns and
refinement together. One key, one meaning: a repository that overrides `tests` is saying it
knows better than the conventions table, and silently keeping `IsTestFile` as a refinement
over their patterns would make that override lie. Ruff ships both `per-file-ignores` and
`extend-per-file-ignores` because replacement alone was not enough; that is evidence for
adding an extend form later if anyone asks for it, not for shipping two forms now.

The known set is therefore built-ins ∪ config keys, resolved once at config load.

### The rule names its kinds

`prompt.md` frontmatter gains `targets`, a list of at least one kind:

```markdown
---
targets: [tests]
include: [tests]
include_source: false
granularity: function
---
```

All seven shipped rules get `targets: [tests]`.

**`targets` is required and is never defaulted**, for exactly the reason the README already
gives for the other two: *"`include_source` and `granularity` are required. Defaulting
either one silently changes what the model sees or what it judges, and nothing would report
it."* A defaulted `targets` silently changes which files are judged, which is the same class
of failure and the more dangerous one, because its symptom is a green run. An empty list is
rejected on the same grounds — a rule matching nothing is a rule that never runs.

**`targets` and `include` stay separate keys** despite every shipped rule setting both to
`[tests]` today. They answer different questions: a rule about comments targets `code` and
wants no fragment at all, and nothing stops a rule targeting `tests` while including a
`code` fragment. Collapsing them would re-assert that every rule is a test rule, which is
the assumption `32927e9` was written to remove.

A rule naming an unknown kind fails at load, listing the known set — the same failure, in
the same place, as `checkIncludesAreKnown` gives for an unknown fragment. The typo case is
what makes this non-negotiable: an unvalidated `targets: [test]` matches nothing, runs
nothing and reports green.

This costs `rule.Load` a parameter. Fragments validate against a compiled-in set
(`prompts.IsKnown`), but kinds are only known after `aritu.yml` is read, so the resolved
set is passed in. `Load` already takes `rulesDir` from its caller; it takes the known kinds
the same way.

### The sweep is derived, and `include` is deleted

Today `targetsFor` expands the CLI patterns, falling back to `include`. With kinds, the
fallback becomes the union of the patterns of every enabled rule's kinds. Enabling a docs
rule then just works; nobody has to remember to widen a second list, and the two lists that
could disagree are one list.

This needs one change in `internal/lib/glob`. `Expand` fails a pattern matching nothing, on
stated grounds — *"silently succeeding over an empty set is how a hook reports green because
its path was wrong"* — and that reasoning holds for a pattern somebody **typed**. It does
not hold for a generated one: a built-in kind spans four ecosystems and a Go repository has
no `.java` files. So generation uses a tolerant expansion and typed patterns keep the strict
one. Both exist because the two cases fail differently, not because the rule is negotiable.

CLI patterns still override the derived list entirely, so `aritu apply $(git diff
--name-only)` keeps working. They are still filtered per rule: naming a `.md` file does not
make a test rule judge it.

For this repository the derived sweep is every `*_test.go` file, against today's
`internal/**/*_test.go` and `cmd/**/*_test.go` — the same set, plus any test file at the
root, which there are none of.

### Pairing, ordering, and files nothing targets

The cross product in `run.Run` becomes an ordered list of `(file, rule)` pairs, built by
walking files in order and rules in order and keeping the pair when the file satisfies one
of the rule's kinds. This is the current traversal with holes removed, so the print order
`observeInOrder` depends on is unchanged; what changes is that `results` is indexed by
position in the pair list rather than by `f*len(rules)+r`.

A **typed** file that no enabled rule targets is reported and exits `2`. It cannot be judged
by anything, which is what code `2` is for — *"one or more targets could not be run, which
outranks 1"* — and silence here would be the failure mode this repository guards against
twice already, in `targetsFor` and in `glob.Expand`. A derived file cannot hit this case:
derivation only produces files some rule targets.

### `selftest` is unaffected

`selftest` runs a rule against its own fixture directories, named directly by
`LoadFixtures`. It does not consult kinds and must not: a docs rule's `.md` fixtures would
otherwise have to satisfy the repository's `docs` patterns, coupling a rule's self-test to
the config of whatever repository it happens to sit in.

### Docs

`README.md` **Configuration** replaces the `include` example with `targets` and explains
replacement semantics. **Rules** gains `targets` in the frontmatter example and in the
required-keys paragraph. The seven-rule table gains no column — all seven are `[tests]`, and
a column of one repeated value teaches nothing.

## Out of scope

- **Repository walking.** Discovery stays glob-driven. A `.gitignore`-aware walk is where
  this ends up if kinds ever need to mean "everything except", and it brings vendor
  directories, submodules and ignore-file precedence with it. Separate feature.
- **Negated patterns and per-kind ignores.** Ruff's `!` prefix is the precedent and the
  answer when someone needs it. Not before.
- **Ruff-style per-file-ignores** — turning one rule off for one path, as an exception
  rather than as a kind. A real want eventually, and a different axis: this feature decides
  what a rule is *about*, that one records where a repository has decided to live with a
  violation.
- **Per-rule severity or votes.** ESLint's `rules: { semi: "error" }` carries a level.
  aritu has one severity and one vote count per run, and nothing here changes that.
- **Shipping a `config` kind.** Plausible, unclaimed by any rule that exists. The vocabulary
  is open, so the repository that wants it writes two lines.
- **A `language:` key, anywhere.** Unchanged from `05-feat-grouped-rule-set`. The model reads
  the file; the filesystem layer resolves paths; the conventions table answers the rest.
- **Writing the first non-test rule.** This feature makes one expressible. Which one to
  write is a separate decision.

## Constraints

- Bound by `CONSTITUTION.md`. Specifically: kinds are a data table driving a lookup, not a
  switch on a kind value (R-4, R-13); the table sits in `internal/lib` and is imported by
  `internal/domain`, never the reverse (R-7).
- **No second copy of the naming conventions** (R-2). `tests` refines through
  `testpath.IsTestFile` and both `tests` and `code` take their patterns from the same
  extension list the conventions table is indexed by.
- `include` is **deleted, not deprecated** (R-17). `config.Config.Include`,
  `allResolvedAgainst`'s use for it, and the fallback in `targetsFor` go with it. A repo
  whose `aritu.yml` still carries `include` fails to load, because `KnownFields(true)` is
  already the contract and *"a setting that silently does nothing is worse than one that
  refuses to load."*
- `glob.Match` exists and is called from nowhere. This feature is what calls it, or it is
  deleted (R-17).
- `targets` patterns resolve against the config file's directory, like `rules.dir` and
  today's `include` — *"each path resolves in the frame it was written in."*
- No change to `verdicts`, `reasons`, the exit codes, or `selftest`'s hold semantics. This
  feature changes which pairs are formed, not what a judged pair returns.
- `run.Options.Files` and `run.Options.Rules` stay as they are. The pairing happens inside
  `Run`, so a caller still hands over two lists.

## Done criteria

A rule directory with `targets: [docs]` and `include: []`, enabled alongside the seven,
judges every `.md` file in the repository and no `.go` file. The seven judge every test file
and no `.md` file. One `aritu apply` with no arguments produces both, grouped by file, and
`README.md` and `internal/domain/rule/rule_test.go` each appear once with only the rules that
target them beneath.

`targets: [test]` — the singular typo — fails at load naming `tests`, `code`, `docs` and any
kind the config defines, before a single model call.

`aritu.yml` carries no `include` key, and a file that still does refuses to load.

`aritu apply README.md` with only the seven test rules enabled reports that no enabled rule
targets it and exits `2`.

`aritu apply internal/domain/rule/rule_test.go` produces seven verdicts, unchanged from
today: adding an axis to rule selection must not narrow the case that already worked.

`selftest` holds for every fixture of every rule, with no `targets` key consulted anywhere
in the run.
