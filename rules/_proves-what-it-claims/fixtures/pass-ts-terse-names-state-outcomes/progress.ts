export function clampPercent(value: number): number {
  const rounded = Math.round(value);

  if (rounded > 100) {
    return 100;
  }
  if (rounded < 0) {
    return 0;
  }

  return rounded;
}
