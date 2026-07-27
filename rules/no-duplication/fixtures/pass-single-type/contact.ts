export interface Contact {
  id: string;
  name: string;
  email: string;
}

export function greeting(contact: Contact): string {
  return `Hello ${contact.name}`;
}

export function mention(contact: Contact): string {
  return `${greeting(contact)} <${contact.email}>`;
}
