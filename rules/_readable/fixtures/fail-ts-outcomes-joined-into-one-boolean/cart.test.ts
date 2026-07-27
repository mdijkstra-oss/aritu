import { describe, it, expect } from "vitest";

import { summarise, type Item } from "./cart";

describe("summarise", () => {
  it("summarises a cart of three in-stock items", () => {
    const items: Item[] = [
      { sku: "ESP-250", unitCents: 1200, quantity: 1, backordered: false },
      { sku: "FLT-500", unitCents: 950, quantity: 2, backordered: false },
      { sku: "GRD-01", unitCents: 1650, quantity: 1, backordered: false },
    ];

    const summary = summarise(items);

    const ok =
      summary.count === 3 &&
      summary.subtotalCents === 4750 &&
      summary.currency === "EUR" &&
      summary.hasBackorder === false;
    expect(ok).toBe(true);
  });
});
