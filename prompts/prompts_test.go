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
			rendered: Verdict("THE-RULE", "- a unit   ->   a_key", "=== FILE: a_test.go ==="),
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
			rendered: Enumerate("=== FILE: a_test.go ==="),
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
	rendered := Enumerate("=== FILE: a_test.go ===")

	if strings.Contains(rendered, "{{rule}}") || strings.Contains(rendered, "rule above") {
		t.Errorf("the enumeration prompt refers to a rule:\n%s", rendered)
	}
}

func TestARenderedPromptNeverReachesTheModelWithBraces(t *testing.T) {
	tests := []struct {
		name     string
		rendered string
	}{
		{"verdict", Verdict("a rule", "a unit", "a file")},
		{"enumerate", Enumerate("a file")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if strings.Contains(tc.rendered, "{{") {
				t.Errorf("prompt carries an unfilled placeholder:\n%s", tc.rendered)
			}
		})
	}
}

func TestRenderPanicsOnAPlaceholderNobodyFilled(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("render did not panic on an unfilled placeholder, so braces would reach the model")
		}
	}()
	_ = render("verdict.md", map[string]string{"rule": "only one of the four"})
}
