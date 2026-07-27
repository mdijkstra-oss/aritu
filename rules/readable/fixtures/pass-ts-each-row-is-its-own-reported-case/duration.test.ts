import { describe, it, expect } from "vitest";

import { formatDuration } from "./duration";

describe("formatDuration", () => {
  it.each([
    { seconds: 0, expected: "0s" },
    { seconds: 59, expected: "59s" },
    { seconds: 60, expected: "1m 0s" },
    { seconds: 3671, expected: "1h 1m 11s" },
  ])("formats $seconds seconds as $expected", ({ seconds, expected }) => {
    expect(formatDuration(seconds)).toBe(expected);
  });
});
