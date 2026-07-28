package main

import (
	"context"
	"fmt"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/run"
	"github.com/matthijn/aritu/internal/lib/service"
)

func runApply(ctx context.Context, cli *CLI, ask service.Ask, out streams) lint.Exit {
	started := time.Now()
	opts, setupErr := applyOptions(cli)
	report := reporterFor(cli.Output, out.stdout, wantsColour(out.stdout))
	if cli.Debug {
		report = run.Silent()
	}

	var results []run.Result
	if setupErr == nil {
		if !cli.Debug {
			run.Announce(out.stderr, opts)
		}
		opts.Observe = report.Observe
		results = run.Run(ctx, ask, opts)
	}
	if err := report.Finish(run.Outcome{Results: results, Options: opts, Elapsed: time.Since(started)}); err != nil {
		fmt.Fprintf(out.stderr, "aritu apply: %v\n", err)
		return lint.ExitError
	}
	if setupErr != nil {
		fmt.Fprintf(out.stderr, "aritu apply: %v\n", setupErr)
		return lint.ExitError
	}
	return run.ExitFor(results)
}
