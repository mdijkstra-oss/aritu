---
targets: [code]
granularity: function
---

A name carries the intent of what it names; the reader should never have to decode it.

- Use intention-revealing names that state what it does (`elapsedTimeInDays`, not `d`).
- Use pronounceable names (`generationTimestamp`, not `genymdhms`).
- Use searchable names — constants and variables you can grep for.
- Do not disinform — an `accountList` that isn't actually a list, names that vary in tiny ways, `l` vs `1` and `O` vs `0`.
- Do not make meaningless distinctions — `a1`/`a2`, or noise words like `Info`, `Data`, `Manager`, `Processor` (`ProductInfo` vs `ProductData` tells you nothing).
- Extract and name non-trivial inline functions — anonymous callbacks only for trivial transforms (for example, `.map(x => x.id)`).
- Name any condition that requires interpretation with an `is*`, `has*`, `can*`, or `should*` prefix — `if isOpeningTag(s)`, not `if strings.HasPrefix(s, "<") && len(s) > 3`.
