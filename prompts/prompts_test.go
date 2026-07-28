package prompts

import (
	"strings"
	"testing"
)

func TestLinterCarriesEverySectionInOrder(t *testing.T) {
	rendered := Linter("test_case", "THE-RULE",
		[]Unit{{Name: "TestParsesHost", Key: "u01_test_parses_host"}},
		[]File{
			{Path: "pkg/parser_test.txt", Content: "the test body"},
			{Path: "pkg/parser.txt", Content: "the source body"},
		})

	wantInOrder(t, rendered,
		"<instructions>", "the whole of your evidence", "never by line number",
		"<unit>", "one test case",
		"<rule>", "THE-RULE",
		"<units>", "TestParsesHost   ->   u01_test_parses_host",
		`<file path="pkg/parser_test.txt">`, "the test body",
		`<source path="pkg/parser.txt">`, "the source body",
		"<task>",
	)
}

func TestSplitterCarriesEverySectionInOrder(t *testing.T) {
	rendered := Splitter("test_case", File{Path: "pkg/parser_test.txt", Content: "the test body"})

	wantInOrder(t, rendered,
		"<instructions>", "declaration order",
		"<unit>", `"Name (case name)"`,
		`<file path="pkg/parser_test.txt">`, "the test body",
		"<task>",
	)
}

// TestTheSplitterCarriesNoRule guards the property that lets one listing serve
// every rule at a granularity: the moment the splitter prompt varies by rule, the
// answer stops being shareable and every rule pays for its own.
func TestTheSplitterCarriesNoRule(t *testing.T) {
	rendered := Splitter("test_case", File{Path: "a_test.txt"})

	if strings.Contains(rendered, "<rule>") {
		t.Errorf("the splitter prompt carries a rule section:\n%s", rendered)
	}
}

// TestEachKindDescribesItsOwnUnitAndNoOther is what makes a rule about something
// other than tests possible: the test vocabulary comes with the test_case kind,
// never with the frame, so a rule at another granularity is told nothing about
// tests.
func TestEachKindDescribesItsOwnUnitAndNoOther(t *testing.T) {
	tests := []struct {
		name       string
		rendered   string
		want       string
		wantAbsent string
	}{
		{
			name:       "the function linter names declarations, not tests",
			rendered:   Linter("function", "THE-RULE", nil, nil),
			want:       "function or method",
			wantAbsent: "table of cases",
		},
		{
			name:       "the file linter asks one verdict for everything",
			rendered:   Linter("file", "THE-RULE", nil, nil),
			want:       "one verdict covering everything",
			wantAbsent: "enclosing scope",
		},
		{
			name:       "the function splitter leaves nested functions out",
			rendered:   Splitter("function", File{Path: "a.txt"}),
			want:       "Nested and anonymous functions",
			wantAbsent: "benchmarks",
		},
		{
			name:       "the test_case splitter carries the whole naming format",
			rendered:   Splitter("test_case", File{Path: "a_test.txt"}),
			want:       "Parser > Address > ParsesHostBeforeColon",
			wantAbsent: "function or method the file declares",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(tc.rendered, tc.want) {
				t.Errorf("missing %q:\n%s", tc.want, tc.rendered)
			}
			if strings.Contains(tc.rendered, tc.wantAbsent) {
				t.Errorf("carries %q, which belongs to another kind:\n%s", tc.wantAbsent, tc.rendered)
			}
		})
	}
}

// TestFileContentReachesTheModelVerbatim covers the input that once killed a
// whole sweep: a source file holding braces of its own — a composite literal, a
// Handlebars view, a template — is the ordinary input this tool exists to read,
// never a placeholder for this package to resolve.
func TestFileContentReachesTheModelVerbatim(t *testing.T) {
	source := "Params: []Param{{Name: \"charset\"}},\ngreeting := `{{unfilled}}`"

	tests := []struct {
		name     string
		rendered string
	}{
		{"linter", Linter("file", "a rule", nil, []File{{Path: "a.txt", Content: source}})},
		{"splitter", Splitter("function", File{Path: "a.txt", Content: source})},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, kept := range []string{`[]Param{{Name: "charset"}}`, "{{unfilled}}"} {
				if !strings.Contains(tc.rendered, kept) {
					t.Errorf("dropped %q from the source it was given:\n%s", kept, tc.rendered)
				}
			}
		})
	}
}

func TestAKindWithoutItsPromptFilePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("an unknown kind rendered a prompt, so a typoed granularity would reach the model half-framed")
		}
	}()
	_ = Splitter("haiku", File{Path: "a.txt"})
}

func wantInOrder(t *testing.T, rendered string, want ...string) {
	t.Helper()
	at := 0
	for _, piece := range want {
		found := strings.Index(rendered[at:], piece)
		if found < 0 {
			t.Fatalf("%q is missing or out of order:\n%s", piece, rendered)
		}
		at += found + len(piece)
	}
}
