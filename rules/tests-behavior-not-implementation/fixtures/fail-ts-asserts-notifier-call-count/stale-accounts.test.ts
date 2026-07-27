import { describe, it, expect, vi } from "vitest";

import { deactivateStaleAccounts, type Account } from "./stale-accounts";

describe("deactivateStaleAccounts", () => {
  it("deactivates the accounts idle since the cutoff", () => {
    const notifier = { accountDeactivated: vi.fn() };
    const accounts: Account[] = [
      { email: "ada@example.com", lastSeen: new Date("2025-03-14"), active: true },
      { email: "linus@example.com", lastSeen: new Date("2026-06-02"), active: true },
      { email: "grace@example.com", lastSeen: new Date("2025-11-30"), active: true },
      { email: "alan@example.com", lastSeen: new Date("2026-07-19"), active: true },
    ];

    deactivateStaleAccounts(accounts, new Date("2026-01-01"), notifier);

    expect(notifier.accountDeactivated).toHaveBeenCalledTimes(2);
  });
});
