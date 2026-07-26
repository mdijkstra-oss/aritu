package selftest

import (
	"context"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/claudecli"
	"github.com/matthijn/aritu/internal/lib/vote"
)

// Result pairs a fixture with the report it produced and whether that report
// matched the expectation the fixture's directory name carries.
type Result struct {
	Fixture rule.Fixture
	Report  lint.Report
	Held    bool
	Err     error
}

// Options configures one selftest run.
type Options struct {
	Rule   rule.Rule
	Base   string
	Votes  int
	Model  string
	Effort string
}

// Run applies the rule to every fixture in order. A fixture that errors is
// recorded and the run continues, so one unreachable call cannot hide the rest
// of the table.
func Run(ctx context.Context, ask claudecli.Ask, opts Options, fixtures []rule.Fixture) []Result {
	results := make([]Result, 0, len(fixtures))
	for _, fixture := range fixtures {
		report, err := lint.Apply(ctx, ask, lint.Options{
			Rule:   opts.Rule,
			Base:   opts.Base,
			File:   fixture.TestFile,
			Votes:  opts.Votes,
			Model:  opts.Model,
			Effort: opts.Effort,
		})
		results = append(results, Result{
			Fixture: fixture,
			Report:  report,
			Held:    err == nil && Holds(fixture.Expect, report.Verdicts, opts.Votes),
			Err:     err,
		})
	}
	return results
}

// Holds reports whether counts match the fixture's expectation: a pass- fixture
// holds at exactly votes, a fail- fixture at zero. A fixture yielding no test
// functions never holds, because it demonstrates nothing. A fail- fixture that
// needed a dissenting vote to fire is one bad test away from missing, so
// anything between the poles fails.
func Holds(expect rule.Expectation, counts map[string]int, votes int) bool {
	if len(counts) == 0 {
		return false
	}
	switch expect {
	case rule.ExpectPass:
		return vote.IsUnanimous(counts, votes)
	case rule.ExpectFail:
		return vote.IsUnanimous(counts, 0)
	default:
		panic(fmt.Sprintf("unknown expectation: %d", int(expect)))
	}
}

// ExitFor derives the exit status for a completed table.
func ExitFor(results []Result) lint.Exit {
	exit := lint.ExitPass
	for _, result := range results {
		if result.Err != nil {
			return lint.ExitError
		}
		if !result.Held {
			exit = lint.ExitFail
		}
	}
	return exit
}

// Format renders the results as an aligned table. It prints whether or not the
// run held, because the counts are the whole diagnostic.
func Format(w io.Writer, opts Options, results []Result) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "rule: %s  model: %s  votes: %d\n\n", opts.Rule.Name, opts.Model, opts.Votes)
	fmt.Fprintln(tw, "FIXTURE\tEXPECT\tRESULT\tVERDICTS")
	for _, result := range results {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", result.Fixture.Name, result.Fixture.Expect, outcomeOf(result), detailOf(result))
	}
	fmt.Fprintf(tw, "\n%d/%d fixtures hold\n", countHeld(results), len(results))
	return tw.Flush()
}

func outcomeOf(result Result) string {
	if result.Err != nil {
		return "ERROR"
	}
	if result.Held {
		return "hold"
	}
	return "MISS"
}

func detailOf(result Result) string {
	if result.Err != nil {
		return singleLine(result.Err.Error())
	}
	return formatVerdicts(result.Report.Verdicts)
}

// singleLine flattens an error whose text this package does not author. A failing
// claude subprocess contributes its stderr verbatim, and an embedded newline would
// otherwise break the row apart and misalign every column after it.
func singleLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func formatVerdicts(counts map[string]int) string {
	pairs := make([]string, 0, len(counts))
	for _, name := range slices.Sorted(maps.Keys(counts)) {
		pairs = append(pairs, fmt.Sprintf("%s=%d", name, counts[name]))
	}
	return strings.Join(pairs, " ")
}

func countHeld(results []Result) int {
	held := 0
	for _, result := range results {
		if result.Held {
			held++
		}
	}
	return held
}
