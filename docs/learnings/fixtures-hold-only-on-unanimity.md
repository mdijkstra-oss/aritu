# A fail- fixture needs every unit in it to fail

`selftest` compares counts, not exit codes. `Holds` asks `vote.IsUnanimous(counts, votes)`
for a `pass-` fixture and `vote.IsUnanimous(counts, 0)` for a `fail-` one, so a fixture holds
only when **every unit the file yields** lands on the same pole in **every vote**.

At `file` granularity that reads the way you expect: one unit, one verdict. At `function` and
`test` granularity it does not. A `fail-` fixture there yields one unit per test — or per leaf
— and every one of them has to be rejected. One innocent test sitting beside the offending
one produces `{offender: 0, bystander: 4}`, which is neither unanimous-for nor
unanimous-against, and the fixture misses.

So a `fail-` fixture is not "a file containing a violation". It is a file in which **nothing**
satisfies the rule. In practice that means one test, or a small set that all exhibit the same
shape. The instinct to write a realistic file with one planted defect produces a fixture that
can never hold.

The mirror is easier to remember but worth stating: in a `pass-` fixture every unit must
satisfy the rule, including the ones that are only there as scenery.

This is also why near-miss fixtures — the ones proving a rule does *not* over-fire — are the
expensive ones to get right. They have to look like the violation and be unanimously judged a
satisfaction, four runs out of four.

See [[go-fixtures-are-live-code]] for the other constraint fixtures have to satisfy.
