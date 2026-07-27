export interface User {
  id: string;
  name: string;
  email: string;
}

export interface Person {
  id: string;
  name: string;
  email: string;
}

export function greetUser(user: User): string {
  return `Hello ${user.name} <${user.email}>`;
}

export function greetPerson(person: Person): string {
  return `Hello ${person.name} <${person.email}>`;
}
