let requestCount = 0;
let lastPath: string | null = null;

export function recordRequest(path: string): void {
  requestCount += 1;
  lastPath = path;
}

export function requestSummary(): string {
  return `${requestCount} requests, last: ${lastPath ?? "none"}`;
}
