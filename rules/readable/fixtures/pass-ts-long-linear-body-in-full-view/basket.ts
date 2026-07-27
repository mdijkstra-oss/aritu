export interface Line {
  sku: string;
  unitCents: number;
  quantity: number;
  category: string;
}

export interface Promotion {
  category: string;
  percentOff: number;
}

export interface Receipt {
  subtotalCents: number;
  discountCents: number;
  shippingCents: number;
  totalCents: number;
}

export function priceBasket(
  lines: Line[],
  promotion: Promotion,
  freeShippingOverCents: number,
  flatShippingCents: number,
): Receipt {
  const subtotalCents = totalOf(lines);
  const promotedCents = totalOf(lines.filter((line) => line.category === promotion.category));
  const discountCents = Math.round((promotedCents * promotion.percentOff) / 100);
  const discountedCents = subtotalCents - discountCents;
  const shippingCents = discountedCents >= freeShippingOverCents ? 0 : flatShippingCents;

  return {
    subtotalCents,
    discountCents,
    shippingCents,
    totalCents: discountedCents + shippingCents,
  };
}

function totalOf(lines: Line[]): number {
  return lines.reduce((cents, line) => cents + line.unitCents * line.quantity, 0);
}
