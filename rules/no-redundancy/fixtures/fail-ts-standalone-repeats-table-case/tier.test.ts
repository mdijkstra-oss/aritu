import { describe, it, expect } from "vitest";

import { tierFor } from "./tier";

describe("tierFor", () => {
  it.each([
    { spendCents: 5_000, expected: "standard" },
    { spendCents: 15_000, expected: "silver" },
    { spendCents: 90_000, expected: "gold" },
  ])("returns the $expected tier for a spend of $spendCents cents", ({ spendCents, expected }) => {
    expect(tierFor(spendCents)).toBe(expected);
  });

  it("classifies a 15000 cent spend as silver", () => {
    expect(tierFor(15_000)).toBe("silver");
  });
});
