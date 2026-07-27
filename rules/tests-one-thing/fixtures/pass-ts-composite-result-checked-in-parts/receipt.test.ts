import { describe, it, expect } from "vitest";

import { renderReceipt } from "./receipt";

describe("renderReceipt", () => {
  it("lays out the header, the item lines and the totals of one order", () => {
    const order = {
      id: "4182",
      lines: [
        { description: "Espresso", quantity: 2, unitCents: 250 },
        { description: "Croissant", quantity: 1, unitCents: 320 },
      ],
    };

    const receipt = renderReceipt(order);

    expect(receipt).toContain("ORDER 4182");
    expect(receipt).toContain("2 x Espresso");
    expect(receipt).toContain("1 x Croissant");
    expect(receipt).toMatch(/Subtotal\s+EUR 8\.20/);
    expect(receipt).toMatch(/VAT 21%\s+EUR 1\.72/);
    expect(receipt).toMatch(/TOTAL\s+EUR 9\.92/);
  });
});
