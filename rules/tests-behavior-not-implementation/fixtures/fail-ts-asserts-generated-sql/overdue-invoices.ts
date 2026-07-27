export interface InvoiceRow {
  id: number;
  customer: string;
  amount_cents: number;
}

export interface InvoiceDb {
  query(sql: string, params: unknown[]): Promise<InvoiceRow[]>;
}

export interface Invoice {
  id: number;
  customer: string;
  amount: number;
}

export async function findOverdueInvoices(db: InvoiceDb, asOf: Date): Promise<Invoice[]> {
  const rows = await db.query(
    "SELECT id, customer, amount_cents FROM invoices WHERE due_at < $1 AND paid_at IS NULL ORDER BY due_at",
    [asOf],
  );

  return rows.map(toInvoice);
}

function toInvoice(row: InvoiceRow): Invoice {
  return { id: row.id, customer: row.customer, amount: row.amount_cents / 100 };
}
