# Driving the Claude CLI as a linter backend

Measured against `claude 2.1.220` on 2026-07-26 while building aritu. Every number
and every error string below came from an actual invocation, not from reasoning
about the CLI.

The test for anything in this file is whether it stays true when aritu changes.
Facts about the CLI, the API and the model belong here; snapshots of our own output
do not, because re-running the tool is faster and cannot be out of date.

## Open-ended key schemas fail; generated fixed-key schemas do not

The spec sketched the model's reply as a map from test name to verdict:

```json
{ "testFuncA": true, "testFuncB": false }
```

Expressed as an object with `additionalProperties`, so the model invents the keys,
the CLI cannot produce it. The call burns all five structured-output retries and
dies:

```
"terminal_reason": "structured_output_retry_exhausted"
"subtype": "error_max_structured_output_retries"
```

118 seconds and $0.27 for a failed call on a five-line file.

**The conclusion first drawn from that was too broad, and it stood for a while
before being caught: "dynamic-key schemas do not work."** What actually fails is
asking the model to *invent* the keys. When the key set is already known — and for
a verdict call it is, because a separate call enumerated the units a moment
earlier — the schema can name every key explicitly:

```json
{ "type": "object",
  "properties": { "TestParseConfig.host_and_port": { "type": "object", ... } },
  "required": ["TestParseConfig.host_and_port"],
  "additionalProperties": false }
```

Generated a millisecond before the call, and indistinguishable from a hand-written
schema as far as the model is concerned. That works, and it is strictly better than
the array shape: an object cannot repeat a key, cannot omit a required one and
cannot carry an extra one, so duplicated, dropped and invented units stop being
errors the caller detects and become schema violations the CLI retries by itself.

The general lesson is not about schemas. A single measurement supports a narrow
claim. "This schema shape failed" was the observation; "this class of schema is
unusable" was an invention laid on top of it, and it closed off the better design
until someone pushed back.

## Schema property keys are constrained, and the error says so

A generated key like `TestParseConfig:extracts_host_before_colon` is rejected:

```
API Error: 400 input_schema.properties:
Property keys should match pattern '^[a-zA-Z0-9_.-]{1,64}$'
```

So: letters, digits, underscore, dot, hyphen, and **64 characters at most**. No
colons, no slashes, no spaces. Two consequences worth knowing before designing a
key scheme:

- A human-readable identifier cannot be the key. `TestFoo (some case name)` has
  spaces and parentheses; a file path has slashes. Both need normalising, which
  makes normalisation load-bearing rather than cosmetic.
- 64 characters is shorter than it sounds once a key is a function name plus a case
  name. Truncation is unavoidable, and truncation manufactures collisions between
  keys that were distinct — long case names in one function tend to share a prefix.
  Whatever disambiguates them has to live inside the same character set, so Go's
  own `#01` convention for duplicate subtests is not available.

## The default system prompt dominates the bill

A bare `claude -p` call carries the whole Claude Code system prompt, CLAUDE.md
discovery, and tool definitions — around 60k cached tokens before the question is
even asked. For a one-shot classification that is pure overhead.

Replacing it collapses the cost:

| invocation | latency | cost |
|---|---|---|
| default system prompt, dynamic-key schema | 118s | $0.2711 |
| `--safe-mode --system-prompt <one line>`, array schema | 2.5s | $0.0058 |

46× cheaper, 47× faster. The flags that matter:

- `--system-prompt` replaces the default prompt outright (`--append-system-prompt` would not).
- `--safe-mode` drops CLAUDE.md, skills, plugins, hooks and MCP while leaving auth working.
  `--bare` looks similar but forces `ANTHROPIC_API_KEY` and breaks subscription auth.
- `--tools ""` removes every tool, so there is nothing to call and no agentic loop to bound.
- `--no-session-persistence` and `--strict-mcp-config` keep a linter run from touching
  session state or picking up ambient MCP servers.

## Failure arrives on two channels at once

On an API error the CLI exits **1** *and* writes a complete, parseable envelope
to stdout with `"is_error": true` and a `terminal_reason`. Checking only the exit
status loses the model's own error text; checking only `is_error` misses failures
that never produced an envelope. Read both — and on a non-zero exit still try to
parse stdout before falling back to the exit status.

## The prompt goes on stdin

`claude -p` with no positional argument reads the prompt from stdin. Worth using
unconditionally: a linter splices whole source files into its prompt, and argv
has a hard size limit that stdin does not.

## Unanimity across votes is realistic

The voting design only pays off if a well-written rule actually produces the same
answer every time. Probing one file holding an obviously-good and an
obviously-bad test name, judged by a two-pole prompt, four independent sonnet
calls returned identical verdicts — 4/4 for the good name, 0/4 for the bad one —
in about 4 seconds wall-clock when run concurrently, for $0.024.

Split votes are therefore a signal about the *prompt*, not noise to be averaged
away, which is what makes the count worth printing on failure.
