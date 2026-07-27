package scenario

import (
	"net"
	"os"
	"strings"
	"testing"
)

func TestListenAcceptsConnectionsOnTheRelayPort(t *testing.T) {
	relay := New(Config{Addr: "127.0.0.1:18085"})
	if err := relay.Listen(); err != nil {
		t.Fatalf("Listen() = %v; want no error", err)
	}
	defer relay.Close()

	conn, err := net.Dial("tcp", "127.0.0.1:18085")
	if err != nil {
		t.Fatalf("Dial(127.0.0.1:18085) = %v; want no error", err)
	}
	conn.Close()
}

func TestSpoolAppendsTheMessageToTheSpoolFile(t *testing.T) {
	relay := New(Config{SpoolPath: "/tmp/relay-spool.jsonl"})

	if err := relay.Spool(Message{ID: "m-1", Body: "checkout timed out"}); err != nil {
		t.Fatalf("Spool() = %v; want no error", err)
	}

	spooled, err := os.ReadFile("/tmp/relay-spool.jsonl")
	if err != nil {
		t.Fatalf("ReadFile(/tmp/relay-spool.jsonl) = %v; want no error", err)
	}
	if !strings.Contains(string(spooled), "checkout timed out") {
		t.Errorf("/tmp/relay-spool.jsonl = %q; want the spooled body in it", spooled)
	}
}

func TestArchiveMovesTheSpoolIntoTheArchiveDirectory(t *testing.T) {
	relay := New(Config{SpoolPath: "/tmp/relay-spool.jsonl", ArchiveDir: "/tmp/relay-archive"})
	if err := relay.Spool(Message{ID: "m-2", Body: "payment declined"}); err != nil {
		t.Fatalf("Spool() = %v; want no error", err)
	}

	if err := relay.Archive(); err != nil {
		t.Fatalf("Archive() = %v; want no error", err)
	}

	archived, err := os.ReadFile("/tmp/relay-archive/messages.jsonl")
	if err != nil {
		t.Fatalf("ReadFile(/tmp/relay-archive/messages.jsonl) = %v; want no error", err)
	}
	if !strings.Contains(string(archived), "payment declined") {
		t.Errorf("/tmp/relay-archive/messages.jsonl = %q; want the archived body in it", archived)
	}
}
