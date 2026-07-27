import { describe, it, expect } from "vitest";

import { dispatchReceipt, type Mailer, type Message } from "./receipts";

class RecordingMailer implements Mailer {
  readonly sent: Message[] = [];

  send(message: Message): void {
    this.sent.push(message);
  }
}

describe("dispatchReceipt", () => {
  it("totals the order lines in the receipt it sends to the buyer", () => {
    const mailer = new RecordingMailer();

    dispatchReceipt(
      {
        id: 4711,
        buyerEmail: "ada@example.com",
        lines: [
          { name: "Notebook", quantity: 2, unitCents: 350 },
          { name: "Pen", quantity: 1, unitCents: 150 },
        ],
      },
      mailer,
    );

    expect(mailer.sent[0]).toEqual({
      to: "ada@example.com",
      subject: "Receipt for order 4711",
      body: "2 x Notebook — € 7.00\n1 x Pen — € 1.50\nTotal € 8.50",
    });
  });
});
