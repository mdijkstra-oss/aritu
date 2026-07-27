export interface CartItem {
  name: string;
  priceCents: number;
}

export interface Cart {
  items: CartItem[];
}

export const MAX_TOTAL_CENTS = 500_000;

export function totalOf(cart: Cart): number {
  return sumCents(cart.items);
}

export function isOverLimit(cart: Cart): boolean {
  return totalOf(cart) > MAX_TOTAL_CENTS;
}

function sumCents(items: CartItem[]): number {
  return items.reduce((sum, item) => sum + item.priceCents, 0);
}
