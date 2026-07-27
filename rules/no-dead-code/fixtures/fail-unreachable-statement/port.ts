export function parsePort(raw: string): number {
  const port = Number(raw);
  if (Number.isNaN(port)) {
    throw new Error(`not a port: ${raw}`);
  }
  return port;
  console.log(`parsed port ${port}`);
}
