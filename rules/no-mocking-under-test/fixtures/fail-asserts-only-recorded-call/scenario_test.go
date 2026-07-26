package scenario

import "testing"

type mockLedger struct {
	recordCalls int
}

var _ Ledger = (*mockLedger)(nil)

func (m *mockLedger) Record(entry Entry) {
	m.recordCalls++
}

func (m *mockLedger) Balance(account string) int { return 0 }

func TestMemoryLedgerRecordsADebitAgainstTheAccount(t *testing.T) {
	ledger := &mockLedger{}

	ledger.Record(Entry{Account: "acct-1", Amount: -250})

	if ledger.recordCalls != 1 {
		t.Errorf("Record call count = %d, want 1", ledger.recordCalls)
	}
}
