package run

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/domain/selftest"
	"github.com/matthijn/aritu/internal/lib/service"
)

// Options configures one multi-target run.
type Options struct {
	Rules  []rule.Rule
	Files  []string
	Votes  int
	Model  string
	Effort string

	// Observe is handed each target as it finishes, in the order Format prints
	// them. Calls are serial, so an implementation writing to a terminal needs no
	// lock of its own.
	Observe func(Result)
}

// Result is one file judged against one rule.
type Result struct {
	Report   lint.Report
	Duration time.Duration
	Err      error
}

// Envelope is the --output json shape. The top level is no longer one report, and
// emitting a bare report for a single target would make every consumer branch.
type Envelope struct {
	Reports []lint.Report `json:"reports"`
}

// Run judges every file against every rule. Results come back ordered by file then
// rule however the calls interleave, and a target that errors is recorded rather
// than aborting the run, so one unreachable file cannot hide the rest.
//
// Each file is enumerated once, at test granularity, however many rules judge it;
// the coarser levels roll up from that list. How many calls run at once is bounded
// by the ask, not here.
func Run(ctx context.Context, ask service.Ask, opts Options) []Result {
	results := make([]Result, len(opts.Files)*len(opts.Rules))
	leaves := newLeafCache()
	landed := make(chan int, len(results))
	var wg sync.WaitGroup
	for f, file := range opts.Files {
		for r, judged := range opts.Rules {
			at := f*len(opts.Rules) + r
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[at] = judge(ctx, ask, leaves, targetFor(opts, judged, file))
				landed <- at
			}()
		}
	}
	go func() {
		wg.Wait()
		close(landed)
	}()
	observeInOrder(landed, results, orNoop(opts.Observe))
	return results
}

// observeInOrder hands each finished target to the observer in the order Format
// prints them, holding one back while a target printed above it is still running.
// Handing them over as they land would order a report by whichever call the model
// answered first, so the same run would print its files in a different order every
// time. Draining until the channel closes is what waits for the workers.
func observeInOrder(landed <-chan int, results []Result, observe func(Result)) {
	hasLanded := make([]bool, len(results))
	next := 0
	for at := range landed {
		hasLanded[at] = true
		for isLanded(hasLanded, next) {
			observe(results[next])
			next++
		}
	}
}

func isLanded(hasLanded []bool, at int) bool {
	return at < len(hasLanded) && hasLanded[at]
}

// orNoop keeps the ordering loop free of a nil check per target: a run nobody is
// watching still has to drain the channel.
func orNoop(observe func(Result)) func(Result) {
	if observe != nil {
		return observe
	}
	return func(Result) {}
}

// ExitFor derives the exit status across every target. ExitError outranks
// ExitFail: a run where one file could not be read did not check everything, and
// reporting that as an ordinary rule failure would let a hook treat a partial
// sweep as a complete one.
func ExitFor(results []Result) lint.Exit {
	exit := lint.ExitPass
	for _, result := range results {
		if result.Err != nil {
			return lint.ExitError
		}
		if lint.ExitFor(result.Report) != lint.ExitPass {
			exit = lint.ExitFail
		}
	}
	return exit
}

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

// EnvelopeOf collects the reports for JSON output, in the same order Format prints.
func EnvelopeOf(results []Result) Envelope {
	reports := make([]lint.Report, 0, len(results))
	for _, result := range results {
		reports = append(reports, result.Report)
	}
	return Envelope{Reports: reports}
}

func targetFor(opts Options, judged rule.Rule, file string) lint.Options {
	return lint.Options{
		Rule:   judged,
		File:   file,
		Votes:  opts.Votes,
		Model:  opts.Model,
		Effort: opts.Effort,
	}
}

// judge times one target from the caller's point of view, so a rule that waited on
// another rule's enumeration of the same file reports the wait it actually took.
func judge(ctx context.Context, ask service.Ask, leaves *leafCache, target lint.Options) Result {
	started := time.Now()
	report, err := reportFor(ctx, ask, leaves, target)
	if err != nil {
		report = withError(report, err)
	}
	return Result{Report: report, Duration: time.Since(started), Err: err}
}

func reportFor(ctx context.Context, ask service.Ask, leaves *leafCache, target lint.Options) (lint.Report, error) {
	units, err := unitsFor(ctx, ask, leaves, target)
	if err != nil {
		return lint.Report{Rule: target.Rule.Name, File: target.File, Votes: target.Votes}, err
	}
	return lint.Judge(ctx, ask, target, units)
}

// unitsFor consults the cache only for a rule whose units the model has to list.
// A file-granularity rule judges the path, so asking would spend a call on an
// answer already in hand.
func unitsFor(ctx context.Context, ask service.Ask, leaves *leafCache, target lint.Options) ([]lint.Unit, error) {
	if !lint.NeedsEnumeration(target.Rule.Granularity) {
		return lint.UnitsAt(target.Rule.Granularity, target.File, nil), nil
	}
	found, err := leaves.leavesOf(ctx, ask, target)
	if err != nil {
		return nil, err
	}
	return lint.UnitsAt(target.Rule.Granularity, target.File, found), nil
}

// withError carries the failure on the report itself, so a target that could not
// be judged is still printed and still serialised alongside the ones that were.
func withError(r lint.Report, err error) lint.Report {
	r.Error = err.Error()
	if r.Verdicts == nil {
		r.Verdicts = map[string]int{}
	}
	return r
}

// leafCache enumerates each file once however many rules judge it. The entry is
// claimed under the lock and filled under its own Once, so rules that start
// together on one file wait on a single call instead of racing into several — a
// check-then-call would ask the same question once per rule.
type leafCache struct {
	mu      sync.Mutex
	entries map[string]*leafEntry
}

// leafEntry holds the answer for one file, error included: an enumeration that
// failed is the answer for every rule over that file, not a call to retry.
type leafEntry struct {
	once   sync.Once
	leaves []string
	err    error
}

func newLeafCache() *leafCache {
	return &leafCache{entries: map[string]*leafEntry{}}
}

func (c *leafCache) leavesOf(ctx context.Context, ask service.Ask, target lint.Options) ([]string, error) {
	entry := c.entryFor(target.File)
	entry.once.Do(func() {
		entry.leaves, entry.err = lint.Enumerate(ctx, ask, target)
	})
	return entry.leaves, entry.err
}

func (c *leafCache) entryFor(file string) *leafEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, claimed := c.entries[file]
	if !claimed {
		entry = &leafEntry{}
		c.entries[file] = entry
	}
	return entry
}

func writeResult(b *strings.Builder, result Result, colour bool) error {
	var rendered strings.Builder
	if err := lint.Format(&rendered, result.Report, colour); err != nil {
		return err
	}
	fmt.Fprintf(b, "  %s  %s\n", result.Report.Rule, selftest.FormatDuration(result.Duration))
	b.WriteString(indent(bodyOf(rendered.String()), "  "))
	b.WriteString("\n\n")
	return nil
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
