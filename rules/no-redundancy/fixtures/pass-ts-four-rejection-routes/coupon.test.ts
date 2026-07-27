import { describe, it, expect } from "vitest";

import { redeem, CouponError, type Coupon, type Order } from "./coupon";

describe("redeem", () => {
  function validCoupon(overrides: Partial<Coupon> = {}): Coupon {
    return {
      code: "SPRING10",
      percentOff: 10,
      expiresOn: "2025-06-30",
      currency: "EUR",
      minimumSpendCents: 5_000,
      redeemed: false,
      ...overrides,
    };
  }

  function validOrder(overrides: Partial<Order> = {}): Order {
    return {
      totalCents: 8_000,
      currency: "EUR",
      placedOn: "2025-05-04",
      ...overrides,
    };
  }

  it("rejects a coupon that has already been redeemed", () => {
    const coupon = validCoupon({ redeemed: true });
    const order = validOrder();

    expect(() => redeem(coupon, order)).toThrow(CouponError);
  });

  it("rejects a coupon that expired before the order was placed", () => {
    const coupon = validCoupon({ expiresOn: "2025-04-30" });
    const order = validOrder();

    expect(() => redeem(coupon, order)).toThrow(CouponError);
  });

  it("rejects a coupon issued in a different currency", () => {
    const coupon = validCoupon({ currency: "USD" });
    const order = validOrder();

    expect(() => redeem(coupon, order)).toThrow(CouponError);
  });

  it("rejects a coupon on an order below its minimum spend", () => {
    const coupon = validCoupon();
    const order = validOrder({ totalCents: 2_500 });

    expect(() => redeem(coupon, order)).toThrow(CouponError);
  });

  it("returns the discounted total for a valid coupon", () => {
    const coupon = validCoupon();
    const order = validOrder();

    expect(redeem(coupon, order)).toBe(7_200);
  });
});
