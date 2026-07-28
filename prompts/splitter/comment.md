A unit is one comment the file carries, named after the thing it belongs to. Every comment is a unit whatever it holds: prose, a licence header, a TODO, or code that has been commented out.

A comment belongs to the declaration it documents — the one directly beneath it — or, where it sits in the middle of a body, to the declaration enclosing it. Write that declaration's name:

- a comment above a plain function or a type      ->  ParseAddress
- a comment above a method                        ->  Parser.ParseAddress, with Type the receiver, class or object
- a comment above a field or an enum member       ->  Options.Timeout
- a comment inside a body                         ->  the name of the declaration it is inside
- a comment belonging to no declaration           ->  (file), for a licence header, a comment above the package or the imports, or one stranded between declarations

Count a run of adjacent single-line comments as one unit: what matters is the comment a reader meets, not how many delimiters it took to write. A blank line, or any code between them, starts a new one.

When several comments resolve to one name, keep the first as written and append #01, #02 and so on to the later ones, in the order they appear in the file.

Do not list the doc comment's absence: a declaration carrying no comment is not a unit. A file with no comments at all yields no units.
