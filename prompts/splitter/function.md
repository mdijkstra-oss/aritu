A unit is one function or method the file declares under its own name. Nested and anonymous functions belong to the declaration they appear inside rather than counting as units of their own.

Write a plain function as its own name. Write a method as Type.Name, with Type the receiver, class or object it is declared on, so two types declaring a method of the same name stay two units. When two units still share a written name, keep the first as written and append #01, #02 and so on to the later ones.
