# Auto-fix

Run aritu against a repository and fix what it flags. Resolve commands from
the repository you are standing in: a make target if one exists, the
ecosystem's own test runner, whatever the repo already uses. Judge only
through a path that rebuilds aritu first. Scope defaults to the files
changed against the default branch, filtered to what the enabled rules
target; a full run means every file they target; and files named by the
user are the queue exactly as given, no seeding.

## Goals

Everything else derives from these. When something comes up that no rule
covers, decide from the goals.

1. **Every resting point is green.** Mid-edit the tree may be anything, but
   wherever the run stops — moving to the next file, giving up on one,
   finishing — it stands on a state that builds, passes the suite, and is
   committed. Something you could ship.

2. **Bank progress, roll back failure.** An edit that verifiably improved
   things — builds, suite green, findings strictly better with none newly
   introduced — is committed immediately: one file, staged by name, message
   naming the file and the change. An edit that didn't earn that costs one
   rollback to the last green commit, never the session's earlier wins.

3. **Fix causes, not counts.** A finding is a symptom: seven arguments can
   mean a function doing two jobs, and the fix is the split, not the
   smallest edit that moves the number. A finding that is wrong gets no
   code change at all — record it as skipped, and as a fixture candidate
   plus prompt change for the rule that misfired.

4. **Never judge the same content twice.** Verdicts attach to content, not
   filenames. Record the blob hash with every clean or skipped verdict; a
   file re-entering the queue at a hash already ruled on is dropped without
   a judge-run. This is what makes the run finite: every session either
   commits an improvement or closes a hash for good.

5. **Keep state a crash could resume from.** One worklist file in the repo
   root — shape is yours, but it holds the queue, the verdicts with their
   hashes, and the skip/rule-issue notes; it is written as things happen,
   outranks your memory on resume, is never committed, and is deleted at
   the end after reporting what it held: done, skipped, rule-issues.

6. **Know when a file is beating you.** A few failed attempts on the same
   file means the fix oscillates or the rule is confused: roll back to the
   last green commit, note it under rule-issues, close it out, move on.

## Facts

Local knowledge the goals cannot give you.

- Nothing under the rules directory is ever queued or fixed: the fail-
  fixtures are wrong on purpose.
- A move mixed with edits renders as delete-plus-add, the one diff shape
  where a behaviour change cannot be seen. Commit moves on their own: code
  relocated verbatim, no logic edited. Files a move created are the run's
  own authorship: written to the rules from the start, and queued for
  judgement even on an explicit run. Pre-existing files it merely edited
  for consistency are kept green but not judged — name them in the
  end-of-run report so widening scope stays the user's call.
- Code landing exactly on a rule's threshold gets flagged again. Fix past
  the edge: five arguments become an options struct at two or three, never
  exactly four.
- The judge votes and is not deterministic. A verdict that wobbles between
  runs is not necessarily your edit's fault — weigh it before reacting.
- Rollback is relative to HEAD. A queued file carrying uncommitted changes
  at the start of the run breaks that: have the user commit or stash them
  first, or exclude the file.
- This file is never modified during a run.
