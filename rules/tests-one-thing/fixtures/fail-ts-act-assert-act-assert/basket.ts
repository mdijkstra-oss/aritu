export type BasketLine = {
  sku: string;
  quantity: number;
  unitCents: number;
};

export type Basket = {
  lines: BasketLine[];
  couponCode: string | null;
};

export type Receipt = {
  orderId: string;
  paidCents: number;
  lineCount: number;
  cardLast4: string;
};

export function newBasket(): Basket {
  return { lines: [], couponCode: null };
}

export function addItem(basket: Basket, sku: string, quantity: number, unitCents: number): Basket {
  return { ...basket, lines: [...basket.lines, { sku, quantity, unitCents }] };
}

export function applyCoupon(basket: Basket, code: string): Basket {
  if (COUPON_DISCOUNTS[code] === undefined) {
    return basket;
  }

  return { ...basket, couponCode: code };
}

export function basketTotal(basket: Basket): number {
  const grossCents = basket.lines.reduce(lineAmountInto, 0);

  return Math.round(grossCents * (1 - discountFor(basket.couponCode)));
}

export function checkout(basket: Basket, cardLast4: string): Receipt {
  const paidCents = basketTotal(basket);

  return {
    orderId: `ORD-${basket.lines.length}-${paidCents}`,
    paidCents,
    lineCount: basket.lines.length,
    cardLast4,
  };
}

const COUPON_DISCOUNTS: Record<string, number | undefined> = {
  WELCOME10: 0.1,
};

function discountFor(code: string | null): number {
  if (code === null) {
    return 0;
  }

  return COUPON_DISCOUNTS[code] ?? 0;
}

function lineAmountInto(total: number, line: BasketLine): number {
  return total + line.quantity * line.unitCents;
}
