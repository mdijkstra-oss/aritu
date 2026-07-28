package main

import (
	"context"
	"fmt"
	"time"

	"github.com/matthijn/aritu/internal/domain/audit"
	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/lib/service"
)

func runApply(ctx context.Context, resolved settings, ask service.Ask, out streams) lint.Exit {
	started := time.Now()
	opts, setupErr := applyOptions(resolved)
	report := reporterFor(resolved.Format, out.stdout, wantsColour(out.stdout))
	if resolved.Debug {
		report = audit.Silent()
	}

	var results []audit.Result
	if setupErr == nil {
		if !resolved.Debug {
			audit.Announce(out.stderr, opts)
		}
		opts.Observe = report.Observe
		results = audit.Run(ctx, ask, opts)
	}
	if err := report.Finish(audit.Outcome{Results: results, Options: opts, Elapsed: time.Since(started)}); err != nil {
		fmt.Fprintf(out.stderr, "aritu apply: %v\n", err)
		return lint.ExitError
	}
	if setupErr != nil {
		fmt.Fprintf(out.stderr, "aritu apply: %v\n", setupErr)
		return lint.ExitError
	}
	return audit.ExitFor(results)
}
