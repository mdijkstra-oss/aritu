package selftest

import (
	"context"

	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/service"
)

type Spec struct {
	Name     string
	RulesDir string
	Known    []string
	Votes    int
	Model    string
	Effort   string
}

// Judge loads the named rule with its fixtures and runs them. The options come
// back even where the load failed, so a caller can report which rule it was
// about to judge.
func Judge(ctx context.Context, ask service.Ask, spec Spec) (Options, []Result, error) {
	opts := Options{
		Rule:   rule.Rule{Name: spec.Name},
		Votes:  spec.Votes,
		Model:  spec.Model,
		Effort: spec.Effort,
	}
	loaded, err := rule.Load(spec.RulesDir, spec.Name, spec.Known)
	if err != nil {
		return opts, nil, err
	}
	opts.Rule = loaded

	fixtures, err := rule.LoadFixtures(loaded)
	if err != nil {
		return opts, nil, err
	}
	return opts, Run(ctx, ask, opts, fixtures), nil
}
