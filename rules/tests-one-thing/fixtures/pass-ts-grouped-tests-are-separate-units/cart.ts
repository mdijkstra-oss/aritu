export type CartLine = {
  sku: string;
  quantity: number;
  unitCents: number;
};

export type Cart = {
  lines: CartLine[];
};

export function emptyCart(): Cart {
  return { lines: [] };
}

export function addItem(cart: Cart, line: CartLine): Cart {
  if (line.quantity < 1) {
    throw new RangeError("quantity must be positive");
  }

  return { lines: [...cart.lines, line] };
}

export function cartTotal(cart: Cart): number {
  return cart.lines.reduce(lineAmountInto, 0);
}

function lineAmountInto(total: number, line: CartLine): number {
  return total + line.quantity * line.unitCents;
}
