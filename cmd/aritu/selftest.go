package main

import (
	"context"
	"fmt"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/domain/selftest"
	"github.com/matthijn/aritu/internal/lib/service"
)

func runSelftest(ctx context.Context, cli *CLI, ask service.Ask, out streams) lint.Exit {
	known, err := knownTargetsFor(cli)
	if err != nil {
		fmt.Fprintf(out.stderr, "aritu selftest: %v\n", err)
		return lint.ExitError
	}
	names, err := ruleNamesFor(cli)
	if err != nil {
		fmt.Fprintf(out.stderr, "aritu selftest: %v\n", err)
		return lint.ExitError
	}

	return selftestRun{ask: ask, cli: cli, known: known, out: out}.rules(ctx, names)
}

type selftestRun struct {
	ask   service.Ask
	cli   *CLI
	known []string
	out   streams
}

func (s selftestRun) rules(ctx context.Context, names []string) lint.Exit {
	exit := lint.ExitPass
	for _, name := range names {
		exit = worse(exit, s.rule(ctx, name))
	}
	return exit
}

func (s selftestRun) rule(ctx context.Context, name string) lint.Exit {
	started := time.Now()
	opts, results, runErr := s.results(ctx, name)

	if err := selftest.Format(s.out.stdout, opts, results, time.Since(started)); err != nil {
		fmt.Fprintf(s.out.stderr, "aritu selftest: %v\n", err)
		return lint.ExitError
	}
	if runErr != nil {
		fmt.Fprintf(s.out.stderr, "aritu selftest: %v\n", runErr)
		return lint.ExitError
	}
	return selftest.ExitFor(results)
}

func (s selftestRun) results(ctx context.Context, name string) (selftest.Options, []selftest.Result, error) {
	opts := selftest.Options{
		Rule:   rule.Rule{Name: name},
		Votes:  s.cli.Votes,
		Model:  s.cli.Model,
		Effort: s.cli.Effort,
	}
	loaded, err := rule.Load(s.cli.Rules, name, s.known)
	if err != nil {
		return opts, nil, err
	}
	opts.Rule = loaded

	fixtures, err := rule.LoadFixtures(loaded)
	if err != nil {
		return opts, nil, err
	}
	return opts, selftest.Run(ctx, s.ask, opts, fixtures), nil
}

func worse(a, b lint.Exit) lint.Exit {
	return max(a, b)
}
