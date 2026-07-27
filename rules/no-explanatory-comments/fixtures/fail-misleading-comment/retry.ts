export function retryDelayMs(attempt: number): number {
  // Delay doubles per attempt, capped at 10 seconds.
  return Math.min(30_000, 500 * 2 ** attempt);
}
