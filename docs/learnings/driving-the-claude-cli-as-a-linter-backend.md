# Driving the Claude CLI as a linter backend

Measured against `claude 2.1.220` on 2026-07-26 while building the first pass of
aritu. Every number below came from an actual invocation, not from reasoning
about the CLI.

## Dynamic-key JSON schemas are unusable

The spec sketched the model's reply as a map from test name to verdict:

```json
{ "testFuncA": true, "testFuncB": false }
```

Expressed as a schema that is an object with `additionalProperties`, and the CLI
cannot produce it. The call burns all five structured-output retries and dies:

```
"terminal_reason": "structured_output_retry_exhausted"
"subtype": "error_max_structured_output_retries"
```

118 seconds and $0.27 for a failed call on a five-line file.

Rephrased as an **array of fixed-key objects**, the same question succeeds on the
first attempt in 2.5s:

```json
{ "results": [ { "name": "TestAlpha", "satisfies": true } ] }
```

So schemas want fixed keys. The map shape the spec describes is still what the
tool prints — it is just rebuilt in Go from the array rather than demanded from
the model. Where a schema and a desired output shape disagree, convert on our
side; the model is far better at filling a fixed shape than at inventing keys.

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
