export type Tier = "standard" | "silver" | "gold";

export function tierFor(spendCents: number): Tier {
  if (spendCents < 0) {
    throw new RangeError("spend must not be negative");
  }
  if (spendCents >= GOLD_THRESHOLD_CENTS) {
    return "gold";
  }
  if (spendCents >= SILVER_THRESHOLD_CENTS) {
    return "silver";
  }

  return "standard";
}

const SILVER_THRESHOLD_CENTS = 10_000;
const GOLD_THRESHOLD_CENTS = 50_000;
