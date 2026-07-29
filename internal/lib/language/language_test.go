package language

import (
	"slices"
	"testing"
)

func TestIsSourceFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "a listed extension is source", path: "internal/parser.go", want: true},
		{name: "a language with no test convention is still source", path: "lib/report.rb", want: true},
		{name: "so is one whose ecosystem this tool has never seen", path: "src/main.zig", want: true},
		{name: "an extension the table does not list is not", path: "legacy/report.rnd"},
		{name: "documentation is not source, whatever else it may be", path: "README.md"},
		{name: "a file with no extension is not, since the table is extensions only", path: "Makefile"},
		{name: "the match is on the extension, not anywhere else in the path", path: "go/notes.txt"},
		{name: "R's capital extension is listed as well as its lowercase", path: "analysis/model.R", want: true},
		{name: "and the lowercase one", path: "analysis/model.r", want: true},
		{name: "an uppercase extension nobody writes that way is not", path: "src/PARSER.GO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSourceFile(tt.path); got != tt.want {
				t.Errorf("IsSourceFile(%q) = %t, want %t", tt.path, got, tt.want)
			}
		})
	}
}

// TestNoExtensionIsClaimedTwice guards a table that outside contributors extend:
// two languages listing one extension would make the second unreachable, and
// nothing else in the package would report it.
func TestNoExtensionIsClaimedTwice(t *testing.T) {
	claimedBy := map[string]string{}
	for _, row := range languages {
		for _, extension := range row.Extensions {
			if first, isClaimed := claimedBy[extension]; isClaimed {
				t.Errorf("extension %q is listed under both %s and %s", extension, first, row.Name)
				continue
			}
			claimedBy[extension] = row.Name
		}
	}
}

func TestEveryLanguageIsNamedAndCarriesAnExtension(t *testing.T) {
	for at, row := range languages {
		if row.Name == "" {
			t.Errorf("language at index %d has no name", at)
		}
		if len(row.Extensions) == 0 {
			t.Errorf("language %s lists no extensions, so nothing can match it", row.Name)
		}
	}
}

func TestEveryExtensionStartsWithADot(t *testing.T) {
	for _, extension := range Extensions() {
		if len(extension) < 2 || extension[0] != '.' {
			t.Errorf("extension %q is not in the form filepath.Ext returns", extension)
		}
	}
}

func TestExtensionsCannotBeMutatedThroughTheReturnedSlice(t *testing.T) {
	returned := Extensions()
	if len(returned) == 0 {
		t.Fatal("Extensions() is empty, so the test below proves nothing")
	}
	returned[0] = ".corrupted"

	if slices.Contains(Extensions(), ".corrupted") {
		t.Error("a caller writing to the returned slice changed the table")
	}
}
