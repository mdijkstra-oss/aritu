# Go fixtures are compiled and run by this repository's own suite

Every `rules/*/fixtures/*/` directory holding `.go` files is a package under the module root,
so `go build ./...`, `go vet ./...` and `go test ./...` all reach it. A fixture that does not
compile breaks the build, and a fixture whose test fails breaks the suite — even though
nothing in aritu ever executes a fixture.

The consequence that catches you out: **a fixture demonstrating a bad test still has to be a
passing test.** The fixtures for `self-contained` are where this bites. A fixture proving that
dialling the network is a violation cannot actually dial the network, and one proving that
reading a checked-in file by relative path is a violation cannot read a file that is not
there — either way `go test ./...` goes red for the wrong reason.

What works is a violation that is real but inert when executed: `time.Now()`, `os.Getenv`,
`os.TempDir()`, an absolute path that only ever appears in a string, a package-level variable
the tests assign. All of them are genuinely the shape the rule rejects, and all of them run
green.

The other three languages have no such constraint — nothing in this repository compiles or
runs `.ts`, `.py` or `.java` — but they should be written as though they did. A fixture with
an invented API teaches the judging model to accept invented APIs.

See [[fixtures-hold-only-on-unanimity]] for the constraint on which units a fixture may
contain.
