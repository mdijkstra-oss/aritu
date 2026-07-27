You are judging the units of a file against one rule. The rule follows this text; the
file follows the rule. For each unit you are given, return a verdict: whether that unit
satisfies that rule.

Only the rule below applies. A unit may be badly named, may do two things at once, may
stub out half the world — none of that is your concern unless the rule below asks about
it. Judge one criterion, on the units you are given, and nothing else.

## What a unit is

A unit is whatever the rule below judges, and the rule declares which. Sometimes it is
each named thing in the file, sometimes each leaf of one, and sometimes the file itself —
in which case you are given one unit, its path, and you return one verdict covering
everything in it. Judge each unit you are given on its own, exactly as identified.

{{unit_model}}

## Writing the reason

Every unit's answer carries a reason, so write one for every unit. For a unit that does not
satisfy the rule the reason is the whole diagnostic and the guidance below applies to it.
For a unit that does satisfy it, one short clause naming what carries it is enough; nobody
reads those, and they exist so that every answer has the same shape. When the unit is a
whole file, the reason is the only guidance a reader gets, so it has to do the locating work
the unit name cannot.

- Exactly one sentence.
- About this unit against this rule. The rule restated in the abstract is not a reason,
  and neither is asserting non-compliance: "does not satisfy the rule", "violates the
  criterion" and "a unit must pin down one behaviour" tell the reader nothing they did
  not already have from the verdict.
- **Name the thing in the file that causes it** — the identifier, the case label, the
  assertion, the second act, the substituted type, the call that reaches outside. A reader
  who has this sentence and nothing else must know where in the file to look and what to
  change. Quote the offending fragment when a name alone would not locate it.
- Write to the person who has to fix this, not about them.

Weak: `the name is not behaviour-focused`.
Better: `names the unit under test with no stated outcome, so it would still read as true
whatever parsing produced`.

---

{{rule}}

---

Judge exactly these units against the rule above. Each line gives the unit, then the key to answer under:
{{units}}

Judge the unit as written on the left. The key on the right is only where the answer goes.

{{sources}}
