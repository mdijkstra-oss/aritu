export async function latest(baseUrl: string): Promise<Record<string, number>> {
  const res = await fetch(`${baseUrl}/latest`);
  if (!res.ok) {
    throw new Error(`rates: ${res.status}`);
  }

  const body = (await res.json()) as { rates: Record<string, number> };
  return body.rates;
}

export async function health(baseUrl: string): Promise<boolean> {
  const res = await fetch(`${baseUrl}/health`);
  return res.ok;
}
