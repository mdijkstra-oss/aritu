export interface Catalog {
  priceOf(sku: string): number;
}

export interface Line {
  sku: string;
  qty: number;
}

export class Cart {
  private readonly items: Line[] = [];

  constructor(private readonly catalog: Catalog) {}

  add(sku: string, qty: number): void {
    const existing = this.items.find((line) => line.sku === sku);
    if (existing) {
      existing.qty += qty;
      return;
    }

    this.items.push({ sku, qty });
  }

  lines(): Line[] {
    return this.items.map((line) => ({ ...line }));
  }

  total(): number {
    return this.items.reduce(
      (sum, line) => sum + this.catalog.priceOf(line.sku) * line.qty,
      0,
    );
  }
}
