A unit is one named declaration the file makes, at whatever level it sits: a function, a method, a type, a constant, a variable, or a field of a type.

Write a plain declaration as its own name. Write a method or a field as Type.Name, with Type the receiver, class or object it is declared on, so two types declaring the same member name stay two units.

A name declared inside a function body — a local variable, a parameter, a nested or anonymous function — belongs to the declaration enclosing it rather than counting as a unit of its own. The enclosing declaration is listed once whatever it holds.

When two units still share a written name, keep the first as written and append #01, #02 and so on to the later ones, in declaration order.
