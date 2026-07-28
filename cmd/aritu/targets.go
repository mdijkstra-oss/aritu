package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/matthijn/aritu/internal/domain/config"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/domain/run"
	"github.com/matthijn/aritu/internal/lib/glob"
	"github.com/matthijn/aritu/internal/lib/kind"
)

func applyOptions(cli *CLI) (run.Options, error) {
	opts := run.Options{Votes: cli.Votes, Model: cli.Model, Effort: cli.Effort}

	dir, err := workingDir()
	if err != nil {
		return opts, err
	}
	kinds, err := kindsFor(cli.Loaded, dir)
	if err != nil {
		return opts, err
	}
	rules, err := rulesFor(cli, kinds.Names())
	if err != nil {
		return opts, err
	}
	opts.Rules = rules
	opts.IsTargeted = targetingBy(kinds, dir)

	files, err := filesFor(cli.Apply.Patterns, derivedSweep{
		kinds:    kinds,
		targeted: targetedKindsOf(rules),
		rulesDir: glob.Rooted(dir, cli.Rules),
	})
	if err != nil {
		return opts, err
	}
	opts.Files = files
	return opts, checkEveryFileIsTargeted(files, rules, opts.IsTargeted)
}

func filesFor(patterns []string, derived derivedSweep) ([]string, error) {
	if len(patterns) > 0 {
		return glob.Expand(patterns)
	}
	return derived.files()
}

type derivedSweep struct {
	kinds    kind.Set
	targeted []string
	rulesDir string
}

func (d derivedSweep) files() ([]string, error) {
	found, err := d.kinds.Expand(d.targeted)
	if err != nil {
		return nil, err
	}
	files := filterOutsideRules(found, d.rulesDir)
	if len(files) == 0 {
		return nil, fmt.Errorf("no targets: nothing here is %s, so name a file or glob pattern",
			strings.Join(d.targeted, " or "))
	}
	return files, nil
}

func filterOutsideRules(files []string, rulesDir string) []string {
	outside := make([]string, 0, len(files))
	for _, file := range files {
		if !isUnder(rulesDir, file) {
			outside = append(outside, file)
		}
	}
	return outside
}

func isUnder(dir, path string) bool {
	return strings.HasPrefix(path, dir+string(filepath.Separator))
}

func kindsFor(loaded config.Config, dir string) (kind.Set, error) {
	return kind.Resolve(repositoryDir(loaded, dir), loaded.Targets)
}

func repositoryDir(loaded config.Config, dir string) string {
	if loaded.Dir == "" {
		return dir
	}
	return glob.Rooted(dir, loaded.Dir)
}

func targetingBy(kinds kind.Set, dir string) func(rule.Rule, string) bool {
	return func(judged rule.Rule, file string) bool {
		return kinds.Covers(judged.Targets, glob.Rooted(dir, file))
	}
}

func targetedKindsOf(rules []rule.Rule) []string {
	targeted := make([]string, 0, len(rules))
	for _, judged := range rules {
		for _, name := range judged.Targets {
			if !slices.Contains(targeted, name) {
				targeted = append(targeted, name)
			}
		}
	}
	slices.Sort(targeted)
	return targeted
}

func checkEveryFileIsTargeted(files []string, rules []rule.Rule, isTargeted func(rule.Rule, string) bool) error {
	untargeted := make([]string, 0, len(files))
	for _, file := range files {
		if !isTargetedByAny(rules, file, isTargeted) {
			untargeted = append(untargeted, file)
		}
	}
	if len(untargeted) == 0 {
		return nil
	}
	return fmt.Errorf("no enabled rule targets %s", strings.Join(untargeted, ", "))
}

func isTargetedByAny(rules []rule.Rule, file string, isTargeted func(rule.Rule, string) bool) bool {
	return slices.ContainsFunc(rules, func(judged rule.Rule) bool { return isTargeted(judged, file) })
}

func workingDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("working directory: %w", err)
	}
	return dir, nil
}
