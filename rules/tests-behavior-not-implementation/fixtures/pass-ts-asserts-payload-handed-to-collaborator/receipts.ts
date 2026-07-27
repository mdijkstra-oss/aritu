export interface Message {
  to: string;
  subject: string;
  body: string;
}

export interface Mailer {
  send(message: Message): void;
}

export interface OrderLine {
  name: string;
  quantity: number;
  unitCents: number;
}

export interface Order {
  id: number;
  buyerEmail: string;
  lines: OrderLine[];
}

export function dispatchReceipt(order: Order, mailer: Mailer): void {
  const totalCents = order.lines.reduce(addLineAmount, 0);

  mailer.send({
    to: order.buyerEmail,
    subject: `Receipt for order ${order.id}`,
    body: [...order.lines.map(renderLine), `Total € ${euros(totalCents)}`].join("\n"),
  });
}

function renderLine(line: OrderLine): string {
  return `${line.quantity} x ${line.name} — € ${euros(line.quantity * line.unitCents)}`;
}

function addLineAmount(total: number, line: OrderLine): number {
  return total + line.quantity * line.unitCents;
}

function euros(cents: number): string {
  return (cents / 100).toFixed(2);
}
