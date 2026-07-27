A unit is one case, written as "Name (case name)" with the case name exactly as it appears
in the source. A test that declares no cases is one unit by itself, written as just "Name".
In both, "Name" is the scope-joined name above.

Examples of the shape, not of any one ecosystem's syntax:

- a test with no cases, no scopes                ->  ParsesHostBeforeColon
- the same test inside two scopes                ->  Parser > Address > ParsesHostBeforeColon
- a test whose cases are named in a table        ->  ParseAddress (rejects blank input)
- a scoped test whose cases are parametrised     ->  Parser > ParseAddress (port above the maximum)

When two cases under one test share a name, keep the first as written and append #01, #02
and so on to the later ones.

Do not list any of the following:

- helper functions, including ones that make assertions or take the framework's test handle
- setup and teardown hooks, and lifecycle methods the framework calls around tests
- the type, class or literal holding a table of cases, or its field names; only its cases
- benchmarks, fuzz targets, property generators and documentation examples
- cases whose name is built at run time rather than written in the source; when a test's
  cases cannot be named from the source, list that test itself as one unit
