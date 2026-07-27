import { describe, it, expect, beforeAll } from "vitest";

import { Cart, Catalog, currency } from "./checkout";

const catalog = new Catalog();

describe("checkout", () => {
  let cart: Cart;
  let lastSku: string;

  beforeAll(() => {
    cart = new Cart(catalog);
  });

  it("registers a product in the catalog", () => {
    catalog.register("apple", 100);
    lastSku = "apple";

    expect(catalog.priceOf("apple")).toBe(100);
  });

  it("prices the product registered a moment ago", () => {
    cart.add(lastSku, 2);

    expect(cart.total()).toBe(200);
  });

  it("keeps the earlier line when another is added", () => {
    cart.add("apple", 1);

    expect(cart.total()).toBe(300);
  });

  it("reports the configured currency", () => {
    process.env.CURRENCY = "EUR";

    expect(currency()).toBe("EUR");

    cart.clear();
  });
});
