---
slug: first-pass
type: feat
status: draft
created: 2026-07-26
---

------

# Feature: First pass

## Purpose

Setting up this repo with the first pass of the system to build. A shield (aritu) to prevent LLMs from writing tests that don't really add value. Think of it as an LLM linter. Eg a subagent that can be called with a file and a linting rule, that then returns structured output — which in turn will make the model know what is up with the test it wrote. (Eg could be part of CI or just a pre-commit hook etc.)

Keep in mind in the prompt for a rule that tests can come in different shapes — eg table-driven tests, functional tests etc.

## Scope

### Rules

Make a `/rules` directory. One rule per dir.

```
/rules/one-reason-to-fail/
  prompt.md                    # prompt to test that a given unit test has:
                               # "One reason to fail — one behavior, however many assertions"
  fixtures/
    pass-single-assert/
      scenario_test.go         # should compile on its own, passes the scenario
    pass-multiple-asserts-one-behavior/
      scenario_test.go
    fail-two-behaviors/
      scenario_test.go         # a scenario that doesn't pass when passed in
    fail-act-assert-chain/
      scenario_test.go
```

Each fixture gets its own subdirectory. Go files in the same directory are the same package, so two fixtures both declaring `func TestFoo` in one `fixtures/` dir won't compile. Directory name carries the pass/fail expectation via prefix.

Make the following 3 rules:

- Named for the behavior it protects, specifically
- One reason to fail — one behavior, however many assertions
- Doesn't mock the thing under test

Every `prompt.md` states both poles: the property the test **must** have, and
the shape or shapes that **disqualify** it. A prompt given only one pole drifts
toward answering everything the same way, and a rule that never fires is
indistinguishable from a rule that always passes.

### Output shape

```json
{
  "rule": "one-reason-to-fail",
  "file": "src/parser_test.go",
  "votes": 4,
  "verdicts": {
    "TestParsesHostAndPort": 4,
    "TestRejectsMalformedPort": 0
  }
}
```

Function name as key, value is how many of the `votes` runs said the test
satisfies the rule. A test passes only at `votes` — full agreement. Every other
count is a fail, `0` and `3` of `4` alike.

The count is not a second verdict; it is how close the prompt is. `3` of `4` is
nearly working, `0` of `4` on a test that should pass is broken. Both fail, and
the number is what tells them apart when tuning.

### Exit codes

- `0` — every test function unanimously satisfies the rule
- `1` — one or more do not, whether the votes were unanimously against or split
- `2` — could not run (model unreachable, file not found, malformed JSON from
  model, name/verdict mismatch)

The distinction matters in a commit hook: a crash exiting `1` looks like a rule
failure, a crash exiting `0` looks like a pass. Neither is acceptable.

A split vote is not a third outcome and must not be routed to `2`. The rule is
"all votes agree the test satisfies it"; a split does not meet that, so it
fails. Sorting it under "could not run" invites a hook to treat an unsure model
as a tooling problem to skip past, which lets exactly the test the tool exists
to catch through.

Output is always written before exiting, including on `2`. The counts are the
whole diagnostic; suppressing them on failure removes the reason to have them.

### CLI

```
# run a rule against a given file
# all votes must agree, else it's a non-verdict
aritu apply [rule] [file] --model [x] --votes n

# run apply against every fixture for a rule, compare each
# result to the expectation carried by its directory prefix
aritu selftest [rule] --model [x] --votes n
```

`selftest` is `apply` in a loop. Same prompt, same voting, same code path — it
adds only the comparison against the `pass-`/`fail-` prefix and a summary table
across fixtures. There is no second set of voting semantics to keep in sync.

It compares counts, never exit codes. A `pass-` fixture holds at `votes`; a
`fail-` fixture holds at `0`. Anything else means the rule is not reliable on
that fixture, including a `fail-` fixture the model only mostly rejected — `1`
of `4` would fail the file under `apply`, but a rule that needs a dissenting
vote to fire is one bad test away from missing.

`selftest` exits `0` when every fixture holds, `1` when any does not, `2` when
it could not run. The table prints either way.

### Context inclusion

For some rules and prompts certain context is best left out (eg sometimes it's better to ONLY have the test file and not the file under test). This is defined in the `prompt.md` frontmatter:

```yaml
include_source: true | false
```

Given Go convention `file_test.go` → `file.go`. When `include_source` is false, only the test file is passed to the model with the given prompt. When true, `file.go` is included too.

If `include_source` is true and the source file can't be resolved, skip the file and report it — don't silently run with partial context.

### Model calling

For now, just call claude-cli through `exec`, forwarding the model arg and (reasoning level arg if possible). Output shape from the model:

```json
{
  "testFuncA": true,
  "testFuncB": false
}
```

The surrounding structure (rule, file) is added programmatically when forwarding to the user. When forwaring to model make sure you limit agentic calling or turn it off. There are no tools to call. Just one output

### Determining function names in test file

For the output shape you need to know which functions in a file are actual test functions. Use the same model calling system with a prompt that returns structured JSON of test function names. Map that over the verdicts.

**Mismatch is an error, exit 2.** If the verdict call returns keys not in the name list, or omits names that are in it, do not merge and continue. Models are unreliable at exhaustive enumeration, and a silently dropped test is the exact failure this tool exists to catch.

## Out of scope

- Don't drag in any kind of code parsing — it's just reading files and asking the LLM for functions that contain tests.
- Don't write internal agentic calling over API — only `exec`.
- Don't write any kind of suppression comment system.

## Constraints

- Go language
- One bin that loads and reads a config dir `/rules`

## Done criteria

Once you have written the 3 rules, and at least 2 pass and 2 fail scenarios for each rule, you must be able to compile the application and run `selftest` against each of them on the sonnet model with 4 votes. 
