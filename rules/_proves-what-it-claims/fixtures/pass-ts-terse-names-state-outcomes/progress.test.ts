import { describe, it, expect } from "vitest";

import { clampPercent } from "./progress";

describe("clampPercent", () => {
  it("caps at 100", () => {
    expect(clampPercent(140)).toBe(100);
  });

  it("floors at 0", () => {
    expect(clampPercent(-12)).toBe(0);
  });

  it("rounds to the nearest whole percent", () => {
    expect(clampPercent(42.6)).toBe(43);
  });
});
