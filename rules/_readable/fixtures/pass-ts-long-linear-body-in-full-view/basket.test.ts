import { describe, it, expect } from "vitest";

import { priceBasket, type Line, type Promotion } from "./basket";

describe("priceBasket", () => {
  it("drops shipping once the discounted subtotal clears the free-shipping threshold", () => {
    const espresso: Line = { sku: "ESP-250", unitCents: 1150, quantity: 2, category: "coffee" };
    const filter: Line = { sku: "FLT-500", unitCents: 1400, quantity: 3, category: "coffee" };
    const grinder: Line = { sku: "GRD-01", unitCents: 8900, quantity: 1, category: "hardware" };
    const tenPercentOffCoffee: Promotion = { category: "coffee", percentOff: 10 };
    const freeShippingOverCents = 12000;
    const flatShippingCents = 599;

    const receipt = priceBasket(
      [espresso, filter, grinder],
      tenPercentOffCoffee,
      freeShippingOverCents,
      flatShippingCents,
    );

    expect(receipt.discountCents).toBe(650);
    expect(receipt.shippingCents).toBe(0);
    expect(receipt.totalCents).toBe(14750);
  });
});
