import dns from "node:dns";
import { describe, it, expect } from "vitest";

import { latest, health } from "./exchange";

describe("exchange", () => {
  it("returns the euro rate published upstream", async () => {
    const rates = await latest("https://api.exchangerate.host");

    expect(rates.EUR).toBeGreaterThan(0);
  });

  it("reports the local gateway as healthy", async () => {
    expect(await health("http://localhost:4000")).toBe(true);
  });

  it("resolves the rates host", async () => {
    const { address } = await dns.promises.lookup("rates.internal");

    expect(address).toBeDefined();
  });
});
