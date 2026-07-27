import { describe, it, expect } from "vitest";

import { findOverdueInvoices, type InvoiceDb, type InvoiceRow } from "./overdue-invoices";

class RecordingDb implements InvoiceDb {
  lastSql = "";

  async query(sql: string): Promise<InvoiceRow[]> {
    this.lastSql = sql;

    return [
      { id: 81, customer: "Kolibri BV", amount_cents: 12_500 },
      { id: 92, customer: "Vermeer Interieur", amount_cents: 4_000 },
    ];
  }
}

describe("findOverdueInvoices", () => {
  it("leaves out invoices that have already been paid", async () => {
    const db = new RecordingDb();

    await findOverdueInvoices(db, new Date("2026-07-27"));

    expect(db.lastSql).toContain("paid_at IS NULL");
    expect(db.lastSql).toContain("ORDER BY due_at");
  });
});
