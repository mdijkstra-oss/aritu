export function totalOf(cart: Cart): number {
  return cart.items.reduce((sum, item) => sum + item.priceCents, 0);
}

export function isOverLimit(cart: Cart): boolean {
  return totalOf(cart) > MAX_TOTAL_CENTS;
}

export const MAX_TOTAL_CENTS = 500_000;

export interface CartItem {
  name: string;
  priceCents: number;
}

export interface Cart {
  items: CartItem[];
}
