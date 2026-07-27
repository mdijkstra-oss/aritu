package scenario

import (
	"os"
	"reflect"
	"testing"
)

func TestAppendMakesTheEntryReadableBack(t *testing.T) {
	journal, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() = %v; want no error", err)
	}

	written := []Entry{{ID: "a1", Body: "first"}, {ID: "a2", Body: "second"}}
	for _, entry := range written {
		if err := journal.Append(entry); err != nil {
			t.Fatalf("Append(%+v) = %v; want no error", entry, err)
		}
	}

	got, err := journal.Entries()
	if err != nil {
		t.Fatalf("Entries() = %v; want no error", err)
	}
	if !reflect.DeepEqual(got, written) {
		t.Errorf("Entries() = %+v; want %+v", got, written)
	}
}

func TestEntriesIsEmptyForAFreshJournal(t *testing.T) {
	dir, err := os.MkdirTemp("", "journal")
	if err != nil {
		t.Fatalf("MkdirTemp() = %v; want no error", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	journal, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() = %v; want no error", err)
	}

	entries, err := journal.Entries()
	if err != nil {
		t.Fatalf("Entries() = %v; want no error", err)
	}
	if len(entries) != 0 {
		t.Errorf("Entries() = %+v; want nothing in a journal nobody has written to", entries)
	}
}
