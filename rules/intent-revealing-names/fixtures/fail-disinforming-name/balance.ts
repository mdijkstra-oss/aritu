export function totalBalance(accounts: { id: string; balanceCents: number }[]): number {
  const accountList = new Map<string, number>();
  for (const account of accounts) {
    accountList.set(account.id, account.balanceCents);
  }
  let totalCents = 0;
  for (const balanceCents of accountList.values()) {
    totalCents += balanceCents;
  }
  return totalCents;
}
