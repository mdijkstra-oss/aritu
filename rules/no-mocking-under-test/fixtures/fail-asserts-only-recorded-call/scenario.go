package scenario

type Entry struct {
	Account string
	Amount  int
}

type Ledger interface {
	Record(entry Entry)
	Balance(account string) int
}

type MemoryLedger struct {
	entries []Entry
}

func NewMemoryLedger() *MemoryLedger {
	return &MemoryLedger{}
}

func (l *MemoryLedger) Record(entry Entry) {
	l.entries = append(l.entries, entry)
}

func (l *MemoryLedger) Balance(account string) int {
	total := 0
	for _, entry := range l.entries {
		if entry.Account == account {
			total += entry.Amount
		}
	}
	return total
}
