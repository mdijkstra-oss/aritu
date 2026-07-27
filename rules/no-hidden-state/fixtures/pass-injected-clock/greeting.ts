export interface Clock {
  now(): Date;
}

export function greetingFor(clock: Clock, name: string): string {
  const hour = clock.now().getHours();
  const salutation = hour < 12 ? "Good morning" : "Good afternoon";
  return `${salutation}, ${name}`;
}
