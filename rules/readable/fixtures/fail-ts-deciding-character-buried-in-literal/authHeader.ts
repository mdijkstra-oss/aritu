export type BearerResult = { token: string } | { error: string };

export function parseBearerToken(header: string): BearerResult {
  if (!header.startsWith(BEARER_PREFIX)) {
    return { error: "missing bearer prefix" };
  }

  const token = header.slice(BEARER_PREFIX.length);
  if (token === "") {
    return { error: "empty token" };
  }
  if (!BASE64URL.test(token)) {
    return { error: "invalid token character" };
  }

  return { token };
}

const BEARER_PREFIX = "Bearer ";
const BASE64URL = /^[A-Za-z0-9_-]+$/;
