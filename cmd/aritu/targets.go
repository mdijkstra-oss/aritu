package main

import (
	"fmt"
	"os"

	"github.com/matthijn/aritu/internal/domain/run"
	"github.com/matthijn/aritu/internal/domain/sweep"
	"github.com/matthijn/aritu/internal/lib/glob"
)

func applyOptions(cli *CLI) (run.Options, error) {
	opts := run.Options{Votes: cli.Votes, Model: cli.Model, Effort: cli.Effort}

	dir, err := workingDir()
	if err != nil {
		return opts, err
	}
	kinds, err := sweep.Kinds(cli.Loaded, dir)
	if err != nil {
		return opts, err
	}
	rules, err := rulesFor(cli, kinds.Names())
	if err != nil {
		return opts, err
	}
	opts.Rules = rules

	resolved, err := sweep.Resolve(sweep.Request{
		Patterns: cli.Apply.Patterns,
		Rules:    rules,
		Kinds:    kinds,
		Dir:      dir,
		RulesDir: glob.Rooted(dir, cli.Rules),
	})
	opts.Files = resolved.Files
	opts.IsTargeted = resolved.IsTargeted
	return opts, err
}

func knownTargetsFor(cli *CLI) ([]string, error) {
	dir, err := workingDir()
	if err != nil {
		return nil, err
	}
	kinds, err := sweep.Kinds(cli.Loaded, dir)
	if err != nil {
		return nil, err
	}
	return kinds.Names(), nil
}

func workingDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("working directory: %w", err)
	}
	return dir, nil
}
