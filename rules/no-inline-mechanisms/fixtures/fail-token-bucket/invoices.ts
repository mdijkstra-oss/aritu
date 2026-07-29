interface Invoice {
  id: string;
  cents: number;
}

class RateLimiter {
  private tokens: number;
  private lastRefill: number;

  constructor(
    private readonly capacity: number,
    private readonly perSecond: number,
  ) {
    this.tokens = capacity;
    this.lastRefill = Date.now();
  }

  take(): boolean {
    const now = Date.now();
    const gained = ((now - this.lastRefill) / 1000) * this.perSecond;
    this.tokens = Math.min(this.capacity, this.tokens + gained);
    this.lastRefill = now;
    if (this.tokens < 1) {
      return false;
    }
    this.tokens -= 1;
    return true;
  }
}

const limiter = new RateLimiter(20, 5);

export async function fetchInvoice(id: string): Promise<Invoice> {
  if (!limiter.take()) {
    throw new Error("invoice lookups are rate limited");
  }
  const response = await fetch(`/invoices/${id}`);
  return response.json();
}
