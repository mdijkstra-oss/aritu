package testpath

import (
	"path/filepath"
	"slices"
	"testing"
)

// TestExtensions pins the list a sweep generates its candidate files from. It is
// read off the same index the table is looked up through, so an ecosystem added as
// a row widens it with no second edit — and this is what would notice if it stopped
// being the same list.
func TestExtensions(t *testing.T) {
	want := []string{".cjs", ".cts", ".go", ".java", ".js", ".jsx", ".mjs", ".mts", ".py", ".ts", ".tsx"}

	if got := Extensions(); !slices.Equal(got, want) {
		t.Errorf("Extensions() = %v, want %v", got, want)
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "underscore suffix", path: "parser_test.go", want: true},
		{name: "underscore suffix nested", path: "internal/domain/parser_test.go", want: true},
		{name: "directory named like a test file", path: "a_test.go/parser_test.go", want: true},
		{name: "implementation beside a test", path: "parser.go", want: false},
		{name: "suffix without its separator", path: "parsertest.go", want: false},
		{name: "affix with nothing left of it", path: "_test.go", want: false},
		{name: "affix with nothing left of it, nested", path: "internal/_test.go", want: false},

		{name: "dotted test suffix", path: "src/parser.test.ts", want: true},
		{name: "dotted spec suffix", path: "src/parser.spec.ts", want: true},
		{name: "component test", path: "src/Button.test.tsx", want: true},
		{name: "module test", path: "src/loader.spec.mjs", want: true},
		{name: "plain module in a tests directory", path: "src/__tests__/parser.ts", want: true},
		{name: "plain module outside a tests directory", path: "src/parser.ts", want: false},
		{name: "a directory called test does not mark its contents", path: "test/parser.ts", want: false},

		{name: "prefixed module", path: "tests/test_parser.py", want: true},
		{name: "suffixed module", path: "tests/parser_test.py", want: true},
		{name: "plain module", path: "src/parser.py", want: false},
		{name: "prefix with nothing after it", path: "test_.py", want: false},

		{name: "suffixed class", path: "src/test/java/com/x/ParserTest.java", want: true},
		{name: "plural suffixed class", path: "src/test/java/com/x/ParserTests.java", want: true},
		{name: "prefixed class", path: "src/test/java/com/x/TestParser.java", want: true},
		{name: "production class", path: "src/main/java/com/x/Parser.java", want: false},
		{name: "the marker alone is not a test class", path: "Test.java", want: false},

		{name: "unknown extension", path: "parser_test.rb", want: false},
		{name: "no extension", path: "Makefile", want: false},
		{name: "empty", path: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTestFile(filepath.FromSlash(tc.path)); got != tc.want {
				t.Errorf("IsTestFile(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestSourceCandidates(t *testing.T) {
	tests := []struct {
		name     string
		testPath string
		want     []string
	}{
		{
			name:     "the implementation sits beside the test",
			testPath: "parser_test.go",
			want:     []string{"parser.go"},
		},
		{
			name:     "nested directories are kept",
			testPath: "internal/domain/parser_test.go",
			want:     []string{"internal/domain/parser.go"},
		},
		{
			name:     "a dotted suffix leaves the stem",
			testPath: "src/parser.test.ts",
			want:     []string{"src/parser.ts"},
		},
		{
			name:     "a spec keeps its own extension",
			testPath: "src/Button.spec.tsx",
			want:     []string{"src/Button.tsx"},
		},
		{
			name:     "a tests directory is dropped before it is swapped",
			testPath: "src/__tests__/parser.test.ts",
			want:     []string{"src/__tests__/parser.ts", "src/parser.ts", "src/parser.test.ts"},
		},
		{
			name:     "a plain module in a tests directory is covered by the file above it",
			testPath: "__tests__/parser.ts",
			want:     []string{"parser.ts"},
		},
		{
			name:     "a test tree is swapped for a source tree",
			testPath: "test/parser.test.ts",
			want:     []string{"test/parser.ts", "src/parser.ts"},
		},
		{
			name:     "a prefixed module resolves beside itself and under the package root",
			testPath: "tests/test_parser.py",
			want:     []string{"tests/parser.py", "parser.py", "src/parser.py"},
		},
		{
			name:     "a suffixed module resolves the same way",
			testPath: "tests/parser_test.py",
			want:     []string{"tests/parser.py", "parser.py", "src/parser.py"},
		},
		{
			name:     "a module beside its implementation needs no move",
			testPath: "app/parser_test.py",
			want:     []string{"app/parser.py"},
		},
		{
			name:     "a mirrored source tree is reached by rewriting the root",
			testPath: "src/test/java/com/x/ParserTest.java",
			want:     []string{"src/test/java/com/x/Parser.java", "src/main/java/com/x/Parser.java"},
		},
		{
			name:     "a plural suffix leaves the same stem",
			testPath: "src/test/java/com/x/ParserTests.java",
			want:     []string{"src/test/java/com/x/Parser.java", "src/main/java/com/x/Parser.java"},
		},
		{
			name:     "a prefixed class leaves the same stem",
			testPath: "src/test/java/com/x/TestParser.java",
			want:     []string{"src/test/java/com/x/Parser.java", "src/main/java/com/x/Parser.java"},
		},
		{
			name:     "an absolute path keeps its root",
			testPath: "/repo/internal/parser_test.go",
			want:     []string{"/repo/internal/parser.go"},
		},
		{
			name:     "the inner tests directory is the one rewritten",
			testPath: "tests/integration/tests/test_parser.py",
			want: []string{
				"tests/integration/tests/parser.py",
				"tests/integration/parser.py",
				"tests/integration/src/parser.py",
			},
		},
		{name: "an implementation is not a test", testPath: "parser.go"},
		{name: "an unknown extension has no convention", testPath: "parser_test.rb"},
		{name: "a bare affix names nothing", testPath: "_test.go"},
		{name: "empty", testPath: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SourceCandidates(filepath.FromSlash(tc.testPath))

			want := make([]string, 0, len(tc.want))
			for _, path := range tc.want {
				want = append(want, filepath.FromSlash(path))
			}
			if !slices.Equal(got, want) {
				t.Errorf("SourceCandidates(%q) = %q, want %q", tc.testPath, got, want)
			}
		})
	}
}

func TestSourceCandidatesNeverOffersTheTestItself(t *testing.T) {
	tests := []struct {
		name     string
		testPath string
	}{
		{name: "a plain module in a tests directory", testPath: "__tests__/parser.ts"},
		{name: "a nested plain module in a tests directory", testPath: "src/__tests__/parser.ts"},
		{name: "an affixed file in a tests directory", testPath: "src/__tests__/parser.test.ts"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.FromSlash(tc.testPath)

			if slices.Contains(SourceCandidates(path), filepath.Clean(path)) {
				t.Errorf("SourceCandidates(%q) offered the test file as its own implementation", tc.testPath)
			}
		})
	}
}

func TestCutAffixPanicsOnAPlacementTheTableDoesNotDefine(t *testing.T) {
	tests := []struct {
		name string
		at   placement
	}{
		{name: "zero value", at: placement(0)},
		{name: "out of range", at: placement(9)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("cutAffix did not panic on an undefined placement")
				}
			}()
			_, _ = cutAffix("parser_test", affix{text: "_test", at: tc.at})
		})
	}
}
