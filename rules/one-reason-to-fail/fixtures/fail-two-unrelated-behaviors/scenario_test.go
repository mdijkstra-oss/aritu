package scenario

import "testing"

func TestLowercasesSurroundedNamesAndRejectsBlankOnes(t *testing.T) {
	got, err := NormalizeUsername("  Ada  ")
	if err != nil {
		t.Fatalf("NormalizeUsername(%q) returned error %v; want none", "  Ada  ", err)
	}
	if got != "ada" {
		t.Errorf("NormalizeUsername(%q) = %q; want %q", "  Ada  ", got, "ada")
	}

	if _, err := NormalizeUsername("   "); err == nil {
		t.Errorf("NormalizeUsername(%q) returned no error; want one", "   ")
	}
}
