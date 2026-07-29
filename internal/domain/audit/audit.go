package audit

import (
	"context"
	"sync"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/service"
)

type Options struct {
	Rules  []rule.Rule
	Files  []string
	Votes  int
	Model  string
	Effort string

	// A nil IsTargeted pairs every rule with every file.
	IsTargeted func(judged rule.Rule, file string) bool

	// Calls are serial and in the order Format prints, so an implementation
	// writing to a terminal needs no lock of its own.
	Observe func(Result)
}

type Result struct {
	Report   lint.Report
	Duration time.Duration
	Err      error
}

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

func orEveryPair(isTargeted func(rule.Rule, string) bool) func(rule.Rule, string) bool {
	if isTargeted != nil {
		return isTargeted
	}
	return func(rule.Rule, string) bool { return true }
}

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

func orNoop(observe func(Result)) func(Result) {
	if observe != nil {
		return observe
	}
	return func(Result) {}
}

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

func withError(r lint.Report, err error) lint.Report {
	r.Error = err.Error()
	if r.Verdicts == nil {
		r.Verdicts = map[string]int{}
	}
	return r
}

// leafCache is safe for concurrent use, and enumerates a key once however many
// rules ask, a failed enumeration included.
type leafCache struct {
	mu      sync.Mutex
	entries map[enumerationKey]*leafEntry
}

type enumerationKey struct {
	file        string
	granularity rule.Granularity
}

type leafEntry struct {
	once   sync.Once
	leaves []string
	err    error
}

func newLeafCache() *leafCache {
	return &leafCache{entries: map[enumerationKey]*leafEntry{}}
}

func (c *leafCache) leavesOf(ctx context.Context, ask service.Ask, target lint.Options) ([]string, error) {
	entry := c.entryFor(keyOf(target))
	entry.once.Do(func() {
		entry.leaves, entry.err = lint.Enumerate(ctx, ask, target)
	})
	return entry.leaves, entry.err
}

func (c *leafCache) entryFor(key enumerationKey) *leafEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, claimed := c.entries[key]
	if !claimed {
		entry = &leafEntry{}
		c.entries[key] = entry
	}
	return entry
}

func keyOf(target lint.Options) enumerationKey {
	return enumerationKey{file: target.File, granularity: target.Rule.Granularity}
}
