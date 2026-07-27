import { describe, it, expect } from "vitest";

import { decodeSettings, encodeSettings, type Settings } from "./settings";

describe("settings codec", () => {
  it("encodes settings to the wire format and decodes them back", () => {
    const settings: Settings = { theme: "dark", fontSize: 14, autosave: true };

    const encoded = encodeSettings(settings);
    expect(encoded).toBe("theme=dark;fontSize=14;autosave=true");

    const decoded = decodeSettings(encoded);
    expect(decoded).toEqual(settings);
  });
});
