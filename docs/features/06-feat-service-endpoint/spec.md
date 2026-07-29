---
slug: service-endpoint
type: feat
status: draft
created: 2026-07-27
---

------

# Feature: Service endpoint

## Purpose

Every model call aritu makes is a subprocess. `claudecli.Exec` shells out to a
binary, writes the prompt to its stdin and parses an `--output-format json`
envelope back (`internal/lib/claudecli/claudecli.go:32-47`). That binds the tool
to one vendor's CLI being installed, authenticated and on `PATH` — the README
says so in its second paragraph: "Requires Go 1.25+ and the `claude` CLI on
`PATH`, already authenticated" (`README.md:22`).

The linter itself has no opinion about who answers. It sends a prompt, a model
name, an optional effort and a JSON schema, and reads back one JSON value —
that is the entire `Request` (`claudecli.go:14-19`). Any Responses-compatible
endpoint can answer that, and a repository that already runs a gateway, a proxy
or a self-hosted model should be able to point aritu at it and pay nothing for a
CLI it does not use.

So the transport becomes configuration: `aritu.yml` names an endpoint, aritu
talks HTTP, and the subprocess goes away entirely.

## Scope

### The `service` block

```yaml
service:
  endpoint: https://gateway.internal/v1
  auth_token_var: ARITU_TOKEN
```

`endpoint` is the base URL of a Responses-compatible API.

`auth_token_var` is **the name of an environment variable**, never a token. The
field is named for what it holds so that a config file read at a glance cannot be
misread as a place secrets go; `token: ARITU_TOKEN` would look like a token
called `ARITU_TOKEN`. Its value is read at startup and sent as
`Authorization: Bearer <value>`.

`auth_token_var` is optional. Omitted, no `Authorization` header is sent at all,
which is what a local endpoint that ignores auth wants. Named but unset or empty
in the environment, the run fails at startup before any request:

```
aritu: service.auth_token_var names $ARITU_TOKEN, which is not set
```

A typo in a variable name is otherwise paid for as a wall of 401s from every call
in the sweep, arriving minutes later. The distinction is between "no auth" and
"auth I asked for and did not get", and only one of those is an error.

`endpoint` is required. There is no default: a run that silently reached
`api.openai.com` because a key was missing from the file would be a surprising
place for a repository's source to end up.

### The transport

`github.com/openai/openai-go/v3` (`v3.46.0`, tagged 2026-07-23; there is no v4),
pointed at the configured base URL:

```go
responses.NewResponseService(
    option.WithBaseURL(endpoint),
    option.WithHeader("Authorization", "Bearer "+token),  // only when configured
)
```

The service is constructed directly rather than through `openai.NewClient`.
`responses` is a self-contained subpackage that does not import the root
`openai` package, so reaching for the client would compile in images, audio,
batches, fine-tuning and the admin surface to make one call that touches none of
them. Same behaviour, four fewer packages.

It is the only Go library that speaks this API. `sashabaranov/go-openai`, the
community alternative, has no Responses support at all — no endpoint, no types —
and Chat Completions is explicitly out of scope below.

`option.WithAPIKey` is deliberately not used. It is one header spelled a
particular way, and this feature already has a general mechanism for the header;
two ways to set `Authorization` is a precedence question nobody needs to answer.

One call per request, non-streaming — `client.Responses.New`. Nothing about
aritu's structure wants a token at a time: a verdict is a single JSON object that
is either parsed or not.

### `Request` maps onto `ResponseNewParams`

The four fields of `Request` (`claudecli.go:14-19`) each have a home, and the
mapping is the whole of the new leaf:

| `Request` | `ResponseNewParams` |
|---|---|
| `Prompt` | `Input` (`OfString`) |
| `Model` | `Model` |
| `Effort` | `Reasoning.Effort`, omitted entirely when empty |
| `Schema` | `Text.Format`, as a strict `json_schema` |

`SystemPrompt` (`claudecli.go:28`) moves from the `--system-prompt` flag to
`Instructions`. Its job does not change: it is what keeps the reply to bare JSON
and off tool use. The `--tools ""`, `--safe-mode`, `--no-session-persistence` and
`--strict-mcp-config` flags have no counterpart and need none — an HTTP request
carries no session, no MCP config and no tools unless one is sent.

**Effort needs no translation.** The Responses API accepts `none`, `minimal`,
`low`, `medium`, `high`, `xhigh` and `max` (`shared.ReasoningParam.Effort`), and
aritu's five levels (`main.go:177`) are a subset. The comment calling them "the
levels the claude CLI accepts" is what changes.

**Schema needs one.** `Request.Schema` is `json.RawMessage`;
`ResponseFormatTextConfigParamOfJSONSchema(name, schema)` wants
`map[string]any`. The leaf unmarshals it, and sets `Strict: true` so the endpoint
enforces the schema rather than merely suggesting it — that is the property the
generated verdict schema depends on for uniqueness and completeness of keys.

The format needs a `name`, which the CLI never asked for. It is one constant in
the leaf, not configured and not per-call.

Naming the two call sites apart — enumeration and verdicts — would mean carrying
the name on `Request`, and a fifth field changes both literals in `lint.go`,
which contradicts the promise below that every call site changes by its import
line and nothing else. The name buys nothing to pay for that: it is a label the
API requires, constrained to `^[a-zA-Z0-9_-]+$`, and it does not affect
validation, strictness or the shape of the reply. Two labels nothing reads would
be two things to keep in sync for no observable difference.

### Reading the answer

`Response.OutputText()` concatenates the `output_text` content and is the reply,
parsed by the same `json.Unmarshal` in `lint` that reads the CLI's envelope
today. `structured_output` has no equivalent and needs none: with a strict
`json_schema` format, the output text *is* the structured value.

### What "retryable" means now

`Retry` (`claudecli.go:58`) exists to start a fresh turn when the model answered
in the wrong shape, and it retries exactly one classified condition —
`errModelFailure` — leaving unstartable binaries, cancelled contexts and
unparseable envelopes alone, because a second attempt would meet them again. That
policy survives; only the classification moves:

| condition | today | with an endpoint |
|---|---|---|
| model answered wrong | `is_error` in the envelope | `status` is `failed` or `incomplete`, or a refusal |
| worth another turn | yes | yes |
| transport failed | binary would not start | connection refused, DNS, 4xx |
| worth another turn | no | no |

Transport retries are the SDK's, not aritu's. `option.WithMaxRetries` already
covers 408/409/429/5xx with backoff, which is a different failure from the one
`Retry` is about, and stacking aritu's turn-level retry on top of it for the same
condition would multiply a rate-limited sweep rather than pace it.

`Throttle` (`claudecli.go:82`) is unchanged and still the one ceiling every call
passes through. Its reason — that fixture-level and vote-level concurrency
multiply — is about the callers, not the transport.

### What is deleted

- `internal/lib/claudecli`'s `Exec`, `Args`, `ParseResult`, `envelope` and
  everything reading an `--output-format json` wrapper.
- The `--claude` flag (`main.go:41`) and its default (`main.go:81`).
- The `claude:` config key (`config.go:27`) and its resolver entry
  (`config.go:151`).
- The README's CLI prerequisite (`README.md:22`) and the `--claude` row of the
  flag table (`README.md:402`).

Per R-17 these go rather than being kept behind a switch. A repository that wants
the old behaviour points `endpoint` at a local proxy; keeping two transports alive
would double every error path and every test table for a path with no second
caller.

### Where the code lives

The package is renamed, because `claudecli` names a thing that is gone. It keeps
its position in the dependency graph (R-7) as a library under `internal/lib`,
imported by `lint`, `run` and `selftest` exactly as now.

`Ask`, `Request`, `Retry` and `Throttle` keep their names and signatures, so
every call site in `lint.go`, `run.go` and `selftest.go` changes by its import
line and nothing else. The seam was already the right shape; this feature only
changes what sits behind it.

## Out of scope

- **Streaming.** No caller wants a partial JSON object.
- **Multiple services, or per-rule endpoints.** One repository, one endpoint,
  the same way there is one model and one vote count.
- **A `Chat Completions` fallback.** Responses is the contract; an endpoint that
  does not speak it is not supported.
- **Extra headers beyond `Authorization`.** If a gateway later needs
  `x-org-id`, that is a second field and a second decision.
- **Sending the token from anywhere but an environment variable.** No literal
  tokens in `aritu.yml`, no file paths, no keychain.
- **Retries, timeouts or concurrency policy changes.** `--timeout`, `--jobs` and
  the three attempts keep their current meanings.
- **Prompt content, rule set, unit model, exit codes and report shape.** None of
  them can tell the difference.

## Constraints

- Go. Bound by `CONSTITUTION.md`.
- The `Ask` seam stays a function type, so domain tests keep running against
  table data with no model and no HTTP (R-3, no mocks).
- The request mapping is a pure function — `Request` in,
  `responses.ResponseNewParams` out — table-testable without a network. This is
  the direct replacement for the `Args` table that tests the CLI flags today.
- Auth resolution is a pure function of the config value and a lookup function,
  not a package-level `os.Getenv` (R-4: dependencies in explicitly, IO at the
  boundary).
- A token never reaches an error message, a log line or a report.
- Config decoding stays strict (`config.go:83`), so `servce:` is an error rather
  than silence, and a `service` block with an unknown key is too.
- Exit codes are unchanged. A missing endpoint or unset variable is a
  could-not-run.

## Done criteria

`aritu selftest --rule no-gaps` holds the same fixtures against a configured
endpoint that it holds against the CLI today.

One rule, not the whole set. Every rule sends the same four fields through the
same mapping, so a second rule re-tests the mapping rather than testing anything
new, and a full sweep is seven rules times fifty fixtures of model latency to
learn it twice. `no-gaps` is the one worth running: it is `granularity: file`
with `include_source: true`, which makes it the largest request the tool
sends — a whole test file plus its implementation — and prompt size is the one
dimension where a subprocess writing to stdin and an HTTP body could genuinely
differ.

`selftest` is the check rather than a pair of `apply` runs because the fixtures
carry ground truth in their directory names: a `pass-` fixture must hold at
exactly `votes`, a `fail-` fixture at zero (`selftest.go:57-59`). Each transport
is therefore compared against the expectation, not against the other transport's
output. Diffing two reports would measure model non-determinism — a unit that
votes 3/3 on one run and 2/3 on the next has told us nothing about the transport
— and `Reasons` is free prose that will never repeat.

The bar is that the same fixtures hold. A fixture that flips is a real signal and
gets investigated, widening to `proves-what-it-claims` (test granularity) and
`readable` (function granularity, no source) to locate it. A differing vote count
on a fixture that still holds is not a signal.

`go build ./...` finds no reference to a `claude` binary anywhere in the tree.

The request mapping is covered by a table including: no effort, each effort
level, a schema present and absent, and prompt text large enough that it would
once have been an `ARG_MAX` concern.

Auth resolution is covered by a table: no `auth_token_var` (no header), a set
variable (header present, value from the environment), a named-but-unset
variable (startup error naming both the key and the variable).

The retry classification is covered by a table pinning which conditions start a
fresh turn and which fail immediately.
