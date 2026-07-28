package main

import (
	"github.com/matthijn/aritu/internal/domain/rule"
)

func rulesFor(cli *CLI, known []string) ([]rule.Rule, error) {
	names, err := ruleNamesFor(cli)
	if err != nil {
		return nil, err
	}
	rules := make([]rule.Rule, 0, len(names))
	for _, name := range names {
		loaded, err := rule.Load(cli.Rules, name, known)
		if err != nil {
			return nil, err
		}
		rules = append(rules, loaded)
	}
	return rules, nil
}

func ruleNamesFor(cli *CLI) ([]string, error) {
	if len(cli.Rule) > 0 {
		return cli.Rule, nil
	}
	if enabled := cli.Loaded.Rules.Enabled; len(enabled) > 0 {
		return enabled, nil
	}
	return rule.List(cli.Rules)
}

func knownTargetsFor(cli *CLI) ([]string, error) {
	dir, err := workingDir()
	if err != nil {
		return nil, err
	}
	kinds, err := kindsFor(cli.Loaded, dir)
	if err != nil {
		return nil, err
	}
	return kinds.Names(), nil
}
