package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/service"
)

const fileSource = `package scenario

import "testing"

func TestAlpha(t *testing.T) {}
`

var (
	byBehaviour = rule.Rule{Name: "named-for-behavior", Prompt: "rule body: a name states the behaviour", Granularity: rule.GranularityTestCase}
	oneReason   = rule.Rule{Name: "one-reason-to-fail", Prompt: "rule body: one reason to fail", Granularity: rule.GranularityFunction}
	noMocking   = rule.Rule{Name: "no-mocking-under-test", Prompt: "rule body: the real unit is exercised", Granularity: rule.GranularityFunction}

	// otherFragment asks the same file a different question: its prompt carries a
	// fragment the others do not, so its enumeration is not theirs to share.
	otherFragment = rule.Rule{Name: "prose-rule", Prompt: "rule body: the prose is legible", Granularity: rule.GranularityFunction, Include: []string{"tests"}}
)

func TestRun(t *testing.T) {
	alpha := target{name: "alpha_test.go", leaves: []string{"TestAlpha (host)", "TestAlpha (port)"}}
	beta := target{name: "beta_test.go", leaves: []string{"TestBeta"}}

	tests := []struct {
		name             string
		rules            []rule.Rule
		targets          []target
		votes            int
		rejects          map[string][]string
		want             []wantResult
		wantEnumerations map[string]int
		wantJudgements   int
	}{
		{
			name:    "every file is judged by every rule, ordered by file then rule",
			rules:   []rule.Rule{byBehaviour, oneReason},
			targets: []target{alpha, beta},
			votes:   1,
			rejects: map[string][]string{"named-for-behavior alpha_test.go": {"TestAlpha (port)"}},
			want: []wantResult{
				{rule: "named-for-behavior", file: "alpha_test.go", verdicts: map[string]int{"TestAlpha (host)": 1, "TestAlpha (port)": 0}},
				{rule: "one-reason-to-fail", file: "alpha_test.go", verdicts: map[string]int{"TestAlpha": 1}},
				{rule: "named-for-behavior", file: "beta_test.go", verdicts: map[string]int{"TestBeta": 1}},
				{rule: "one-reason-to-fail", file: "beta_test.go", verdicts: map[string]int{"TestBeta": 1}},
			},
			wantEnumerations: map[string]int{"alpha_test.go": 1, "beta_test.go": 1},
			wantJudgements:   4,
		},
		{
			name:    "one enumeration serves every rule for a file even when they start together",
			rules:   []rule.Rule{byBehaviour, oneReason, noMocking},
			targets: []target{alpha.after(30 * time.Millisecond)},
			votes:   1,
			want: []wantResult{
				{rule: "named-for-behavior", file: "alpha_test.go", verdicts: map[string]int{"TestAlpha (host)": 1, "TestAlpha (port)": 1}},
				{rule: "one-reason-to-fail", file: "alpha_test.go", verdicts: map[string]int{"TestAlpha": 1}},
				{rule: "no-mocking-under-test", file: "alpha_test.go", verdicts: map[string]int{"TestAlpha": 1}},
			},
			wantEnumerations: map[string]int{"alpha_test.go": 1},
			wantJudgements:   3,
		},
		{
			name:    "rules including different fragments each get their own enumeration",
			rules:   []rule.Rule{oneReason, otherFragment},
			targets: []target{alpha},
			votes:   1,
			want: []wantResult{
				{rule: "one-reason-to-fail", file: "alpha_test.go", verdicts: map[string]int{"TestAlpha": 1}},
				{rule: "prose-rule", file: "alpha_test.go", verdicts: map[string]int{"TestAlpha": 1}},
			},
			wantEnumerations: map[string]int{"alpha_test.go": 2},
			wantJudgements:   2,
		},
		{
			name:    "results keep file and rule order when the replies complete out of order",
			rules:   []rule.Rule{byBehaviour, oneReason},
			targets: []target{alpha.after(40 * time.Millisecond), beta},
			votes:   1,
			want: []wantResult{
				{rule: "named-for-behavior", file: "alpha_test.go", verdicts: map[string]int{"TestAlpha (host)": 1, "TestAlpha (port)": 1}},
				{rule: "one-reason-to-fail", file: "alpha_test.go", verdicts: map[string]int{"TestAlpha": 1}},
				{rule: "named-for-behavior", file: "beta_test.go", verdicts: map[string]int{"TestBeta": 1}},
				{rule: "one-reason-to-fail", file: "beta_test.go", verdicts: map[string]int{"TestBeta": 1}},
			},
			wantEnumerations: map[string]int{"alpha_test.go": 1, "beta_test.go": 1},
			wantJudgements:   4,
		},
		{
			name:    "every vote is asked once per target",
			rules:   []rule.Rule{byBehaviour},
			targets: []target{alpha, beta},
			votes:   3,
			want: []wantResult{
				{rule: "named-for-behavior", file: "alpha_test.go", verdicts: map[string]int{"TestAlpha (host)": 3, "TestAlpha (port)": 3}},
				{rule: "named-for-behavior", file: "beta_test.go", verdicts: map[string]int{"TestBeta": 3}},
			},
			wantEnumerations: map[string]int{"alpha_test.go": 1, "beta_test.go": 1},
			wantJudgements:   6,
		},
		{
			name:    "an unreadable file fails its own rules and leaves the others judged",
			rules:   []rule.Rule{byBehaviour, oneReason},
			targets: []target{alpha, {name: "gone_test.go", missing: true}},
			votes:   1,
			want: []wantResult{
				{rule: "named-for-behavior", file: "alpha_test.go", verdicts: map[string]int{"TestAlpha (host)": 1, "TestAlpha (port)": 1}},
				{rule: "one-reason-to-fail", file: "alpha_test.go", verdicts: map[string]int{"TestAlpha": 1}},
				{rule: "named-for-behavior", file: "gone_test.go", errText: "no such file"},
				{rule: "one-reason-to-fail", file: "gone_test.go", errText: "no such file"},
			},
			wantEnumerations: map[string]int{"alpha_test.go": 1},
			wantJudgements:   2,
		},
		{
			name:    "a refused enumeration fails every rule for that file and is asked once",
			rules:   []rule.Rule{byBehaviour, oneReason},
			targets: []target{{name: "alpha_test.go", leaves: alpha.leaves, refuses: true}, beta},
			votes:   1,
			want: []wantResult{
				{rule: "named-for-behavior", file: "alpha_test.go", errText: "the model refused"},
				{rule: "one-reason-to-fail", file: "alpha_test.go", errText: "the model refused"},
				{rule: "named-for-behavior", file: "beta_test.go", verdicts: map[string]int{"TestBeta": 1}},
				{rule: "one-reason-to-fail", file: "beta_test.go", verdicts: map[string]int{"TestBeta": 1}},
			},
			wantEnumerations: map[string]int{"alpha_test.go": 1, "beta_test.go": 1},
			wantJudgements:   2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			answers := writeCorpus(t, tc.targets)
			table := &model{files: answers, rules: tc.rules, rejects: tc.rejects, enumerations: map[string]int{}}
			opts := Options{Rules: tc.rules, Files: pathsOf(answers), Votes: tc.votes, Model: "sonnet"}

			results := Run(context.Background(), table.ask, opts)

			checkResults(t, results, tc.want)
			if !maps.Equal(table.enumerationCounts(), tc.wantEnumerations) {
				t.Errorf("enumerations = %v, want %v", table.enumerationCounts(), tc.wantEnumerations)
			}
			if got := table.judgementCount(); got != tc.wantJudgements {
				t.Errorf("verdict calls = %d, want %d", got, tc.wantJudgements)
			}
			for _, result := range results {
				if result.Duration <= 0 {
					t.Errorf("%s over %s was not timed", result.Report.Rule, result.Report.File)
				}
			}
		})
	}
}

func checkResults(t *testing.T, results []Result, want []wantResult) {
	t.Helper()
	if len(results) != len(want) {
		t.Fatalf("got %d results, want %d", len(results), len(want))
	}
	for i, wanted := range want {
		got := results[i]
		if got.Report.Rule != wanted.rule || filepath.Base(got.Report.File) != wanted.file {
			t.Errorf("result %d = %s over %s, want %s over %s",
				i, got.Report.Rule, filepath.Base(got.Report.File), wanted.rule, wanted.file)
			continue
		}
		if wanted.errText != "" {
			checkErrored(t, got, wanted.errText)
			continue
		}
		if got.Err != nil {
			t.Errorf("result %d: unexpected error: %v", i, got.Err)
			continue
		}
		if !maps.Equal(got.Report.Verdicts, wanted.verdicts) {
			t.Errorf("result %d verdicts = %v, want %v", i, got.Report.Verdicts, wanted.verdicts)
		}
	}
}

func checkErrored(t *testing.T, got Result, errText string) {
	t.Helper()
	if got.Err == nil || !strings.Contains(got.Err.Error(), errText) {
		t.Errorf("%s over %s: error = %v, want one containing %q",
			got.Report.Rule, filepath.Base(got.Report.File), got.Err, errText)
		return
	}
	if !strings.Contains(got.Report.Error, errText) {
		t.Errorf("%s over %s: report carries %q, want the error so it still prints",
			got.Report.Rule, filepath.Base(got.Report.File), got.Report.Error)
	}
}

func TestRunJudgesAFileGranularityRuleWithoutEnumerating(t *testing.T) {
	wholeFile := rule.Rule{Name: "shared-state", Prompt: "rule body: the file holds no shared state", Granularity: rule.GranularityFile}
	written := writeCorpus(t, []target{{name: "alpha_test.go", leaves: []string{"TestAlpha"}}})
	table := &model{files: written, rules: []rule.Rule{wholeFile}, enumerations: map[string]int{}}
	opts := Options{Rules: []rule.Rule{wholeFile}, Files: pathsOf(written), Votes: 1, Model: "sonnet"}

	results := Run(context.Background(), table.ask, opts)

	if counts := table.enumerationCounts(); len(counts) > 0 {
		t.Errorf("enumerations = %v, want none: the unit is the path", counts)
	}
	checkResults(t, results, []wantResult{{
		rule:     "shared-state",
		file:     "alpha_test.go",
		verdicts: map[string]int{written[0].path: 1},
	}})
}

func TestRunHandsEachTargetOverInReportOrder(t *testing.T) {
	alpha := target{name: "alpha_test.go", leaves: []string{"TestAlpha (host)"}}
	beta := target{name: "beta_test.go", leaves: []string{"TestBeta"}}

	tests := []struct {
		name    string
		rules   []rule.Rule
		targets []target
		want    []string
	}{
		{
			name:    "a file that finished first still waits for the file printed above it",
			rules:   []rule.Rule{byBehaviour, oneReason},
			targets: []target{alpha.after(40 * time.Millisecond), beta},
			want: []string{
				"named-for-behavior over alpha_test.go",
				"one-reason-to-fail over alpha_test.go",
				"named-for-behavior over beta_test.go",
				"one-reason-to-fail over beta_test.go",
			},
		},
		{
			name:    "every rule over one file is handed over once",
			rules:   []rule.Rule{byBehaviour, oneReason, noMocking},
			targets: []target{alpha},
			want: []string{
				"named-for-behavior over alpha_test.go",
				"one-reason-to-fail over alpha_test.go",
				"no-mocking-under-test over alpha_test.go",
			},
		},
		{
			name:    "a target that could not run is handed over like any other",
			rules:   []rule.Rule{byBehaviour},
			targets: []target{{name: "gone_test.go", missing: true}, beta},
			want: []string{
				"named-for-behavior over gone_test.go",
				"named-for-behavior over beta_test.go",
			},
		},
		{
			name:    "a run over nothing hands nothing over",
			rules:   []rule.Rule{byBehaviour},
			targets: nil,
			want:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			answers := writeCorpus(t, tc.targets)
			table := &model{files: answers, rules: tc.rules, enumerations: map[string]int{}}
			var observed []string
			opts := Options{
				Rules: tc.rules, Files: pathsOf(answers), Votes: 1, Model: "sonnet",
				Observe: func(result Result) {
					observed = append(observed, fmt.Sprintf("%s over %s", result.Report.Rule, filepath.Base(result.Report.File)))
				},
			}

			results := Run(context.Background(), table.ask, opts)

			if !slices.Equal(observed, tc.want) {
				t.Errorf("observed %v, want %v", observed, tc.want)
			}
			if len(observed) != len(results) {
				t.Errorf("observed %d of %d results", len(observed), len(results))
			}
		})
	}
}

// TestReporterWritesEachResultAsItArrives feeds one reporter in sequence, because
// what it writes for a result depends on the file the result before it carried.
func TestReporterWritesEachResultAsItArrives(t *testing.T) {
	passing := func(name, file string, elapsed time.Duration, unit string) Result {
		return Result{
			Report: lint.Report{
				Rule: name, File: file, Votes: 2,
				Verdicts: map[string]int{unit: 2},
			},
			Duration: elapsed,
		}
	}

	steps := []struct {
		name   string
		result Result
		want   string
	}{
		{
			name:   "the first result opens its file heading",
			result: passing("named-for-behavior", "alpha_test.go", 100*time.Millisecond, "TestAlpha"),
			want: "alpha_test.go\n" +
				"  named-for-behavior  100ms\n" +
				"    ✓ TestAlpha\n" +
				"\n" +
				"    1 passed  ·  1 unit, 2 votes\n" +
				"\n",
		},
		{
			name:   "a second rule over the same file writes no second heading",
			result: passing("one-reason-to-fail", "alpha_test.go", 400*time.Millisecond, "TestAlpha"),
			want: "  one-reason-to-fail  400ms\n" +
				"    ✓ TestAlpha\n" +
				"\n" +
				"    1 passed  ·  1 unit, 2 votes\n" +
				"\n",
		},
		{
			name:   "the next file opens a heading of its own",
			result: passing("named-for-behavior", "beta_test.go", 300*time.Millisecond, "TestBeta"),
			want: "beta_test.go\n" +
				"  named-for-behavior  300ms\n" +
				"    ✓ TestBeta\n" +
				"\n" +
				"    1 passed  ·  1 unit, 2 votes\n" +
				"\n",
		},
	}

	var out bytes.Buffer
	reporter := NewReporter(&out, false)
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			out.Reset()

			if err := reporter.Result(step.result); err != nil {
				t.Fatalf("Result returned %v", err)
			}

			if out.String() != step.want {
				t.Errorf("wrote:\n%s\nwant:\n%s", out.String(), step.want)
			}
		})
	}
}

func TestAnnounceSaysWhatTheSweepCovers(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want string
	}{
		{
			name: "one file against one rule reads in the singular",
			opts: Options{Rules: []rule.Rule{byBehaviour}, Files: []string{"alpha_test.go"}, Votes: 1},
			want: "judging 1 file against 1 rule, 1 vote\n\n",
		},
		{
			name: "a sweep names its counts before any of it has run",
			opts: Options{
				Rules: []rule.Rule{byBehaviour, oneReason},
				Files: []string{"alpha_test.go", "beta_test.go", "gamma_test.go"},
				Votes: 4,
			},
			want: "judging 3 files against 2 rules, 4 votes\n\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer

			Announce(&out, tc.opts)

			if out.String() != tc.want {
				t.Errorf("Announce wrote %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestExitFor(t *testing.T) {
	passing := Result{Report: lint.Report{Votes: 2, Verdicts: map[string]int{"TestAlpha": 2}}}
	failing := Result{Report: lint.Report{Votes: 2, Verdicts: map[string]int{"TestAlpha": 0}}}
	split := Result{Report: lint.Report{Votes: 2, Verdicts: map[string]int{"TestAlpha": 1}}}
	errored := Result{Report: lint.Report{Votes: 2, Verdicts: map[string]int{}}, Err: fmt.Errorf("no such file")}

	tests := []struct {
		name    string
		results []Result
		want    lint.Exit
	}{
		{name: "every unit of every target satisfied its rule", results: []Result{passing, passing}, want: lint.ExitPass},
		{name: "nothing to judge passes", results: nil, want: lint.ExitPass},
		{name: "one unanimous rejection fails the run", results: []Result{passing, failing}, want: lint.ExitFail},
		{name: "one split vote fails the run", results: []Result{passing, split}, want: lint.ExitFail},
		{name: "one target that could not run errors the run", results: []Result{passing, errored}, want: lint.ExitError},
		{name: "an error outranks a failure whichever came first", results: []Result{errored, failing}, want: lint.ExitError},
		{name: "an error outranks a failure that was recorded before it", results: []Result{failing, errored}, want: lint.ExitError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitFor(tc.results); got != tc.want {
				t.Errorf("ExitFor = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name    string
		results []Result
		opts    Options
		elapsed time.Duration
		want    string
	}{
		{
			name: "reports group under the file, then the rule",
			results: []Result{
				{
					Report: lint.Report{
						Rule: "named-for-behavior", File: "alpha_test.go", Votes: 2,
						Verdicts: map[string]int{"TestAlpha (host)": 2, "TestAlpha (port)": 0},
						Reasons:  map[string][]string{"TestAlpha (port)": {"names the input the case supplies"}},
					},
					Duration: 1200 * time.Millisecond,
				},
				{
					Report: lint.Report{
						Rule: "one-reason-to-fail", File: "alpha_test.go", Votes: 2,
						Verdicts: map[string]int{"TestAlpha": 2},
					},
					Duration: 400 * time.Millisecond,
				},
				{
					Report: lint.Report{
						Rule: "named-for-behavior", File: "beta_test.go", Votes: 2,
						Verdicts: map[string]int{"TestBeta": 1},
					},
					Duration: 300 * time.Millisecond,
				},
				{
					Report: lint.Report{
						Rule: "one-reason-to-fail", File: "beta_test.go", Votes: 2,
						Verdicts: map[string]int{"TestBeta": 2},
					},
					Duration: 200 * time.Millisecond,
				},
			},
			opts: Options{
				Rules: []rule.Rule{byBehaviour, oneReason},
				Files: []string{"alpha_test.go", "beta_test.go"},
				Votes: 2,
			},
			elapsed: 1500 * time.Millisecond,
			want: "alpha_test.go\n" +
				"  named-for-behavior  1.2s\n" +
				"    TestAlpha\n" +
				"      ✓ host\n" +
				"      ✗ port\n" +
				"        names the input the case supplies\n" +
				"\n" +
				"    1 passed  ·  1 failed  ·  2 units, 2 votes\n" +
				"\n" +
				"  one-reason-to-fail  400ms\n" +
				"    ✓ TestAlpha\n" +
				"\n" +
				"    1 passed  ·  1 unit, 2 votes\n" +
				"\n" +
				"beta_test.go\n" +
				"  named-for-behavior  300ms\n" +
				"    ! TestBeta (1 of 2)\n" +
				"\n" +
				"    0 passed  ·  1 failed  ·  1 split  ·  1 unit, 2 votes\n" +
				"\n" +
				"  one-reason-to-fail  200ms\n" +
				"    ✓ TestBeta\n" +
				"\n" +
				"    1 passed  ·  1 unit, 2 votes\n" +
				"\n" +
				"  3 passed  ·  2 failed  ·  1 split  ·  2 files, 2 rules, 2 votes  ·  1.5s\n",
		},
		{
			name: "a target that could not run is printed and counted apart from the failures",
			results: []Result{
				{
					Report: lint.Report{
						Rule: "named-for-behavior", File: "alpha_test.go", Votes: 2,
						Verdicts: map[string]int{"TestAlpha": 2},
					},
					Duration: 100 * time.Millisecond,
				},
				{
					Report: lint.Report{
						Rule: "named-for-behavior", File: "gone_test.go", Votes: 2,
						Verdicts: map[string]int{},
						Error:    "open gone_test.go: no such file or directory",
					},
					Duration: 50 * time.Millisecond,
					Err:      fmt.Errorf("open gone_test.go: no such file or directory"),
				},
			},
			opts: Options{
				Rules: []rule.Rule{byBehaviour},
				Files: []string{"alpha_test.go", "gone_test.go"},
				Votes: 2,
			},
			elapsed: 200 * time.Millisecond,
			want: "alpha_test.go\n" +
				"  named-for-behavior  100ms\n" +
				"    ✓ TestAlpha\n" +
				"\n" +
				"    1 passed  ·  1 unit, 2 votes\n" +
				"\n" +
				"gone_test.go\n" +
				"  named-for-behavior  50ms\n" +
				"    ✗ could not run\n" +
				"      open gone_test.go: no such file or directory\n" +
				"\n" +
				"  1 passed  ·  1 errored  ·  2 files, 1 rule, 2 votes  ·  200ms\n",
		},
		{
			name: "a file with nothing to judge still prints under its own heading",
			results: []Result{
				{
					Report: lint.Report{
						Rule: "named-for-behavior", File: "empty_test.go", Votes: 2,
						Verdicts: map[string]int{},
					},
					Duration: 100 * time.Millisecond,
				},
			},
			opts: Options{
				Rules: []rule.Rule{byBehaviour},
				Files: []string{"empty_test.go"},
				Votes: 2,
			},
			elapsed: 100 * time.Millisecond,
			want: "empty_test.go\n" +
				"  named-for-behavior  100ms\n" +
				"    no test units found\n" +
				"\n" +
				"  0 passed  ·  1 file, 1 rule, 2 votes  ·  100ms\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := Format(&out, tc.results, tc.opts, tc.elapsed, false); err != nil {
				t.Fatalf("Format returned %v", err)
			}
			if out.String() != tc.want {
				t.Fatalf("Format wrote:\n%s\nwant:\n%s", out.String(), tc.want)
			}
		})
	}
}

func TestFormatEmitsColourOnlyWhenAsked(t *testing.T) {
	results := []Result{{
		Report: lint.Report{
			Rule: "named-for-behavior", File: "alpha_test.go", Votes: 2,
			Verdicts: map[string]int{"TestAlpha": 2},
		},
		Duration: 100 * time.Millisecond,
	}}
	opts := Options{Rules: []rule.Rule{byBehaviour}, Files: []string{"alpha_test.go"}, Votes: 2}

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
			if err := Format(&out, results, opts, time.Second, tc.colour); err != nil {
				t.Fatalf("Format returned %v", err)
			}
			if strings.Contains(out.String(), "\x1b[") != tc.wantEscape {
				t.Errorf("colour=%t produced %q", tc.colour, out.String())
			}
		})
	}
}

func TestEnvelopeOf(t *testing.T) {
	tests := []struct {
		name    string
		results []Result
		want    []string
	}{
		{
			name:    "no targets carry an empty envelope rather than a missing one",
			results: nil,
			want:    []string{},
		},
		{
			name: "reports keep the order the run produced them in",
			results: []Result{
				{Report: lint.Report{Rule: "named-for-behavior", File: "alpha_test.go"}},
				{Report: lint.Report{Rule: "one-reason-to-fail", File: "alpha_test.go"}},
				{Report: lint.Report{Rule: "named-for-behavior", File: "beta_test.go"}},
			},
			want: []string{
				"named-for-behavior over alpha_test.go",
				"one-reason-to-fail over alpha_test.go",
				"named-for-behavior over beta_test.go",
			},
		},
		{
			name: "a target that could not run is carried too",
			results: []Result{
				{Report: lint.Report{Rule: "named-for-behavior", File: "gone_test.go", Error: "no such file"},
					Err: fmt.Errorf("no such file")},
			},
			want: []string{"named-for-behavior over gone_test.go"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := make([]string, 0, len(tc.results))
			for _, report := range EnvelopeOf(tc.results).Reports {
				got = append(got, fmt.Sprintf("%s over %s", report.Rule, report.File))
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("reports = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEnvelopeOfEncodesEveryReportUnderOneKey(t *testing.T) {
	results := []Result{
		{Report: lint.Report{Rule: "named-for-behavior", File: "alpha_test.go", Votes: 2, Verdicts: map[string]int{"TestAlpha": 2}}},
		{Report: lint.Report{Rule: "one-reason-to-fail", File: "alpha_test.go", Votes: 2, Verdicts: map[string]int{}, Error: "no such file"}},
	}

	encoded, err := json.Marshal(EnvelopeOf(results))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"reports":[` +
		`{"rule":"named-for-behavior","file":"alpha_test.go","votes":2,"verdicts":{"TestAlpha":2}},` +
		`{"rule":"one-reason-to-fail","file":"alpha_test.go","votes":2,"verdicts":{},"error":"no such file"}` +
		`]}`
	if string(encoded) != want {
		t.Errorf("envelope = %s, want %s", encoded, want)
	}
}

// target is one file the table judges: what the model enumerates it to, how long
// each answer about it takes, and whether it can be read or enumerated at all.
// Path is the temp file writeCorpus wrote, Name the base the counts report under.
type target struct {
	name    string
	path    string
	leaves  []string
	missing bool
	refuses bool
	delay   time.Duration
}

func (t target) after(delay time.Duration) target {
	t.delay = delay
	return t
}

type wantResult struct {
	rule     string
	file     string
	verdicts map[string]int
	errText  string
}

// model answers from the table rather than from a process: a file's prompt gets
// that file's leaves, and a verdict prompt gets one answer per unit the rule's
// granularity derives from them. It counts what it was asked, because one
// enumeration per file is the property the run exists to hold.
type model struct {
	files   []target
	rules   []rule.Rule
	rejects map[string][]string

	mu           sync.Mutex
	enumerations map[string]int
	judgements   int
}

func (m *model) ask(_ context.Context, req service.Request) (json.RawMessage, error) {
	asked, isKnown := m.fileIn(req.Prompt)
	if !isKnown {
		return nil, fmt.Errorf("the prompt names no file the table knows")
	}
	if string(req.Schema) == lint.NamesSchema {
		return m.enumerate(asked)
	}
	judged, isKnown := m.ruleIn(req.Prompt)
	if !isKnown {
		return nil, fmt.Errorf("the prompt carries no rule body the table knows")
	}
	return m.judge(asked, judged)
}

func (m *model) enumerate(asked target) (json.RawMessage, error) {
	m.mu.Lock()
	m.enumerations[asked.name]++
	m.mu.Unlock()

	time.Sleep(asked.delay)
	if asked.refuses {
		return nil, fmt.Errorf("the model refused to enumerate %s", asked.name)
	}
	return json.Marshal(namesReply{Names: asked.leaves})
}

func (m *model) judge(asked target, judged rule.Rule) (json.RawMessage, error) {
	m.mu.Lock()
	m.judgements++
	m.mu.Unlock()

	time.Sleep(asked.delay)
	rejected := m.rejects[judged.Name+" "+asked.name]
	answers := map[string]verdictAnswer{}
	for _, unit := range lint.UnitsAt(judged.Granularity, asked.path, asked.leaves) {
		answers[unit.Key] = answerFor(slices.Contains(rejected, unit.Name))
	}
	return json.Marshal(answers)
}

func (m *model) fileIn(prompt string) (target, bool) {
	for _, candidate := range m.files {
		if strings.Contains(prompt, "=== FILE: "+candidate.path+" ===") {
			return candidate, true
		}
	}
	return target{}, false
}

func (m *model) ruleIn(prompt string) (rule.Rule, bool) {
	for _, candidate := range m.rules {
		if strings.Contains(prompt, candidate.Prompt) {
			return candidate, true
		}
	}
	return rule.Rule{}, false
}

func (m *model) enumerationCounts() map[string]int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Clone(m.enumerations)
}

func (m *model) judgementCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.judgements
}

func answerFor(isRejected bool) verdictAnswer {
	if isRejected {
		return verdictAnswer{Reason: "the table rejects this unit"}
	}
	return verdictAnswer{Satisfies: true}
}

type namesReply struct {
	Names []string `json:"names"`
}

type verdictAnswer struct {
	Satisfies bool   `json:"satisfies"`
	Reason    string `json:"reason"`
}

func writeCorpus(t *testing.T, targets []target) []target {
	t.Helper()
	dir := t.TempDir()
	written := make([]target, 0, len(targets))
	for _, wanted := range targets {
		wanted.path = filepath.Join(dir, wanted.name)
		if !wanted.missing {
			if err := os.WriteFile(wanted.path, []byte(fileSource), 0o600); err != nil {
				t.Fatalf("writing %s: %v", wanted.path, err)
			}
		}
		written = append(written, wanted)
	}
	return written
}

func pathsOf(targets []target) []string {
	paths := make([]string, 0, len(targets))
	for _, one := range targets {
		paths = append(paths, one.path)
	}
	return paths
}
