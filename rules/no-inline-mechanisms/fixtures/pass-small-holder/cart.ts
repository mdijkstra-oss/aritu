interface Line {
  sku: string;
  quantity: number;
}

export class Cart {
  private readonly lines: Line[] = [];

  constructor(private readonly currency: string) {}

  add(line: Line): void {
    this.lines.push(line);
  }

  total(priceOf: (sku: string) => number): string {
    const cents = this.lines.reduce(
      (running, line) => running + priceOf(line.sku) * line.quantity,
      0,
    );
    return `${(cents / 100).toFixed(2)} ${this.currency}`;
  }
}
