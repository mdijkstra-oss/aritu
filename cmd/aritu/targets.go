package main

import (
	"fmt"
	"os"

	"github.com/matthijn/aritu/internal/domain/audit"
	"github.com/matthijn/aritu/internal/domain/sweep"
	"github.com/matthijn/aritu/internal/lib/glob"
)

func applyOptions(resolved settings) (audit.Options, error) {
	opts := audit.Options{Votes: resolved.Votes, Model: resolved.Model, Effort: resolved.Effort}

	dir, err := workingDir()
	if err != nil {
		return opts, err
	}
	kinds, err := sweep.Kinds(resolved.Config, dir)
	if err != nil {
		return opts, err
	}
	rules, err := rulesFor(resolved, kinds.Names())
	if err != nil {
		return opts, err
	}
	opts.Rules = rules

	swept, err := sweep.Resolve(sweep.Request{
		Patterns: resolved.Patterns,
		Rules:    rules,
		Kinds:    kinds,
		Dir:      dir,
		RulesDir: glob.Rooted(dir, resolved.RulesDir),
	})
	opts.Files = swept.Files
	opts.IsTargeted = swept.IsTargeted
	return opts, err
}

func knownTargetsFor(resolved settings) ([]string, error) {
	dir, err := workingDir()
	if err != nil {
		return nil, err
	}
	kinds, err := sweep.Kinds(resolved.Config, dir)
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
