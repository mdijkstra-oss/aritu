package selftest

import (
	"context"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/service"
	"github.com/matthijn/aritu/internal/lib/vote"
)

// Result pairs a fixture with the report it produced and whether that report
// matched the expectation the fixture's directory name carries.
type Result struct {
	Fixture  rule.Fixture
	Report   lint.Report
	Held     bool
	Err      error
	Duration time.Duration
}

// Options configures one selftest run.
type Options struct {
	Rule   rule.Rule
	Votes  int
	Model  string
	Effort string
}

// Run applies the rule to every fixture, concurrently. Results come back in
// fixture order however the calls interleave, and a fixture that errors is
// recorded rather than aborting the run, so one unreachable call cannot hide the
// rest of the table. How many calls actually run at once is bounded by the ask,
// not here.
func Run(ctx context.Context, ask service.Ask, opts Options, fixtures []rule.Fixture) []Result {
	results := make([]Result, len(fixtures))
	var wg sync.WaitGroup
	for i, fixture := range fixtures {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = judge(ctx, ask, opts, fixture)
		}()
	}
	wg.Wait()
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
// run held, because the counts are the whole diagnostic. Elapsed is the wall
// clock for the whole run, which with concurrent fixtures is less than the sum of
// the rows and is the number a caller actually waited.
func Format(w io.Writer, opts Options, results []Result, elapsed time.Duration) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "rule: %s  model: %s  votes: %d\n\n", opts.Rule.Name, opts.Model, opts.Votes)
	fmt.Fprintln(tw, "FIXTURE\tEXPECT\tRESULT\tTIME\tVERDICTS")
	for _, result := range results {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			result.Fixture.Name, result.Fixture.Expect, outcomeOf(result),
			FormatDuration(result.Duration), detailOf(result))
	}
	fmt.Fprintf(tw, "\n%d/%d fixtures hold in %s\n", countHeld(results), len(results), FormatDuration(elapsed))
	return tw.Flush()
}

// FormatDuration renders a duration at a precision worth reading: whole
// milliseconds below a second, tenths of a second above it.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Round(100 * time.Millisecond).String()
}

func judge(ctx context.Context, ask service.Ask, opts Options, fixture rule.Fixture) Result {
	started := time.Now()
	report, err := lint.Apply(ctx, ask, lint.Options{
		Rule:   opts.Rule,
		File:   fixture.TestFile,
		Votes:  opts.Votes,
		Model:  opts.Model,
		Effort: opts.Effort,
	})
	return Result{
		Fixture:  fixture,
		Report:   report,
		Held:     err == nil && Holds(fixture.Expect, report.Verdicts, opts.Votes),
		Err:      err,
		Duration: time.Since(started),
	}
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

// singleLine flattens an error whose text this package does not author. An
// endpoint contributes its own message verbatim, and an embedded newline would
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
