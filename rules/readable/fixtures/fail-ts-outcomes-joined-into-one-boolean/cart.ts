export interface Item {
  sku: string;
  unitCents: number;
  quantity: number;
  backordered: boolean;
}

export interface Summary {
  count: number;
  subtotalCents: number;
  currency: string;
  hasBackorder: boolean;
}

export function summarise(items: Item[]): Summary {
  return {
    count: items.length,
    subtotalCents: items.reduce((cents, item) => cents + item.unitCents * item.quantity, 0),
    currency: CURRENCY,
    hasBackorder: items.some((item) => item.backordered),
  };
}

const CURRENCY = "EUR";
