package main

import (
	"context"
	"fmt"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/selftest"
	"github.com/matthijn/aritu/internal/lib/service"
)

func runSelftest(ctx context.Context, resolved settings, ask service.Ask, out streams) lint.Exit {
	known, err := knownTargetsFor(resolved)
	if err != nil {
		fmt.Fprintf(out.stderr, "aritu selftest: %v\n", err)
		return lint.ExitError
	}
	names, err := ruleNamesFor(resolved)
	if err != nil {
		fmt.Fprintf(out.stderr, "aritu selftest: %v\n", err)
		return lint.ExitError
	}

	return selftestRun{ask: ask, settings: resolved, known: known, out: out}.rules(ctx, names)
}

type selftestRun struct {
	ask      service.Ask
	settings settings
	known    []string
	out      streams
}

func (r selftestRun) rules(ctx context.Context, names []string) lint.Exit {
	exit := lint.ExitPass
	for _, name := range names {
		exit = worse(exit, r.rule(ctx, name))
	}
	return exit
}

func (r selftestRun) rule(ctx context.Context, name string) lint.Exit {
	started := time.Now()
	opts, results, runErr := selftest.Judge(ctx, r.ask, r.specFor(name))

	if err := selftest.Format(r.out.stdout, opts, results, time.Since(started)); err != nil {
		fmt.Fprintf(r.out.stderr, "aritu selftest: %v\n", err)
		return lint.ExitError
	}
	if runErr != nil {
		fmt.Fprintf(r.out.stderr, "aritu selftest: %v\n", runErr)
		return lint.ExitError
	}
	return selftest.ExitFor(results)
}

func (r selftestRun) specFor(name string) selftest.Spec {
	return selftest.Spec{
		Name:     name,
		RulesDir: r.settings.RulesDir,
		Known:    r.known,
		Votes:    r.settings.Votes,
		Model:    r.settings.Model,
		Effort:   r.settings.Effort,
	}
}

func worse(a, b lint.Exit) lint.Exit {
	return max(a, b)
}
