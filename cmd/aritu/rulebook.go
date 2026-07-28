package main

import (
	"context"
	"fmt"
	"io"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
)

func runRulebook(_ context.Context, resolved settings, out streams) lint.Exit {
	if err := writeRulebook(resolved, out.stdout); err != nil {
		fmt.Fprintf(out.stderr, "aritu rulebook: %v\n", err)
		return lint.ExitError
	}
	return lint.ExitPass
}

func writeRulebook(resolved settings, w io.Writer) error {
	known, err := knownTargetsFor(resolved)
	if err != nil {
		return err
	}
	rules, err := rulesFor(resolved, known)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, rule.Rulebook(rules))
	return err
}
