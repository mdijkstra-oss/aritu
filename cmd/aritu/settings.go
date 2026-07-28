package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/matthijn/aritu/internal/domain/config"
)

type settings struct {
	Config   config.Config
	Patterns []string
	Rule     []string
	RulesDir string
	Format   string
	Model    string
	Effort   string
	Votes    int
	Parallel int
	Timeout  time.Duration
	Debug    bool
}

const (
	defaultModel  = "sonnet"
	defaultEffort = "medium"
)

func settingsFrom(cli CLI) settings {
	return settings{
		Config:   cli.Config,
		Patterns: cli.Apply.Patterns,
		Rule:     cli.Rule,
		RulesDir: cli.RulesDir,
		Format:   cli.Format,
		Model:    stringOr(cli.Config.Service.Model, defaultModel),
		Effort:   stringOr(cli.Config.Service.Effort, defaultEffort),
		Votes:    cli.Votes,
		Parallel: cli.Parallel,
		Timeout:  cli.Timeout,
		Debug:    cli.Debug,
	}
}

func stringOr(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}

func validate(resolved settings) error {
	if resolved.Votes < 1 {
		return fmt.Errorf("votes must be at least 1, got %d", resolved.Votes)
	}
	if _, isKnown := reporters[resolved.Format]; !isKnown {
		return fmt.Errorf("unknown output %q, want pretty or json", resolved.Format)
	}
	if !isKnownEffort(resolved.Effort) {
		return fmt.Errorf("unknown effort %q, want one of %s, or empty for the endpoint default", resolved.Effort, strings.Join(efforts, ", "))
	}
	return nil
}

var efforts = []string{"minimal", "low", "medium", "high", "xhigh", "max"}

func isKnownEffort(effort string) bool {
	return effort == "" || slices.Contains(efforts, effort)
}
