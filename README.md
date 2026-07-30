# aritu

<p style="color: red;">Not sure if this is the path to go down to. May abandon this.</p>

An LLM linter. Give it a rule written in prose and a file; it asks a model whether
each unit of the file satisfies the rule, several times over, and reports how many
runs agreed. Run it in CI or a pre-commit hook to enforce what no parser can check:
the comment that restates its code, the name that says nothing, the test that
proves nothing.

No AST, no matcher DSL, no suppression comments, no language flag — a model reads
the file and reports what it sees, so one rule set judges Go, TypeScript, Python,
anything.

## Build & use

```sh
go build -o aritu ./cmd/aritu    # Go 1.25+, any Responses-compatible endpoint

aritu apply                      # every enabled rule over everything it targets
aritu apply 'internal/**/*.go'   # globs, ** included; matching nothing is an error
aritu apply --rule intent-revealing-names parser.go
aritu selftest --votes 4         # judge the rules against their own fixtures
aritu rulebook > AGENTS.md       # write the same rules out as instructions
```

A unit passes on a strict majority of votes: `✓` pass, `✗` fail, `!` tie — which
fails, half is not a majority. Exit `0` all pass, `1` any fail, `2` something could
not run, which outranks `1` so a partial sweep never reads as a complete one.

`rulebook` renders the exact text the model judges against, so the standard you
hand an agent and the standard it is held to are one file and cannot drift apart.

## Configuration — `aritu.yml`

```yaml
service:
  endpoint: https://gateway.internal/v1  # required; deliberately no default
  auth_token_var: ARITU_TOKEN            # env var NAME, never a token; omit for none
  model: sonnet
votes: 2
rules: { dir: ./rules, enabled: [intent-revealing-names] }  # omit enabled for all
targets: { migrations: ['db/**/*.sql'] } # replace built-in kinds or add your own
exclude: ['vendor/**', '**/*.gen.go']    # .gitignore already respected, via git
```

Precedence: defaults, then file, then flags. Unknown keys are errors. `exclude`
and `.gitignore` bound what a sweep derives, never a file you name explicitly.

## Rules

One directory per rule under `rules/`: a `prompt.md` and `fixtures/` of `pass-` /
`fail-` directories, which `selftest` holds to their prefix — a `pass-` fixture
must pass unanimously, a `fail-` fixture must get zero votes.

```markdown
---
targets: [code]         # tests | code | docs | your own
granularity: function   # file | function | test_case
priority: med           # severe | high | med — no low: not worth fixing, not worth a rule
---
The criterion, stated in both directions: the property required and the shapes
that disqualify it.
```

The rule's name is its directory name; the body is the whole rule, judged and
preached from the same text. Prefix the directory with `_` to park it: kept on
disk, runnable by name, out of sweeps and the rulebook.

Flags: `--rule` (repeatable), `--votes` (1), `--parallel` (5), `--timeout` (10m),
`--output json`, `--rules`, `--config`, `--debug`.
