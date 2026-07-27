import { describe, it, expect, beforeEach, vi, type Mock } from "vitest";

import { Cart } from "./cart";

function listedPrice(sku: string): number {
  switch (sku) {
    case "apple":
      return 100;
    case "pear":
      return 150;
    default:
      return 0;
  }
}

describe("Cart", () => {
  let priceOf: Mock<(sku: string) => number>;
  let cart: Cart;

  beforeEach(() => {
    priceOf = vi.fn(listedPrice);
    cart = new Cart({ priceOf });
  });

  it("totals a single line", () => {
    cart.add("apple", 2);

    expect(cart.total()).toBe(200);
  });

  it("totals two lines together", () => {
    cart.add("apple", 1);
    cart.add("pear", 2);

    expect(cart.total()).toBe(400);
  });

  it("merges repeated additions of a sku into one line", () => {
    cart.add("pear", 1);
    cart.add("pear", 4);

    expect(cart.lines()).toEqual([{ sku: "pear", qty: 5 }]);
  });

  it("asks the catalog once for a sku added twice", () => {
    cart.add("apple", 1);
    cart.add("apple", 2);
    cart.total();

    expect(priceOf).toHaveBeenCalledTimes(1);
  });

  it("reports zero for a cart with nothing in it", () => {
    const empty = new Cart({ priceOf: () => 0 });

    expect(empty.total()).toBe(0);
  });
});
