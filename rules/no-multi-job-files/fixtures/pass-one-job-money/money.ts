export function formatCents(cents: number, currency: "EUR" | "USD"): string {
  const amount = (cents / 100).toFixed(2);
  const symbol = currency === "EUR" ? "€" : "$";
  return `${symbol}${amount}`;
}

export function parseCents(formatted: string): number {
  const digits = formatted.replace(/[^0-9.]/g, "");
  return Math.round(Number(digits) * 100);
}
