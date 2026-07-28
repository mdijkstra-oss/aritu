---
name: aritu-review
description: Run aritu against this repository file by file and fix what it flags. Use when asked to review code quality / cleanliness, lint and fix, or clean the repo against its own rules.
---

# Auto-fix

Run aritu against a repository and fix what it flags. Resolve commands from
the repository you are standing in: a make target if one exists, otherwise
the ecosystem's own runner. Judge only through a path that rebuilds aritu
first. Scope defaults to the files changed against the default branch,
filtered to what the enabled rules target; a full run means every file they
target; files named by the user are the queue exactly as given, no seeding.

## Init
Before starting fixing familiarize yourself with the rules by running `make rulebook`

## Goals

Everything else derives from these. When something comes up that no rule
covers, decide from the goals.

1. **Every resting point is green.** Mid-edit the tree may be anything, but
   wherever the run stops — moving to the next file, giving up on one,
   finishing — it stands on a state that builds, passes the suite, and is
   committed.

2. **Bank progress, roll back failure.** An edit that builds, keeps the suite
   green, and leaves findings strictly better with none newly introduced is
   committed immediately: one file, staged by name, message naming the file
   and the change. An edit that didn't earn that costs one rollback to the
   last green commit, never the session's earlier wins.

3. **Fix causes, not counts.** A finding is a symptom: seven arguments can
   mean a function doing two jobs, and the fix is the split, not the smallest
   edit that moves the number. Read findings together — they describe one
   shape. A finding that is wrong gets no code change at all: record it as
   skipped, and as a fixture candidate plus prompt change for the rule that
   misfired.

4. **Never judge the same content twice.** Verdicts attach to content, not
   filenames: change a file and its verdict is void, find it unchanged and the
   verdict stands. Record enough beside every clean or skipped verdict to tell
   those apart, and drop a file re-entering the queue unchanged without a
   judge-run. This is what makes the run finite.

5. **Keep state a crash could resume from.** One worklist file in the repo
   root — shape is yours, but it holds the queue, the verdicts, and the
   skip/rule-issue notes. Write it as things happen; it outranks your memory
   on resume, is never committed, and is deleted at the end after reporting
   what it held: done, skipped, rule-issues.

6. **Know when a file is beating you.** A few failed attempts on the same file
   means the fix oscillates or the rule is confused: roll back to the last
   green commit, note it under rule-issues, close it out, move on.

## Order

Taking findings in the order the report prints them spends the run on the
smallest thing in view. Every rule declares a priority, and a finding carries
its rule's band beside its name: work `severe`, then `high`, then `med`.
Inside a band work outward in — the shape of the file, then of a type or a
function, then of the expression inside one.

A file doing several jobs is the biggest mover there is, which is why that
rule is severe. Splitting it relocates whole declarations, and the findings
underneath it either travel to the file they belong in or stop existing once
the split lands. Fixing those first is work thrown away, and it buries the
split's diff under edits nobody can separate from it.

So a severe finding is a reason not to act on the rest yet. Land it, re-judge,
and the lower bands are often shorter than they were. Where two findings
overlap the outer one wins and the inner one is re-judged rather than fixed.
A split is a move, so it commits on its own before anything inside it is
touched.

## Closing

The commits a run makes as it goes are working state rather than history:
they exist so a failed edit costs one rollback instead of the session. Once
the queue is empty and the tree is green, squash them into one and leave
whatever was already there untouched.

Squash only what this run authored, and only while it is unpushed — a commit
that has been pushed belongs to whoever has fetched it. A move is the one
exception and keeps its own commit through the squash, for the reason it got
its own commit in the first place: relocated code renders as delete-plus-add,
and folding edits into that hides them. Verify the squashed tree matches the
one that was green before it, since that is the only thing the squash may not
change.

The message names what changed, not which rules fired — a reader wants the
shape of the edit, and the rules are already the reason the file was opened.

## Facts

Local knowledge the goals cannot give you.

- The rules are readable before they are enforced: the rulebook path prints
  every enabled rule as prose under its band, calling no model. Read it before
  the first edit, and hold what you write to it as you write — a violation you
  author yourself costs a whole judge pass to find.
- A comment you add is a finding waiting to happen. Write none that restates
  the declaration it sits on, and none that mentions a rule or a finding —
  that is code commenting on its own linting. When a fix relocates an existing
  comment, carry the why across and drop the sentence that introduced it.
- Nothing under the rules directory is ever queued or fixed: the fail-fixtures
  are wrong on purpose.
- A move mixed with edits renders as delete-plus-add, the one diff shape where
  a behaviour change cannot be seen. Commit moves on their own: code relocated
  verbatim, no logic edited. Files a move created are the run's own
  authorship — written to the rules from the start, and queued for judgement
  even on an explicit run. Pre-existing files it merely edited for consistency
  are kept green but not judged; name them in the end-of-run report so
  widening scope stays the user's call.
- Code landing exactly on a rule's threshold gets flagged again. Fix past the
  edge.
- The judge votes and is not deterministic. A verdict that wobbles between
  runs is not necessarily your edit's fault — weigh it before reacting.
- Rollback is relative to HEAD. A queued file carrying uncommitted changes at
  the start of the run breaks that: have the user commit or stash them first,
  or exclude the file.
