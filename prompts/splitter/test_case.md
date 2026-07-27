A unit is one leaf of a single test, and three levels decide what that means:

A **test** is the smallest thing this file's framework runs and reports under its own name.

An **enclosing scope** is anything that groups tests and qualifies their names without being run as a test itself: a grouping block, a suite, a fixture class, a module. Scopes are namespaces, not tests.

A **case** is one leaf of a single test: one row of a table of cases, one generated or parametrised argument set, or one subdivision declared inside the test body. A case is not a test of its own; it is one execution of one test.

Write a unit as "Name (case name)", with the case name exactly as it appears in the source. A test that declares no cases is one unit by itself, written as just "Name". In both, "Name" joins the test's enclosing scopes with " > ", outermost first, then the test's own name:

- a test with no cases, no scopes                ->  ParsesHostBeforeColon
- the same test inside two scopes                ->  Parser > Address > ParsesHostBeforeColon
- a test whose cases are named in a table        ->  ParseAddress (rejects blank input)
- a scoped test whose cases are parametrised     ->  Parser > ParseAddress (port above the maximum)

When two cases under one test share a name, keep the first as written and append #01, #02 and so on to the later ones.

Do not list any of the following:

- helper functions, including ones that make assertions or take the framework's test handle
- setup and teardown hooks, and lifecycle methods the framework calls around tests
- the type, class or literal holding a table of cases, or its field names; only its cases
- benchmarks, fuzz targets, property generators and documentation examples
- cases whose name is built at run time rather than written in the source; when a test's cases cannot be named from the source, list that test itself as one unit
