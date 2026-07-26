---
slug: generated-verdict-schema
type: feat
status: draft
created: 2026-07-26
---

------

# Feature: Generated verdict schema

## Purpose

Running the linter over its own `cmd/aritu/main_test.go` exited `2`:

```
verdict for test unit "TestRun (apply with a surplus positional)" given twice
```

Nothing was wrong with the test. The model returned the same unit twice in one
reply, aritu could not tell which verdict to keep, and refused to guess. That file
has nineteen leaves whose names share their first eleven characters, and copying
nineteen near-identical strings without slipping is the clerical work models are
worst at.

The verdict reply is an array of `{name, satisfies, reason}` objects, and an array
carries no uniqueness constraint, so the reply above validates against the schema
perfectly. The schema checks shape; the damage is in the content. That is why the
duplicate has to be caught in Go, after the fact, as a could-not-run.

The array shape was chosen because an **open-ended** object schema — one where the
model invents the keys — exhausts the CLI's structured-output retries. That
measurement was real, but the conclusion drawn from it was too broad. By the time
the verdict call is made the unit list is already known, so the schema can name
every key explicitly. A schema generated a millisecond earlier is still a fixed
schema as far as the model is concerned.

Naming the keys moves three failure classes out of aritu's error handling and into
the schema, where the CLI retries them automatically instead of aritu exiting `2`:

| failure | today | with named keys |
|---|---|---|
| a unit judged twice | exit `2` | impossible, object keys are unique |
| a unit dropped | exit `2` | impossible, every key is `required` |
| a unit invented | exit `2` | impossible, `additionalProperties: false` |

## Scope

### One object per file, keyed by unit

The verdict reply becomes a single object whose keys are the units enumerated for
that file:

```json
{
  "TestParseConfig.extracts_host_before_colon": { "satisfies": true,  "reason": "" },
  "TestParseConfig.host_and_port":              { "satisfies": false, "reason": "names the input the case supplies, not the outcome it protects" }
}
```

One object per file, not per unit — the call is already per file, and splitting it
per unit would multiply requests by the leaf count for no gain.

### Key format

**Measured constraint.** The API rejects a schema whose property keys fall outside
`^[a-zA-Z0-9_.-]{1,64}$`:

```
API Error: 400 input_schema.properties:
Property keys should match pattern '^[a-zA-Z0-9_.-]{1,64}$'
```

So normalisation is not cosmetic, it is what makes the key legal at all. Colons,
slashes and spaces are all out, which rules out both the raw identifier and a file
path used verbatim.

`<TestFuncName>.<case name, snake_cased>`, and the bare function name where a test
declares no cases. A dot separates the halves because a colon is not permitted.
Anything still outside the character set becomes `_`, and the result is cut to 64.

The function name is kept verbatim. Only the case name is normalised, because it
is the half that carries arbitrary prose.

**No filename prefix.** The request already covers exactly one file, so a filename
scopes nothing and costs tokens on every key.

**Collisions get `-01`.** Two cases in one function can normalise to the same key —
`"empty input"` and `"empty  input"` both reach `empty_input` — and cutting to 64
characters creates more, since long case names in one function tend to share a
prefix. Truncation is in fact the likelier source of the two. Left alone the second
overwrites the first while building the schema, `required` lists one key where two
units exist, and a test silently vanishes from the run with every count still
looking healthy. That is precisely the silent drop the mismatch check was written
to prevent, reintroduced through the back door. Append `-01`, `-02` on collision. Go uses `#01` for duplicate subtest names, but
`#` is outside the permitted character set, and a hyphen never appears in a
snake-cased key so it stays unambiguous.

### The judged text stays the original

The prompt lists each unit as it appears in CI, alongside the key to answer under:

```
- TestParseConfig (extracts host before colon)   →   TestParseConfig.extracts_host_before_colon
```

This is not redundancy. `named-for-behavior` judges the identifier **as text**, so
a model shown only `TestParseConfig.extracts_host_before_colon` would be judging the
normalised handle rather than what a reader actually sees. The key is transport; the
original is the subject.

### Mapping back

`Report.Verdicts` and `Report.Reasons` keep the original identifiers. The
normalised keys never leave the package — they exist only between building the
schema and reading the reply.

### What happens to the mismatch check

`checkNamesMatch` becomes unreachable in normal operation: the schema now enforces
exactly the set it used to verify. It stays anyway, as an assertion against a
remote contract this code does not own, and its error path stays exit `2`. If it
ever fires, the CLI's schema validation has failed and a could-not-run is still the
honest answer.

The duplicate check inside the verdict parse can go: a JSON object cannot express
the condition it tests for.

## Out of scope

- **Schema size.** A rule at `test` granularity over `internal/domain/rule/rule_test.go`
  produces 61 units, so 61 property definitions in one schema. The 64-character key
  ceiling is now known and handled; whether the property *count* has a ceiling of its
  own is a question for later.
- **A retry policy.** Retries now belong to the CLI's schema validation. Adding a
  second layer inside aritu would change what exit `2` means, from "I could not
  check" to "I could not check twice".
- Batching several files into one request.
- Changing the enumeration call, which still returns an array. Its content is
  genuinely unknown ahead of time, which is what array shape is for.
- Changing the printed output shape, the exit codes, or `selftest` semantics.

## Constraints

- Go. Bound by `CONSTITUTION.md`.
- `Report.Verdicts` stays `map[string]int` keyed by the original identifier, and
  `reasons` stays beside it. Nothing about the printed output changes.
- Exit codes are unchanged, and nothing here may route a judgement into exit `2`.
- Key generation is a pure function of the enumerated unit list, table-testable
  without a model.
- The schema is built from data, not string-concatenated: marshal a typed value so
  a case name containing a quote cannot produce invalid JSON.

## Done criteria

`aritu apply named-for-behavior cmd/aritu/main_test.go` completes instead of exiting
`2` on a duplicated unit.

Key generation is covered by a table including the collision case, and the
round trip from original to key and back is pinned.

`selftest` still holds for every fixture of every rule on sonnet, as before.
