export function invoiceSummaries(
  invoices: {
    number: string;
    dueDate: string;
    paidAt: string | null;
    lines: { cents: number; quantity: number }[];
  }[],
  today: string,
): string[] {
  return invoices.map((invoice) => {
    const totalCents = invoice.lines.reduce((sum, line) => sum + line.cents * line.quantity, 0);
    const isOverdue = invoice.paidAt === null && Date.parse(invoice.dueDate) < Date.parse(today);
    const status = invoice.paidAt !== null ? "paid" : isOverdue ? "overdue" : "open";
    return `${invoice.number}: ${(totalCents / 100).toFixed(2)} (${status})`;
  });
}
