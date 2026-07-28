package rule

import "testing"

// preamble is spelled out here rather than read from the source, so that changing
// what the document tells its reader has to be a deliberate edit in two places
// rather than a silent one the test agrees with by construction.
const preamble = "# Coding rules\n\n" +
	"The following rules must be abided by. Read them before writing anything they are\n" +
	"about, and check what you wrote against them before calling it done.\n\n" +
	"They are grouped by what a violation costs. Where a file breaks rules in more\n" +
	"than one band, the higher band goes first: its fix moves the code the lower ones\n" +
	"are about.\n"

// The band headings are spelled out for the same reason the preamble is.
const (
	severeBand = "\n## Severe\n\nFix these before anything below them. The fix relocates the code around it, and findings nested inside one often go with it.\n"
	highBand   = "\n## High\n\nA shape callers depend on. The fix reaches past the declaration that carries it, so it lands after the structural work above.\n"
	medBand    = "\n## Medium\n\nLocal enough to fix where it stands: a rename, a move, a deletion. Nothing here blocks the work above it.\n"
)

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
	oneThing := Rule{Name: "tests-one-thing", Prompt: "A test must have one reason to fail.", Priority: PriorityMed}
	selfContained := Rule{Name: "self-contained", Prompt: "A suite must produce the same verdict anywhere.", Priority: PriorityMed}
	oneJob := Rule{Name: "one-job", Prompt: "A file has one job.", Priority: PrioritySevere}
	fewArguments := Rule{Name: "few-arguments", Prompt: "A signature tells the whole story.", Priority: PriorityHigh}

	tests := []struct {
		name  string
		rules []Rule
		want  string
	}{
		{
			name:  "a rule becomes its derived heading under the band it declared",
			rules: []Rule{oneThing},
			want:  preamble + medBand + "\n### Tests one thing\n\nA test must have one reason to fail.\n",
		},
		{
			name:  "rules in one band keep the order they were enabled in",
			rules: []Rule{selfContained, oneThing},
			want: preamble + medBand +
				"\n### Self contained\n\nA suite must produce the same verdict anywhere.\n" +
				"\n### Tests one thing\n\nA test must have one reason to fail.\n",
		},
		{
			name:  "the hardest band leads however the rules were ordered",
			rules: []Rule{oneThing, fewArguments, oneJob},
			want: preamble +
				severeBand + "\n### One job\n\nA file has one job.\n" +
				highBand + "\n### Few arguments\n\nA signature tells the whole story.\n" +
				medBand + "\n### Tests one thing\n\nA test must have one reason to fail.\n",
		},
		{
			name:  "a band no rule declared is left out rather than printed empty",
			rules: []Rule{oneJob},
			want:  preamble + severeBand + "\n### One job\n\nA file has one job.\n",
		},
		{
			name:  "a rule that declared no priority still reaches the reader",
			rules: []Rule{{Name: "undeclared", Prompt: "Something worth writing down."}},
			want:  preamble + medBand + "\n### Undeclared\n\nSomething worth writing down.\n",
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
