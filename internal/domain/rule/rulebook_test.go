package rule

import "testing"

// preamble is spelled out here rather than read from the source, so that changing
// what the document tells its reader has to be a deliberate edit in two places
// rather than a silent one the test agrees with by construction.
const preamble = "# Coding rules\n\n" +
	"The following rules must be abided by. Read them before writing anything they are\n" +
	"about, and check what you wrote against them before calling it done.\n"

func TestTitle(t *testing.T) {
	tests := []struct {
		name string
		rule string
		want string
	}{
		{name: "words are joined by hyphens", rule: "my-pretty-rule", want: "My pretty rule"},
		{name: "a one word rule is capitalised and left alone", rule: "readable", want: "Readable"},
		{name: "underscores separate words too", rule: "my_pretty_rule", want: "My pretty rule"},
		{name: "a parked rule is titled by what it asks, not by whether it runs", rule: "_tests-one-thing", want: "Tests one thing"},
		{name: "however deeply it is parked", rule: "__do-not-duplicate-code", want: "Do not duplicate code"},
		{name: "a name that is already capitalised keeps its other letters", rule: "HTTP-status-codes", want: "HTTP status codes"},
		{name: "a name that is nothing but a parking prefix has no title", rule: "__", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Title(tc.rule); got != tc.want {
				t.Errorf("Title(%q) = %q, want %q", tc.rule, got, tc.want)
			}
		})
	}
}

// TestSectionIsTheWholeRuleUnderItsDerivedHeading pins what both readers get. The
// model judging a file and the person about to write one are handed this same
// block, so a rule cannot come to mean two things.
func TestSectionIsTheWholeRuleUnderItsDerivedHeading(t *testing.T) {
	judged := Rule{Name: "__do-not-duplicate-code", Prompt: "\n\n- Write a given piece of logic once.\n"}

	got := Section(judged)

	if want := "## Do not duplicate code\n\n- Write a given piece of logic once."; got != want {
		t.Errorf("Section() = %q, want %q", got, want)
	}
}

func TestRulebook(t *testing.T) {
	oneThing := Rule{Name: "tests-one-thing", Prompt: "A test must have one reason to fail."}
	selfContained := Rule{Name: "self-contained", Prompt: "A suite must produce the same verdict anywhere."}

	tests := []struct {
		name  string
		rules []Rule
		want  string
	}{
		{
			name:  "a rule becomes its derived heading with the whole rule under it",
			rules: []Rule{oneThing},
			want:  preamble + "\n## Tests one thing\n\nA test must have one reason to fail.\n",
		},
		{
			name:  "several rules keep the order they were enabled in",
			rules: []Rule{selfContained, oneThing},
			want: preamble +
				"\n## Self contained\n\nA suite must produce the same verdict anywhere.\n" +
				"\n## Tests one thing\n\nA test must have one reason to fail.\n",
		},
		{
			name:  "no rules leaves the document its preamble and nothing to follow",
			rules: nil,
			want:  preamble,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Rulebook(tc.rules); got != tc.want {
				t.Errorf("Rulebook() = %q, want %q", got, tc.want)
			}
		})
	}
}
