package rule

import "testing"

// preamble is spelled out here rather than read from the source, so that changing
// what the document tells its reader has to be a deliberate edit in two places
// rather than a silent one the test agrees with by construction.
const preamble = "# Coding rules\n\n" +
	"The following rules must be abided by. Read them before writing anything they are\n" +
	"about, and check what you wrote against them before calling it done.\n"

func TestRulebook(t *testing.T) {
	oneThing := Rule{Name: "tests-one-thing", Description: "Give every test one reason to fail."}
	selfContained := Rule{Name: "self-contained", Description: "Create the state your test uses."}

	tests := []struct {
		name  string
		rules []Rule
		want  string
	}{
		{
			name:  "a rule becomes a heading with its description under it",
			rules: []Rule{oneThing},
			want:  preamble + "\n## tests-one-thing\n\nGive every test one reason to fail.\n",
		},
		{
			name:  "several rules keep the order they were enabled in",
			rules: []Rule{selfContained, oneThing},
			want: preamble +
				"\n## self-contained\n\nCreate the state your test uses.\n" +
				"\n## tests-one-thing\n\nGive every test one reason to fail.\n",
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

// TestRulebookCarriesNoJudgingCriterion pins the split between the two pieces of
// prose a rule holds. The criterion is written to settle a verdict about a file
// that already exists, and leaking it into the document handed to whoever is about
// to write one would hand them the argument rather than the instruction.
func TestRulebookCarriesNoJudgingCriterion(t *testing.T) {
	judged := Rule{
		Name:        "tests-one-thing",
		Description: "Give every test one reason to fail.",
		Prompt:      "DISQUALIFIES the rule: several unrelated behaviours in one function.",
	}

	got := Rulebook([]Rule{judged})

	if want := preamble + "\n## tests-one-thing\n\nGive every test one reason to fail.\n"; got != want {
		t.Errorf("Rulebook() = %q, want %q", got, want)
	}
}
