export interface RetryOptions {
  attempts: number;
  backoffMs: number;
}

export async function withRetry<T>(
  fn: () => Promise<T>,
  opts: RetryOptions,
): Promise<T> {
  let lastError: unknown;

  for (let attempt = 1; attempt <= opts.attempts; attempt += 1) {
    try {
      return await fn();
    } catch (error) {
      lastError = error;
      if (attempt < opts.attempts) {
        await sleep(delayFor(attempt, opts.backoffMs));
      }
    }
  }

  throw lastError;
}

function delayFor(attempt: number, backoffMs: number): number {
  return backoffMs * 2 ** (attempt - 1);
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
