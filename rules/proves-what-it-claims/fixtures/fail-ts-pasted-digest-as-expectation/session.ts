import { createHmac } from "node:crypto";

export interface Session {
  id: string;
  userId: string;
  expiresAt: number;
  scopes: string[];
}

export function toWireString(session: Session): string {
  return [
    "v1",
    session.id,
    session.userId,
    String(session.expiresAt),
    session.scopes.join(","),
  ].join("|");
}

export function encodeSessionToken(session: Session, secret: string): string {
  const payload = Buffer.from(toWireString(session), "utf8").toString(
    "base64url",
  );
  return `${payload}.${truncatedHmac(payload, secret)}`;
}

function truncatedHmac(payload: string, secret: string): string {
  return createHmac("sha256", secret).update(payload).digest("hex").slice(0, 32);
}
