package audit

import (
	"context"
	"sync"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/service"
)

// Options configures one multi-target run.
type Options struct {
	Rules  []rule.Rule
	Files  []string
	Votes  int
	Model  string
	Effort string

	// IsTargeted reports whether a rule is about a file, so that a rule and a file
	// it has nothing to say about form no target at all. A run naming no predicate
	// pairs everything with everything, which is what a caller handing over two
	// lists it has already narrowed means.
	IsTargeted func(judged rule.Rule, file string) bool

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

// Run judges every file against every rule that is about it. Results come back
// ordered by file then rule however the calls interleave, and a target that errors
// is recorded rather than aborting the run, so one unreachable file cannot hide the
// rest.
//
// Each file is enumerated once per granularity, however many rules judge it at
// that level. How many calls run at once is bounded by the ask, not here.
func Run(ctx context.Context, ask service.Ask, opts Options) []Result {
	targets := targetsOf(opts)
	results := make([]Result, len(targets))
	leaves := newLeafCache()
	landed := make(chan int, len(results))
	var wg sync.WaitGroup
	for at, target := range targets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[at] = judge(ctx, ask, leaves, target)
			landed <- at
		}()
	}
	go func() {
		wg.Wait()
		close(landed)
	}()
	observeInOrder(landed, results, orNoop(opts.Observe))
	return results
}

// targetsOf pairs each file with the rules that are about it, walking files in
// order and rules in order. That is the reading order Format prints in, with the
// pairs nobody asked for left out rather than judged.
func targetsOf(opts Options) []lint.Options {
	isTargeted := orEveryPair(opts.IsTargeted)
	targets := make([]lint.Options, 0, len(opts.Files)*len(opts.Rules))
	for _, file := range opts.Files {
		for _, judged := range opts.Rules {
			if isTargeted(judged, file) {
				targets = append(targets, targetFor(opts, judged, file))
			}
		}
	}
	return targets
}

// orEveryPair keeps the pairing loop free of a nil check per pair, the way orNoop
// keeps the ordering loop free of one per target.
func orEveryPair(isTargeted func(rule.Rule, string) bool) func(rule.Rule, string) bool {
	if isTargeted != nil {
		return isTargeted
	}
	return func(rule.Rule, string) bool { return true }
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
		return lint.ReportFor(target), err
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

// leafCache enumerates each file once per granularity however many rules judge
// it there. The entry is claimed under the lock and filled under its own Once, so
// rules that start together on one file wait on a single call instead of racing
// into several — a check-then-call would ask the same question once per rule.
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
	entry := c.entryFor(enumerationKey(target))
	entry.once.Do(func() {
		entry.leaves, entry.err = lint.Enumerate(ctx, ask, target)
	})
	return entry.leaves, entry.err
}

func (c *leafCache) entryFor(key string) *leafEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, claimed := c.entries[key]
	if !claimed {
		entry = &leafEntry{}
		c.entries[key] = entry
	}
	return entry
}

// enumerationKey is the file and the granularity the splitter prompt was built
// from. Two rules over one file share an answer only when they asked the same
// question, and a rule at a different granularity is asking for a different kind
// of unit.
func enumerationKey(target lint.Options) string {
	return target.File + "\x00" + target.Rule.Granularity.String()
}
