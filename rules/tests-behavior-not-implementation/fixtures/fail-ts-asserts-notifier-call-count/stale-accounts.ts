export interface Account {
  email: string;
  lastSeen: Date;
  active: boolean;
}

export interface Notifier {
  accountDeactivated(email: string): void;
}

export function deactivateStaleAccounts(
  accounts: Account[],
  cutoff: Date,
  notifier: Notifier,
): Account[] {
  const deactivated = accounts
    .filter((account) => isStale(account, cutoff))
    .map(withoutAccess);

  deactivated.forEach((account) => notifier.accountDeactivated(account.email));

  return deactivated;
}

function isStale(account: Account, cutoff: Date): boolean {
  return account.active && account.lastSeen < cutoff;
}

function withoutAccess(account: Account): Account {
  return { ...account, active: false };
}
