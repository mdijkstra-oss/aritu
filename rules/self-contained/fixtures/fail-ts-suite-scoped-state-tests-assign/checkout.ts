export class Catalog {
  private readonly prices = new Map<string, number>();

  register(sku: string, cents: number): void {
    this.prices.set(sku, cents);
  }

  priceOf(sku: string): number {
    return this.prices.get(sku) ?? 0;
  }
}

export class Cart {
  private readonly items = new Map<string, number>();

  constructor(private readonly catalog: Catalog) {}

  add(sku: string, qty: number): void {
    this.items.set(sku, (this.items.get(sku) ?? 0) + qty);
  }

  total(): number {
    let sum = 0;
    for (const [sku, qty] of this.items) {
      sum += this.catalog.priceOf(sku) * qty;
    }

    return sum;
  }

  clear(): void {
    this.items.clear();
  }
}

export function currency(): string {
  return process.env.CURRENCY ?? "USD";
}
