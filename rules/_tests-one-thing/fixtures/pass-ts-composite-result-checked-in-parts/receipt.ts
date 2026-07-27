export type Line = {
  description: string;
  quantity: number;
  unitCents: number;
};

export type Order = {
  id: string;
  lines: Line[];
};

export function renderReceipt(order: Order): string {
  const subtotalCents = order.lines.reduce(lineAmountInto, 0);
  const vatCents = Math.round((subtotalCents * VAT_PERCENT) / 100);

  return [
    `ORDER ${order.id}`,
    ...order.lines.map(renderLine),
    renderRow("Subtotal", subtotalCents),
    renderRow(`VAT ${VAT_PERCENT}%`, vatCents),
    renderRow("TOTAL", subtotalCents + vatCents),
  ].join("\n");
}

const VAT_PERCENT = 21;
const DESCRIPTION_WIDTH = 24;

function lineAmountInto(total: number, line: Line): number {
  return total + line.quantity * line.unitCents;
}

function renderLine(line: Line): string {
  return renderRow(`${line.quantity} x ${line.description}`, line.quantity * line.unitCents);
}

function renderRow(label: string, amountCents: number): string {
  return `${label.padEnd(DESCRIPTION_WIDTH)}${formatEuros(amountCents)}`;
}

function formatEuros(amountCents: number): string {
  return `EUR ${(amountCents / 100).toFixed(2)}`;
}
