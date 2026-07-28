package main

import (
	"github.com/matthijn/aritu/internal/domain/rule"
)

func rulesFor(cli *CLI, known []string) ([]rule.Rule, error) {
	names, err := ruleNamesFor(cli)
	if err != nil {
		return nil, err
	}
	return rule.LoadAll(cli.Rules, names, known)
}

func ruleNamesFor(cli *CLI) ([]string, error) {
	return rule.Names(cli.Rules, cli.Rule, cli.Loaded.Rules.Enabled)
}
