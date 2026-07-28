package audit

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/selftest"
)

// Format renders the results grouped by file and then by rule, because the reader
// is looking at their own code rather than at the rule set. Elapsed is the wall
// clock for the whole run.
//
// It is one pass of the same Reporter a watched run feeds a result at a time, so
// a sweep prints the same bytes whether it was rendered as it went or at the end.
func Format(w io.Writer, results []Result, opts Options, elapsed time.Duration, colour bool) error {
	reporter := NewReporter(w, colour)
	for _, result := range results {
		if err := reporter.Result(result); err != nil {
			return err
		}
	}
	return reporter.Summary(results, opts, elapsed)
}

// Reporter writes a report one target at a time, opening a file heading whenever
// the file changes. A sweep of any size is otherwise silent until its last model
// call returns, which reads exactly like a hung CLI.
//
// Results have to arrive in the order Format prints them, which is the order Run
// observes them in.
type Reporter struct {
	w          io.Writer
	colour     bool
	file       string
	hasWritten bool
}

// NewReporter renders to w. Nothing is written until the first result arrives, so
// a run that could not start prints no heading it never earned.
func NewReporter(w io.Writer, colour bool) *Reporter {
	return &Reporter{w: w, colour: colour}
}

// Result writes one target's block.
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

// Summary closes the run with the one line that answers what the whole sweep did.
func (r *Reporter) Summary(results []Result, opts Options, elapsed time.Duration) error {
	var b strings.Builder
	writeSummary(&b, results, opts, elapsed)
	_, err := io.WriteString(r.w, b.String())
	return err
}

// Announce says what a sweep covers before its first model call, so the wait for
// the first file to land is not spent wondering whether anything is happening.
// It names no file: which files are in flight cannot be shown without redrawing,
// and each one names itself when its block lands.
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

// A clean target has nothing to triage, so stamping every passing rule with a
// severity would bury the handful that need one under a column of noise.
func banner(result Result) string {
	if !hasFallen(result) {
		return ""
	}
	if result.Report.Priority == "" {
		return ""
	}
	return "  " + result.Report.Priority
}

// hasFallen reads the report rather than the Result's error, because the report
// is what gets rendered: a target carrying a could-not-run has one either way.
func hasFallen(result Result) bool {
	return result.Report.Error != "" || lint.ExitFor(result.Report) != lint.ExitPass
}

// bodyOf drops the single-report header, which names the rule and the file that
// the group headers above it already carry, and normalises the trailing blank
// lines that differ between a judged report and one that could not run.
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

// writeSummary answers what the whole sweep did in one line. Targets that could
// not run are counted separately rather than folded into the failures, because a
// partial sweep reading as a clean one is the failure mode exit code 2 exists for.
func writeSummary(b *strings.Builder, results []Result, opts Options, elapsed time.Duration) {
	counts := totalsOf(results)

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
		plural(len(opts.Files), "file"), plural(len(opts.Rules), "rule"), plural(opts.Votes, "vote")))
	parts = append(parts, selftest.FormatDuration(elapsed))

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
