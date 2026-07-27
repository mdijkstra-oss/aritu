---
include: [tests]
include_source: false
granularity: file
---
A suite must produce the same verdict on any machine, in any order, at any time. Run these
tests on a colleague's laptop, run them in reverse, run two copies of the file at once, run
them a year from now: every test that passes today has to pass then, and for the same
reason. A suite that fails this reports on the machine that happened to run it rather than
on the code it exercises, and the red it eventually produces costs someone a day to explain
before they learn it was never about the code at all.

**The distinction that decides almost every case: a test that creates the state it uses is
self-contained; a test that depends on state it did not create is not.** A temporary
directory the test makes and removes is created state. A hardcoded absolute path is state
someone else must have arranged. A record the test inserts is created state; a record the
test expects to already be there is not. A value built inside the test is created state; a
variable declared outside every test and assigned by them is state each test inherits from
whichever test ran last. Cost and realism are not the criterion — a test that stands up a
whole server is self-contained if it stood that server up itself, and a test that reads one
small file is not if it did not put that file there.

You are given the file as a single unit and you return one verdict for all of it. That is
deliberate: ordering dependence and shared mutable state are relations *between* tests and
cannot be stated at any finer level, since each test on its own looks blameless and only the
pair reveals the coupling. One offending test fails the file. Because the verdict covers
everything, the reason has to do the locating work that the unit name cannot.

SATISFIES the rule:

- **State the test creates and cleans up.** A temporary directory made per test and torn
  down after it, a record the test inserts and deletes, a file written into a location the
  test itself made. The path being on disk is not the defect; inheriting a path is.
- **A fresh instance per test.** The subject and its collaborators constructed inside the
  test, or in setup that runs again before each test. Two tests that each build their own
  copy share nothing, however identical the construction lines look.
- **Determinism supplied by the test.** A fixed instant handed in as the current time, a
  generator seeded with a constant, an identifier passed in where the code would otherwise
  invent one. Substituting an unpredictable source with a predictable one is the fix for
  this rule, not a violation of it.
- **A real dependency the test starts for itself.** Standing up a server, an in-memory
  store or a temporary database and then talking to it at an address learned from what was
  started. The test brought its own world with it.
- **Ambient state the test sets and restores.** An environment value or working location
  changed at the start of a test and put back when it finishes is created state, provided
  the restore runs however the test ends. What is created is the setting itself, not
  whatever the test then finds at the location it moved to.
- **An unavoidable real value asserted relationally.** Capturing the instant before the act
  and asserting the returned timestamp is not earlier than it holds on every machine at
  every moment. What passes is the assertion that cannot be made false by the clock, never
  merely the fact that a clock was involved.
- **Shared read-only data.** A constant table of cases declared in the file, a helper that
  returns a freshly built value, a value defined once and never assigned to. Sharing a value
  the file itself defines is not the defect; shared mutation is. A file on disk the test did
  not write is judged by the location shape below, whether or not any test writes to it.

DISQUALIFIES the rule:

- **The real clock.** Asserting an exact value derived from the current instant — an
  equality against a timestamp the code recorded, an expected string built from today's date
  so the test is true today and false tomorrow — asserting that something finished within a
  wall-clock budget, or sleeping to let the code under test catch up. Elapsed time is a
  property of the machine, and a deadline that holds on a developer's laptop is a coin flip
  on a loaded build agent. An assertion the passage of time cannot falsify is the relational
  near-miss above, not this shape.
- **An unpredictable source.** Random inputs, randomly generated identifiers, a shuffled
  collection, or anything else drawn from a generator the test did not seed. A test that
  invents its own input cannot be reproduced from the failure it prints.
- **The network as found.** Dialling a host, fetching a URL, resolving a name for a service
  the test did not start. The verdict then depends on connectivity, on credentials, and on
  whoever last changed what is at the other end.
- **A location the test did not create.** An absolute path written as a literal, the user's
  home directory, a location named by an environment value the test assumes is set, or a
  checked-in file the test did not put there — reached by an absolute path or by a relative
  one, it makes no difference which. Establishing the working location first does not cure
  this: the file is still state someone else arranged, and the verdict still turns on their
  having arranged it.
- **A fixed name in a shared namespace.** A listening port chosen by hand, a fixed
  temporary filename, a fixed database or table name, a fixed key in a shared registry. One
  run is fine and two at once collide, which is why this shape survives for years and then
  fails only under load.
- **A test that only passes because an earlier one ran.** Reading a record an earlier test
  inserted, expecting a counter, cache or connection an earlier test warmed, or relying on
  an earlier test to have performed the setup. Labels or numbers that exist to pin an order
  are an admission of this, not a remedy: run alone, or run second, the test goes red
  without the code having changed.
- **Mutable state shared across tests.** A variable at file or suite scope that tests assign
  to, one client, connection or registry built once for the whole file and mutated by
  individual tests, a global left modified, or cleanup that runs only when the test succeeds
  and so leaks its mess into the next one whenever it fails.

You are given the test file and not the code it exercises. Judge the ambient dependencies
the test file itself names — the clock it reads, the socket it dials, the location it opens,
the variable it assigns. A subject that reaches for the clock or the network behind the
test's back is a real defect and is not one you can see from here, so do not speculate about
it and do not fail a file because some call it makes *might* touch the outside world. Fail
it for what is written in front of you.

Nothing else about these tests moves this verdict. A stand-in for a collaborator, a thin
assertion, a vague name, the same behaviour pinned down twice — none of that belongs here,
and a file of weak tests satisfies this rule as long as they are weak in the same way
everywhere, every time.

In the reason, name the offending test as a runner would print it, then the exact thing it
reached for: the call that read the clock, the literal path opened, the address dialled, the
variable assigned outside the tests, or — for a coupling — both tests and which one has to
run first. A reason that characterises the file in general leaves the reader searching a
file you have already searched. When several tests share one shape, name the clearest
instance and say how many others repeat it; the reader will find the rest once they know
what they are looking for.

Weak: `the tests share state and depend on ordering`.
Better: `returns_the_cached_entry reads the entry that stores_the_entry inserted into the
cache shared by the whole file, so it fails whenever it runs alone or first`.
