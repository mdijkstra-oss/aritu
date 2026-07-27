# Rules aritu cannot judge

`~/.claude/CONSTITUTION.md` still exists and still loads into every session, so
these rules reach an agent the way they always have: as prose in a prompt, handed
over ahead of the work.

That is the weakness this repository exists to close. **A rule in a prompt is a
request.** Nothing reads the code back afterwards, nothing disagrees, and nothing
fails: whether the rule was followed comes down to whether the model was inclined
to follow it, and the only account of that is a diff summarised by the same model
that wrote it. The rules under `rules/` are those same rules made checkable — a
model reads each file, returns a verdict per unit with a reason, several runs have
to agree before anything passes, and a repository that breaks one gets a non-zero
exit code out of a hook rather than a paragraph of good intentions. Being told
becomes being held to it.

These four cannot make that move. aritu judges a file, and none of them is a
property of one — they are about how the work is done: what happens before a
commit, what to do on discovering a cycle, when to ask rather than guess, and what
a claim has to rest on. A model handed a source file and asked whether it "grounds
every claim in evidence" has nothing to read the answer off, so the verdict would
be noise. So these stay on the prompt side, where they are asked for and not
enforced, and this file records which ones those are and why — rather than leaving
the split to be rediscovered later by whoever wonders where R-1 went.

They keep the numbers of the constitution they came from, so a rule here and a rule
under `rules/` can still be told apart by name.

## R-1. Code history is controlled

- When asked to commit, stage the changes and show the proposed commit message. Do not commit until the maintainer approves the message.
- Use a single-line conventional commit. No body, no description.
- Never add attribution of any kind. Specifically: never add "Generated with Claude Code", "Co-Authored-By: Claude", or any similar trailer or credit line.
- Destructive git commands (`checkout`, `reset`, `clean`, `stash drop`) require two explicit confirmations from the maintainer. A single instruction such as "revert that" counts as one confirmation; ask again before executing.

## R-6. Import cycles are escalated, not fixed

- If you detect an import cycle, report it: state where it occurs and which modules are involved. Do not refactor to resolve it. Wait for the maintainer's direction on the correct fix.

## R-15. Investigate and act; ask only what cannot be determined

- Use the available tools (Read, Grep, Glob, Bash, WebSearch, WebFetch) to find answers rather than guessing.
- Do not use hedging language ("likely", "probably", "might be") in place of investigation. Look it up.
- The only valid questions to the maintainer concern requirements, preferences, and product decisions.
- If the correct answer is knowable, determine it and act. Do not present a menu of options when one is clearly correct.
- Present genuine trade-offs only. If a simpler approach exists, state it.
- For external APIs, models, or providers, consult the current official documentation. Do not rely on training data, which may be outdated; model names, API shapes, and fields change.

## R-18. Ground every claim in evidence

- Every factual claim must be grounded in a file path with line numbers, a quoted snippet, a tool result, or a documentation URL. If you cannot cite it, do not assert it.
- Do not use "I think", "it seems", "this probably does", or "from memory" as a basis for a claim.
- Before stating what code does, open the file, cite `path:line`, and quote the relevant line.
- Before stating what an API returns, fetch the documentation and quote the field.
- If asked a question for which you lack a citation, answer "I don't know yet — looking it up", then perform the lookup. Do not present a guess as fact.
- Treat memory (auto-memory, earlier turns, training data) as a hypothesis to verify against the live file, not as a citation.
