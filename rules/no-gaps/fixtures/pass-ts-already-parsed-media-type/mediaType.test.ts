import { describe, it, expect } from "vitest";

import { isJsonPayload, type MediaType } from "./mediaType";

const cases: ReadonlyArray<{ shape: string; media: MediaType; expected: boolean }> = [
  {
    shape: "the plain json subtype",
    media: { type: "application", subtype: "json", suffix: null },
    expected: true,
  },
  {
    shape: "a vendor subtype carrying the json suffix",
    media: { type: "application", subtype: "vnd.api", suffix: "json" },
    expected: true,
  },
  {
    shape: "an application subtype that is json in neither part",
    media: { type: "application", subtype: "xml", suffix: null },
    expected: false,
  },
  {
    shape: "a json subtype under a type other than application",
    media: { type: "text", subtype: "json", suffix: null },
    expected: false,
  },
];

describe("isJsonPayload", () => {
  it.each(cases)("returns $expected for $shape", ({ media, expected }) => {
    expect(isJsonPayload(media)).toBe(expected);
  });
});
