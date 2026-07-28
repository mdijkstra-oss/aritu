package main

import (
	"github.com/matthijn/aritu/internal/domain/rule"
)

func rulesFor(resolved settings, known []string) ([]rule.Rule, error) {
	names, err := ruleNamesFor(resolved)
	if err != nil {
		return nil, err
	}
	return rule.LoadAll(resolved.RulesDir, names, known)
}

func ruleNamesFor(resolved settings) ([]string, error) {
	return rule.Names(rule.Selection{
		RulesDir: resolved.RulesDir,
		Explicit: resolved.Rule,
		Enabled:  resolved.Config.Rules.Enabled,
	})
}
