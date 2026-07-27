import { describe, it, expect } from "vitest";

import { averageScore } from "./scores";

describe("averageScore", () => {
  it("averages every score in the list", () => {
    expect(averageScore([10, 20, 30])).toBe(20);
  });

  it("returns the score itself when the list holds one", () => {
    expect(averageScore([7])).toBe(7);
  });
});
