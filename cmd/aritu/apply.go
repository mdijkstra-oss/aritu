package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/run"
	"github.com/matthijn/aritu/internal/lib/service"
)

func runApply(ctx context.Context, cli *CLI, ask service.Ask, stdout, stderr io.Writer) lint.Exit {
	started := time.Now()
	opts, setupErr := applyOptions(cli)
	report := reporterFor(cli.Output, stdout, wantsColour(stdout))
	if cli.Debug {
		report = silentReporter()
	}

	var results []run.Result
	if setupErr == nil {
		if !cli.Debug {
			run.Announce(stderr, opts)
		}
		opts.Observe = report.observe
		results = run.Run(ctx, ask, opts)
	}
	if err := report.finish(sweep{Results: results, Options: opts, Elapsed: time.Since(started)}); err != nil {
		fmt.Fprintf(stderr, "aritu apply: %v\n", err)
		return lint.ExitError
	}
	if setupErr != nil {
		fmt.Fprintf(stderr, "aritu apply: %v\n", setupErr)
		return lint.ExitError
	}
	return run.ExitFor(results)
}
