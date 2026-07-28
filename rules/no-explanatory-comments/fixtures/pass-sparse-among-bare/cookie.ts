export interface Cookie {
  name: string;
  value: string;
}

export function parse(header: string): Cookie[] {
  return header
    .split(";")
    .map((pair) => pair.trim())
    .filter((pair) => pair.includes("="))
    .map(toCookie);
}

function toCookie(pair: string): Cookie {
  const at = pair.indexOf("=");
  return { name: pair.slice(0, at), value: pair.slice(at + 1) };
}

export function serialize(cookies: Cookie[]): string {
  return cookies.map(({ name, value }) => `${name}=${value}`).join("; ");
}

// Values arrive percent-encoded from some user agents and raw from others,
// and decoding twice corrupts a raw value holding a percent sign, so
// decoding is left to the caller who knows which agent sent it.
export function decode(value: string): string {
  return decodeURIComponent(value);
}
