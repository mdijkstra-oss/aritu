# A schema property name becomes a tool parameter, and a mangled one breaks it

`VerdictSchemaFor` names every unit as a required top-level property. The CLI turns each of
those properties into a **parameter of its `StructuredOutput` tool**. That mapping is the
part nobody sees, and it is where a bad key does its damage.

A full sweep made the cost visible. The two `file`-granularity rules — whose key was the file
path, sanitised and cut to the API's 64-character ceiling — scored 8/14 and 6/12 with
**twelve** `structured_output_retry_exhausted` failures between them. The `function` and
`test` rules, in the same run at the same votes and concurrency, scored 14/14 and 14/14 with
**none**. Changing the file key to the constant `file` took the worst rule to 14/14, zero
errors, nothing else altered.

## What is actually happening, from the stream

`--output-format stream-json --verbose` shows every rejected attempt. All five attempts on
the failing call emitted the key **exactly right**, as **valid JSON**, with exactly the
required fields and no extras. The model was never at fault. What differed was the tool the
CLI had built:

| key | tool parameters | outcome |
|---|---|---|
| `file` | `["file"]` | success on the first attempt |
| `rules_no-gaps_fixtures_fail-go-insufficient-funds-never-reached_` | `["$PARAMETER_NAME"]` | rejected five times |

The long name failed to become a parameter and left an unsubstituted `$PARAMETER_NAME`
placeholder. The model then packs the whole object into that placeholder as a string. That
degraded path is not an automatic failure — it survived a trivial one-line answer — but it
failed 5/5 on a real 265-character verdict.

## The trigger is narrower than "long"

Probing one property at a time, at exactly 64 characters:

| key shape | parameter |
|---|---|
| hyphens **and** a trailing `_` | `$PARAMETER_NAME` |
| hyphens, ends alphanumeric | mapped |
| trailing `_`, no hyphens | mapped |
| hyphens, `_` present but not last | mapped |

Sixty-four `a`s map fine, so length alone is not it. It takes the ceiling, hyphens and a
trailing separator together — which is exactly what truncating a path produces, and pure
chance that aritu generated it.

## What follows

- **Where there is one unit, the key has no work to do.** At `file` granularity the path is
  already on the left of the arrow in the prompt and in the report, so a constant is strictly
  better than a derived name.
- **Never let a key end on a separator.** `truncateKey` now trims trailing `_-.` after the
  cut, which removes the observed trigger for every granularity rather than only the one that
  hit it.
- **Measure the rate, do not test the case.** Both keys answer correctly most of the time; a
  single-shot reproduction shows nothing. Only a run of fifty-odd calls separates a 100% key
  from a 60% one.
- **Read the rejected attempts before believing an explanation.** The first story here — that
  the model was mistyping a hard-to-copy key — fitted every number and was wrong. The stream
  said so in one line.

See [[a-required-reason-must-be-asked-for]] for the other prompt/schema mismatch that this
same failure mode was hiding.
