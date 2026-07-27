import { describe, it, expect } from "vitest";

import { nextStatus } from "./status";

describe("nextStatus", () => {
  it("moves a submitted draft into review", () => {
    expect(nextStatus("draft", "submit")).toBe("inReview");
  });

  it("archives a draft without sending it for review", () => {
    expect(nextStatus("draft", "archive")).toBe("archived");
  });

  it("approves an item that is in review", () => {
    expect(nextStatus("inReview", "approve")).toBe("approved");
  });

  it("returns a rejected item to draft", () => {
    expect(nextStatus("inReview", "reject")).toBe("draft");
  });

  it("archives an item that is still in review", () => {
    expect(nextStatus("inReview", "archive")).toBe("archived");
  });

  it("publishes an approved item", () => {
    expect(nextStatus("approved", "publish")).toBe("published");
  });

  it("returns an approved item to draft when it is revised", () => {
    expect(nextStatus("approved", "revise")).toBe("draft");
  });

  it("archives a published item", () => {
    expect(nextStatus("published", "archive")).toBe("archived");
  });

  it("restores an archived item to draft", () => {
    expect(nextStatus("archived", "restore")).toBe("draft");
  });

  it("rejects publishing a draft", () => {
    expect(() => nextStatus("draft", "publish")).toThrow("cannot publish from draft");
  });
});
