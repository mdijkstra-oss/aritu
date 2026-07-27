import { describe, it, expect } from "vitest";

import { addItem, applyCoupon, basketTotal, checkout, newBasket } from "./basket";

describe("basket", () => {
  it("handles a customer's shopping session", () => {
    let basket = newBasket();

    basket = addItem(basket, "espresso", 2, 250);
    expect(basketTotal(basket)).toBe(500);

    basket = addItem(basket, "croissant", 1, 320);
    expect(basketTotal(basket)).toBe(820);

    basket = applyCoupon(basket, "WELCOME10");
    expect(basketTotal(basket)).toBe(738);

    const receipt = checkout(basket, "4242");
    expect(receipt.paidCents).toBe(738);
    expect(receipt.lineCount).toBe(2);
  });
});
