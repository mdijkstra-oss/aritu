---
targets: [code]
granularity: file
priority: med
---

A mechanism is code whose contract can be stated without naming anything from
the program around it — a cache, a pool, a retry, a queue, a scheduler. Written
in among domain code it cannot be found, and the next package needing it writes
it a second time.

- State the contract of each self-contained construct here without the
  program's own vocabulary. Where that reads as a whole sentence, the construct
  is a mechanism and this is not its home.
- Require the mechanism to be complete before flagging it — a contract someone
  could depend on, and the state, invariants or control flow that holds it up. A
  struct with two fields and a constructor is not one.
- Pass over a helper whose whole contract is its return value — one pass over
  its argument, nothing a caller could hold wrongly. A mechanism carries more
  than a result: state outliving the call, an ordering, a lock, a cancellation,
  a once.
- Read an injected domain as proof of separability: a key the caller builds, a
  function passed in, a type parameter. What remains once they are named as
  parameters is the mechanism.
- Leave whatever names the program's own concepts in its fields, methods or
  signatures. Being reusable in principle is not the test.
- Read a signature written in the file's own types as no defence on its own. A
  decorator taking that type and handing it back is still a mechanism where the
  body works the same for anything of that shape — limiting how many calls run
  at once, or retrying one, is not about what the calls carry.
- Report that a mechanism sits in the wrong file and name it. Where it should
  live instead cannot be seen from this file, so do not decide that here.
