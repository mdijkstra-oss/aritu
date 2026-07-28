package lint

import (
	"bytes"
	"strings"
	"testing"
)

func TestOutcomeFor(t *testing.T) {
	tests := []struct {
		name  string
		count int
		votes int
		want  Outcome
	}{
		{name: "every vote agreed", count: 4, votes: 4, want: OutcomePass},
		{name: "a single vote agreeing is unanimous when one was asked", count: 1, votes: 1, want: OutcomePass},
		{name: "no vote agreed", count: 0, votes: 4, want: OutcomeFail},
		{name: "a majority carries the unit", count: 3, votes: 4, want: OutcomePass},
		{name: "a minority falls short", count: 1, votes: 4, want: OutcomeFail},
		{name: "an exact tie is split", count: 2, votes: 4, want: OutcomeSplit},
		{name: "two of three is a majority", count: 2, votes: 3, want: OutcomePass},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := OutcomeFor(tc.count, tc.votes); got != tc.want {
				t.Errorf("OutcomeFor(%d, %d) = %d, want %d", tc.count, tc.votes, got, tc.want)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name   string
		report Report
		want   string
	}{
		{
			name: "only what fell short is listed, grouped under its function",
			report: Report{
				Rule: "named-for-behavior", File: "internal/lib/vote/vote_test.go", Votes: 2,
				Verdicts: map[string]int{
					"TestCollect (rejects zero rounds)":            2,
					"TestIsUnanimous (every count at the total)":   0,
					"TestIsUnanimous (a single vote at the total)": 1,
					"TestIsUnanimous (an empty tally)":             2,
					"TestSlugify":                                  0,
				},
				Reasons: map[string][]string{
					"TestIsUnanimous (every count at the total)":   {"names the input and never says what it returns"},
					"TestIsUnanimous (a single vote at the total)": {"describes the setup alone"},
					"TestSlugify": {"names the unit under test with no stated outcome"},
				},
			},
			want: "named-for-behavior  internal/lib/vote/vote_test.go\n" +
				"\n" +
				"  TestIsUnanimous\n" +
				"    ! a single vote at the total (1 of 2)\n" +
				"      describes the setup alone\n" +
				"    ✗ every count at the total\n" +
				"      names the input and never says what it returns\n" +
				"\n" +
				"  ✗ TestSlugify\n" +
				"    names the unit under test with no stated outcome\n" +
				"\n" +
				"  2 passed  ·  3 failed  ·  1 split  ·  5 units, 2 votes\n",
		},
		{
			name: "a clean file is its count alone, no unit listed",
			report: Report{
				Rule: "no-mocking-under-test", File: "parser_test.go", Votes: 4,
				Verdicts: map[string]int{"TestParsesHost": 4},
			},
			want: "no-mocking-under-test  parser_test.go\n" +
				"\n" +
				"  1 passed  ·  1 unit, 4 votes\n",
		},
		{
			name: "a could-not-run reports the error instead of verdicts",
			report: Report{
				Rule: "named-for-behavior", File: "/nope/foo_test.go", Votes: 2,
				Verdicts: map[string]int{},
				Error:    "open /nope/foo_test.go: no such file or directory",
			},
			want: "named-for-behavior  /nope/foo_test.go\n" +
				"\n" +
				"  ✗ could not run\n" +
				"    open /nope/foo_test.go: no such file or directory\n" +
				"\n",
		},
		{
			name: "a file with nothing to judge says so",
			report: Report{
				Rule: "named-for-behavior", File: "empty_test.go", Votes: 2,
				Verdicts: map[string]int{},
			},
			want: "named-for-behavior  empty_test.go\n" +
				"\n" +
				"  no units to judge\n" +
				"\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := Format(&out, tc.report, false); err != nil {
				t.Fatalf("Format returned %v", err)
			}
			if out.String() != tc.want {
				t.Fatalf("Format wrote:\n%s\nwant:\n%s", out.String(), tc.want)
			}
		})
	}
}

func TestFormatAttachesEachReasonToItsOwnFunction(t *testing.T) {
	report := Report{
		Rule: "named-for-behavior", File: "parser_test.go", Votes: 2,
		Verdicts: map[string]int{
			"TestParse (empty input)":    0,
			"TestValidate (empty input)": 0,
		},
		Reasons: map[string][]string{
			"TestParse (empty input)":    {"the parse reason"},
			"TestValidate (empty input)": {"the validate reason"},
		},
	}

	var out bytes.Buffer
	if err := Format(&out, report, false); err != nil {
		t.Fatalf("Format returned %v", err)
	}

	want := "  TestParse\n    ✗ empty input\n      the parse reason\n\n" +
		"  TestValidate\n    ✗ empty input\n      the validate reason\n"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("two functions sharing a case label got their reasons crossed:\n%s", out.String())
	}
}

func TestFormatEmitsColourOnlyWhenAsked(t *testing.T) {
	report := Report{
		Rule: "named-for-behavior", File: "parser_test.go", Votes: 2,
		Verdicts: map[string]int{"TestParsesHost": 2, "TestSlugify": 0},
	}

	tests := []struct {
		name       string
		colour     bool
		wantEscape bool
	}{
		{name: "a terminal gets escape sequences", colour: true, wantEscape: true},
		{name: "a pipe gets plain text", colour: false, wantEscape: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := Format(&out, report, tc.colour); err != nil {
				t.Fatalf("Format returned %v", err)
			}
			if strings.Contains(out.String(), "\x1b[") != tc.wantEscape {
				t.Errorf("colour=%t produced %q", tc.colour, out.String())
			}
		})
	}
}
