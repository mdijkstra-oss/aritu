package prompts

import (
	"strings"
	"testing"
)

func TestRenderedPromptsCarryEveryLayerInOrder(t *testing.T) {
	tests := []struct {
		name     string
		rendered string
		want     []string
	}{
		{
			name:     "the verdict prompt runs generic, then test-shaped, then the rule, then the units",
			rendered: Verdict([]string{"tests"}, "THE-RULE", "- a unit   ->   a_key", "=== FILE: a_test.go ==="),
			want: []string{
				"return a verdict",
				"## Test shapes",
				"THE-RULE",
				"- a unit   ->   a_key",
				"=== FILE: a_test.go ===",
			},
		},
		{
			name:     "the enumeration prompt runs generic, then test-shaped, then the file",
			rendered: Enumerate([]string{"tests"}, "=== FILE: a_test.go ==="),
			want: []string{
				"List every unit",
				"## Test shapes",
				`"Name (case name)"`,
				"Report units in declaration order",
				"=== FILE: a_test.go ===",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			at := 0
			for _, want := range tc.want {
				found := strings.Index(tc.rendered[at:], want)
				if found < 0 {
					t.Fatalf("%q is missing or out of order:\n%s", want, tc.rendered)
				}
				at += found + len(want)
			}
		})
	}
}

// TestNoPromptCarriesARuleItWasNotGiven guards the property that lets one
// enumeration serve every rule in a run: the moment the enumeration prompt varies
// by rule, the answer stops being shareable and every rule pays for its own.
func TestNoPromptCarriesARuleItWasNotGiven(t *testing.T) {
	rendered := Enumerate([]string{"tests"}, "=== FILE: a_test.go ===")

	if strings.Contains(rendered, "{{rule}}") || strings.Contains(rendered, "rule above") {
		t.Errorf("the enumeration prompt refers to a rule:\n%s", rendered)
	}
}

func TestEveryPlaceholderAShippedPromptCarriesIsFilled(t *testing.T) {
	tests := []struct {
		name     string
		rendered string
	}{
		{"verdict", Verdict([]string{"tests"}, "a rule", "a unit", "a file")},
		{"enumerate", Enumerate([]string{"tests"}, "a file")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if strings.Contains(tc.rendered, "{{") {
				t.Errorf("prompt carries an unfilled placeholder:\n%s", tc.rendered)
			}
		})
	}
}

// TestBracesInTheSourceAreCodeRatherThanPlaceholders covers the input that killed
// a whole sweep: a source file is substituted into the prompt, so scanning the
// result for braces made []Param{{...}} — ordinary Go, and Handlebars, and a Go
// template — panic the process mid-run rather than be judged.
func TestBracesInTheSourceAreCodeRatherThanPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantKept string
	}{
		{
			name:     "a slice of structs",
			source:   `Params: []Param{{Name: "charset"}},`,
			wantKept: `[]Param{{Name: "charset"}}`,
		},
		{
			name:     "a name that looks like a placeholder nobody supplied",
			source:   "greeting := `{{unfilled}}`",
			wantKept: "{{unfilled}}",
		},
		{
			name:     "an opening brace the source never closes",
			source:   "left := `{{`",
			wantKept: "{{",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for label, rendered := range map[string]func() string{
				"verdict":   func() string { return Verdict([]string{"tests"}, "a rule", "a unit", tc.source) },
				"enumerate": func() string { return Enumerate([]string{"tests"}, tc.source) },
			} {
				out := rendered()
				if !strings.Contains(out, tc.wantKept) {
					t.Errorf("the %s prompt dropped %q from the source it was given", label, tc.wantKept)
				}
			}
		})
	}
}

func TestRenderPanicsOnAPlaceholderTheTemplateCarriesAndNobodyFilled(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("render did not panic on an unfilled placeholder, so braces would reach the model")
		}
	}()
	_ = render("verdict.md", map[string]string{"rule": "only one of the four"})
}

// TestOmittingEveryFragmentLeavesTheGenericPromptAlone is what makes a rule about
// something other than tests possible: the test vocabulary is included, never
// assumed, so a rule that asks for nothing is told nothing about tests.
func TestOmittingEveryFragmentLeavesTheGenericPromptAlone(t *testing.T) {
	tests := []struct {
		name     string
		rendered string
	}{
		{"verdict", Verdict(nil, "THE-RULE", "- a unit   ->   a_key", "=== FILE: notes.md ===")},
		{"enumerate", Enumerate(nil, "=== FILE: notes.md ===")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, absent := range []string{"Test shapes", "enclosing scope", "table of cases", "benchmarks"} {
				if strings.Contains(tc.rendered, absent) {
					t.Errorf("carries %q with no fragment included:\n%s", absent, tc.rendered)
				}
			}
			if strings.Contains(tc.rendered, "\n\n\n") {
				t.Errorf("the omitted fragment left a blank run behind:\n%s", tc.rendered)
			}
		})
	}
}

func TestKnownNamesEveryFragmentARuleMayInclude(t *testing.T) {
	tests := []struct {
		name  string
		given string
		want  bool
	}{
		{"a fragment that ships", "tests", true},
		{"the listing half is not includable on its own", "tests.enumerate", false},
		{"a fragment nobody wrote", "haiku", false},
		{"the empty name", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsKnown(tc.given); got != tc.want {
				t.Errorf("IsKnown(%q) = %v, want %v (known: %v)", tc.given, got, tc.want, Known())
			}
		})
	}
}
