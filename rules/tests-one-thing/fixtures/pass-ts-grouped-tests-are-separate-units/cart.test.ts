import { describe, it, expect } from "vitest";

import { addItem, cartTotal, emptyCart } from "./cart";

describe("cart", () => {
  it("sums the line prices of every item", () => {
    const cart = {
      lines: [
        { sku: "espresso", quantity: 2, unitCents: 250 },
        { sku: "croissant", quantity: 1, unitCents: 320 },
      ],
    };

    expect(cartTotal(cart)).toBe(820);
  });

  it("rejects a quantity below one", () => {
    const cart = emptyCart();

    expect(() => addItem(cart, { sku: "espresso", quantity: 0, unitCents: 250 })).toThrow(
      "quantity must be positive",
    );
  });
});
