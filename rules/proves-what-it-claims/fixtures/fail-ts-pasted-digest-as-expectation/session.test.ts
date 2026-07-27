import { createHash } from "node:crypto";
import { describe, it, expect } from "vitest";

import { encodeSessionToken, toWireString, type Session } from "./session";

function activeSession(): Session {
  return {
    id: "s_7f3a91",
    userId: "u_4021",
    expiresAt: 1_767_225_600,
    scopes: ["read:profile", "write:notes"],
  };
}

describe("toWireString", () => {
  it("writes the session as a compact wire string", () => {
    const wire = toWireString(activeSession());

    expect(createHash("sha256").update(wire).digest("hex")).toBe(
      "3e313a8b1020bbeba1679365a75017f305c2ba76e4ada47fa89c0aa24ecf5b2b",
    );
  });
});

describe("encodeSessionToken", () => {
  it("signs the encoded payload with the shared secret", () => {
    expect(encodeSessionToken(activeSession(), "test-secret")).toMatchInlineSnapshot(
      `"djF8c183ZjNhOTF8dV80MDIxfDE3NjcyMjU2MDB8cmVhZDpwcm9maWxlLHdyaXRlOm5vdGVz.673501da199d6dfe31bc123749bbaefd"`,
    );
  });
});
