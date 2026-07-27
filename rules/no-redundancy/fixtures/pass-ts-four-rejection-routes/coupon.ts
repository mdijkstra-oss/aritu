export class CouponError extends Error {}

export interface Coupon {
  code: string;
  percentOff: number;
  expiresOn: string;
  currency: string;
  minimumSpendCents: number;
  redeemed: boolean;
}

export interface Order {
  totalCents: number;
  currency: string;
  placedOn: string;
}

export function redeem(coupon: Coupon, order: Order): number {
  if (coupon.redeemed) {
    throw new CouponError("coupon already redeemed");
  }
  if (coupon.expiresOn < order.placedOn) {
    throw new CouponError("coupon expired");
  }
  if (coupon.currency !== order.currency) {
    throw new CouponError("currency mismatch");
  }
  if (order.totalCents < coupon.minimumSpendCents) {
    throw new CouponError("minimum spend not met");
  }

  return Math.round(order.totalCents * (1 - coupon.percentOff / 100));
}
