package scenario

import "testing"

func TestRetainsARecordOnItsThirtiethDay(t *testing.T) {
	got := IsExpired(30)
	if got {
		t.Fatalf("IsExpired(%d) = %v; want %v", 30, got, false)
	}
}

func TestExpiresARecordOnItsThirtyFirstDay(t *testing.T) {
	got := IsExpired(31)
	if !got {
		t.Fatalf("IsExpired(%d) = %v; want %v", 31, got, true)
	}
}
