package selftest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/service"
	"github.com/matthijn/aritu/internal/lib/vote"
)

type Result struct {
	Fixture  rule.Fixture
	Report   lint.Report
	Held     bool
	Err      error
	Duration time.Duration
}

type Options struct {
	Rule   rule.Rule
	Votes  int
	Model  string
	Effort string
}

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

func judge(ctx context.Context, ask service.Ask, opts Options, fixture rule.Fixture) Result {
	started := time.Now()
	report, err := lint.Apply(ctx, ask, lint.Options{
		Rule:   opts.Rule,
		File:   fixture.File,
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
