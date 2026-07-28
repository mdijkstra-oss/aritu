---
targets: [code]
granularity: declaration
priority: med
---

A name carries the intent of what it names; the reader should never have to decode it.

- Use intention-revealing names that state what it does (`elapsedTimeInDays`, not `d`).
- Accept short names the language's idiom or the domain's own notation sanctions — loop indices, `err`, `ctx`, method receivers, `dx` in vector math.
- Use pronounceable names (`generationTimestamp`, not `genymdhms`); an established domain acronym (`rgba`, `utc`) counts as pronounceable.
- Use searchable names — constants and variables you can grep for.
- Do not disinform — an `accountList` that isn't actually a list, names that vary in tiny ways, `l` vs `1` and `O` vs `0`.
- Do not make meaningless distinctions — `a1`/`a2`, or noise words like `Info`, `Data`, `Manager`, `Processor` — but flag a noise suffix only when both look-alike names appear in this file; a lone `UserData` may be distinguishing a counterpart elsewhere.
- Extract and name an inline function that branches or runs longer than a few lines — anonymous callbacks only for short single-expression transforms (for example, `.map(x => x.id)`).
- Name a condition that combines two or more clauses and requires interpretation with an `is*`, `has*`, `can*`, or `should*` prefix — `if isOpeningTag(s)`, not `if strings.HasPrefix(s, "<") && len(s) > 3`.
