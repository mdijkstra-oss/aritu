package audit

import (
	"fmt"
	"io"
	"strings"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/selftest"
)

func Format(w io.Writer, o Outcome, colour bool) error {
	reporter := NewReporter(w, colour)
	for _, result := range o.Results {
		if err := reporter.Result(result); err != nil {
			return err
		}
	}
	return reporter.Summary(o)
}

// Reporter must be fed results in the order Run observes them, and writes nothing
// until the first one arrives.
type Reporter struct {
	w          io.Writer
	colour     bool
	file       string
	hasWritten bool
}

func NewReporter(w io.Writer, colour bool) *Reporter {
	return &Reporter{w: w, colour: colour}
}

func (r *Reporter) Result(result Result) error {
	var b strings.Builder
	if r.startsNewFile(result.Report.File) {
		fmt.Fprintf(&b, "%s\n", result.Report.File)
		r.file = result.Report.File
		r.hasWritten = true
	}
	if err := writeResult(&b, result, r.colour); err != nil {
		return err
	}
	_, err := io.WriteString(r.w, b.String())
	return err
}

func (r *Reporter) Summary(o Outcome) error {
	var b strings.Builder
	writeSummary(&b, o)
	_, err := io.WriteString(r.w, b.String())
	return err
}

// Announce goes out before Run, not after.
func Announce(w io.Writer, opts Options) {
	fmt.Fprintf(w, "judging %s against %s, %s\n\n",
		plural(len(opts.Files), "file"), plural(len(opts.Rules), "rule"), plural(opts.Votes, "vote"))
}

func (r *Reporter) startsNewFile(file string) bool {
	return !r.hasWritten || r.file != file
}

func writeResult(b *strings.Builder, result Result, colour bool) error {
	var rendered strings.Builder
	if err := lint.Format(&rendered, result.Report, colour); err != nil {
		return err
	}
	fmt.Fprintf(b, "  %s%s  %s\n", result.Report.Rule, banner(result), selftest.FormatDuration(result.Duration))
	b.WriteString(indent(bodyOf(rendered.String()), "  "))
	b.WriteString("\n\n")
	return nil
}

func banner(result Result) string {
	if !hasFallen(result) {
		return ""
	}
	if result.Report.Priority == "" {
		return ""
	}
	return "  " + result.Report.Priority
}

func hasFallen(result Result) bool {
	return result.Report.Error != "" || lint.ExitFor(result.Report) != lint.ExitPass
}

func bodyOf(rendered string) string {
	_, body, _ := strings.Cut(rendered, "\n\n")
	return strings.TrimRight(body, "\n")
}

func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func writeSummary(b *strings.Builder, o Outcome) {
	counts := totalsOf(o.Results)

	parts := []string{fmt.Sprintf("%d passed", counts.passed)}
	if fell := counts.failed + counts.split; fell > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", fell))
	}
	if counts.split > 0 {
		parts = append(parts, fmt.Sprintf("%d split", counts.split))
	}
	if counts.errored > 0 {
		parts = append(parts, fmt.Sprintf("%d errored", counts.errored))
	}
	parts = append(parts, fmt.Sprintf("%s, %s, %s",
		plural(len(o.Options.Files), "file"), plural(len(o.Options.Rules), "rule"), plural(o.Options.Votes, "vote")))
	parts = append(parts, selftest.FormatDuration(o.Elapsed))

	fmt.Fprintf(b, "  %s\n", strings.Join(parts, "  ·  "))
}

type totals struct {
	passed  int
	split   int
	failed  int
	errored int
}

func totalsOf(results []Result) totals {
	var counted totals
	for _, result := range results {
		if result.Err != nil {
			counted.errored++
			continue
		}
		for _, count := range result.Report.Verdicts {
			switch lint.OutcomeFor(count, result.Report.Votes) {
			case lint.OutcomePass:
				counted.passed++
			case lint.OutcomeSplit:
				counted.split++
			case lint.OutcomeFail:
				counted.failed++
			default:
				panic(fmt.Sprintf("unknown outcome for count %d", count))
			}
		}
	}
	return counted
}

func plural(n int, singular string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %ss", n, singular)
}
