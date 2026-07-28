package selftest

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"text/tabwriter"
	"time"
)

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
