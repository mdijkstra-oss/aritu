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
	"github.com/matthijn/aritu/internal/lib/claudecli"
)

// Options configures one multi-target run.
type Options struct {
	Rules  []rule.Rule
	Base   string
	Files  []string
	Votes  int
	Model  string
	Effort string
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
func Run(ctx context.Context, ask claudecli.Ask, opts Options) []Result {
	results := make([]Result, len(opts.Files)*len(opts.Rules))
	leaves := newLeafCache()
	var wg sync.WaitGroup
	for f, file := range opts.Files {
		for r, judged := range opts.Rules {
			at := f*len(opts.Rules) + r
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[at] = judge(ctx, ask, leaves, targetFor(opts, judged, file))
			}()
		}
	}
	wg.Wait()
	return results
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
func Format(w io.Writer, results []Result, opts Options, elapsed time.Duration, colour bool) error {
	var b strings.Builder
	for _, group := range groupsOf(results) {
		b.WriteString(group.File + "\n")
		for _, result := range group.Results {
			if err := writeResult(&b, result, colour); err != nil {
				return err
			}
		}
	}
	writeSummary(&b, results, opts, elapsed)

	_, err := io.WriteString(w, b.String())
	return err
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
		Base:   opts.Base,
		File:   file,
		Votes:  opts.Votes,
		Model:  opts.Model,
		Effort: opts.Effort,
	}
}

// judge times one target from the caller's point of view, so a rule that waited on
// another rule's enumeration of the same file reports the wait it actually took.
func judge(ctx context.Context, ask claudecli.Ask, leaves *leafCache, target lint.Options) Result {
	started := time.Now()
	report, err := reportFor(ctx, ask, leaves, target)
	if err != nil {
		report = withError(report, err)
	}
	return Result{Report: report, Duration: time.Since(started), Err: err}
}

func reportFor(ctx context.Context, ask claudecli.Ask, leaves *leafCache, target lint.Options) (lint.Report, error) {
	units, err := unitsFor(ctx, ask, leaves, target)
	if err != nil {
		return lint.Report{Rule: target.Rule.Name, File: target.File, Votes: target.Votes}, err
	}
	return lint.Judge(ctx, ask, target, units)
}

// unitsFor consults the cache only for a rule whose units the model has to list.
// A file-granularity rule judges the path, so asking would spend a call on an
// answer already in hand.
func unitsFor(ctx context.Context, ask claudecli.Ask, leaves *leafCache, target lint.Options) ([]lint.Unit, error) {
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

func (c *leafCache) leavesOf(ctx context.Context, ask claudecli.Ask, target lint.Options) ([]string, error) {
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

type fileGroup struct {
	File    string
	Results []Result
}

// groupsOf splits the results into consecutive runs of one file. Run already
// orders them by file, so grouping never reorders what the caller was handed.
func groupsOf(results []Result) []fileGroup {
	groups := make([]fileGroup, 0, len(results))
	for _, result := range results {
		if startsNewFile(groups, result.Report.File) {
			groups = append(groups, fileGroup{File: result.Report.File})
		}
		last := &groups[len(groups)-1]
		last.Results = append(last.Results, result)
	}
	return groups
}

func startsNewFile(groups []fileGroup, file string) bool {
	return len(groups) == 0 || groups[len(groups)-1].File != file
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
